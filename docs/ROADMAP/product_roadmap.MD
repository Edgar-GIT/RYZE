# RYZE Product Roadmap

This is the single canonical roadmap for RYZE. It replaces ad-hoc backlog lists;
the prioritized findings behind it are recorded in
`docs/AUDIT/product_completeness.MD`. Priorities may change only after a product
decision and an update to this document.

The roadmap does not introduce product scope that is explicitly excluded by the
project instructions: no Premium Level 2, no RYZE Feed without a separate
product decision, no additional payment methods and no trainer-authored
nutrition.

## Phase A — Training Content Backend (v2.95.x)

Goal: make the persisted exercise catalogue and trainer program structure
usable by the backend without losing authorization or historical data.

Scope:

- Canonical trainer workout-exercise prescription persistence and reads.
- Client read of the assigned program, including instructions, notes and sets.
- Shared prescription validation for trainer and platform-owned programs.
- Server-side exercise-media gating across every program and catalogue response.
- Input, ownership, soft-delete and historical-tail guarantees.
- Documentation and release records for the above.

Exit criteria:

- A trainer can add an exercise and replace its instructions, notes and ordered
  sets only through their own program chain.
- A client receives the assigned program's prescription.
- Invalid set values and foreign assignments are rejected.
- `go test ./...` and `go vet ./...` pass.
- No schema migration is required for this phase.

Status: backend implemented. The trainer-facing prescription editor and its
browser-level coverage were completed in v2.97.0; the editor consumes the Phase A
contract without backend changes.

## Phase B — Product Consistency and Content Operations

Goal: remove remaining mismatch between the documented product and the
implemented surfaces.

Scope:

- Trainer program detail prescription editor consuming the Phase A API.
- Home/Services content from the API; remove hardcoded prices and any Premium
  Level 2 placeholder.
- Marketplace server-side filtering, sorting and pagination. **Closed in v2.98.0.**
- Premium Level 1 lists every returned program, not only the first.
- Contact page submission backend. **Closed in v2.98.0** (live SMTP delivery still unverified).
- Import reporting improvements for rejected/unchanged records, with tests.
- Test Mode persona-email reservation and fail-closed ownership checks, with
  regression tests.
- Documentation alignment for completed surfaces.

Acceptance criteria:

- No user-facing product price or offer copy is hardcoded when the API owns it.
- Marketplace filters and pagination are server-driven and tested.
- Premium Level 1 renders the complete returned catalogue.
- Trainer prescription editing is covered by browser-level tests.
- Import failures are observable without partially applying a transaction.

Status: v2.96.0 completed the backend Test Mode persona-email reservation and
fail-closed ownership checks, plus the importer reporting improvements. v2.97.0
completed the trainer prescription editor, API-driven Home/Services content, the
complete Premium Level 1 catalogue and browser-level coverage for catalogue
browsing and trainer prescription editing. v2.98.0 completed marketplace
server-side filtering/sorting/pagination and the contact submission backend;
contact delivery still requires a real SMTP configuration before a live send can
be verified.

## Phase C — Commerce and Operations

Goal: make the platform operable and payment integration verifiable.

Scope:

- Self-service password recovery and transactional email delivery.
- Live Stripe/PayPal verification in non-production accounts.
- Production CORS, cookie, proxy and secret-management verification.
- Encrypted off-host backup and a documented restore drill.
- Observability for failed payments, nutrition delivery failures, imports and
  evidence storage.
- Credential rotation for values exposed in historical repository state.

Acceptance criteria:

- A purchase completes against the real provider sandbox and the entitlement is
  issued exactly once.
- A restore drill recovers the development data into a disposable environment.
- Failed background work is visible to operators without exposing client data.
- No production secret is read from the repository or printed in logs.

## Phase D — Private Beta

Goal: validate the complete client and trainer journeys with bounded users.

Scope:

- Browser E2E coverage for catalogue browsing, program access, checkout,
  trainer program editing and evidence review.
- Accessibility and responsive regression pass.
- Load/performance test of catalogue and program access.
- Trainer onboarding support and evidence-storage operations.
- Resolution of the Gym visual media licence before any public media delivery.
- Security review of authentication, authorization, uploads and payment paths.

Acceptance criteria:

- Critical journeys pass in a browser against a production-like environment.
- No unresolved critical or high-severity security finding.
- Media delivery is either explicitly licensed and enabled, or remains disabled
  with a documented placeholder experience.
- Backup, restore and incident contacts are exercised.

Status: browser coverage has started with deterministic mocked API tests for
public catalogue browsing and trainer prescription editing. Program access,
checkout, evidence review, accessibility/responsive regression and a
production-like live run remain open.

## Phase E — Public Launch

Goal: release RYZE to the public with operational and legal readiness.

Scope:

- Production deployment and rollback procedure.
- Final legal/licence confirmation for exercise media and dataset attribution.
- Production monitoring, alerting, backups and support runbook.
- Launch acceptance pass over public marketing, marketplace, checkout, program
  access and trainer surfaces.
- Documentation freeze for the released version.

Acceptance criteria:

- All deployment blockers are closed or explicitly accepted by the operator.
- A production purchase and program-access flow is verified end to end.
- Monitoring and backup/restore are active.
- The released API version is passed through build flags and matches the
  registry entry.

## Cross-Cutting Rules

- Every phase preserves soft-delete, ownership scoping, client identity from the
  authentication context and server-side validation.
- Database changes use the migration system; no manual schema edits.
- Documentation is updated in the same development cycle as behavior changes.
- Verification claims distinguish unit, integration, database, browser and
  external-provider evidence.

## Current Blockers

- Gym visual media is limited to 180×180 and its commercial redistribution terms
  must be confirmed before production delivery.
- Live payment-provider verification requires external credentials.
- A disposable database is required for a safe restore/rollback drill.
- Historical credential exposure requires operator rotation.
