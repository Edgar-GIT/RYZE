# RYZE Product Completeness Audit

Version: v2.111.0
Date: 2026-11-13
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

No live payment provider, public SMTP relay or container runtime is available in
this environment. Deterministic browser E2E coverage exists for the public
catalogue, contact outcomes and trainer prescription editing. v2.99.0 also
served the production bundle with the documented CSP and exercised public routes
and a contact submission. The Docker image was not built and nginx did not load
its configuration locally, so deployment-only paths remain marked unverified.

## Matrix

### Foundation and infrastructure

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Backend API (Go/Gin/GORM/PostgreSQL) | Complete/Verified | `go test ./...` green against the isolated `ryze_test` database (v2.109.0) |
| Database migrations | Complete/Verified | schema version 40, `dirty=false` on `ryze` and `ryze_test`; every new `down` validated inside a rolled-back transaction; migrations embedded in the release binary |
| PWA (manifest, icons, service worker, update prompt) | Complete/Verified (local) | production preview: manifest served, worker controls the page, caches hold only hashed assets/icons and the page shell (no `/api/`), app opens offline; real install on devices unverified |
| Frontend SPA (React 19/Vite/TS/CSS Modules) | Complete/Verified | `tsc --noEmit` clean, vitest green |
| CI (race detector + frontend tests + Playwright) | Implemented (hosted run unverified) | `.github/workflows/ci.yml`; local CI-equivalent checks passed in v2.99.0 |
| Docker compose stack | Implemented (unverified) | `docker compose config` parsed in v2.99.0; image/build/runtime blocked by unavailable daemon |
| systemd unit (`deploy/ryze-backend.service`) | Implemented (unverified) | `systemd-analyze verify` accepted the unit apart from the absent deployed binary |
| Database backup script | Complete/Verified (local drill) | v2.99.0 plain and GPG restores matched 33 tables, version 32 and row counts; not rerun after migration 000034; off-host storage still operator-owned |
| Security baseline (v2.93.0) | Complete/Verified (unit level) | CSRF OriginGuard, JWT claims/TTL cap, rate-limiter body bound, parameterized SQL logs; v2.99.0 adds mandatory SMTP STARTTLS and a CSP browser sweep; git-history rotation remains an operator action |

### Authentication and accounts

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Registration / login / logout / session versioning | Complete/Verified | handler tests, session_version migration |
| Change password | Complete/Verified | service + handler tests |
| Account deletion (soft delete) | Complete/Verified | delete_account service |
| Profile management | Complete/Verified | profile page + API |
| Self-service password recovery | Complete/Verified | forgot/reset pages, single-use hashed fragment tokens, session revocation; service integration tests |
| Email verification and language preference | Complete/Verified | 48-hour links, resend, `PATCH /me/preferences`; integration tests |
| Transactional email (Resend over SMTP) | Implemented (relay unverified) | outbox with dedupe, retries and worker (unit tests); templates EN/PT for verification, recovery, password change, access granted, Premium Level 2 steps and support replies; no live Resend send in this environment (MAIL_* not configured, emails recorded as `skipped`) |
| Google Sign-In | Implemented (provider unverified) | OIDC + PKCE + JWKS verification, linking from the profile, no auto-linking; unit/integration tests with a local fake provider; no real Google client configured locally |

### Public marketing and commerce pages

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Home page | Complete/Verified | API-backed program highlights and category preview, including the Premium Level 2 service; no static prices; browser E2E |
| Services page | Complete/Verified | API-backed category cards for Generic, Premium Level 1 and Premium Level 2; no static `SERVICE_PROGRAMS`; browser E2E |
| Programs overview section | Complete/Verified | API-backed published programs; no hardcoded BUILD MUSCLE / LOSE FAT / TRAIN FREE; browser E2E |
| Our Vision | Complete/Verified | static content page |
| Feedback | Complete/Verified | form + ADMIN_2 feedback page |
| Contact | Implemented, real relay unverified | Multipart API, mandatory STARTTLS, validation, rate limit and browser states; positive and failure paths verified against controlled local SMTP sinks |
| Coach showcase (featured coach programs, coach profiles, view counter) | Complete/Verified (local) | v2.111.0: `program_view_stats`, dedupe + owner exclusion, `GET /coaches/featured-programs`, `GET /coaches/:id`, coach on program detail; service, handler and repository tests; 27-step browser E2E on `ryze_test` (publish → featured → view counted once → coach profile) at 390 and 1440 px |
| Generic marketplace | Complete/Verified (mocked) | server-side filters, sort and pagination; deterministic rows; browser E2E with mocked API; live positive path unverified because the DB has no published programs |
| Generic program detail + checkout | Complete/Verified | API-driven detail, `useProgramCheckout`, entitlement states |
| Premium Level 1 page | Complete/Verified | objective-based catalogue, query-based objective/package selection, curated and legacy questionnaire delivery, shared checkout and status-aware access; mocked browser E2E; live purchase still unverified |
| Premium Level 2 page and request | Complete/Verified (mocked) | API-scoped offer catalogue, shared checkout, catalog-driven versioned intake, status-aware request page, delivered nutrition and training access; frontend unit tests; live purchase still unverified |
| Premium Level 2 administration | Implemented (unverified) | offer authoring in the ADMIN_2 plan builder, request queue/search/status, trainer allocation, intake review, delivery state and history |
| Premium Level 2 trainer workspace | Implemented (unverified) | allocated-request queue and detail, draft programme attachment, nutrition authoring and confirmed delivery |
| Purchase history (`/account/purchases`) | Complete/Verified | API-driven |
| My Programs / Program Access (training + nutrition) | Complete/Verified | unified entitlement/assignment/Free access, Premium Level 1 package routing and Premium Level 2 service-request routing, metadata-only public detail, backend e2e and handler/service tests |
| RYZE Feed | Missing | route exists as an explicit `UnderDevelopmentPage` placeholder (deferred by product decision; needs video hosting and moderation decisions) |
| Unknown addresses | Complete | unknown `/services/:slug` and other unknown routes show "Page not found" instead of an "under development" message (v2.109.0) |

