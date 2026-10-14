#!/usr/bin/env bash
#
# Creates a compressed snapshot of the RYZE database.
#
# The dump is transactionally consistent (--single-transaction) and includes
# routines, triggers and events, so a restore reproduces the full schema and the
# platform-owned data, not just the tables.
#
# Configuration is read from the repository .env file (or the environment when
# the variables are already exported). Optional:
#   BACKUP_DIR             output directory (default: <repo>/backups)
#   BACKUP_GPG_RECIPIENT   encrypt with this GPG recipient when set
#   BACKUP_RETENTION_DAYS  delete local backups older than N days (0 = keep all)
#
# security.MD requires backups to be encrypted and kept off the production host:
# this script only produces the local artefact and prints the reminder.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "${ROOT_DIR}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "${ROOT_DIR}/.env"
  set +a
fi

: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=3306}"
: "${DB_NAME:?DB_NAME must be set}"
: "${DB_USER:?DB_USER must be set}"
: "${DB_PASSWORD:=}"

BACKUP_DIR="${BACKUP_DIR:-${ROOT_DIR}/backups}"
GPG_RECIPIENT="${BACKUP_GPG_RECIPIENT:-}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-0}"

dump_bin="$(command -v mariadb-dump || command -v mysqldump || true)"
if [[ -z "${dump_bin}" ]]; then
  echo "error: mariadb-dump or mysqldump is required" >&2
  exit 1
fi

mkdir -p "${BACKUP_DIR}"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="${BACKUP_DIR}/${DB_NAME}-${timestamp}.sql.gz"

# MYSQL_PWD keeps the password out of the process list.
export MYSQL_PWD="${DB_PASSWORD}"

"${dump_bin}" \
  --host="${DB_HOST}" \
  --port="${DB_PORT}" \
  --user="${DB_USER}" \
  --single-transaction \
  --quick \
  --routines \
  --triggers \
  --events \
  --default-character-set=utf8mb4 \
  "${DB_NAME}" | gzip -9 > "${target}"

unset MYSQL_PWD

if [[ -n "${GPG_RECIPIENT}" ]]; then
  if ! command -v gpg >/dev/null 2>&1; then
    echo "error: BACKUP_GPG_RECIPIENT is set but gpg is not installed" >&2
    exit 1
  fi
  gpg --yes --encrypt --recipient "${GPG_RECIPIENT}" "${target}"
  rm -f "${target}"
  target="${target}.gpg"
fi

if [[ "${RETENTION_DAYS}" -gt 0 ]]; then
  find "${BACKUP_DIR}" -maxdepth 1 -type f -name "${DB_NAME}-*.sql.gz*" \
    -mtime "+${RETENTION_DAYS}" -delete
fi

echo "backup written: ${target}"
echo "reminder: copy this file off-host and encrypt it if BACKUP_GPG_RECIPIENT is unset."
