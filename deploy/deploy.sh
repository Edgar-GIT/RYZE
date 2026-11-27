#!/usr/bin/env bash
#
# Deploys one RYZE API image to one environment on the VPS.
#
#   /opt/ryze/deploy/deploy.sh <production|staging> <image-tag>
#
# Called by GitHub Actions over SSH (user "deploy"), or by hand. Steps:
#   1. pull the new image (nothing changes if this fails);
#   2. back up the database and uploads (backup.sh);
#   3. apply pending migrations with the NEW image (migrations are embedded
#      in the binary; "migrate up" does nothing when none are pending);
#   4. replace the API container with the new image;
#   5. check GET /api/v1/health; if it fails, go back to the previous image.
#
# A rollback restores the previous API image only. It never undoes a
# migration: migrations must stay backward compatible (add, do not rename or
# drop, in the same release). Restoring data is a manual decision
# (production.MD, "Situações de emergência").
#
# The previous and current tags are kept in <env>/release.env and every run is
# appended to <env>/deployments.log.
set -euo pipefail
umask 077

ENVIRONMENT="${1:-}"
TAG="${2:-}"
case "${ENVIRONMENT}" in
  production) SUBNET="172.31.10.0/24" ;;
  staging) SUBNET="172.31.20.0/24" ;;
  *) echo "usage: $0 <production|staging> <image-tag>" >&2; exit 2 ;;
esac
if [[ ! "${TAG}" =~ ^[A-Za-z0-9._-]{1,128}$ ]]; then
  echo "error: invalid image tag" >&2
  exit 2
fi

ROOT=/opt/ryze
DIR="${ROOT}/${ENVIRONMENT}"
COMPOSE_FILE="${ROOT}/deploy/compose.app.yml"
IMAGE_REPO="$(cat "${ROOT}/image-repository")"   # e.g. ghcr.io/owner/ryze-api
NEW_IMAGE="${IMAGE_REPO}:${TAG}"
RELEASE="${DIR}/release.env"
LOG="${DIR}/deployments.log"

for required in "${DIR}/app.env" "${DIR}/db.env" "${COMPOSE_FILE}"; do
  [[ -f "${required}" ]] || { echo "error: missing ${required}" >&2; exit 1; }
done

previous_image=""
if [[ -f "${RELEASE}" ]]; then
  previous_image="$(grep -E '^RYZE_IMAGE=' "${RELEASE}" | cut -d= -f2- || true)"
fi

write_release() {
  {
    echo "RYZE_ENV=${ENVIRONMENT}"
    echo "RYZE_IMAGE=$1"
    echo "RYZE_SUBNET=${SUBNET}"
    echo "RYZE_PREVIOUS_IMAGE=$2"
    echo "RYZE_DEPLOYED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "${RELEASE}.tmp"
  mv "${RELEASE}.tmp" "${RELEASE}"
}

compose() {
  docker compose --project-directory "${DIR}" --env-file "${RELEASE}" -f "${COMPOSE_FILE}" "$@"
}

health() {
  # Up to 60 s for the API to answer with the database reachable.
  for _ in $(seq 1 30); do
    if docker run --rm --network "ryze-${ENVIRONMENT}-net" curlimages/curl:8.11.1 \
         -fsS --max-time 3 "http://ryze-${ENVIRONMENT}-backend:8080/api/v1/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

log() { echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $*" | tee -a "${LOG}"; }

log "deploy ${ENVIRONMENT}: ${previous_image:-<none>} -> ${NEW_IMAGE}"

# 1. Pull first: a missing image or registry error leaves everything as it is.
docker pull "${NEW_IMAGE}"

# Prepare the release file with the new image, keeping the previous one.
write_release "${NEW_IMAGE}" "${previous_image}"

# The database must run before backups and migrations (first deploy included).
compose up -d db

# 2. Back up before touching the schema.
if ! "${ROOT}/deploy/backup.sh" "${ENVIRONMENT}" pre-deploy; then
  log "backup failed: deploy stopped before any change"
  if [[ -n "${previous_image}" ]]; then write_release "${previous_image}" ""; fi
  exit 1
fi

# 3. Migrations with the new image.
log "schema before: $(compose run --rm --no-deps backend migrate version 2>&1 | tail -1)"
if ! compose run --rm --no-deps backend migrate up; then
  log "migration FAILED: API left on ${previous_image:-<none>}; see production.MD (migration failed)"
  if [[ -n "${previous_image}" ]]; then write_release "${previous_image}" ""; fi
  exit 1
fi
log "schema after: $(compose run --rm --no-deps backend migrate version 2>&1 | tail -1)"

# 4. Replace the API container.
compose up -d --no-deps backend

# 5. Health check, rollback on failure.
if health; then
  log "deploy OK: ${NEW_IMAGE}"
  docker image prune -f --filter "until=168h" >/dev/null 2>&1 || true
  exit 0
fi

log "health check FAILED for ${NEW_IMAGE}"
if [[ -n "${previous_image}" ]]; then
  write_release "${previous_image}" ""
  compose up -d --no-deps backend
  if health; then
    log "rolled back to ${previous_image}"
  else
    log "ROLLBACK ALSO FAILED: manual action needed (production.MD, emergencies)"
  fi
fi
exit 1