### Exercise library and training content

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Exercise catalogue backend | Complete/Verified | catalogue with admin management API and imported dataset (1,353 rows locally); handler tests |
| Public Exercise Library UI | Complete/Verified | list and detail pages with tests (protected area) |
| Exercise media / demonstrations | Blocked (licence) | media delivery exists behind `EXERCISE_MEDIA_ENABLED`, which stays off until the dataset licence is confirmed |
| 3D exercise demonstrations | Implemented | lazy Three.js viewer with fallback in the Exercise Library (protected area) |
| Trainer program builder — exercise selection | Implemented (unverified) | trainer workout-exercise routes + program detail builder; picker quality limited by catalogue content |
| ADMIN_2 plan builder — exercise selection | Implemented (unverified) | `plan_exercise_editor.tsx`; same limitation |
| Workout history (client side) | Complete/Verified | legacy `workout_history` plus session history; service, repository and e2e tests |
| Programs: generic admin CRUD | Complete/Verified | generic_programs service + handler tests |
| Programs: trainer CRUD + structure (weeks/workouts/sets) | Complete/Verified | program_structure/workout_exercises services + tests |
| Entitlements and purchase gating | Complete/Verified | entitlement repository + expiry/restore + access e2e tests |
| Payments (Stripe/PayPal providers) | Implemented (unverified) | providers implemented; no live credentials exercised. v2.99.0 verified the provider-absent state fails closed (no method advertised, pending purchase, generic 502, no entitlement) |
| Free programme activation | Complete/Verified | one-active-per-user DB constraint, idempotent activation, replacement, repository/access/e2e tests |
| Interactive workout sessions | Complete/Verified | snapshot sessions, one live session per user, set logging, pause/resume/complete/abandon, rest timer, backend e2e |
| Commission rules / pricing admin | Complete/Verified | admin_program_pricing + commission_rules tests |

### Trainer experience

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Trainer application (apply) | Complete/Verified | application plus private evidence documents (upload/list/download/remove while pending); tests |
| Trainer approval / rejection | Complete/Verified | approve/reject with stored decisions (`trainer_application_decisions`, `GET …/decisions`) and evidence documents; e2e test |
| Verification lifecycle (more-info, suspend, reinstate) | Missing | statuses are only PENDING/APPROVED/REJECTED; suspension is done by disabling the trainer in ADMIN_1 |
| Trainer dashboard | Complete/Verified | data from clients/programs services |
| Trainer clients | Complete/Verified | trainer_clients service + tests; client list UI |
| Trainer client training insight | Complete/Verified | relationship-scoped overview/statistics/session/exercise-history APIs, repository + service + handler tests, client detail page tests |
| Trainer private client programs | Complete/Verified | trainer_client_program handler tests; assign/remove controls in the client detail UI |
| Trainer profile | Complete/Verified | trainer_profile service |

### Administration

