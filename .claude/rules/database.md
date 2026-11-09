---
paths:
  - "main/src/backend/database/**/*.go"
  - "main/src/backend/database/**/*.sql"
  - "main/src/backend/models/**/*.go"
  - "main/src/backend/repositories/**/*.go"
---

# RYZE database rules

- Read `docs/DATABASE/database.MD` and `docs/INSTRUCTIONS/arquitecture.MD`.
- The verified current runtime is MariaDB 11 in Compose, accessed through the
  Go MySQL driver and GORM. Do not write PostgreSQL SQL, add a Postgres driver,
  alter connection settings, convert migrations, or start a cutover in routine
  work.
- Schema changes must use the established migration workflow. Never edit a
  shared database directly, reset a database, or use destructive migrations.
- Preserve UUIDs, timestamp precision, soft deletes, foreign keys, indexes,
  unique constraints, JSON semantics, generated columns and transaction
  behavior. Review raw SQL and dialect-specific GORM clauses explicitly.
- Tests must use an isolated database or an explicitly controlled transaction;
  never point destructive test setup at shared or production data.
- For migration planning, use the staged read-only inventory and compatibility
  process in `docs/AI/database-migration-inventory.md` and the matching skill.
