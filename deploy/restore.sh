#!/usr/bin/env bash
#
# Restores or tests a RYZE backup made by deploy/backup.sh.
#
#   /opt/ryze/deploy/restore.sh test <db-backup.sql.gz.age> <age-identity-file>
#       Restores the dump into a THROWAWAY PostgreSQL container, counts the
#       users and purchases, and deletes the container. Touches nothing else.
#       Run it every month (restore test).
#
#   /opt/ryze/deploy/restore.sh database <production|staging> <db-backup> <age-identity-file> --yes
#       Replaces that environment's database with the backup. DESTRUCTIVE:
#       the API is stopped first, then the dump (which drops and recreates
#       every object) runs in one transaction, then the API starts again.
#
#   /opt/ryze/deploy/restore.sh uploads <production|staging> <uploads-backup.tar.gz.age> <age-identity-file> --yes
#       Extracts the uploads archive over the API volume (existing files with
#       the same names are replaced; others are kept).
#
# The age identity (private key) is never stored on the server: copy it for
# the restore, use it, and delete it (shred) right after.
# Download a backup first:  rclone copy r2:ryze-backups/production/daily/<file> .
set -euo pipefail
umask 077

MODE="${1:-}"
ROOT=/opt/ryze

decrypt() { age --decrypt --identity "$2" "$1"; }

case "${MODE}" in
  test)
    BACKUP="${2:?backup file}"; IDENTITY="${3:?age identity file}"
    NAME="ryze-restore-test-$$"
    PASS="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')"
    docker run -d --name "${NAME}" -e POSTGRES_PASSWORD="${PASS}" -e POSTGRES_USER=restore -e POSTGRES_DB=restore postgres:17 >/dev/null
    trap 'docker rm -f "${NAME}" >/dev/null 2>&1 || true' EXIT
    for _ in $(seq 1 60); do docker exec "${NAME}" pg_isready -U restore -d restore >/dev/null 2>&1 && break; sleep 1; done
    # citext is created by the dump (--clean --if-exists keeps it idempotent).
    decrypt "${BACKUP}" "${IDENTITY}" | gunzip | docker exec -i "${NAME}" psql -v ON_ERROR_STOP=1 --single-transaction -q -U restore -d restore >/dev/null
    echo "restore test OK:"
    docker exec "${NAME}" psql -U restore -d restore -Atc \
      "SELECT 'users=' || COUNT(*) FROM users UNION ALL SELECT 'purchases=' || COUNT(*) FROM purchases UNION ALL SELECT 'schema_version=' || version FROM schema_migrations"
    ;;
  database)
    ENVIRONMENT="${2:?environment}"; BACKUP="${3:?backup file}"; IDENTITY="${4:?age identity file}"
    [[ "${5:-}" == "--yes" ]] || { echo "refusing without --yes (this replaces the ${ENVIRONMENT} database)" >&2; exit 2; }
    DIR="${ROOT}/${ENVIRONMENT}"
    compose() { docker compose --project-directory "${DIR}" --env-file "${DIR}/release.env" -f "${ROOT}/deploy/compose.app.yml" "$@"; }
    compose stop backend
    decrypt "${BACKUP}" "${IDENTITY}" | gunzip | compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 --single-transaction -q --username="$POSTGRES_USER" --dbname="$POSTGRES_DB"'
    compose start backend
    echo "database restored for ${ENVIRONMENT}; check https://<api>/api/v1/health"
    ;;
  uploads)
    ENVIRONMENT="${2:?environment}"; BACKUP="${3:?backup file}"; IDENTITY="${4:?age identity file}"
    [[ "${5:-}" == "--yes" ]] || { echo "refusing without --yes" >&2; exit 2; }
    decrypt "${BACKUP}" "${IDENTITY}" | docker run --rm -i -v "ryze-${ENVIRONMENT}_backend_data:/data" alpine:3.20 \
      sh -c 'tar -C /data -xzf - && chown -R 65532:65532 /data'
    echo "uploads restored for ${ENVIRONMENT}"
    ;;
  *)
    sed -n '3,22p' "$0" >&2
    exit 2
    ;;
esac
