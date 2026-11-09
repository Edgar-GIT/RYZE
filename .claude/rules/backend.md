---
paths:
  - "main/src/backend/**/*.go"
---

# RYZE backend rules

- Read `docs/INSTRUCTIONS/project.MD`, `docs/INSTRUCTIONS/arquitecture.MD`,
  the relevant API/security/database sections, and procedure guidance first.
- Follow the existing handler/service/repository/model separation. Keep
  handlers thin and reuse existing services, views and query patterns.
- Validate all input server-side. Derive identity, role and ownership from the
  authenticated context; never trust client-supplied user, trainer or owner IDs.
- Preserve UUID and timestamp conventions, soft deletion, foreign keys,
  transactions, idempotency and entitlement/access boundaries.
- Return established response envelopes and reason-code behavior. Do not expose
  sensitive questionnaire/nutrition data, internal failure reasons, raw paths or
  credentials in logs or public responses.
- Add or update focused tests for confirmed backend changes. Do not silently
  change payment, authorization or product-family behavior.
- Keep the current MariaDB/MySQL implementation unchanged until the dedicated
  PostgreSQL migration is explicitly started.
