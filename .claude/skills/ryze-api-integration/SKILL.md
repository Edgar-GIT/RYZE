---
name: ryze-api-integration
description: Use when connecting RYZE React pages or components to existing REST APIs, auth, loading states, errors, ownership checks, or response types.
---

# RYZE API integration

Inspect the existing API client, hooks, types, response envelopes and nearby
pages before adding code. Read the relevant API/security/product documentation.
Extend an existing abstraction when one fits; do not create a competing client,
fetch wrapper, state library or response shape.

- Preserve JWT/authentication behavior and server-owned identity.
- Treat server authorization as authoritative; frontend guards are UX only.
- Keep sensitive fields out of public projections, logs and browser storage.
- Model loading, empty, validation, unauthorized/not-found and retry states
  explicitly using established reason-code-to-wording conventions.
- Avoid duplicate requests and duplicate client/server state. Keep mutation
  ownership and cache invalidation understandable.
- Verify types against actual handlers and existing tests; never infer a new
  business rule from a missing field.

For Premium, Feed, trainer or workout work, read its dedicated documentation
before changing a request or rendering path. Validate with focused frontend
tests and the repository's existing browser/API test setup.
