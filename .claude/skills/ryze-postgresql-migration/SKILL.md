---
name: ryze-postgresql-migration
description: Reference for the completed MariaDB -> PostgreSQL migration of RYZE (2026-11-11) - use when repeating the data copy on another environment or reviewing PostgreSQL compatibility of database code.
---

# RYZE PostgreSQL migration

The migration was approved by the owner and completed on 2026-11-11; the
runtime is now PostgreSQL. `docs/AI/database-migration-inventory.md` records
what was converted, the tooling (`scripts/postgres_setup.sh`,
`scripts/migrate_mysql_to_postgres.sh`) and how it was verified. Use the stages
below when repeating the copy on another environment (for example production)
or when reviewing database code for PostgreSQL compatibility.

Required stages:

1. Read-only inventory of engine/version, driver, GORM config, migrations,
   models, views, raw SQL, generated columns, JSON, enums/checks, indexes,
   collations, timestamps, locks, transactions, test/deployment config.
2. Compatibility matrix and application/schema design decisions, with reviewed
   alternatives and no unverified SQL conversion.
3. Isolated schema conversion and data migration rehearsal; validate row counts,
   keys, constraints, nullability, UUIDs, JSON, timestamps and relationships.
4. Application compatibility changes and focused/unit/integration/performance
   tests against isolated MariaDB and PostgreSQL environments.
5. Backup/restore, cutover, observability, rollback and recovery rehearsal.

Never combine conversion, application changes and production cutover in one
unverified operation. Do not add a Postgres dependency, edit current migrations,
change connection configuration, reset a database, or touch production during
inventory. Record assumptions and unresolved incompatibilities rather than
guessing. Pay particular attention to MySQL `ON DUPLICATE KEY UPDATE`,
`DATETIME(6)`, `ENUM`, `JSON`, generated columns, collations and locking.
