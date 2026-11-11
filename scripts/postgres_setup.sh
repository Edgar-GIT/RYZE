#!/usr/bin/env bash
#
# Creates the RYZE PostgreSQL login role and databases from the repository
# .env file:
#   - the role DB_USER with the password DB_PASSWORD (updated if it exists);
#   - the application database DB_NAME, owned by that role;
#   - an isolated test database (DB_TEST_NAME, default "<DB_NAME>_test") for
#     the backend integration tests, so they never touch DB_NAME.
#
# Run it on the database host by a user allowed to use sudo; psql runs as the
# "postgres" superuser. The values travel on psql's standard input, never on a
# command line, so they do not show up in the process list. Re-running it is
# safe: existing databases are kept.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck source=env_loader.sh
. "${ROOT_DIR}/scripts/env_loader.sh"
load_project_env "${ROOT_DIR}" DB_NAME DB_USER DB_PASSWORD DB_TEST_NAME

: "${DB_NAME:?DB_NAME must be set}"
: "${DB_USER:?DB_USER must be set}"
: "${DB_PASSWORD:?DB_PASSWORD must be set: PostgreSQL logins over TCP need a password}"
TEST_DB="${DB_TEST_NAME:-${DB_NAME}_test}"

if ! command -v psql >/dev/null 2>&1; then
  echo "error: psql not found; install PostgreSQL first" >&2
  exit 1
fi

# SQL quoting helpers: literals double single quotes, identifiers double
# double quotes.
literal() {
  local value="$1"
  printf "'%s'" "${value//\'/\'\'}"
}

create_database_sql() {
  local name="$1"
  printf "SELECT format('CREATE DATABASE %%I OWNER %%I ENCODING %%L TEMPLATE template0', %s, %s, 'UTF8')\n" \
    "$(literal "${name}")" "$(literal "${DB_USER}")"
  printf "WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = %s) \\gexec\n" "$(literal "${name}")"
}

{
  printf "DO \$ryze\$\nBEGIN\n"
  printf "  IF EXISTS (SELECT FROM pg_roles WHERE rolname = %s) THEN\n" "$(literal "${DB_USER}")"
  printf "    EXECUTE format('ALTER ROLE %%I WITH LOGIN PASSWORD %%L', %s, %s);\n" "$(literal "${DB_USER}")" "$(literal "${DB_PASSWORD}")"
  printf "  ELSE\n"
  printf "    EXECUTE format('CREATE ROLE %%I WITH LOGIN PASSWORD %%L', %s, %s);\n" "$(literal "${DB_USER}")" "$(literal "${DB_PASSWORD}")"
  printf "  END IF;\nEND\n\$ryze\$;\n"
  create_database_sql "${DB_NAME}"
  create_database_sql "${TEST_DB}"
} | sudo -u postgres psql --quiet --no-psqlrc -v ON_ERROR_STOP=1 -d postgres

echo "PostgreSQL role \"${DB_USER}\" ready; databases \"${DB_NAME}\" and \"${TEST_DB}\" exist."
