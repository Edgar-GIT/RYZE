#!/usr/bin/env bash
#
# Restores the RYZE database from a snapshot produced by scripts/backup_db.sh.
#
# This is destructive: it overwrites the existing objects in DB_NAME. The
# restore therefore requires an explicit --yes flag.
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

client_bin="$(command -v mariadb || command -v mysql || true)"
if [[ -z "${client_bin}" ]]; then
  echo "error: mariadb or mysql client is required" >&2
  exit 1
fi

echo "restoring ${backup_file} into ${DB_HOST}:${DB_PORT}/${DB_NAME}"

export MYSQL_PWD="${DB_PASSWORD}"

case "${backup_file}" in
  *.gpg)
    if ! command -v gpg >/dev/null 2>&1; then
      echo "error: gpg is required to decrypt ${backup_file}" >&2
      exit 1
    fi
    gpg --quiet --decrypt "${backup_file}" | gzip -dc | "${client_bin}" \
      --host="${DB_HOST}" --port="${DB_PORT}" --user="${DB_USER}" "${DB_NAME}"
    ;;
  *.gz)
    gzip -dc "${backup_file}" | "${client_bin}" \
      --host="${DB_HOST}" --port="${DB_PORT}" --user="${DB_USER}" "${DB_NAME}"
    ;;
  *)
    "${client_bin}" \
      --host="${DB_HOST}" --port="${DB_PORT}" --user="${DB_USER}" "${DB_NAME}" < "${backup_file}"
    ;;
esac

unset MYSQL_PWD

echo "restore complete"
