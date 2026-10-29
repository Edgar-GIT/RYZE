# RYZE Product Completeness Audit

Version: v2.97.0
Date: 2026-10-29
Status: living document — update whenever a backlog item is closed.

This document records the product-completeness audit of the RYZE platform
performed for v2.94.0. It is the reference for "what exists, how far it goes,
and what is missing", and it drives the prioritized backlog.

## Method

Every area was inspected in the repository (routes, services, repositories,
pages, tests) and, where possible, exercised against the local environment.
Statuses are conservative:

- **Complete/Verified** — implemented and covered by passing tests or
  reproduced end-to-end locally.
- **Implemented (unverified)** — implemented, but not reproducible locally
  (external provider, deployment-only path, browser-level behaviour).
- **Partial** — a working implementation exists but with documented gaps.
- **Missing** — no implementation exists.
- **Blocked** — implementation is blocked on an external condition (licence,
  credential, third party).

No live payment provider or production deployment is available in this
environment. Deterministic browser E2E coverage now exists for the public
catalogue and trainer prescription editing, but a production-like live browser
run is still unavailable, so anything that depends on it is not marked
Complete/Verified here.

## Matrix

### Foundation and infrastructure

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Backend API (Go/Gin/GORM/MariaDB) | Complete/Verified | `go test ./...` green (40 packages) |
| Database migrations | Complete/Verified | schema version 28, `dirty=false` |
| Frontend SPA (React 19/Vite/TS/CSS Modules) | Complete/Verified | `tsc --noEmit` clean, vitest green |
| CI (race detector + frontend tests) | Complete/Verified | `.github/workflows/ci.yml` |
| Docker compose stack | Implemented (unverified) | no production deployment in this environment |
| systemd unit (`deploy/ryze-backend.service`) | Implemented (unverified) | deployment-only path |
| Database backup script | Implemented (unverified) | `umask 077` hardening; no live restore drill |
| Security baseline (v2.93.0) | Complete/Verified (unit level) | CSRF OriginGuard, JWT claims/TTL cap, rate-limiter body bound, parameterized SQL logs; residual: git-history rotation is an operator action, no browser-level E2E |

### Authentication and accounts

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Registration / login / logout / session versioning | Complete/Verified | handler tests, session_version migration |
| Change password | Complete/Verified | service + handler tests |
| Account deletion (soft delete) | Complete/Verified | delete_account service |
| Profile management | Complete/Verified | profile page + API |
| Self-service password recovery | Missing | no forgot/reset flow; only admin-triggered `POST /admin/users/:id/password` |
| Email delivery (verification, receipts, notifications) | Missing | `main/src/backend/email/` is empty; no mail integration |

### Public marketing and commerce pages

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Home page | Complete/Verified | API-backed program highlights and category preview; no static prices or Premium Level 2 link; browser E2E |
| Services page | Complete/Verified | API-backed category cards; no static `SERVICE_PROGRAMS` or Premium Level 2; browser E2E |
| Programs overview section | Complete/Verified | API-backed published programs; no hardcoded BUILD MUSCLE / LOSE FAT / TRAIN FREE; browser E2E |
| Our Vision | Complete/Verified | static content page |
| Feedback | Complete/Verified | form + ADMIN_2 feedback page |
| Contact | Partial | UI-only section; no submission backend |
| Generic marketplace | Partial | catalog from API, but filtering/sorting/pagination run client-side; server filter params unused |
| Generic program detail + checkout | Complete/Verified | API-driven detail, `useProgramCheckout`, entitlement states |
| Premium Level 1 page | Complete/Verified | complete returned catalogue, query-based selection, questionnaire/checkout/delivery unchanged; mocked browser E2E; live purchase still unverified |
| Purchase history (`/account/purchases`) | Complete/Verified | API-driven |
| My Programs / Program Access (training + nutrition) | Complete/Verified | prior milestone e2e tests |
| RYZE Feed | Missing | route exists as an explicit `UnderDevelopmentPage` placeholder (deferred by product decision) |

