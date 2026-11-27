#!/usr/bin/env bash
#
# Off-site, encrypted backup of one RYZE environment on the VPS.
#
#   /opt/ryze/deploy/backup.sh <production|staging> [daily|pre-deploy]
#
# Produces two files, both encrypted with age for the public key in
# /opt/ryze/backup.env (the private key is NOT on the server):
#   <env>-db-<UTC time>.sql.gz.age       pg_dump of the database
#   <env>-uploads-<UTC time>.tar.gz.age  the uploads and trainer evidence
# and copies them to Cloudflare R2 with rclone:
#   r2:<bucket>/<env>/daily/    every run (pre-deploy runs go to pre-deploy/)
#   r2:<bucket>/<env>/weekly/   on Sundays
#   r2:<bucket>/<env>/monthly/  on the 1st of the month
# Retention (proposal, production.MD section 12): daily 14 days, pre-deploy
# 14 days, weekly 8 weeks, monthly 12 months; local copies 3 days.
#
# /opt/ryze/backup.env (chmod 600) provides:
#   BACKUP_AGE_RECIPIENT     age public key (age1...)
#   BACKUP_R2_REMOTE         rclone remote and bucket, e.g. r2:ryze-backups
#   BACKUP_HEALTHCHECK_URL   optional: pinged on success, /fail on failure
set -euo pipefail
umask 077

ENVIRONMENT="${1:-}"
KIND="${2:-daily}"
case "${ENVIRONMENT}" in production|staging) ;; *) echo "usage: $0 <production|staging> [daily|pre-deploy]" >&2; exit 2 ;; esac
case "${KIND}" in daily|pre-deploy) ;; *) echo "error: kind must be daily or pre-deploy" >&2; exit 2 ;; esac

ROOT=/opt/ryze
DIR="${ROOT}/${ENVIRONMENT}"
# shellcheck disable=SC1091
. "${ROOT}/backup.env"
: "${BACKUP_AGE_RECIPIENT:?set BACKUP_AGE_RECIPIENT in ${ROOT}/backup.env}"
: "${BACKUP_R2_REMOTE:?set BACKUP_R2_REMOTE in ${ROOT}/backup.env}"
HEALTHCHECK="${BACKUP_HEALTHCHECK_URL:-}"

LOCAL="${ROOT}/backups/${ENVIRONMENT}"
mkdir -p "${LOCAL}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DB_FILE="${LOCAL}/${ENVIRONMENT}-db-${STAMP}.sql.gz.age"
UP_FILE="${LOCAL}/${ENVIRONMENT}-uploads-${STAMP}.tar.gz.age"

fail() {
  echo "backup FAILED (${ENVIRONMENT}, ${KIND}): $*" >&2
  rm -f "${DB_FILE}" "${UP_FILE}"
  if [[ -n "${HEALTHCHECK}" ]]; then curl -fsS -m 10 "${HEALTHCHECK}/fail" >/dev/null 2>&1 || true; fi
  exit 1
}
trap 'fail "unexpected error on line ${LINENO}"' ERR

# The database credentials stay inside the db container (db.env).
docker compose --project-directory "${DIR}" --env-file "${DIR}/release.env" -f "${ROOT}/deploy/compose.app.yml" \
  exec -T db sh -c 'pg_dump --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" --format=plain --clean --if-exists --no-owner --no-privileges --encoding=UTF8' \
  | gzip -9 | age --recipient "${BACKUP_AGE_RECIPIENT}" --output "${DB_FILE}"
[[ -s "${DB_FILE}" ]] || fail "empty database dump"

# Uploads and trainer evidence, read-only from the API's volume.
docker run --rm -v "ryze-${ENVIRONMENT}_backend_data:/data:ro" alpine:3.20 tar -C /data -czf - . \
  | age --recipient "${BACKUP_AGE_RECIPIENT}" --output "${UP_FILE}"
[[ -s "${UP_FILE}" ]] || fail "empty uploads archive"

REMOTE="${BACKUP_R2_REMOTE}/${ENVIRONMENT}"
if [[ "${KIND}" == "pre-deploy" ]]; then
  rclone copy "${DB_FILE}" "${REMOTE}/pre-deploy/"
  rclone copy "${UP_FILE}" "${REMOTE}/pre-deploy/"
else
  rclone copy "${DB_FILE}" "${REMOTE}/daily/"
  rclone copy "${UP_FILE}" "${REMOTE}/daily/"
  if [[ "$(date -u +%u)" == "7" ]]; then
    rclone copy "${DB_FILE}" "${REMOTE}/weekly/"
    rclone copy "${UP_FILE}" "${REMOTE}/weekly/"
  fi
  if [[ "$(date -u +%d)" == "01" ]]; then
    rclone copy "${DB_FILE}" "${REMOTE}/monthly/"
    rclone copy "${UP_FILE}" "${REMOTE}/monthly/"
  fi
fi

# Retention (only files older than the limits are removed).
rclone delete --min-age 14d "${REMOTE}/daily/" || true
rclone delete --min-age 14d "${REMOTE}/pre-deploy/" || true
rclone delete --min-age 56d "${REMOTE}/weekly/" || true
rclone delete --min-age 365d "${REMOTE}/monthly/" || true
find "${LOCAL}" -type f -name '*.age' -mtime +3 -delete

trap - ERR
if [[ -n "${HEALTHCHECK}" ]]; then curl -fsS -m 10 "${HEALTHCHECK}" >/dev/null 2>&1 || true; fi
echo "backup OK (${ENVIRONMENT}, ${KIND}): $(basename "${DB_FILE}") $(basename "${UP_FILE}")"
