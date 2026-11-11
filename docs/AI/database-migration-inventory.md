# Database migration record: MariaDB -> PostgreSQL

The owner approved the migration on 2026-11-11 and it was completed the same
day (v2.108.0). This file records what changed, why, and how it was verified.
The earlier read-only inventory (2026-11-09, MariaDB 11, migrations up to
000035) is in Git history.

## Runtime now

- PostgreSQL 18.6 locally (Arch `postgresql` package, systemd service on
  5432); `postgres:17` in `docker-compose.yml`.
- Backend: `gorm.io/driver/postgres` v1.6.0 over `github.com/jackc/pgx/v5`
  v5.7.6 (`database/database.go`); golang-migrate v4.19.1 with its
  `database/pgx/v5` driver (`database/migrate.go`). The MySQL drivers were
  removed from the backend module.
- Configuration: `DB_HOST`, `DB_PORT` (default 5432), `DB_NAME`, `DB_USER`,
  `DB_PASSWORD`, `DB_SSLMODE` (default `disable`). Sessions run in UTC and a
  registered `timestamptz` codec returns UTC values. GORM's `NowFunc` rounds
  to the microsecond.
- Databases `ryze` and `ryze_test` are owned by the application role and use
  the ICU root collation (`und`).

## Conversion decisions

| MariaDB | PostgreSQL | Why |
| --- | --- | --- |
| `CHAR(36)` / `VARCHAR(36)` ids | `VARCHAR(36)` | Keeps string ids; a native `uuid` would turn an invalid id in a URL into a database error instead of "not found". |
| `DATETIME(6)` | `TIMESTAMPTZ` | Stores instants; UTC sessions keep API output identical. |
| `ON UPDATE CURRENT_TIMESTAMP(6)` | `ryze_touch_updated_at()` + one `BEFORE UPDATE` trigger per table (28) | No column-level equivalent. |
| `JSON` | `JSONB` | Both normalise documents. |
| `DECIMAL` | `NUMERIC` | Same precision. |
| `INT/BIGINT/TINYINT UNSIGNED` | `INTEGER`/`BIGINT`/`SMALLINT` + `CHECK (>= 0)` | No unsigned types. |
| `BOOLEAN` (tinyint) | `BOOLEAN` | Raw SQL `test = 0` became `NOT test`. |
| Generated `IF(cond, CONCAT(...), NULL)` | `CASE WHEN cond THEN a \|\| ':' \|\| b END` (stored) | `IF`/`CONCAT` are not usable; `\|\|` is immutable. |
| `utf8mb4_unicode_ci` (case-insensitive) | `CITEXT` for `users.email`; `active_name = lower(name)` for exercises; `ILIKE` for searches; ICU `und` collation for sorting | Keeps case-insensitive login, uniqueness, search and ordering. Accent-insensitive equality was not carried over. |
| `KEY`/`UNIQUE KEY` inside `CREATE TABLE` | `CREATE INDEX` / `CONSTRAINT ... UNIQUE` | Same names (index names are schema-wide in PostgreSQL; none collided). |
| `UTC_TIMESTAMP(6)`, `NOW(6)` in code | `CURRENT_TIMESTAMP` | — |
| `ON DUPLICATE KEY UPDATE` | `ON CONFLICT (provider, event_id) DO UPDATE` | Reconciliation upsert. |
| `SUM(status = 'x')` | `COUNT(*) FILTER (WHERE ...)` | Boolean sums are not valid. |
| MySQL error 1062 | SQLSTATE `23505` (`isDuplicateEntry`, `violatedConstraint`) | — |

The 36 migration files keep their numbers, so the migration history (and the
`schema_migrations` version) is unchanged.

Transaction semantics: PostgreSQL aborts a transaction after any failed
statement, while MariaDB let it continue. Repository statements whose unique
violation is mapped to a domain error now run in a savepoint (`inSavepoint`).
One production path depended on continuing: the Premium Level 2 delivery
tolerates a concurrent entitlement insert. Tests that provoke a database error
and keep using their transaction wrap that statement in a savepoint as well.

## Tooling

- `scripts/postgres_setup.sh` — creates or updates the login role from `.env`
  and the databases `DB_NAME` and `<DB_NAME>_test` (ICU `und`), through
  `sudo -u postgres psql` with the values on standard input.
- `scripts/migrate_mysql_to_postgres.sh` (module `scripts/mysql_to_postgres/`)
  — copies every row: consistent source snapshot, one target transaction,
  `TRUNCATE` and refill in foreign-key order, generated columns recomputed,
  per-table row counts compared before `COMMIT`; `--dry-run` rolls back.
- `scripts/backup_db.sh` / `scripts/restore_db.sh` — `pg_dump` plain SQL with
  `--clean --if-exists` / `psql --single-transaction -v ON_ERROR_STOP=1`.

## Verification (2026-11-11)

- Migrations: `migrate up` to version 36, a full `migrate down` and `up` again
  on `ryze_test`, then `migrate up` on `ryze`.
- Data: dry run, then copies into `ryze_test` and `ryze`: 3791 rows in 40
  tables, every count equal to MariaDB (version 36 on both sides). Value-level
  aggregates matched for users, exercises (including JSONB step counts and
  microsecond timestamps), programs and trainers.
- Tests: `DB_NAME=ryze_test go test ./... -p 1` — 48 packages pass, including
  the purchase concurrency test. `go vet ./...` passes.
- API: backend restarted on PostgreSQL; public endpoints return 200 with
  `Z` timestamps; on a second instance against `ryze_test`, register, a
  case-different duplicate registration (409), login with a lower-cased
  email, `/me`, feedback eligibility and account deletion behaved as before.
- Backup: backup of `ryze_test` restored into it with equal counts, version
  36, 28 triggers and `citext`.

## Rollback

MariaDB was left intact (no writes after the cutover) and two dumps were taken
before the copy in `backups/ryze-mariadb-final-*.sql.gz` (data and triggers;
the stored-routine dump fails until `mariadb-upgrade` is run, and RYZE has no
routines). Returning to MariaDB means checking out the backend code before
v2.108.0, setting `DB_PORT=3306` and restarting; data written to PostgreSQL
after the cutover would have to be copied back by hand.
