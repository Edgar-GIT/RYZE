#!/usr/bin/env bash
#
# Copies every row of the RYZE MariaDB database into PostgreSQL (one-off data
# step of the MariaDB -> PostgreSQL migration). See
# scripts/mysql_to_postgres/main.go for the guarantees: consistent source
# snapshot, single target transaction, row counts verified before COMMIT.
#
# Prerequisites:
#   1. scripts/postgres_setup.sh has created the role and databases.
#   2. The PostgreSQL schema exists:
#        (cd main/src/backend && DB_PORT=5432 go run . migrate up)
#   3. The backend is stopped, so nothing writes to MariaDB during the copy.
#
# Usage:
#   scripts/migrate_mysql_to_postgres.sh [--source-port 3306] [--target-port 5432]
#                                        [--target-db NAME] [--dry-run]
#
# Host, database, user and password come from the repository .env (DB_*); set
# SOURCE_DB_* or TARGET_DB_* to point one side elsewhere.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=env_loader.sh
. "${ROOT_DIR}/scripts/env_loader.sh"
load_project_env "${ROOT_DIR}" \
  DB_HOST DB_NAME DB_USER DB_PASSWORD DB_SSLMODE \
  SOURCE_DB_HOST SOURCE_DB_NAME SOURCE_DB_USER SOURCE_DB_PASSWORD \
  TARGET_DB_HOST TARGET_DB_NAME TARGET_DB_USER TARGET_DB_PASSWORD TARGET_DB_SSLMODE

cd "${ROOT_DIR}/scripts/mysql_to_postgres"
exec go run . "$@"
