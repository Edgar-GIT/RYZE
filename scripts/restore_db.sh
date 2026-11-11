#!/usr/bin/env bash
#
# Restores the RYZE PostgreSQL database from a snapshot produced by
# scripts/backup_db.sh.
#
# This is destructive: the dump drops and recreates the existing objects in
# DB_NAME. The restore therefore requires an explicit --yes flag. It runs in a
# single transaction and stops at the first error, so a failed restore leaves
# the database as it was.
#
# Usage:
#   scripts/restore_db.sh <backup-file> --yes
#
# The file may be plain SQL, gzip (.gz) or GPG-encrypted (.gpg). Configuration
# is read from the repository .env file (or the environment). Restore into a
# throwaway database first when validating a backup: security.MD requires
# periodic restoration tests.
set -euo pipefail

usage() {
  echo "usage: $0 <backup-file> --yes" >&2
  exit 2
}

if [[ $# -lt 2 ]]; then
  usage
fi

backup_file="$1"
confirm="$2"

if [[ "${confirm}" != "--yes" ]]; then
  echo "refusing to restore without --yes" >&2
  usage
fi

if [[ ! -f "${backup_file}" ]]; then
  echo "error: backup file not found: ${backup_file}" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=env_loader.sh
. "${ROOT_DIR}/scripts/env_loader.sh"
load_project_env "${ROOT_DIR}" DB_HOST DB_PORT DB_NAME DB_USER DB_PASSWORD DB_SSLMODE

: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=5432}"
: "${DB_NAME:?DB_NAME must be set}"
: "${DB_USER:?DB_USER must be set}"
: "${DB_PASSWORD:=}"
: "${DB_SSLMODE:=disable}"

if ! command -v psql >/dev/null 2>&1; then
  echo "error: psql is required" >&2
  exit 1
fi

echo "restoring ${backup_file} into ${DB_HOST}:${DB_PORT}/${DB_NAME}"

export PGPASSWORD="${DB_PASSWORD}"
export PGSSLMODE="${DB_SSLMODE}"

restore() {
  psql --quiet --no-psqlrc -v ON_ERROR_STOP=1 --single-transaction \
    --host="${DB_HOST}" --port="${DB_PORT}" --username="${DB_USER}" --dbname="${DB_NAME}"
}

case "${backup_file}" in
  *.gpg)
    if ! command -v gpg >/dev/null 2>&1; then
      echo "error: gpg is required to decrypt ${backup_file}" >&2
      exit 1
    fi
    gpg --quiet --decrypt "${backup_file}" | gzip -dc | restore
    ;;
  *.gz)
    gzip -dc "${backup_file}" | restore
    ;;
  *)
    restore < "${backup_file}"
    ;;
esac

unset PGPASSWORD PGSSLMODE

echo "restore complete"
