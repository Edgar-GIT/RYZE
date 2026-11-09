---
name: ryze-documentation-sync
description: Use when a confirmed RYZE implementation changes product behavior, API, security, architecture, database, UI, or the AI engineering environment.
---

# RYZE documentation synchronization

Identify the smallest authoritative document before editing. Product and
engineering rules live under `docs/INSTRUCTIONS/`, `docs/DATABASE/`, API and
security documentation; AI workflow references live under `docs/AI/`.

Update documentation in the same development cycle as confirmed behavior
changes. Preserve existing capitalization, headings and version conventions.
Add the newest concise entry first in `docs/REGISTRY/update_logs.MD` with the
version, modified files, summary and reason. Do not bump the product version for
an AI-only setup unless the project procedure explicitly requires it.

Keep claims evidence-based: distinguish installed from recommended, current
from planned, and verified from inferred. Never document secrets, credentials,
raw sensitive answers or unverified feature behavior. Review affected docs and
the final diff for contradictions before handoff.
