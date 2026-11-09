---
name: ryze-postgresql-migration
description: Use only for the explicitly approved future migration of RYZE from its verified MariaDB/MySQL runtime to PostgreSQL; begin with read-only inventory and staged compatibility planning.
---

# RYZE PostgreSQL migration

This skill prepares and guides a dedicated migration milestone; it does not
authorize migration work by itself. Start with the factual inventory in
`docs/AI/database-migration-inventory.md`, then read database, architecture,
security and procedure documentation.

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