### Exercise library and training content

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Exercise catalogue backend | Partial | migration 000009/000017 schema + 48-row starter seed (000018); public read API; no source identity, no media fields, no instruction steps, no publication state, no admin write path |
| Public Exercise Library UI | Missing | no navigation entry, no list/detail pages |
| Exercise media / demonstrations | Missing | dataset added under `main/src/resources/exercises/`; licensing gate required before any delivery |
| 3D exercise demonstrations | Missing | milestone scope (in progress) |
| Trainer program builder — exercise selection | Implemented (unverified) | trainer workout-exercise routes + program detail builder; picker quality limited by catalogue content |
| ADMIN_2 plan builder — exercise selection | Implemented (unverified) | `plan_exercise_editor.tsx`; same limitation |
| Workout history (client side) | Complete/Verified | workout_history service + tests |
| Programs: generic admin CRUD | Complete/Verified | generic_programs service + handler tests |
| Programs: trainer CRUD + structure (weeks/workouts/sets) | Complete/Verified | program_structure/workout_exercises services + tests |
| Entitlements and purchase gating | Complete/Verified | entitlement repository + e2e tests |
| Payments (Stripe/PayPal providers) | Implemented (unverified) | providers implemented; no live credentials exercised |
| Commission rules / pricing admin | Complete/Verified | admin_program_pricing + commission_rules tests |

### Trainer experience

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Trainer application (apply) | Partial | always creates a PENDING application; no evidence documents, no reviewer fields |
| Trainer approval / rejection | Partial | transactional approval exists but the final status update is guarded only by `id` (concurrent-review race); no decision history; rejection atomic |
| Verification lifecycle (more-info, suspend, reinstate, history) | Missing | only PENDING/APPROVED/REJECTED, no stored reviewer decisions |
| Trainer dashboard | Complete/Verified | data from clients/programs services |
| Trainer clients | Complete/Verified | trainer_clients service + tests |
| Trainer private client programs | Complete/Verified | trainer_client_program handler tests |
| Trainer profile | Complete/Verified | trainer_profile service |

### Administration

| Area | Status | Evidence / gap |
| --- | --- | --- |
| ADMIN_1 dashboard/users/trainers/system/configuration/test-mode/development/audit | Implemented (unverified) | pages + APIs present; audit page backing not independently verified |
| ADMIN_1 trainer applications review | Partial | approve/reject only; no evidence viewing (no evidence exists) |
| ADMIN_2 dashboard/plans/marketplace/trainers/clients/sales/analytics/feedback/settings | Implemented (unverified) | pages + APIs present |
| ADMIN_2 catalogue management (exercises) | Missing | no page, no API |
| Role/permission model | Complete/Verified | `adminroles.go` grants + middleware; catalogue permission to be added |

### Cross-cutting

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Statistics / analytics | Complete/Verified | statistics repository + handler tests |
| File uploads / media pipeline | Missing | no upload surface exists (v2.93.0 audit); needed for verification evidence |
| Internationalisation | Missing | English-only UI; dataset provides 10 languages for future use |
| Notifications (in-app/email/push) | Missing | not part of current product scope |

## Prioritized backlog

### P0 — v2.94.0 milestone (this cycle)

1. Exercise catalogue: source identity, body part, instruction steps, media
   references, publication/archive state, ownership separation (platform vs
   reserved trainer content), admin management API and UI.
2. Idempotent import of the 1,324-record dataset with duplicate handling,
   preserving the 48 existing platform exercises and program references.
3. Conditional media integration: thumbnails for browsing, lazy animation
   playback, maintainable asset-reference abstraction, explicit
   media-unavailable states, licensing gate disabled by default.
4. Genuine Three.js viewer for bodyweight squat, push-up, dumbbell biceps curl
   and bodyweight lunge, with lazy loading and WebGL fallback.
5. Trainer verification: evidence documents with private storage and strict
   validation, durable decision history, full lifecycle, self-approval
   prevention, concurrent-decision safety (fix the existing approval race),
   review UI and trainer status UI.
6. Security/integration pass and full validation for all of the above.

### P1 — product consistency

7. Home/Services content from the API; remove hardcoded prices and the
   Premium Level 2 placeholder link (Premium Level 2 must not exist).
   **Closed in v2.97.0.**
8. Marketplace: use server-side filtering/sorting/pagination.
9. Premium Level 1: render all returned programs, not only the first.
   **Closed in v2.97.0.**

### P2 — product capability gaps

10. Self-service password recovery and email delivery.
11. Contact page submission backend.
12. RYZE Feed (deferred by design; requires a product decision first).

### P3 — verification and operations

13. Live payment provider verification (Stripe/PayPal) in a non-production
    account.
14. Browser-level E2E coverage for the highest-value streams.
15. Live backup/restore drill; production deployment verification.

## Deployment blockers

- **Exercise media licensing (external):** the Gym visual media bundled with
  the dataset may not be served, bundled or redistributed in a public or
  production RYZE deployment until the required commercial-use and
  redistribution permissions are obtained and documented. Media delivery is
  therefore shipped disabled by default and must remain disabled until the
  operator confirms the licence. See `docs/INSTRUCTIONS/security.MD`.
- **Git history rotation (operator):** credentials that ever reached the
  repository history remain compromised until rotated by the operator
  (carried over from v2.93.0).
