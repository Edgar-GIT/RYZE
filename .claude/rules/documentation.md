---
paths:
  - "docs/**/*.MD"
  - "docs/**/*.md"
  - "CLAUDE.md"
  - "AGENTS.md"
---

# RYZE documentation rules

- Treat documentation as the product source of truth. Read the relevant
  section, preserve its conventions, and do not claim behavior without evidence.
- Keep implementation and documentation synchronized when confirmed behavior,
  API contracts, security, architecture, UI or database behavior changes.
- Put the newest concise environment or implementation entry first in
  `docs/REGISTRY/update_logs.MD`; do not invent a product version.
- Keep AI-environment guidance in `docs/AI/`. Do not duplicate complete product
  manuals into `CLAUDE.md`; link to targeted files instead.
- Resolve contradictory instructions explicitly. Owner-defined protected UI
  areas and brand colors must remain easy to find in frontend documentation.
