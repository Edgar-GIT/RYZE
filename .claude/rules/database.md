---
paths:
  - "main/src/backend/database/**/*.go"
  - "main/src/backend/database/**/*.sql"
  - "main/src/backend/models/**/*.go"
  - "main/src/backend/repositories/**/*.go"
---

# RYZE database rules

- Read `docs/DATABASE/database.MD` and `docs/INSTRUCTIONS/arquitecture.MD`.
- The runtime is PostgreSQL (migrated from MariaDB on 2026-11-11; `postgres:17`
  in Compose, PostgreSQL 18 locally), accessed through GORM with the pgx
  driver. Write PostgreSQL SQL only; follow the engine conventions in
  `docs/DATABASE/database.MD` ("ENGINE"): `VARCHAR(36)` UUIDs, `TIMESTAMPTZ`,
  the `ryze_touch_updated_at()` trigger for every table with `updated_at`,
  `JSONB`, `CITEXT` email, stored generated columns for one-active rules.
- PostgreSQL aborts a transaction after any failed statement: a statement
  whose constraint violation is handled and followed by more work in the same
  transaction must run in a savepoint (`inSavepoint` in
  `repositories/pg_errors.go`). Detect unique violations with
  `isDuplicateEntry` (SQLSTATE 23505), never by message text.
- Schema changes must use the established migration workflow. Never edit a
  shared database directly, reset a database, or use destructive migrations.
- Preserve UUIDs, timestamp precision, soft deletes, foreign keys, indexes,
  unique constraints, JSON semantics, generated columns and transaction
  behavior. Review raw SQL and dialect-specific GORM clauses explicitly.
- Tests must use an isolated database or an explicitly controlled transaction;
  never point destructive test setup at shared or production data. Run the
  integration suite against `<DB_NAME>_test` (`DB_NAME=ryze_test go test ./...`).
- `docs/AI/database-migration-inventory.md` records the completed MariaDB ->
  PostgreSQL migration and its tooling.
