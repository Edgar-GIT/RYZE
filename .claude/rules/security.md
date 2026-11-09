---
paths:
  - "main/src/backend/api/**/*.go"
  - "main/src/backend/middleware/**/*.go"
  - "main/src/backend/services/**/*.go"
  - "main/src/frontend/src/api/**/*.ts"
  - "main/src/frontend/src/api/**/*.tsx"
---

# RYZE security rules

- Server-side authorization is mandatory. Re-check ownership and role on every
  read, write, upload, trainer action and entitlement-gated response.
- Use authenticated identity, never client-provided owner/user/trainer IDs.
  Treat every frontend permission check as presentation only; prevent IDOR.
- Validate and normalize input at the server boundary. Treat uploaded media and
  filenames as untrusted. Do not expose raw storage paths.
- Keep sensitive answers, health/nutrition data, tokens, passwords and internal
  errors out of logs, URLs, public response projections and queue payloads.
- Preserve JWT, CORS, rate-limit, upload and access-control behavior unless an
  explicit security change is requested and documented.
- Never read or print `.env`, credentials, private keys or token stores.