| Area | Status | Evidence / gap |
| --- | --- | --- |
| ADMIN_1 dashboard/users/trainers/system/configuration/test-mode | Complete/Verified (local) | real data, including `GET /admin/system/status` (v2.110.0); every route opened at 390 and 1440 px |
| ADMIN_1 Audit log | Complete/Verified (local) | v2.111.0: immutable `admin_audit_logs` (trigger rejects UPDATE/DELETE), middleware records every admin mutation, evidence download and second-factor sign-in (denials included); `GET /admin/logs` (ADMIN_1 only); service, middleware, handler and repository tests; page exercised at 390 and 1440 px on `ryze_test` |
| ADMIN_1 Development | Missing | explicit placeholder page |
| ADMIN_1 trainer applications review | Complete/Verified | approve/reject, evidence download, decision history |
| ADMIN_2 dashboard/plans/marketplace/trainers/clients/sales/analytics/feedback/settings | Implemented (unverified) | pages + APIs present; Generic, Premium Level 1 and Premium Level 2 family tabs, objective/mode/Level 2 offer authoring, curated nutrition editor, Premium Level 2 request queue and trainer allocation, content-lock UX |
| ADMIN_2 catalogue management (exercises) | Complete/Verified | admin2 exercises page and API with tests |
| Support chat (clients and trainers ↔ RYZE, answered by ADMIN_2) | Complete/Verified (local) | eligibility on every call, idempotent messages, SSE live stream with resume; service + handler tests and a live two-browser E2E on `ryze_test` (reply delivered in ~0.1 s, recovery after an offline period, closed and ineligible paths) |
| Role/permission model | Complete/Verified | `adminroles.go` grants + middleware; catalogue permission to be added |

### Cross-cutting

| Area | Status | Evidence / gap |
| --- | --- | --- |
| Statistics / analytics | Complete/Verified | programme-aware repository, all-time session/set counters + handler tests |
| File uploads | Complete/Verified | trainer application evidence (private storage, validation, download as attachment) and the contact attachment; exercise media stays licence-blocked |
| Internationalisation | Partial | PT \| EN block in the navigation and a first-visit prompt (v2.110.0); public site, checkout, Premium Level 1/2, login/registration, footer, Exercise Library chrome, member area, training app and trainer navigation translated; trainer page content and admin consoles English-only; protected Services cards English |
| Notifications (in-app + email) | Complete/Verified | 24-hour notifications with dedupe tombstones and hourly purge; created on purchase completion, Premium Level 2 intake/assignment/delivery and support replies; bell in the account and trainer headers; tests; push notifications not implemented |

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
   Premium Level 2 placeholder link. **Closed in v2.97.0**; the real Premium
   Level 2 service replaced the placeholder in v2.103.0.
8. Marketplace: use server-side filtering/sorting/pagination. **Closed in v2.98.0.**
9. Premium Level 1: render all returned programs, not only the first.
   **Closed in v2.97.0.**
- **v2.101.0:** objective-based Premium Level 1 catalogue, curated nutrition
  packages, ADMIN_2 authoring and legacy questionnaire compatibility. Closed.
- **v2.102.0:** secure trainer client detail with access sources, programme
  assignment controls, statistics, session history and exercise performance.
  Closed.
- **v2.103.0:** Premium Level 2 trainer-delivered service: platform-owned offer
  authoring, purchased versioned intake, ADMIN_2 allocation, trainer programme
  and nutrition authoring, validated delivery, entitlement access, customer,
  admin and trainer surfaces, migration 000035 and L2 tests. Closed.

### P2 — product capability gaps

10. Self-service password recovery and email delivery. **Closed in v2.109.0**
    (live Resend delivery still unverified).
11. Contact page submission backend. **Closed in v2.98.0** (live SMTP delivery still unverified).
12. RYZE Feed (deferred by design; requires a product decision first).

### v2.111.0 additions

- Administrative audit log, coach showcase, local test stack fixes
  (`start_test_mode.sh --accounts-only`), Feedback page motion and the 3D Our
  Vision hero. Closed.
- Open: trainer-page and admin-console translation; legal pages (Terms,
  Privacy, Cookies) — owner content required; Feed; audit-log retention
  policy (owner decision); trainer profile fields (bio, photo) for the coach
  profile; test-database isolation between Go packages.

### v2.110.0 additions

- Lyfta-style training app (plan, player, report, history, progress), PT | EN
  toggle and first-visit prompt, public-site translation, private-area design
  unification, lazy-loaded consoles, API security headers and extra rate
  limits. Closed.
- Open: trainer-page and admin-console translation; legal pages (Terms,
  Privacy, Cookies) — owner content required; audit-log service; Feed;
  test-database isolation between Go packages.

### v2.109.0 additions

- Google Sign-In, notifications, support chat, PWA and the account-area
  translation. Closed.
- Pending owner decisions: Free "limited access duration" (no duration value
  defined; activations support an optional expiry), Free "PDF export" (not
  implemented), translation of the public website and protected login copy,
  RYZE Feed.

### P3 — verification and operations

13. Live payment provider verification (Stripe/PayPal) in a non-production
    account.
14. Browser-level E2E coverage for the highest-value streams.
15. Live backup/restore drill; production deployment verification.
    **Local plain and encrypted restore drill closed in v2.99.0; container
    build/runtime and production deployment still unverified.**

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
- **Container runtime (environment):** the Docker daemon is not accessible and
  nginx is not installed locally, so the image, compose stack and nginx config
  were not executed in v2.99.0. `docker compose config` and the CSP policy were
  validated separately.
