# Database migration inventory

This is a read-only preparation record, verified on 2026-11-09. No database,
schema, migration, dependency or connection configuration was changed.

## Current runtime

- Product baseline: v2.103.0; latest verified migration: 000035.
- Compose database image: `mariadb:11`, service port 3306, persistent
  `/var/lib/mysql` volume.
- Go backend: `main/src/backend/`; Go 1.26.5.
- Driver/GORM: `github.com/go-sql-driver/mysql` v1.8.1,
  `gorm.io/driver/mysql` v1.6.0, `gorm.io/gorm` v1.31.2.
- Migration library: `github.com/golang-migrate/migrate/v4` v4.19.1.
- Connection setup: `database/database.go` opens a MySQL GORM connection with
  `ParseTime`, UTC location and `MultiStatements`; it configures pooling and
  pings the database. `database/migrate.go` embeds `database/migrations/*.sql`
  and uses the MySQL migrate driver.
- Environment defaults: host `127.0.0.1`, port `3306`; secrets are read from
  environment and were not inspected or printed.
- Backups: `scripts/backup_db.sh` and `restore_db.sh` use MariaDB/MySQL dump and
  client tooling.
- There is no PostgreSQL driver, connection configuration or migration path in
  the verified backend.

## Schema and query inventory

There are 35 up/down migration pairs under
`main/src/backend/database/migrations/`. The SQL uses MariaDB/MySQL-specific
features including InnoDB/utf8mb4/collation declarations, `DATETIME(6)`,
`CURRENT_TIMESTAMP(6) ON UPDATE`, `AFTER`, `UNSIGNED`, JSON, ENUM values,
generated columns and MySQL foreign-key/index syntax. Recent migrations include
nutrition and Premium Level 2 JSON data and generated/unique keys.

Models use application-generated UUID strings, GORM timestamps and soft-delete
fields where documented. Repositories include GORM row-locking clauses and raw
SQL. A reconciliation repository uses `ON DUPLICATE KEY UPDATE`; user and
exercise repositories recognize MySQL duplicate-key error code 1062. These are
explicit compatibility-review items, not conversion instructions.

Tests currently use the configured MariaDB environment and transactional cleanup
patterns in places; this is not proof of a fully isolated database. A future
migration must establish isolated databases and safe fixtures before destructive
or parity testing.

## Required future stages

1. Inventory all schema objects, models, views, raw SQL, GORM clauses, jobs,
   scripts, deployment variables, tests and operational tooling without writes.
2. Build a reviewed compatibility matrix for UUIDs, JSON, ENUM/checks,
   generated columns, indexes, collations, timestamps, locking and transactions.
3. Rehearse schema/data conversion in isolated environments, validating counts,
   keys, constraints, nullability, relationships and representative content.
4. Adapt application code only in a dedicated milestone, with both-engine tests,
   performance checks and query-plan review.
5. Rehearse backup/restore, cutover, monitoring, rollback and recovery before
   any production decision. Keep schema conversion, application conversion and
   cutover separately reviewable.

Until that milestone is explicitly approved, keep the MariaDB/MySQL driver,
configuration, migrations and runtime unchanged.
