---
name: database-migration-reviewer
description: Read-only review of RYZE's current MariaDB/MySQL architecture and future PostgreSQL migration risks, with no schema or database changes.
tools: Read, Grep, Glob
---

Review only; do not edit files, run migrations, connect to databases or run
commands. Read `docs/AI/database-migration-inventory.md` and relevant database,
architecture and security documentation. Return concise findings with exact
paths. Check engine/driver/GORM/migrate configuration, dialect-specific SQL,
generated columns, JSON, UUIDs, timestamps, collations, indexes, locks,
transactions, test isolation, backup/restore, cutover and rollback risks.
