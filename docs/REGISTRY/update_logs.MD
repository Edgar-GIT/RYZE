# Version 2.35.0

Title: Backend Step 47 — Stripe Checkout Provider Integration

Date: 2026-08-20

Summary:

Integrated Stripe Checkout as the first real payment provider behind the
`payments.Provider` abstraction created in Step 46. A new `StripeProvider`
struct implements the `Provider` interface using Stripe Checkout Sessions in
payment mode. Each initiation request creates a Checkout Session with a single
line item whose amount and currency come exclusively from the immutable purchase
snapshot, and returns the hosted Checkout URL for client-side redirect. The
provider uses Stripe idempotency keys derived from the purchase ID to prevent
duplicate session creation. The `notConfiguredPaymentProvider` placeholder is
replaced by a conditional `resolvePaymentProvider` function in routes that
selects the Stripe provider when `STRIPE_SECRET_KEY` is set, or falls back to
the placeholder otherwise. A `StripeConfig` struct and `LoadStripe` function
follow the existing config conventions.

Database:

- No new migration.

Behavior:

- `services/payments/stripe_provider.go` — `StripeProvider` struct implementing
  the `Provider` interface. Creates Stripe Checkout Sessions in payment mode
  with a single line item. Returns the Checkout URL as `CheckoutURL` and the
  Stripe session ID as `PaymentID`. Status is always `PaymentStatusRequiresAction`
  since the user must visit the Checkout URL. Uses idempotency keys
  `ryze-purchase-{purchaseID}`. Validates request fields (purchase ID, amount,
  currency) before calling Stripe. Wraps all Stripe API errors with
  `ErrProviderFailure`.
- `config/config.go` — `StripeConfig` struct with `SecretKey`, `SuccessURL`,
  `CancelURL` fields.
- `config/env.go` — `LoadStripe()` reads `STRIPE_SECRET_KEY`,
  `STRIPE_SUCCESS_URL`, `STRIPE_CANCEL_URL` from environment. All optional.
- `main.go` — loads `StripeConfig` and passes it to routes.
- `routes/routes.go` — `resolvePaymentProvider` function conditionally creates
  a `StripeProvider` when `STRIPE_SECRET_KEY` is set, otherwise returns the
  `notConfiguredPaymentProvider` placeholder. Sets `stripe.Key` when Stripe is
  configured. `Setup` now accepts `config.StripeConfig` parameter.

Tests:

- 8 Stripe provider unit tests (success, empty purchase ID, zero amount, empty
  currency, Stripe API error, empty checkout URL, idempotency key, no URLs).
- All tests use a mock HTTP server configured via `stripe.SetBackend` to avoid
  network access and real credentials.

Modified files:

- services/payments/stripe_provider.go (new)
- services/payments/stripe_provider_test.go (new)
- config/config.go (modified)
- config/env.go (modified)
- main.go (modified)
- routes/routes.go (modified)
- routes/trainer_application_e2e_test.go (modified)
- .env.example (modified)
- docs/REGISTRY/update_logs.MD (modified)

# Version 2.34.0

Title: Backend Step 46 — Payment Provider Abstraction & Payment Initiation

Date: 2026-08-20

Summary:

Introduced a provider-independent payment abstraction layer. A new
`services/payments` package defines the `Provider` interface, payment request
and result types, and a test-only `FakeProvider`. The `purchases` service gains
an `InitiatePayment` operation that validates an existing pending purchase
belonging to the authenticated user, builds a trusted payment request from the
immutable purchase snapshot, and delegates to the configured payment provider.
An HTTP endpoint (`POST /api/v1/me/purchases/:purchaseID/payment`) exposes
payment initiation to the client. The purchase status is NOT modified during
initiation — it remains "pending" until a verified provider event flows through
`CompletePurchase()`. No real payment provider is implemented; a
`notConfiguredPaymentProvider` placeholder is wired in production routes.

Database:

- No new migration. The existing `purchases` and `entitlements` tables already
  contain the required fields.

Behavior:

- `services/payments/provider.go` — `Provider` interface with a single
  `InitiatePayment(ctx, PaymentRequest) (PaymentResult, error)` method.
- `services/payments/types.go` — `PaymentRequest`, `PaymentResult`, and
  `PaymentStatus` types (`pending`, `requires_action`, `failed`).
- `services/payments/errors.go` — `ErrProviderFailure` sentinel error.
- `services/payments/fake_provider.go` — deterministic test-only fake provider.
- `services/purchases/purchases.go` — `InitiatePayment(userID, purchaseID)`
  validates ownership and pending status, calls the provider, returns the
  result. Errors: `ErrPurchaseNotFound`, `ErrPurchaseNotPending`,
  `ErrPaymentProvider`.
- `api/auth/purchase_handler.go` — `InitiatePayment` handler maps service
  errors to HTTP responses (404, 409, 502, 500).
- `routes/routes.go` — wired with `notConfiguredPaymentProvider` placeholder.
  New route: `POST /api/v1/me/purchases/:purchaseID/payment` behind
  `Authenticate` middleware.

Tests:

- 3 provider unit tests (success, deterministic ids, failure).
- 9 service unit tests (success, empty inputs, not found, repo failure, IDOR,
  not pending, provider failure, snapshot values).
- 5 handler unit tests (identity forwarding, missing context, error mapping,
  repo failure not exposed, no sensitive data).
- 5 handler integration tests (success, unauthenticated, not found, not
  pending, IDOR).

Modified files:

- services/payments/provider.go (new)
- services/payments/types.go (new)
- services/payments/errors.go (new)
- services/payments/fake_provider.go (new)
- services/payments/provider_test.go (new)
- services/purchases/purchases.go (modified)
- services/purchases/purchases_test.go (modified)
- api/auth/purchase_handler.go (modified)
- api/auth/purchase_handler_test.go (modified)
- routes/routes.go (modified)
- docs/REGISTRY/update_logs.MD (modified)

# Version 2.33.0

Title: Backend Step 45 — Purchase Completion Foundation

Date: 2026-08-20

Summary:

Added the internal `CompletePurchase` service operation that a future payment
provider adapter will call once external payment confirmation is received. The
method atomically transitions a pending purchase to "completed" and provisions
the corresponding entitlement. Idempotency is guaranteed: calling
`CompletePurchase` on an already-completed purchase with an existing entitlement
returns a success; if the entitlement is missing a data-integrity error is
raised. Entitlement reactivation is supported for previously soft-deleted
entitlements. No client-facing endpoint is exposed; the operation is
internal-only and designed for webhook or adapter invocation.

Database:

- No new migration. The existing `purchases` and `entitlements` tables already
  contain the required fields.

Behavior:

- `CompletePurchase(purchaseID)` — internal service method (no HTTP endpoint).
  Loads the purchase, validates it is in "pending" status, provisions the
  entitlement atomically and returns the updated purchase snapshot. Returns
  `ErrPurchaseNotFound` for missing or soft-deleted purchases,
  `ErrPurchaseNotCompleted` for terminal states (failed, refunded),
  `ErrEntitlementIntegrity` when the purchase is completed but the entitlement
  is missing. Idempotent when called repeatedly on the same completed purchase
  with an existing entitlement.
- `RestoreByUserAndProgram` — new entitlement repository method that clears
  `deleted_at` on a soft-deleted entitlement, enabling reactivation when a user
  repurchases a program.
- `Complete` — new purchase repository method that atomically transitions a
  pending purchase to "completed".
- `CompleteWithEntitlement` — new purchase repository method that atomically
  completes the purchase and creates the entitlement inside a single database
  transaction.

Files:

- `repositories/purchase_repository.go` — added `Complete` and
  `CompleteWithEntitlement` methods.
- `repositories/entitlement_repository.go` — added `RestoreByUserAndProgram`
  method.
- `services/purchases/purchases.go` — added `CompletePurchase` service method,
  `completeAndCreateEntitlement` private helper, expanded repository interfaces,
  added completion-specific error types (`ErrPurchaseNotFound`,
  `ErrPurchaseNotCompleted`, `ErrEntitlementIntegrity`), added
  `validatePurchaseID` validator.
- `api/auth/purchase_handler_test.go` — extended stub to satisfy the expanded
  `Service` interface.

Tests:

- 10 existing `CreatePurchaseIntent` tests pass unchanged.
- 10 new `CompletePurchase` service tests covering: success via
  `CompleteWithEntitlement`, idempotent success (already completed + existing
  entitlement), entitlement integrity error (completed + missing entitlement),
  purchase not found, empty purchase ID validation, failed status rejection,
  refunded status rejection, entitlement reactivation path, snapshot
  preservation, internal error non-exposure.
- 15 existing handler tests (7 integration with real DB) pass unchanged.

Reason:

Establishes the internal purchase completion operation that bridges the
commercial transaction foundation (Step 44) with a future payment-provider
adapter. The idempotency semantics and atomic entitlement provisioning ensure
the system remains consistent under retries and partial failures.

---

Title: Backend Step 44 — Purchase & Entitlement Transaction Foundation

Date: 2026-08-20

Summary:

Connected program pricing, commission rules, and entitlement creation into a
trusted purchase transaction. The `POST /api/v1/me/programs/:programID/purchase`
endpoint validates the program, snapshots the current price and currency,
resolves the applicable commission (trainer-specific override or global default),
calculates the deterministic split, and persists a pending purchase record. The
purchase status is always "pending"; no payment is processed and no entitlement
is created. A future payment provider will transition the purchase to "completed"
and create the entitlement atomically. Duplicate active entitlements and
duplicate pending purchases per (user, program) pair are prevented.

Database:

- Migration `000016` — extended the `purchases` table with snapshot fields:
  `price_minor_units` (`BIGINT NOT NULL DEFAULT 0`), `currency`
  (`VARCHAR(3) NOT NULL DEFAULT 'EUR'`), `commission_bps`
  (`INT UNSIGNED NOT NULL DEFAULT 0`), `platform_amount`
  (`BIGINT NOT NULL DEFAULT 0`), `trainer_amount`
  (`BIGINT NOT NULL DEFAULT 0`). Changed default `status` from `'completed'`
  to `'pending'` and migrated existing rows.

Behavior:

- `POST /api/v1/me/programs/:programID/purchase` (authenticated client) —
  validates the program is published and not free, checks no active entitlement
  exists, checks no active purchase exists, snapshots price/currency/commission,
  calculates split, persists a pending purchase. Returns 201 with the purchase
  snapshot or appropriate error codes: `PROGRAM_NOT_FOUND` (404),
  `PROGRAM_NOT_PURCHASABLE` (409), `DUPLICATE_ENTITLEMENT` (409),
  `DUPLICATE_PURCHASE` (409).
- Free programs cannot be purchased through the commercial transaction path.
- Draft or soft-deleted programs are invisible (404).
- The response never exposes the owning user id or any internal data.

Files:

- `database/migrations/000016_extend_purchases_with_commercial_snapshots.up.sql`
- `database/migrations/000016_extend_purchases_with_commercial_snapshots.down.sql`
- `models/purchase.go` — added snapshot fields and changed default status.
- `repositories/purchase_repository.go` — new repository with `Create`,
  `FindByID`, `FindActiveByUserAndProgram`.
- `repositories/entitlement_repository.go` — added
  `FindActiveByUserAndProgram`.
- `services/purchases/purchases.go` — new service with validation, program
  lookup, duplicate checks, commission resolution and purchase creation.
- `api/auth/purchase_handler.go` — new handler for purchase creation.
- `routes/routes.go` — wired purchase service, handler and route; added
  `commissionAdapter` to bridge commission and purchase types.

Tests:

- 10 service tests covering success, validation, duplicate detection,
  commission resolution and error handling.
- 13 handler tests covering authentication, error mapping, identity
  forwarding, security, and integration scenarios.

Reason:

Establishes the internal commercial transaction correctly so that a future
payment provider can plug into it. No payment processing, checkout or webhook
handling is implemented — this is a foundation for future commerce.

---

# Version 2.31.0

Title: Backend Step 43 — Commercial Rules & Trainer Commission Foundation

Date: 2026-08-19

Summary:

Added the trainer commission rules system as the foundation for commercial
rules. The platform now supports trainer-specific commission overrides expressed
as basis points (1 bps = 0.01%). When no override exists for a trainer, the
global default commission applies (env var `DEFAULT_PLATFORM_COMMISSION_BPS`,
default 2000 = 20%). Commission resolution is deterministic: trainer-specific
active override → global default. Commission calculation uses floor division to
avoid rounding ambiguity: `platform = floor(price * bps / 10000)`,
`trainer = price - platform`. The Management Administrator (ADMIN_2) manages
commission rules through the `plans.commission.manage` permission. No payment
processing, purchase creation, revenue tracking or payout logic is implemented
— this is a foundation for future commerce.

Database:

- Migration `000015` — created the `commission_rules` table with `id`
  (UUID PK), `trainer_id` (FK to `trainers.id`), `commission_bps`
  (`INT UNSIGNED NOT NULL`), `valid_from` (`DATETIME(6)`, default now),
  `valid_until` (`DATETIME(6) NULL`), timestamps, soft-delete, and a
  generated `active_commission_rule` column with a unique index enforcing
  at most one active override per trainer. Check constraint ensures
  `commission_bps <= 10000`.

Behavior:

- `GET /api/v1/admin/trainers/:id/commission` (admin,
  `plans.commission.manage` permission) — returns the active commission rule
  for a trainer, or 404 when no override exists.
- `PATCH /api/v1/admin/trainers/:id/commission` (admin,
  `plans.commission.manage` permission) — creates or replaces the active
  commission rule for a trainer. Previous active rules are soft-deleted.
- `DELETE /api/v1/admin/trainers/:id/commission` (admin,
  `plans.commission.manage` permission) — soft-deletes the active commission
  rule; the trainer falls back to the global default.
- `GET /api/v1/admin/trainers/:id/commission/resolve` (admin,
  `plans.commission.manage` permission) — returns the resolved commission
  (override or default) with `is_override` flag.
- Environment variable `DEFAULT_PLATFORM_COMMISSION_BPS` controls the global
  default commission (default 2000 = 20%, max 10000 = 100%).

Files:

- `database/migrations/000015_create_commission_rules_table.up.sql`
- `database/migrations/000015_create_commission_rules_table.down.sql`
- `models/commission_rule.go` — new model.
- `config/config.go` — added `CommissionConfig` struct.
- `config/env.go` — added `LoadCommission()` function.
- `repositories/commission_rule_repository.go` — new repository.
- `services/commission_rules/commission_rules.go` — new service with
  validation, rule resolution and deterministic calculation.
- `api/auth/admin_commission_handler.go` — new handler for admin commission
  operations.
- `middleware/adminroles/adminroles.go` — added
  `PermissionPlansCommissionManage` permission.
- `routes/routes.go` — wired admin commission routes.
- `main.go` — loads commission config and passes to routes.
- `services/commission_rules/commission_rules_test.go` — 22 service tests.
- `api/auth/admin_commission_handler_test.go` — 13 handler tests.

---

# Version 2.30.0

Title: Backend Step 42 — Program Pricing & Currency Foundation

Date: 2026-08-19

Summary:

Added configurable selling price and currency to programs using the integer
minor-unit model (cents for EUR). Programs now carry `price_minor_units`
(`BIGINT`) and `currency` (`VARCHAR(3)`, EUR only initially). Free programs
must have price 0; paid programs must meet the configurable minimum. Both
trainers (via existing PATCH) and authorized administrators (Management
Administrator via `plans` permission) can change prices. Pricing is exposed in
all public, client and entitlement responses. No payment processing or purchase
creation is implemented — this is a foundation for future commerce.

Database:

- Migration `000014` — added `price_minor_units BIGINT NOT NULL DEFAULT 0`
  and `currency VARCHAR(3) NOT NULL DEFAULT 'EUR'` to the `programs` table,
  with an index on `currency`. Existing programs receive default values
  (price=0, currency='EUR').

Behavior:

- `POST /api/v1/trainer/programs` — accepts optional `price_minor_units` and
  `currency` fields. Free programs must have price 0; paid programs must meet
  the configured minimum.
- `PATCH /api/v1/trainer/programs/:programID` — accepts optional
  `price_minor_units` and `currency` fields with the same validation.
- `GET /api/v1/programs` (public) — includes `price_minor_units` and `currency`
  in response.
- `GET /api/v1/programs/:programID` (public) — includes pricing in response.
- `GET /api/v1/me/program` (client) — includes pricing in response.
- `GET /api/v1/me/entitlements` — nested program includes pricing.
- `GET /api/v1/admin/programs/:programID` (admin, `plans` permission) — returns
  program with pricing.
- `PATCH /api/v1/admin/programs/:programID/pricing` (admin, `plans` permission)
  — updates `price_minor_units` and `currency` of any active program.
- Environment variable `MIN_PROGRAM_PRICE_MINOR_UNITS` controls the minimum
  price for paid programs (default 100 = €1.00).

Files:

- `database/migrations/000014_add_pricing_to_programs.up.sql`
- `database/migrations/000014_add_pricing_to_programs.down.sql`
- `models/program.go` — added `PriceMinorUnits`, `Currency` fields and
  `ProgramCurrencyEUR` constant.
- `config/config.go` — added `PricingConfig` struct.
- `config/env.go` — added `LoadPricing()` function.
- `repositories/program_repository.go` — added `FindByID` and `UpdatePricing`.
- `services/programs/programs.go` — extended DTOs, added pricing validation,
  `UpdateProgramPricing`, `GetProgramByID`.
- `services/admin_program_pricing/admin_program_pricing.go` — new service for
  admin pricing operations.
- `api/auth/trainer_programs_handler.go` — extended request/response DTOs.
- `api/auth/admin_program_pricing_handler.go` — new handler for admin pricing.
- `api/auth/public_programs_handler.go` — exposed pricing in response.
- `api/auth/client_program_handler.go` — exposed pricing in response.
- `api/auth/entitlements_handler.go` — exposed pricing in safe Program DTO.
- `routes/routes.go` — wired admin pricing routes.
- `main.go` — loads pricing config and passes to routes.

---

# Version 2.29.0

Title: Backend Step 41 — program purchase / entitlement foundation

Date: 2026-08-19

Summary:

Established the domain separation between published programs, purchases and
entitlements without implementing real payment processing. Added `purchases`
and `entitlements` tables, `EntitlementRepository`, entitlements service,
`GET /api/v1/me/entitlements` endpoint and comprehensive tests. Entitlements
currently provide only safe metadata (entitlement id + nested safe Program);
they do NOT yet grant client access to `GET /me/program` — assignment remains
the sole access mechanism.

Database:

- Migration `000013` — created `purchases` and `entitlements` tables.
- `purchases` — records a user's purchase of a published program with status
  (`pending`, `completed`, `failed`, `refunded`), CHECK-constrained. No
  price/currency/amount/provider fields yet. No creation path exists; the table
  is prepared for future commerce integration.
- `entitlements` — records that a user has a purchase-backed right to access a
  program. The `active_entitlement` generated column enforces at most one active
  entitlement per `(user_id, program_id)` pair. Foreign keys reference `users`
  and `programs`.

Behavior:

- `GET /api/v1/me/entitlements`, mounted behind `Authenticate`:
  returns `200 OK` with a safe entitlement metadata array (entitlement id,
  program_id, nested safe Program object with id, name, description, type,
  status, created_at, updated_at); returns `401 AUTHENTICATION_REQUIRED` when
  unauthenticated. Entitlements do NOT grant access to `GET /me/program` — the
  regression test `TestClientProgramReadEntitlementDoesNotGrantAccess` proves
  an entitlement without an assignment yields 404 on `/me/program`.

Files:

- `database/migrations/000013_create_purchases_and_entitlements_tables.up.sql` —
  purchases and entitlements DDL.
- `database/migrations/000013_create_purchases_and_entitlements_tables.down.sql` —
  rollback.
- `models/purchase.go` — `Purchase` model with status constants, foreign keys
  and `BeforeCreate` UUID generation.
- `models/entitlement.go` — `Entitlement` model with foreign keys and
  `BeforeCreate` UUID generation.
- `repositories/entitlement_repository.go` — dedicated
  `EntitlementRepository` interface and implementation with `Create`,
  `ListByUser`, `FindByIDAndUser`, `SoftDelete`. Uses `isDuplicateEntry` for
  the generated-column unique constraint.
- `repositories/entitlement_repository_test.go` — 12 tests covering create,
  duplicate rejection, list by user, find by id, cross-user isolation, soft
  delete, re-creation after delete and empty list.
- `services/entitlements/entitlements.go` — entitlements service providing
  `ListEntitlements`, `CreateEntitlement`, `RevokeEntitlement` with input
  validation and domain error mapping.
- `services/entitlements/entitlements_test.go` — 11 stub-repo unit tests
  covering success, empty list, invalid input, duplicate collapse, not-found
  collapse and repository failure non-exposure.
- `api/auth/entitlements_handler.go` — `EntitlementsHandler` exposing
  `ListEntitlements` with standard auth-context identity extraction.
- `api/auth/entitlements_handler_test.go` — 9 tests: E2E (success, empty list,
  unauthenticated, IDOR isolation), handler-only (context forwarding, missing
  context, error mapping, repo failure non-exposure, secrets never exposed).
- `routes/routes.go` — wired `GET /me/entitlements` behind `Authenticate`.
- `api/auth/client_program_handler_test.go` — added
  `TestClientProgramReadEntitlementDoesNotGrantAccess` regression test.

---

# Version 2.28.0

Title: Backend Step 40 — public program discovery foundation

Date: 2026-08-18

Summary:

Added two unauthenticated, read-only public endpoints for program discovery:
`GET /api/v1/programs` (paginated list) and `GET /api/v1/programs/:programID`
(single program). These endpoints expose only published, non-deleted programs
and require no authentication. The catalog is global: programs from all trainers
are visible to every caller. Draft, soft-deleted and unknown programs all return
the same generic 404 Not Found (`PROGRAM_NOT_FOUND`) to prevent information
leakage. No purchase, assignment, payment or access semantics are introduced.

Database:

- No new tables or migrations. The existing `programs.status` CHECK constraint
  already supports the `published` value used by the public catalog.

Behavior:

- `GET /api/v1/programs`, mounted without authentication on the public `v1`
  group: returns `200 OK` with a paginated array of published programs and
  pagination metadata (`page`, `limit`, `total`, `total_pages`); returns `400
  Bad Request` (`VALIDATION_ERROR`) for invalid pagination parameters.
- `GET /api/v1/programs/:programID`, mounted without authentication: returns
  `200 OK` with one published program; returns `404 Not Found`
  (`PROGRAM_NOT_FOUND`) for missing, draft, soft-deleted or unknown programs;
  returns `400 Bad Request` (`VALIDATION_ERROR`) for a malformed program id.

Files:

- `repositories/program_repository.go` — added `ListPublished(ctx, page, limit)`
  and `FindPublishedByID(ctx, programID)` methods to the `ProgramRepository`
  interface and `programRepository` implementation. `ListPublished` returns only
  programs with `status = 'published'` ordered by `created_at DESC`. Both
  methods are unscoped: no trainer ownership is checked.
- `services/public_programs/public_programs.go` — new service package providing
  `ListPublishedPrograms` and `GetPublishedProgram` through a read-only
  `ProgramRepository` interface limited to the two public methods.
- `api/auth/public_programs_handler.go` — new handler exposing
  `ListPublishedPrograms` and `GetPublishedProgram` with the same pagination and
  error-mapping conventions as the exercises handler.
- `routes/routes.go` — wired `GET /programs` and `GET /programs/:programID` in
  the public `v1` route group, outside any authentication middleware.
- `repositories/program_repository_test.go` — added tests for `ListPublished`
  (published programs only, excludes drafts, excludes soft-deleted, pagination,
  cross-trainer visibility) and `FindPublishedByID` (published, draft hidden,
  soft-deleted hidden, unknown id, platform-owned draft).
- `services/public_programs/public_programs_test.go` — added unit tests for
  `ListPublishedPrograms` (success, limit clamping, invalid input, repository
  failure) and `GetPublishedProgram` (success, invalid IDs, not found,
  repository failure, no secrets exposed).
- `api/auth/public_programs_handler_test.go` — added E2E and unit tests:
  `TestPublicProgramsListSuccess`, `TestPublicProgramsListExcludesDrafts`,
  `TestPublicProgramsListInvalidPagination`, `TestPublicProgramsGetSuccess`,
  `TestPublicProgramsGetNotFound`, `TestPublicProgramsGetDraftHidden`,
  `TestPublicProgramsGetSoftDeletedHidden`, `TestPublicProgramsGetInvalidProgramID`,
  `TestPublicProgramsNeverRequireAuthentication`,
  `TestPublicProgramsHandlerErrorMapping`,
  `TestPublicProgramsHandlerRepositoryFailureNotExposed`,
  `TestPublicProgramsHandlerGetForwardsPathProgramID`,
  `TestPublicProgramsHandlerListForwardsPagination`,
  `TestPublicProgramsHandlerInvalidPagination`,
  `TestPublicProgramsHandlerInvalidPathID`,
  `TestPublicProgramsNeverExposesSecrets`,
  `TestPublicProgramsFindsCrossTrainerPrograms`.

---

# Version 2.27.0

Title: Backend Step 39 — trainer program publishing foundation

Date: 2026-08-18

Summary:

Added a dedicated publish endpoint `POST /api/v1/trainer/programs/:programID/publish`
that transitions a draft program to published through a race-safe conditional
update. The operation is scoped exclusively to the authenticated trainer; the
program id in the URL path identifies the requested resource and never proves
access. A client-supplied `trainer_id`, `user_id` or `client_id` (query, body or
header) is ignored and can never influence the publish operation. The only
allowed transition is draft → published; publishing an already-published program
returns a deterministic 409 Conflict (`PROGRAM_ALREADY_PUBLISHED`). A missing,
soft-deleted or foreign program returns 404 Not Found (`PROGRAM_NOT_FOUND`).

Database:

- No new tables or migrations. The existing `programs.status` CHECK constraint
  already supports `draft` and `published` values.

Behavior:

- `POST /api/v1/trainer/programs/:programID/publish`, mounted behind
  `Authenticate → TrainerAuthenticate → RequireTrainerPermission(trainer.programs)`:
  returns `200 OK` with the updated program data on success; returns `400 Bad
  Request` (`VALIDATION_ERROR`) for a malformed program id; returns `404 Not
  Found` (`PROGRAM_NOT_FOUND`) for missing, soft-deleted or foreign programs;
  returns `409 Conflict` (`PROGRAM_ALREADY_PUBLISHED`) when the program is
  already in published state; returns `401 Unauthorized`
  (`AUTHENTICATION_REQUIRED`) when unauthenticated; returns `403 Forbidden`
  (`FORBIDDEN`) when the trainer role does not include `trainer.programs`.

Files:

- `repositories/program_repository.go` — added `Publish(ctx, trainerID, programID)`
  method with conditional `UPDATE ... SET status='published' WHERE id=? AND
  trainer_id=? AND deleted_at IS NULL AND status='draft'` query.
- `services/programs/programs.go` — added `ErrProgramAlreadyPublished` error,
  `PublishProgram` method on both the `Service` interface and `service`
  implementation. Distinguishes "already published" from other failure modes by
  reloading the program after a repository `ErrProgramNotFound` response.
- `api/auth/trainer_programs_handler.go` — added `PublishProgram` handler
  method and `PROGRAM_ALREADY_PUBLISHED` error mapping (409 Conflict).
- `routes/routes.go` — wired `POST /programs/:programID/publish` in the trainer
  programs route group.
- `repositories/program_repository_test.go` — added tests for successful
  publish, idempotent already-published, foreign publish, missing program and
  soft-deleted program scenarios.
- `services/programs/programs_test.go` — added `TestPublishProgramSuccess`,
  `TestPublishProgramAlreadyPublished`, `TestPublishProgramNotFound`,
  `TestPublishProgramRejectsInvalidIDs`, `TestPublishProgramRepositoryFailure`.
- `api/auth/trainer_programs_handler_test.go` — added `TestTrainerProgramsPublishSuccess`,
  `TestTrainerProgramsPublishIDOR`, `TestTrainerProgramsPublishAlreadyPublished`,
  `TestTrainerProgramsPublishMissing`, `TestTrainerProgramsPublishInvalidProgramID`,
  `TestTrainerProgramsPublishSoftDeleted`,
  `TestTrainerProgramsHandlerPublishForwardsPathProgramID`; updated
  `TestTrainerProgramsNotAuthenticated` and
  `TestTrainerProgramsHandlerMissingContext` with publish cases; added
  `ErrProgramAlreadyPublished` to `TestTrainerProgramsHandlerErrorMapping`.

---

# Version 2.26.0

Title: Backend Step 38 — client workout statistics foundation

Date: 2026-08-18

Summary:

Added a read-only statistics endpoint `GET /api/v1/me/statistics` that returns
derived workout statistics for the authenticated client. Statistics are computed
entirely from existing tables — no new database tables or migrations were added.
The client identity always comes exclusively from the authentication context; a
client-supplied `user_id`, `client_id` or `trainer_id` (query, body or header)
is ignored and can never influence which statistics are returned. The response
distinguishes total executions from unique workouts completed, computes a
completion percentage based on the number of unique workouts completed against
the total active workouts in the currently assigned program, and reports the
last workout date. A user with no active assignment receives zero-value stats
with `has_active_assignment=false`.

Database:

- No new tables or migrations. All statistics are derived from existing tables
  (`workout_history`, `program_assignments`, `programs`, `program_weeks`,
  `program_workouts`) through SQL aggregation.

Behavior:

- `GET /api/v1/me/statistics`, mounted behind `Authenticate` only (no trainer
  permission): returns `200 OK` with `has_active_assignment`,
  `current_program_name`, `total_executions`, `unique_workouts_completed`,
  `total_workouts_in_program`, `completion_percentage` and `last_workout_date`.
- Repository `GetClientStats(userID)` finds the user's most recent active
  assignment, resolves the assigned program, counts active workouts in the
  program through the week chain, counts total executions and distinct completed
  workouts from history, and finds the most recent completion timestamp.
- Errors: `400 VALIDATION_ERROR`, `401 AUTHENTICATION_REQUIRED`,
  `500 INTERNAL_ERROR` (details never exposed).
- Security: identity always comes from the authentication context; the response
  never exposes the owning `user_id`, assignment id, trainer id or any deletion
  markers. IDOR tests verify a stranger can never see another user's statistics.

Architecture notes:

- No new frameworks or libraries. The `services/statistics` service and
  `api/auth/statistics_handler.go` follow the existing service/handler/repository
  layering, the safe-representation pattern and the response conventions of the
  client-facing endpoints.
- The endpoint lives in the authenticated-user area (`/api/v1/me/statistics`),
  matching the Step 36/37 client endpoints convention.
- The statistics layer is deliberately minimal and read-only: no speculative
  metrics (sets, reps, weight, RPE, calories, duration, HR, PRs, body
  measurements) are included. The layer stays extensible so later steps can add
  richer metrics without a schema redesign.

Files:

- `main/src/backend/repositories/statistics_repository.go`
- `main/src/backend/services/statistics/statistics.go`
- `main/src/backend/api/auth/statistics_handler.go`
- `main/src/backend/routes/routes.go`
- Tests: `repositories/statistics_repository_test.go`,
  `services/statistics/statistics_test.go`,
  `api/auth/statistics_handler_test.go`
- Docs: `docs/REGISTRY/update_logs.MD`

---

# Version 2.25.0

Title: Backend Step 37 — client workout history (workout completion and history read)

Date: 2026-08-13

Summary:

Added the client workout history foundation: a `workout_history` table plus two
authenticated-user endpoints — `POST /api/v1/me/workouts/{workoutID}/complete`
records that the authenticated user completed one workout of their currently
assigned program, and `GET /api/v1/me/workouts/history` returns the user's own
completed workouts, newest first, with pagination. Completion access is derived
exclusively from the assignment tree `assignments → programs → program_weeks →
program_workouts`: a workout is only executable when it belongs to the active
structure of the user's most recent active assignment. The client identity
always comes exclusively from the authentication context; a client-supplied
`user_id`, `client_id` or `trainer_id` (query, body or header) is ignored and
can never influence which workout is completed or which history is returned.
Repeated executions are preserved as separate append-only entries; completion is
recorded server-side. Missing, soft-deleted or foreign workouts and missing
assignments are all indistinguishable and map to `404 WORKOUT_NOT_FOUND`.

Database:

- New migration `000012_create_workout_history_table` (up/down): the
  `workout_history` table with `id`, `user_id` (FK `users.id`), `program_workout_id`
  (FK `program_workouts.id`), `completed_at`, `created_at`, `updated_at`,
  `deleted_at`. Indexes on `user_id` and `program_workout_id`. No unique
  constraint on `(user_id, program_workout_id)` because repeated executions are
  preserved. No cascade in either direction so historical references stay
  coherent.
- New `models/workout_history.go` with a `BeforeCreate` hook that generates the
  UUID and sets the server-side `completed_at`, and an explicit
  `TableName` returning `workout_history`.

Behavior:

- `POST /api/v1/me/workouts/{workoutID}/complete`, mounted behind `Authenticate`
  only (no trainer permission): verifies the chain
  `user → most recent active assignment → active program → active week → active
  workout` and inserts one append-only entry. Returns `201 Created` with `id`,
  `program_workout_id`, `completed_at`, `created_at`, `updated_at`.
- `GET /api/v1/me/workouts/history`, mounted behind `Authenticate` only: returns
  one page of the authenticated user's own entries ordered by `completed_at DESC`
  plus `pagination` (`page`, `limit`, `total`, `total_pages`). Defaults
  `page=1`, `limit=20`, capped at `100`.
- Repository `Create(userID, programWorkoutID)` reuses `ErrWorkoutNotFound` (the
  existing trainer-area error) for every break in the chain; `ListByUser(userID,
  page, limit)` is scoped strictly to the authenticated user.
- Errors: `400 VALIDATION_ERROR`, `401 AUTHENTICATION_REQUIRED`,
  `404 WORKOUT_NOT_FOUND` (no active assignment, soft-deleted assignment/program/
  week/workout, foreign workout and unknown workout id are all
  indistinguishable), `500 INTERNAL_ERROR` (details never exposed).
- Security: identity always comes from the authentication context; the entry and
  history responses never expose the owning `user_id`, the trainer id or any
  parent identifier. IDOR tests verify a stranger can never complete another
  user's workout or read another user's history, that client-supplied identity
  is ignored, and that no secrets leak.

Architecture notes:

- No new frameworks or libraries. The `services/workout_history` service and
  `api/auth/workout_history_handler.go` follow the existing
  service/handler/repository layering, the safe-representation pattern and the
  pagination conventions of the trainer-facing list endpoints.
- The endpoints live in the authenticated-user area (`/api/v1/me/...`), matching
  the Step 36 client program read convention; the aspirational `# WORKOUT
  ENDPOINTS` paths in api.MD were updated to match.
- The write path is deliberately minimal and append-only: no entry can be
  modified, removed or reordered by the client, and no per-exercise training
  data (sets, reps, rest, weights, duration) exists yet — the entity stays
  minimal so later steps can add it without a schema redesign.

Files:

- `main/src/backend/database/migrations/000012_create_workout_history_table.up.sql`,
  `...down.sql`
- `main/src/backend/models/workout_history.go`
- `main/src/backend/repositories/workout_history_repository.go`
- `main/src/backend/services/workout_history/workout_history.go`
- `main/src/backend/api/auth/workout_history_handler.go`
- `main/src/backend/routes/routes.go`
- Tests: `repositories/workout_history_repository_test.go`,
  `services/workout_history/workout_history_test.go`,
  `api/auth/workout_history_handler_test.go`,
  `routes/workout_history_e2e_test.go`
- Docs: `docs/INSTRUCTIONS/api.MD`, `docs/INSTRUCTIONS/security.MD`,
  `docs/INSTRUCTIONS/arquitecture.MD`, `docs/DATABASE/database.MD`,
  `docs/REGISTRY/update_logs.MD`

---

# Version 2.24.0

Title: Backend Step 36 — client program read (the client's view of the assigned program)

Date: 2026-08-13

Summary:

Added a read-only, authenticated-user endpoint `GET /api/v1/me/program` that
returns the full active structure of the program currently assigned to the
authenticated user: weeks, workouts, workout exercises and the safe public
catalog summary of each exercise. The client identity always comes exclusively
from the authentication context; a client-supplied `user_id`, `client_id` or
`trainer_id` (query, body or header) is ignored and can never influence which
program is returned. The view is strictly read-only and reuses the existing
assignment tree — no schema, table or migration was added. A user holding
several active assignments (one per trainer) reads the most recently created
one; missing, soft-deleted or unreadable assignments/programs are
indistinguishable and map to `404 PROGRAM_NOT_FOUND`.

Database:

- No migration. The read reuses the existing `program_assignments`,
  `programs`, `program_weeks`, `program_workouts`, `workout_exercises` and
  `exercises` tables.
- The `Program` and `ProgramWorkout` models gained the `Weeks` and `Exercises`
  preload associations (mapped columns already existed), so the whole structure
  is loaded in one tree via GORM preloading.

Behavior:

- `GET /api/v1/me/program`, mounted behind `Authenticate` only (no trainer
  permission): returns `200 OK` with the assigned program (`id`, `name`,
  `description`, `type`, `status`, `created_at`, `updated_at`, `weeks`), each
  week with `id`, `week_number`, `created_at`, `updated_at`, `workouts`, each
  workout with `id`, `position`, `created_at`, `updated_at`, `exercises`, each
  workout exercise with `id`, `position`, `created_at`, `updated_at` and the
  safe public `exercise` summary (`id`, `name`, `description`,
  `target_muscles`, `equipment`, `difficulty`, `video_url`, `image_url`).
- Weeks, workouts and workout exercises are returned in display order
  (`week_number`/`position` ascending). Programs without structure yet return
  empty arrays. Soft-deleted programs, weeks, workouts, workout exercises,
  exercises and assignments are excluded.
- The program `status` carries no access semantics: a draft program is as
  readable as a published one.
- The owning `trainer_id` and every parent identifier (`program_id`,
  `program_week_id`, `program_workout_id`) are never exposed; only the leaf
  node ids of the returned tree appear.
- Repository `FindAssignedProgram(userID)` returns the most recent active
  assignment of the user (no trainer scope, no trainer-client relationship
  check — the user owns the read on their own account) with the full program
  structure preloaded.
- Errors: `401 AUTHENTICATION_REQUIRED`, `404 PROGRAM_NOT_FOUND` (covers no
  active assignment, a soft-deleted assignment and a soft-deleted/unreadable
  program, all indistinguishable), `500 INTERNAL_ERROR` (details never
  exposed).
- Security: identity always comes from the authentication context; IDOR tests
  verify a stranger's cookie can never read another user's program, that
  client-supplied identity is ignored, and that no secrets leak.

Architecture notes:

- No new frameworks or libraries. The `services/client_programs` service and
  `api/auth/client_program_handler.go` follow the existing
  service/handler/repository layering and the safe-representation pattern,
  reusing `trainerExerciseSummary`.
- The endpoint lives in the authenticated-user area, not the trainer area: it
  is the client's own read view and is guarded by the user authentication alone.
- The one-active-assignment rule keeps living in the database (unique generated
  column); the client read just picks the most recent active assignment.

Files:

- `main/src/backend/models/program.go`, `models/program_workout.go`
- `main/src/backend/repositories/program_assignment_repository.go`
- `main/src/backend/services/client_programs/client_programs.go`
- `main/src/backend/api/auth/client_program_handler.go`
- `main/src/backend/routes/routes.go`
- Tests: `services/client_programs/client_programs_test.go`,
  `api/auth/client_program_handler_test.go`,
  `routes/client_program_e2e_test.go`,
  `repositories/program_assignment_repository_test.go`
- Docs: `docs/INSTRUCTIONS/api.MD`, `docs/INSTRUCTIONS/security.MD`,
  `docs/INSTRUCTIONS/arquitecture.MD`, `docs/REGISTRY/update_logs.MD`

---

# Version 2.23.0

Title: Backend Step 35 — trainer program assignments (one active assigned program per client)

Date: 2026-08-13

Summary:

Introduced program assignments: a `program_assignments` table plus a complete,
tested trainer-area API that assigns one of the trainer's own programs to one
of the trainer's active clients. This gives trainers a first-class way to
deliver one of their own programs to one of their clients. A trainer can have
at most one **active** assigned program per client — enforced by the database
itself through a unique generated column, mirroring the established
`trainer_clients`/`program_weeks`/`program_workouts` patterns. The client keeps
being a regular `User`; no second identity is ever created. Removing an
assignment only soft-deletes the assignment row; the program and the
trainer-client relationship are never touched.

Entity and database:

- Migration `000011_create_program_assignments_table` (`up`/`down`, applied at
  version 11 dirty=false): `program_assignments` table with `id` (UUID,
  generated by the backend), `trainer_id` (`VARCHAR(36)`, FK
  `fk_program_assignments_trainer` referencing `trainers(id)`, indexed),
  `program_id` (`VARCHAR(36)`, FK `fk_program_assignments_program` referencing
  `programs(id)`, indexed), `user_id` (`VARCHAR(36)`, FK
  `fk_program_assignments_user` referencing `users(id)`, indexed),
  `created_at`/`updated_at` (`DATETIME(6)`), `deleted_at` for soft-delete and
  the generated column `active_assignment`
  (`IF(deleted_at IS NULL, CONCAT(trainer_id, ':', user_id), NULL)`, unique
  `uq_program_assignments_active_assignment`) — the database itself guarantees
  one active assigned program per client per trainer, while removed and
  historical assignments are preserved.
- The trainer id is stored explicitly on every row so every query is scoped by
  it; it is always set from the authenticated trainer context, never accepted
  from the client.
- No cascade is defined in either direction: soft-deleting a program or
  removing a client relationship never touches existing assignment rows, and
  soft-deleting an assignment never touches the program or the relationship.
- `models.ProgramAssignment` mirrors the table, generates its UUID in
  `BeforeCreate` and preloads the safe program data.
- Repository `ProgramAssignmentRepository` (`Create`, `ListByClient`,
  `FindByIDAndClient`, `SoftDelete`) scopes every query by the trainer id and
  by the active trainer-client relationship; a client that does not exist, is
  not managed by the trainer or whose relationship was soft-deleted is
  indistinguishable from a client without assignments.

Behavior:

- Endpoints under `/api/v1/trainer/clients/:userID/programs`, all guarded by
  `Authenticate` + `TrainerAuthenticate` +
  `RequireTrainerPermission(trainer.clients)` (the same permission as the
  client relationship itself; no new permission was invented):
  - `POST` — accepts only the `program_id`; the trainer identity and the client
    identity come from the trainer context and the path respectively. The
    program must be active and owned by the trainer and the client must be an
    active client of the trainer; the one-active-assignment rule is enforced.
  - `GET` — every active assignment of the client in assignment order, each
    with the safe program data embedded. A client-supplied `trainer_id` query
    is ignored; a foreign or removed client yields an empty list.
  - `DELETE .../programs/:assignmentID` — soft-deletes only the assignment row;
    the program and the trainer-client relationship survive.
- Errors: `400 VALIDATION_ERROR`, `404 CLIENT_RELATION_NOT_FOUND`,
  `404 PROGRAM_NOT_FOUND`, `404 ASSIGNMENT_NOT_FOUND`,
  `409 ASSIGNMENT_ALREADY_ACTIVE`, `500 INTERNAL_ERROR` (details never
  exposed).
- Security: the trainer identity always comes from the trainer context; a
  client-supplied `trainer_id` in the body or query is ignored. An assignment
  that is missing, soft-deleted, belongs to another client or whose
  relationship is not active is indistinguishable and maps to the same
  not-found error (no enumeration). Responses never expose deletion markers,
  passwords, sessions or JWT data.

Architecture notes:

- No new frameworks or libraries. The service
  (`services/program_assignments`) and handler
  (`api/auth/trainer_client_program_handler.go`) follow the existing
  service/handler/repository layering and the safe-representation pattern.
- The one-active-assignment rule lives in the database (unique generated
  column), not in the application.
- The database DSN now enables `MultiStatements` because the golang-migrate
  runner executes every migration file as a single multi-statement batch; GORM
  itself never uses multi-statement batches.

Files:

- `main/src/backend/database/migrations/000011_create_program_assignments_table.up.sql`
  and `.down.sql`
- `main/src/backend/models/program_assignment.go`
- `main/src/backend/repositories/program_assignment_repository.go`
- `main/src/backend/services/program_assignments/program_assignments.go`
- `main/src/backend/api/auth/trainer_client_program_handler.go`
- `main/src/backend/routes/routes.go`
- `main/src/backend/database/database.go`
- Tests: `repositories/program_assignment_repository_test.go`,
  `services/program_assignments/program_assignments_test.go`,
  `api/auth/trainer_client_program_handler_test.go`
- Docs: `docs/DATABASE/database.MD`, `docs/INSTRUCTIONS/api.MD`,
  `docs/INSTRUCTIONS/security.MD`, `docs/INSTRUCTIONS/arquitecture.MD`,
  `docs/REGISTRY/update_logs.MD`

---

# Version 2.22.0

Title: Backend Step 34 — workout exercise foundation (`ProgramWorkout → WorkoutExercise → Exercise`)

Date: 2026-08-13

Summary:

Established the workout exercise foundation: a `workout_exercises` table plus
a complete, tested trainer-area API that assigns global catalog exercises to a
program workout. This completes the `Program → Week → Workout → WorkoutExercise
→ Exercise` chain. The documentation defines no per-assignment training
parameters (sets, reps, rest, tempo, weights, notes, progression) for this
entity, so the assignment stays minimal (`id`, `program_workout_id`,
`exercise_id`, `position` and lifecycle fields) and later steps can add
training data without a schema redesign.

Entity and database:

- Migration `000010_create_workout_exercises_table` (`up`/`down`,
  up/down/up verified, version 10 dirty=false): `workout_exercises` table with
  `id` (UUID, generated by the backend), `program_workout_id` (`VARCHAR(36)`,
  FK `fk_workout_exercises_workout` referencing `program_workouts(id)`,
  indexed), `exercise_id` (`VARCHAR(36)`, FK `fk_workout_exercises_exercise`
  referencing `exercises(id)`, indexed), `position` (`INT NOT NULL`,
  `CHECK (position > 0)`), `created_at`/`updated_at` (`DATETIME(6)`),
  `deleted_at` for soft-delete and the generated column
  `active_workout_exercise` (`IF(deleted_at IS NULL, CONCAT(program_workout_id,
  ':', position), NULL)`, unique `uq_workout_exercises_active_position`) — the
  database itself guarantees one active exercise per position per workout.
- No cascade is defined in either direction: the catalog is independent and its
  soft delete never touches existing workout exercises, so historical
  references stay coherent.
- The same exercise may appear more than once in the same workout (position
  distinguishes each assignment); there is no `UNIQUE(program_workout_id,
  exercise_id)`. This is a documented decision.
- `models.WorkoutExercise` mirrors the table, generates its UUID in
  `BeforeCreate` and preloads the safe catalog `Exercise` data.
- Repository `WorkoutExerciseRepository` (`AddExercise`, `ListByWorkout`,
  `FindByIDAndWorkout`, `Reorder`, `SoftDelete`) scopes every query through the
  full ownership chain `trainer → program → week → workout → workout_exercise`.
  No trainer id is stored on the rows.

Behavior:

- Endpoints under `/api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises`,
  all guarded by `Authenticate` + `TrainerAuthenticate` +
  `RequireTrainerPermission(trainer.programs)`:
  - `POST` (append) — accepts only the catalog `exercise_id`; position is
    computed server-side (`max(active position) + 1`) inside a transaction that
    locks the owning workout row; a missing or soft-deleted exercise is
    rejected (`404 EXERCISE_NOT_FOUND`).
  - `GET` (list) — every active entry in position order, each with the safe
    catalog exercise data embedded.
  - `PATCH .../exercises/order` — two-phase reorder that requires the exact
    active set (`409 REORDER_CONFLICT` otherwise).
  - `GET .../exercises/:workoutExerciseID` — one entry scoped by the full chain.
  - `DELETE .../exercises/:workoutExerciseID` — soft-deletes only the entry row.
- There is no update endpoint: like weeks and workouts, changing an assignment
  is delete + create. The API follows the existing week/workout patterns.
- Errors: `400 VALIDATION_ERROR`, `404 WORKOUT_NOT_FOUND`,
  `404 EXERCISE_NOT_FOUND`, `404 WORKOUT_EXERCISE_NOT_FOUND`,
  `409 REORDER_CONFLICT`, `500 INTERNAL_ERROR` (details never exposed).
- Security: the trainer identity always comes from the trainer context; a
  client-supplied `trainer_id`/`program_id`/`week_id`/`workout_id` in the body
  is ignored. A workout exercise that is missing, soft-deleted, under another
  workout or under a foreign chain is indistinguishable and maps to the same
  not-found error (no enumeration). Responses never expose deletion markers,
  passwords, sessions or JWT data.

Architecture notes:

- No new frameworks or libraries. The service (`services/workout_exercises`)
  and handler (`api/auth/trainer_workout_exercise_handler.go`) follow the
  existing service/handler/repository layering, the safe-representation pattern
  and the established error-envelope and reorder patterns (two-phase release).
- Ownership is derived through the tree, never stored: the whole area keeps the
  single `trainer.programs` permission; no `trainer.workout_exercises`
  permission was invented.

Files:

- `main/src/backend/database/migrations/000010_create_workout_exercises_table.up.sql`
  and `.down.sql`
- `main/src/backend/models/workout_exercise.go`
- `main/src/backend/repositories/workout_exercise_repository.go`
- `main/src/backend/services/workout_exercises/workout_exercises.go`
- `main/src/backend/api/auth/trainer_workout_exercise_handler.go`
- `main/src/backend/routes/routes.go`
- Tests: `repositories/workout_exercise_repository_test.go`,
  `services/workout_exercises/workout_exercises_test.go`,
  `api/auth/trainer_workout_exercise_handler_test.go`
- Docs: `docs/DATABASE/database.MD`, `docs/INSTRUCTIONS/api.MD`,
  `docs/INSTRUCTIONS/security.MD`, `docs/INSTRUCTIONS/arquitecture.MD`,
  `docs/REGISTRY/update_logs.MD`

---

# Version 2.21.0

Title: Backend Step 33 — exercise library foundation (public read-only catalog)

Date: 2026-08-13

Summary:

Established the platform exercise library foundation: a single `exercises`
table plus a complete, tested public read-only catalog API
(`list`, `get` and `search` by name). An exercise is a global,
platform-owned catalog entry that exists only once and is reused across
programs and packs. In this foundation the catalog is deliberately
read-only through the API: no trainer or admin write endpoint exists, the
repository exposes only read operations, and the catalog is populated
outside the application (seed data / future admin tooling).

Entity and database:

- Migration `000009_create_exercises_table` (`up`/`down`): `exercises` table
  with `id` (UUID, generated by the backend), `name` (`VARCHAR(255)`,
  required), `description` (optional text), `target_muscles`,
  `equipment`, `difficulty` (optional free text), `video_url`/`image_url`
  (optional URL references), `created_at`/`updated_at` (`DATETIME(6)`),
  `deleted_at` for soft-delete, and the generated column `active_name`
  (`IF(deleted_at IS NULL, name, NULL)`, unique
  `uq_exercises_active_name`) — the database itself guarantees every active
  exercise exists only once, and soft-deleting an exercise frees its name for
  a new catalog entry.
- `models.Exercise` mirrors the table and generates its UUID in
  `BeforeCreate`. No foreign keys: the catalog has no ownership relationship.
- Repository `ExerciseRepository` exposes only reads (`FindByID`, `List`
  alphabetical, `Search` by case-insensitive name substring with LIKE
  wildcards escaped) so that no write path can be invented.

Behavior:

- Public endpoints under `/api/v1/exercises`, requiring no authentication and
  identical for every caller:
  - `GET /api/v1/exercises?page=&limit=` — one page of the catalog in
    alphabetical order.
  - `GET /api/v1/exercises/:exerciseID` — one active catalog entry.
  - `GET /api/v1/exercises/search?q=&page=&limit=` — one page of entries whose
    name contains the query, case-insensitively; the `q` parameter is
    required.
- Pagination follows the platform rules (page ≥ 1, limit clamped to 100,
  `data.pagination` reports `page`, `limit`, `total`, `total_pages`).
- Errors: malformed ids/pagination/blank search map to
  `400 VALIDATION_ERROR`; unknown or soft-deleted entries map to
  `404 EXERCISE_NOT_FOUND`; internal details are never exposed.
- Soft-deleted exercises disappear from every public read.

Architecture notes:

- No new frameworks or libraries. The service (`services/exercises`) follows
  the existing service/repository/handler layering and the established
  safe-representation pattern (only public metadata leaves the service).
- The catalog schema uses free text for `target_muscles`, `equipment` and
  `difficulty` per the confirmed product decision; no enums or normalized
  tables were invented.

---

# Version 2.20.0

Title: Backend Step 32 — trainer program structure foundation (weeks and workouts)

Date: 2026-08-13

Summary:

Established the program structure foundation: `program_weeks` and
`program_workouts` tables plus a complete, tested week/workout CRUD + reorder
area for the trainer programs endpoint space. A program now contains ordered
week slots and each week contains ordered workout slots; both are empty
containers for now (no exercise assignments, sets or names yet) whose ordering
and lifecycle are the product of this step.

Entity and database:

- Migration `000007_create_program_structure_tables` (`up`/`down`):
  `program_weeks` table with `id` (UUID), `program_id` (`VARCHAR(36)`, FK
  `fk_program_weeks_program` referencing `programs(id)`, indexed), `week_number`
  (`INT NOT NULL`, `CHECK (week_number > 0)`), `created_at`/`updated_at`
  (`DATETIME(6)`), `deleted_at` for soft-delete and the generated column
  `active_week` (`IF(deleted_at IS NULL, CONCAT(program_id, ':', week_number),
  NULL)`, unique `uq_program_weeks_active_week`) — a real stored constraint so
  the database itself guarantees one active week per `week_number` per program.
- Migration `000008_create_program_workouts_table` (`up`/`down`):
  `program_workouts` table with `id` (UUID), `program_week_id` (`VARCHAR(36)`,
  FK `fk_program_workouts_week` referencing `program_weeks(id)`, indexed),
  `position` (`INT NOT NULL`, `CHECK (position > 0)`), timestamps, `deleted_at`
  and the generated column `active_workout` (`IF(deleted_at IS NULL,
  CONCAT(program_week_id, ':', position), NULL)`, unique
  `uq_program_workouts_active_workout`).
- Each migration file carries exactly one statement because the golang-migrate
  MySQL driver executes one statement per file (the earlier multi-statement
  draft failed with `Error 1064` and was split into the two migrations above).
- `models.ProgramWeek` and `models.ProgramWorkout` mirror the tables, generate
  their UUIDs in `BeforeCreate`; `ProgramWeek.Workouts` preloads the active
  workouts in position order.
- Soft-deletes via `deleted_at` everywhere; deleting a week leaves its workouts
  preserved but unreachable because every workout query is scoped through an
  active week.

Behavior:

- Week endpoints under `/api/v1/trainer/programs/:programID/weeks`, workout
  endpoints under `/weeks/:weekID/workouts`, all protected by the chain
  `Authenticate` → `TrainerAuthenticate` →
  `RequireTrainerPermission(trainer.programs)`:
  - `POST .../weeks` — appends an empty week at the end (`week_number` =
    `max(active)+1`) inside a transaction locking the owning program row
    (`201`).
  - `GET .../weeks` — active weeks in week order, each with active workouts in
    position order.
  - `PATCH .../weeks/order` (`ids` body) — replaces the order of every active
    week.
  - `GET` / `DELETE .../weeks/:weekID` — read one week; soft-delete one week.
  - `POST .../weeks/:weekID/workouts` — appends an empty workout at the end
    (`position` = `max(active)+1`) inside a transaction locking the owning week
    row (`201`).
  - `GET .../weeks/:weekID/workouts`, `GET`/`DELETE` of
    `.../workouts/:workoutID` — read/list/soft-delete workouts.
  - `PATCH .../weeks/:weekID/workouts/order` (`ids` body) — replaces the order
    of every active workout.
- Reorder semantics: the list must be the exact set of active entries, each
  exactly once; anything else is rejected before any write with
  `409 REORDER_CONFLICT`. The reassignment runs in two phases inside one
  transaction — every active slot is first released to `slot + 1000000` (a
  value far outside the final `1..n` range, keeping the `> 0` checks valid),
  then each entry is assigned its final slot — so the unique generated column
  never observes a collision.
- Ownership/IDOR protection follows the programs step: identity always comes
  exclusively from `trainercontext`; every repository query is scoped by the
  trainer through program/week subqueries; missing, soft-deleted or foreign
  weeks/workouts are indistinguishable (`404 WEEK_NOT_FOUND` /
  `404 WORKOUT_NOT_FOUND`); a foreign program yields `404 PROGRAM_NOT_FOUND`.
- Errors: malformed identifiers or reorder lists → `400 VALIDATION_ERROR`;
  reorder mismatch → `409 REORDER_CONFLICT`; missing/invalid auth → `401`;
  non-trainer or unheld permission → `403 FORBIDDEN`; internal failures →
  `500 INTERNAL_ERROR` with no details exposed. Responses expose only safe
  structural metadata (`id`, parent id, `week_number`/`position`, timestamps)
  and never include deletion markers.

Not implemented yet:

- No exercise assignments, sets, reps, rest or any workout content.
- No week/workout names or descriptions.
- No client program assignment, marketplace, purchases or payments.
- No frontend changes.

Verification: `gofmt -w`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes against MariaDB. Migrations verified
up/down/up to version 8. Repository tests cover append ordering, owner/week
scoping, cross-trainer and cross-program isolation, exact-permutation reorder
(including rejected lists), two-phase reorder persistence, soft-delete hiding
with row preservation and unreachable-but-preserved workouts after a week
delete. Service tests cover validation, error mapping, identity forwarding and
a structural guarantee that the safe `Week`/`Workout` representations expose no
sensitive fields. HTTP tests cover the success paths, cross-trainer IDOR for
weeks and workouts, reorder success and conflict, soft-delete effects, 401/403
paths, invalid JSON, no-secret-leak guarantees and the handler's own error
mapping and context/path/order forwarding.

Modified files:

- `main/src/backend/database/migrations/000007_create_program_structure_tables.up.sql`
- `main/src/backend/database/migrations/000007_create_program_structure_tables.down.sql`
- `main/src/backend/database/migrations/000008_create_program_workouts_table.up.sql`
- `main/src/backend/database/migrations/000008_create_program_workouts_table.down.sql`
- `main/src/backend/models/program_week.go`
- `main/src/backend/models/program_workout.go`
- `main/src/backend/repositories/program_week_repository.go`
- `main/src/backend/repositories/program_week_repository_test.go`
- `main/src/backend/repositories/program_workout_repository.go`
- `main/src/backend/repositories/program_workout_repository_test.go`
- `main/src/backend/services/program_structure/program_structure.go`
- `main/src/backend/services/program_structure/program_structure_test.go`
- `main/src/backend/api/auth/trainer_program_structure_handler.go`
- `main/src/backend/api/auth/trainer_program_structure_handler_test.go`
- `main/src/backend/routes/routes.go`
- `docs/DATABASE/database.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Programs are a core trainer capability and a program without structure is not
yet usable by clients. This step gives every trainer program an ordered
week/workout skeleton with database-enforced ordering integrity (the generated
unique columns make the `1..n` uniqueness a real constraint, not an application
rule), safe server-side ownership scoping and a documented reorder contract.
Keeping weeks and workouts as empty, position-identified containers means the
future content fields (exercises, sets, rest) extend these entities without
structural change, and the reorder mechanics are locked by tests before any
content feature is built on top. The `trainer.programs` permission and the
middleware chain from Step 31 are reused unchanged, and this step also closes
the documentation gap by adding the `Programs` database section and the full
trainer programs API contract that Step 31 had left only in the log.

---

# Version 2.19.0

Title: Backend Step 31 — trainer programs foundation

Date: 2026-08-13

Summary:

Established the `programs` entity as the foundation for trainer program
management. A program is a training or nutrition offer created by a trainer,
with an ownership model that is **nullable**: `trainer_id` may be NULL, which
is structurally reserved for future platform-owned programs, while the trainer
API always creates programs owned by the authenticated trainer and can never
produce a platform-owned one. The entity, repository, service, handler and five
`trainer.programs`-protected routes now form a complete, tested trainer program
CRUD area.

Entity and database:

- Migration `000006_create_programs_table` (`up`/`down`): `programs` table with
  `id` (UUID), `trainer_id` (`VARCHAR(36) NULL`, FK `fk_programs_trainer`
  referencing `trainers(id)`, indexed), `name` (`VARCHAR(255) NOT NULL`),
  `description` (`TEXT NULL`), `type` (`VARCHAR(20)`, CHECK
  `('free','premium','personalized')`), `status` (`VARCHAR(20)`, CHECK
  `('draft','published')`), `created_at`/`updated_at` (`DATETIME(6)`) and
  `deleted_at` for soft-delete.
- Nullable ownership model: platform-owned programs are allowed at the database
  level (`trainer_id = NULL`); a repository test inserts a real NULL via raw SQL
  and proves such a row is structurally valid yet never leaks into any trainer
  list.
- Soft-delete via `deleted_at`: the model uses GORM `DeletedAt`, regular queries
  exclude soft-deleted rows, and deleting never removes the row from the
  database.
- `status` is `draft` or `published`; publishing is only a state and carries no
  purchase, assignment or access semantics.
- `models.Program` mirrors the table and generates its UUID in `BeforeCreate`.

Behavior:

- Trainer program CRUD is currently implemented under `/api/v1/trainer`,
  protected by the chain `Authenticate` → `TrainerAuthenticate` →
  `RequireTrainerPermission(trainer.programs)`:
  - `GET /programs` — paginated active programs of the authenticated trainer,
    newest first, with `data.pagination` (`page`, `limit`, `total`,
    `total_pages`); the limit is clamped to the platform maximum.
  - `POST /programs` — creates a program owned by the authenticated trainer
    (`201`); status defaults to `draft` when omitted; the body never accepts
    `trainer_id`.
  - `GET /programs/:programID` — one of the authenticated trainer's own active
    programs.
  - `PATCH /programs/:programID` — updates only the whitelisted fields (`name`,
    `description`, `type`, `status`); at least one field is required and
    `trainer_id` is immutable.
  - `DELETE /programs/:programID` — soft-deletes the program row only; the
    trainer profile is never touched.
- Ownership/IDOR protection: the trainer identity always comes exclusively from
  `trainercontext`; client-supplied `trainer_id` values in the body, query or
  header are ignored; every read/update/delete is scoped by both the owner
  trainer and the program id; a missing, soft-deleted or foreign program all map
  to the same `404 PROGRAM_NOT_FOUND` so the endpoints can never enumerate or
  reveal another trainer's programs.
- Errors: malformed identifiers, invalid JSON, invalid fields or invalid
  pagination → `400 VALIDATION_ERROR`; missing/invalid auth → `401`; non-trainer
  or unheld `trainer.programs` → `403 FORBIDDEN`; internal failures →
  `500 INTERNAL_ERROR` with no details exposed. Responses expose only safe
  program metadata and never include deletion markers.
- The `trainer.programs` permission and the middleware chain already existed
  (Steps 25/26/27); this step mounts the first program routes on them.

Not implemented yet:

- No program → client assignment relationship.
- No marketplace, purchases or payments.
- No platform-owned program creation endpoint, no program content (workouts,
  nutrition), no admin program management and no frontend changes.

Verification: `gofmt -w`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes against MariaDB. Repository tests cover the
create/list/find/update/soft-delete lifecycle, newest-first ordering, ownership
scoping, soft-deleted exclusion, foreign-trainer isolation, NULL `trainer_id`
structural allowance without list leakage, unknown-trainer FK rejection and the
`name` NOT NULL invariant. Service tests cover validation, status defaulting,
pagination clamping, error mapping and a structural guarantee that the safe
`Program` representation exposes no sensitive fields. HTTP tests cover the
success paths, the mandatory IDOR tests in both directions for get/update/
delete, body/query/header identity ignoring, immutable `trainer_id`, soft-delete
hiding, invalid input/JSON/pagination, 401/403 paths, no-secret-leak guarantees
and the handler's own error mapping and context forwarding.

Modified files:

- `main/src/backend/database/migrations/000006_create_programs_table.up.sql`
- `main/src/backend/database/migrations/000006_create_programs_table.down.sql`
- `main/src/backend/models/program.go`
- `main/src/backend/repositories/program_repository.go`
- `main/src/backend/repositories/program_repository_test.go`
- `main/src/backend/services/programs/programs.go`
- `main/src/backend/services/programs/programs_test.go`
- `main/src/backend/api/auth/trainer_programs_handler.go`
- `main/src/backend/api/auth/trainer_programs_handler_test.go`
- `main/src/backend/routes/routes.go`
- `docs/REGISTRY/update_logs.MD`

Reason:

Trainer programs are a core trainer capability (matching the `trainer.programs`
permission and the "create/manage training and nutrition programs" product
scope). The program is modeled as an independent, soft-deletable, trainer-owned
entity so trainer and client lifecycles never couple, ownership is guaranteed
server-side by the trainer context, and the nullable `trainer_id` leaves a
documented, DB-validated path for future platform-owned programs. The endpoint
contract, ownership rules and security guarantees are locked with repository,
service and HTTP tests before any assignment, marketplace or purchase feature
is built on top.

---

# Version 2.18.0

Title: Backend Step 30 — trainer client management foundation

Date: 2026-08-13

Summary:

Analysis-first step: audited the trainer→client relationship against the
documentation and the real implementation to confirm the `trainer_clients`
table and the `trainer.clients` area already form a complete, consistent
foundation. No code, schema or behavior changed; the foundation principles
were made explicit and locked in the architecture documentation.

Analysis conclusions:

- `trainer_clients` (migration `000005`) already represents "trainer manages
  client": a first-class relationship with an independent lifecycle
  (active/removed via soft-delete) and stored constraints (unique active
  relation, no self-relationship).
- No states or fields are missing: add, list, read, remove and reactivate
  fully cover the current scope; the relationship carries no business fields
  and none was added.
- Docs and code are consistent: `database.MD`, `api.MD`, `security.MD`,
  `arquitecture.MD` and `project.MD` match the model, repository, service,
  handler, routes and `trainerroles`.
- No migration is needed.

Foundation decisions (now explicit in `arquitecture.MD`):

- The `TrainerClient` lifecycle is fully independent of the `User` and
  `Trainer` lifecycles; relationship operations never touch the account or the
  profile.
- A trainer can never create, update or delete a client's user data through the
  trainer area; the client profile is a read-only safe view.
- The area is guarded by the single `trainer.clients` permission; no
  sub-permissions and no branching on finer-grained roles.
- Business fields (progress, goals, diet, statistics) are only introduced when
  a real feature requires them.

Modified files:

- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Client management was built incrementally (Steps 26–29) and the relationship,
endpoints, security rules and tests already follow the intended model. This
step locks that model down as an explicit, documented foundation so future
trainer-client work stays within its bounds and does not duplicate the user
lifecycle, invent permissions or add speculative fields.

---

# Version 2.17.0

Title: Backend Step 29 — trainer client profile endpoint (`GET /trainer/clients/:userID`)

Date: 2026-08-13

Summary:

Added the single-client profile read so an authenticated trainer can retrieve
the safe profile of one of their active clients. The endpoint reuses the
existing `TrainerClientsHandler`, `trainer_clients` service and
`TrainerClientRepository`; ownership is guaranteed server-side because the
trainer identity always comes exclusively from `trainercontext` and the query
is scoped by both the trainer id and the requested user id. The active
relationship and the linked user are verified together in one query.

Behavior:

- `GET /api/v1/trainer/clients/:userID` returns the same client shape as the
  list and add endpoints (`id`, `trainer_id`, `user_id`, nested safe `user`,
  `created_at`/`updated_at`); the `userID` path parameter only identifies the
  requested resource and never proves access. A client-supplied `trainer_id`
  query value is ignored.
- Ownership rule: access exists only when an active `trainer_clients`
  relationship matches the authenticated trainer and the requested user
  (`deleted_at IS NULL` on the relationship).
- A user that does not exist, is soft-deleted, is not an active client of the
  trainer or whose relationship was soft-deleted all map to the same
  `404 CLIENT_RELATION_NOT_FOUND`, so the endpoint can never enumerate other
  users' existence (anti-IDOR, anti-enumeration).
- Malformed identifiers → `400 VALIDATION_ERROR`; missing/invalid auth or
  trainer context → `401`; non-trainer or unheld `trainer.clients` → `403`;
  internal failures → `500 INTERNAL_ERROR` with no details exposed.
- Response exposes only the client's public user fields; password hashes,
  session versions, deletion markers and trainer-specific fitness data (weight,
  goals, diet, progress, PRs, programs, statistics, payments) are never
  returned.
- No new migration: the `trainer_clients` table already existed.

Verification: `gofmt -w .`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes. Service tests cover success, invalid
identifiers, missing relationship and soft-deleted-user hiding; HTTP tests cover
the success path, the mandatory IDOR test in both directions (trainer A can
never read trainer B's client and vice versa), query/header/body identity
ignoring, a user without any relationship, unknown user, soft-deleted user,
soft-deleted relationship, malformed user id, invalid token, non-trainer,
permission-missing and no-secret-leak guarantees.

Modified files:

- `main/src/backend/services/trainer_clients/trainer_clients.go`
- `main/src/backend/services/trainer_clients/trainer_clients_test.go`
- `main/src/backend/api/auth/trainer_clients_handler.go`
- `main/src/backend/api/auth/trainer_clients_handler_test.go`
- `main/src/backend/routes/routes.go`
- `docs/INSTRUCTIONS/api.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Trainer client management already exposed list, add, remove and reactivate
operations; a single-client profile read completes the `trainer.clients` area
with the smallest coherent change, reusing the existing relationship model,
ownership guarantees and safe response shape while locking the anti-enumeration
and IDOR protections with dedicated tests.

---

# Version 2.16.0

Title: Backend Step 28 — trainer client management (relationship, endpoints and tests)

Date: 2026-08-13

Summary:

Implemented the trainer→client relationship as a first-class entity: the
`trainer_clients` table (migration `000005`, with a MariaDB stored generated
unique column enforcing one **active** relation per `(trainer_id, user_id)` and
a `CHECK` constraint forbidding self-relationships), the `TrainerClient` model,
`TrainerClientRepository`, the `trainer_clients` service and four trainer-area
routes under `trainer.clients`. The trainer identity always comes exclusively
from `trainercontext`; client-supplied `trainer_id` values (query, body or
path) are never accepted. Removing a client soft-deletes only the relationship
row — the user account and trainer profile are never touched — and
reactivation restores the exact same row.

Behavior:

- `GET /api/v1/trainer/clients` — paginated active clients, newest first;
  client-supplied `trainer_id` query values are ignored.
- `POST /api/v1/trainer/clients` — adds an existing active user as a client
  (`201`); soft-deleted or missing user → `404 CLIENT_NOT_FOUND`; duplicate
  active relation → `409 CLIENT_ALREADY_ADDED`.
- `DELETE /api/v1/trainer/clients/:userID` — soft-deletes only the relation;
  missing relation → `404 CLIENT_RELATION_NOT_FOUND`.
- `POST /api/v1/trainer/clients/:userID/reactivate` — restores the same row;
  an already-active relation can never be reactivated.
- Malformed identifiers, invalid JSON body or invalid pagination →
  `400 VALIDATION_ERROR`; missing/invalid auth → `401`; non-trainer or unheld
  `trainer.clients` → `403`; internal failures → `500 INTERNAL_ERROR` with no
  details exposed.
- Responses expose only the relationship identity and the client's public user
  fields; password hashes, session versions and deletion markers never leave
  the server.

Also fixed a test-suite-wide connection leak: every `api/auth` handler-test
router opened a fresh `database.Connect` pool and never closed it, so the
cumulative open connections exceeded MariaDB's `max_connections` (151) and the
package failed with `Error 1040 (08004): Too many connections`. The ten router
helpers now close the `*sql.DB` handle in their cleanup; the whole suite passes
again.

Verification: `gofmt -w .`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes. The migration was applied to the development
MariaDB and validated with a down/up cycle (version 5, dirty=false, table
restored). Repository, service and HTTP tests cover lifecycle, isolation
(other trainers can never reach a relation), self-relationship rejection,
active-user invariants and no-leak/security guarantees.

Modified files:

- `main/src/backend/database/migrations/000005_create_trainer_clients_table.up.sql`
- `main/src/backend/database/migrations/000005_create_trainer_clients_table.down.sql`
- `main/src/backend/models/trainer_client.go`
- `main/src/backend/repositories/trainer_client_repository.go`
- `main/src/backend/repositories/trainer_client_repository_test.go`
- `main/src/backend/services/trainer_clients/trainer_clients.go`
- `main/src/backend/services/trainer_clients/trainer_clients_test.go`
- `main/src/backend/api/auth/trainer_clients_handler.go`
- `main/src/backend/api/auth/trainer_clients_handler_test.go`
- `main/src/backend/routes/routes.go`
- `main/src/backend/api/auth/admin_trainers_handler_test.go`
- `main/src/backend/api/auth/admin_users_handler_test.go`
- `main/src/backend/api/auth/change_password_handler_test.go`
- `main/src/backend/api/auth/delete_account_handler_test.go`
- `main/src/backend/api/auth/login_handler_test.go`
- `main/src/backend/api/auth/me_handler_test.go`
- `main/src/backend/api/auth/register_handler_test.go`
- `main/src/backend/api/auth/trainer_application_handler_test.go`
- `main/src/backend/api/auth/trainer_profile_handler_test.go`
- `docs/DATABASE/database.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Trainer client management is a core trainer capability (matching the
`trainer.clients` permission and the "manage clients" product scope). The
relationship is modeled as an independent, soft-deletable entity so trainer and
client lifecycles never couple, ownership is guaranteed server-side by the
trainer context, and the endpoint contract and security guarantees are locked
with repository, service and HTTP tests.

---

# Version 2.15.0

Title: Backend Step 27 — trainer profile endpoint and HTTP test coverage

Date: 2026-08-13

Summary:

Added `GET /api/v1/trainer/profile`, the first trainer-area route protected by
the trainer authorization chain from v2.14.0 and v2.13.0, and a full HTTP
test suite for it. The endpoint is served by
`auth.NewTrainerProfileHandler` over `services/trainer_profile`, reads the
trainer identity exclusively from `trainercontext.IdentityFromContext`, never
trusts client-supplied `user_id` / `trainer_id` query values, and returns the
trainer's own profile with nested safe user details (password hashes, session
versions and soft-delete flags never leave the server).

Behavior:

- Authenticated active trainer with `trainer.profile` → HTTP 200 with the
  trainer `id`, `user_id`, `created_at`, `updated_at` and nested `user`
  object.
- Missing/invalid token → HTTP 401 `AUTHENTICATION_REQUIRED`.
- Authenticated non-trainer user → HTTP 403 `FORBIDDEN`.
- Trainer without the required permission (route mounted with an unheld
  permission) → HTTP 403 `FORBIDDEN`, no permission leak.
- Query-supplied `user_id` / `trainer_id` are ignored; the authenticated
  identity always wins.
- Missing, wrong-type or empty trainer context identity → HTTP 401.
- Trainer not found → HTTP 401 `AUTHENTICATION_REQUIRED`.
- Repository/internal failure → HTTP 500 `INTERNAL_ERROR` with no internal
  details exposed.

Verification: `gofmt -w .`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes. `api/auth/trainer_profile_handler_test.go`
covers the full middleware chain (Authenticate + TrainerAuthenticate +
RequireTrainerPermission) against a database transaction, the handler's own
error mapping via a stubbed service, identity-ignoring and no-leak checks.

Modified files:

- `main/src/backend/api/auth/trainer_profile_handler_test.go`
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

First protected trainer route: prove the authorization chain end-to-end at the
HTTP level, lock the profile response contract and security guarantees with
tests, and keep the documentation consistent with the implementation.

---

# Version 2.14.0

Title: Backend Step 26 — trainer authorization middleware

Date: 2026-08-13

Summary:

Implemented `middleware.RequireTrainerPermission`, the trainer route
authorization middleware that runs strictly after `TrainerAuthenticate` and
mirrors the admin authorization layer. It reads the authenticated trainer
identity exclusively from `trainercontext.IdentityFromContext` and never
trusts client-supplied user IDs, trainer IDs, permissions, roles, headers,
query or body values.

Behavior:

- Valid trainer identity + granted required permission (at least one is
  enough, OR semantics) → `c.Next()`.
- Missing or invalid trainer identity (absent, wrong type, empty `UserID` /
  `TrainerID` or non-UUID values) → HTTP 401 `AUTHENTICATION_REQUIRED`.
- Unknown or unheld required permissions → HTTP 403 `FORBIDDEN`, fail-closed,
  with the generic envelope that never reveals the required permissions or the
  trainer identity.
- The middleware depends only on `trainercontext`, `trainerroles`, Gin and the
  shared error envelope; it never touches the database, JWT, token service or
  any repository. No trainer routes are protected yet.

Verification: `gofmt -w .`, `go build ./...` and `go vet ./...` clean;
`go test ./... -count=1` passes. `middleware/trainer_authorization_test.go`
covers every official permission granted, OR semantics, unknown permissions
denied, no required permissions denied, request-supplied identity and
permission claims ignored, missing/wrong-type/empty/invalid identities
rejected with 401, and fail-closed misconfigured mounts, plus no-leak checks
for both error codes.

Modified files:

- `main/src/backend/middleware/trainer_authorization.go`
- `main/src/backend/middleware/trainer_authorization_test.go`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Step 26 of the backend roadmap: enforce trainer route authorization server-side
before any trainer endpoints exist, reusing the permission model from v2.13.0
and the trainer authentication chain from v2.12.1, so future trainer routes
only need to mount `TrainerAuthenticate` + `RequireTrainerPermission`.

---

# Version 2.13.0

Title: Backend Step 25 — trainer permission model foundation

Date: 2026-08-13

Summary:

Established the trainer authorization foundation as a separate concern from
trainer authentication (implemented in v2.12.1: `middleware/trainercontext` +
`middleware.TrainerAuthenticate`, resolving an authenticated user into an
active trainer and storing the `UserID` + `TrainerID` in the trainer context);
this step adds the permission model an active trainer may hold.

The new `middleware/trainerroles` package follows the same architectural
pattern as `middleware/adminroles` and depends on no framework, HTTP, database,
JWT or repository:

- `Permission` — the trainer permission type.
- `PermissionProfile` / `trainer.profile` — the trainer's own profile and
  dashboard area.
- `PermissionClients` / `trainer.clients` — the trainer's client management
  area.
- `PermissionPrograms` / `trainer.programs` — the trainer's programs area.
- `PermissionStatistics` / `trainer.statistics` — the trainer's statistics
  area.
- `AllPermissions` — the deterministic official permission set.
- `IsKnownPermission` / `PermissionFromString` — official-permission
  recognition; unknown strings are rejected with `ErrUnknownPermission` so no
  arbitrary string can become a valid trainer permission.
- `HasPermission` — the single centralized trainer grant check. The initial
  grant mapping grants every base permission to every active trainer (there
  are no trainer tiers or sub-roles yet).

A `RequireTrainerPermission` middleware is intentionally deferred: when it
arrives it will run strictly after `TrainerAuthenticate`, mirroring the admin
authorization layer.

Verification: `go build ./...`, `go vet ./middleware/trainerroles/` and
`gofmt -l` clean; `go test ./middleware/trainerroles/ -count=1` passes. New
package tests cover the deterministic official set, no duplicates, every
official permission recognized, valid permissions granted, unknown permissions
denied, `PermissionFromString` round-trips and rejection of unknown strings,
and the initial grant mapping covering the official set.

Modified files:

- `main/src/backend/middleware/trainerroles/trainerroles.go`
- `main/src/backend/middleware/trainerroles/trainerroles_test.go`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Step 25 of the backend roadmap: build the trainer authorization layer
separately from trainer authentication, providing the centralized permission
definitions and grant mapping the future trainer endpoints and
`RequireTrainerPermission` middleware will rely on, without introducing
business logic, middleware or any dependency on the existing admin or trainer
systems.

---

# Version 2.12.1

Title: Backend Step 24 — trainer authentication

Date: 2026-08-12

Summary:

Implemented trainer authentication as a separate concern from user
authentication and from trainer authorization. A trainer is an extension of an
existing user account: trainer authentication proves that the authenticated
user owns an active trainer profile, and nothing else.

The new `middleware/trainercontext` package stores the authenticated trainer
identity in the request context through a dedicated context/helper package:

- `Identity` — the authenticated trainer identity (`UserID` + `TrainerID`).
- `SetIdentity` — stores the identity in the Gin context (used by
  `TrainerAuthenticate`).
- `IdentityFromContext` / `UserIDFromContext` / `TrainerIDFromContext` —
  safe reads; both UUIDs must be valid and non-empty, otherwise
  `ErrNoAuthenticatedTrainer` is returned.

The new `middleware.TrainerAuthenticate` middleware runs strictly after the
user authentication middleware (`Authenticate`) and requires the user
authentication context. It:

- obtains the `UserID` exclusively from the authenticated user context;
- resolves the trainer through `TrainerRepository.FindByUserID`;
- only active trainers pass — an authenticated user without a trainer profile
  receives HTTP 403 `FORBIDDEN`;
- a soft-deleted trainer also receives HTTP 403 `FORBIDDEN` (the repository
  only returns active trainers);
- stores `UserID` + `TrainerID` in the trainer context through
  `trainercontext.SetIdentity`;
- never trusts a pre-existing trainer context: any stale identity is
  overwritten;
- missing or invalid user authentication receives HTTP 401
  `AUTHENTICATION_REQUIRED` (handled by `Authenticate` itself);
- internal repository failures receive HTTP 500 `INTERNAL_ERROR` with a
  generic message that never leaks internal details.

The layer chain is:

```text
Authenticate
    ↓
TrainerAuthenticate
    ↓
Trainer Authorization (separate layer, added in a later step)
    ↓
Handler
```

Trainer authentication only proves the authenticated user is an active
trainer. Trainer authorization is a separate concern built in a later step
(`middleware/trainerroles`); no permission checks exist in this version and no
endpoint is protected by trainer permissions yet.

Verification: `go build ./...` and `go vet ./...` clean;
`go test ./middleware -count=1` passes. Middleware tests cover: active trainer
continues with the trainer identity stored in the context, missing user
authentication 401, invalid user context 401, non-trainer 403, soft-deleted
trainer 403, internal repository failure 500 with no sensitive leaks, trainer
context alone never authenticates a request, and stale trainer context is
overwritten from the repository lookup.

Modified files:

- `main/src/backend/middleware/trainercontext/trainercontext.go`
- `main/src/backend/middleware/trainer_auth.go`
- `main/src/backend/middleware/trainer_auth_test.go`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Step 24 of the backend roadmap: prove that an authenticated user owns an active
trainer profile and expose the trainer identity to downstream layers, keeping
authentication strictly separate from the future trainer authorization layer.
This is the authentication foundation the trainer permission model and a future
`RequireTrainerPermission` middleware will build upon.

---

# Version 2.12.0

Title: Backend Step 23 — trainer application & approval foundation

Date: 2026-08-11

Summary:

Implemented the backend foundation for the trainer application flow. Any
authenticated user may apply to become a trainer and both administrators
(Technical ADMIN_1 and Management ADMIN_2) review the applications. A trainer
is an extension of an existing user identity: approving an application creates
a trainer profile linked to the applicant's **same** user account (same UUID,
email, password and session) — it never creates a second account.

The new `trainer_applications` table (migration `000004`) stores the
application (`id`, `user_id`, `status` in PENDING/APPROVED/REJECTED, timestamps
and soft-delete marker). The one-active-application rule is enforced by the
database through the `active_user_id` MariaDB generated column
(`IF(deleted_at IS NULL, user_id, NULL)`, unique), following the established
`active_user_id`/`active_email` pattern from `trainers`/`users`. Rejected
applications stay in history so the user can apply again; an approved
application is final.

Permanent routes:

- `POST /api/v1/trainer/apply` — authenticated user endpoint. The applicant is
  resolved exclusively from the authenticated session; client-provided
  `user_id` and `status` are ignored. Returns HTTP 201 with the safe
  application representation. Errors: 401 `AUTHENTICATION_REQUIRED` (no valid
  session, including soft-deleted users), 409 `ALREADY_TRAINER`, 409
  `APPLICATION_ALREADY_EXISTS`.
- `GET /api/v1/admin/trainer-applications` and
  `GET /api/v1/admin/trainer-applications/{id}` — read-only review,
  `trainer_applications.read`, both roles. Pagination and optional status
  filter; malformed ids 400 `VALIDATION_ERROR`, unknown applications 404
  `APPLICATION_NOT_FOUND`.
- `POST /api/v1/admin/trainer-applications/{id}/approve` —
  `trainer_applications.manage`, both roles. Approves a PENDING application and
  inserts the trainer profile for the same `user_id` in a single transaction
  (atomic; a duplicate trainer rolls the approval back and the application
  stays PENDING). Errors: 409 `APPLICATION_STATE_CONFLICT`, 409
  `USER_ALREADY_TRAINER`, 409 `USER_NOT_ACTIVE` (soft-deleted applicant).
- `POST /api/v1/admin/trainer-applications/{id}/reject` —
  `trainer_applications.manage`, both roles. Marks the application REJECTED
  (no trainer created); the user can re-apply.

The `trainer_applications` permission is split into `trainer_applications.read`
and `trainer_applications.manage`, both granted to both administrator roles in
`middleware/adminroles` — application review is a shared responsibility,
unlike the Technical-Administrator-only `users.manage`/`trainers.manage`
lifecycle permissions. No authorization logic exists in handlers or services;
enforcement is purely the `AdminAuthenticate` +
`RequireAdminPermission(...)` middleware chain.

Layering follows the established convention:
`AdminTrainerApplicationHandler`/`TrainerApplicationHandler` (in `api/auth`) →
`trainer_applications` service (in `services/trainer_applications`) →
`TrainerApplicationRepository` (in `repositories`), reusing the existing
`UserRepository` (active-user check) and `TrainerRepository` (duplicate-trainer
guard).

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes against MariaDB. New repository tests cover the
full lifecycle including atomic approval failure; service unit tests cover
validation, mapping, state transitions and the reject→re-apply→approve
lifecycle; handler/integration tests cover the user apply endpoint and both
roles for list/get/approve/reject, status filters, pagination and clamping,
state conflicts, atomicity, internal-error handling with no sensitive leaks,
and 401s for missing/invalid/stage/user tokens; a full-router E2E test
(`routes/trainer_application_e2e_test.go`) verified register → login → apply →
ADMIN_1 list/get/approve → trainer created for the same `user_id` with the
original login and `/me` intact, and ADMIN_2 reject → re-apply → approve with
exactly one trainer. Migration `000004` was applied (schema version 4).

Modified files:

- `main/src/backend/database/migrations/000004_create_trainer_applications_table.up.sql`
- `main/src/backend/database/migrations/000004_create_trainer_applications_table.down.sql`
- `main/src/backend/models/trainer_application.go`
- `main/src/backend/repositories/trainer_application_repository.go`
- `main/src/backend/repositories/trainer_application_repository_test.go`
- `main/src/backend/services/trainer_applications/trainer_applications.go`
- `main/src/backend/services/trainer_applications/trainer_applications_test.go`
- `main/src/backend/api/auth/trainer_application_handler.go`
- `main/src/backend/api/auth/trainer_application_handler_test.go`
- `main/src/backend/middleware/adminroles/adminroles.go`
- `main/src/backend/routes/routes.go`
- `main/src/backend/routes/trainer_application_e2e_test.go`
- `docs/INSTRUCTIONS/api.MD`
- `docs/DATABASE/database.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/how_to_run.MD`

Reason:

Step 23 of the backend roadmap: establish the trainer application and approval
flow so the platform can onboard trainers through a review process while
keeping the trainer strictly an extension of the existing user identity, with
database-enforced uniqueness, atomic approval and server-side authorization
shared by both administrators. This is the foundation for the trainer
lifecycle and future trainer features (profiles, RYZE Feed content ownership).

---

# Version 2.11.0

Title: Backend Step 22 — admin trainer management foundation

Date: 2026-08-11

Summary:

Implemented the backend foundation for administrator trainer management on top
of the Step 21 authorization layer and the Step 22 trainer data model.

A trainer is a distinct entity from the user account: the new `trainers` table
(`000003_create_trainers_table`) stores only the trainer identity (`id`,
`user_id`, `created_at`, `updated_at`, `deleted_at`) while the linked `users`
row owns authentication, contact and profile data. The one-to-one rule is
enforced by the database through the `active_user_id` MariaDB generated column
(`IF(deleted_at IS NULL, user_id, NULL)`, unique), following the established
`active_email` pattern from the `users` table. The new trainer GORM model and
`repositories.TrainerRepository` (create, active/deleted listing,
find-by-id/user-id, soft delete, reactivation guarded by `deleted_at IS NOT
NULL`) are the foundation for future trainer profiles and RYZE Feed content.

The `trainers` permission is split into two finer-grained permissions:

- `trainers.read` (list trainers, view one trainer) — granted to both
  administrator roles.
- `trainers.manage` (create, update, soft-delete, reactivate,
  deleted-trainer listing) — granted only to the Technical Administrator
  (ADMIN_1).

ADMIN_2 is strictly read-only: every trainer mutation attempt (including the
deleted-trainer listing) returns HTTP 403 `FORBIDDEN`, enforced server-side by
the existing reusable `RequireAdminPermission(...)` middleware. No role checks
exist in handlers or services.

Permanent routes under `/api/v1/admin`:

- `GET /trainers`, `GET /trainers/{id}` — read-only, `trainers.read`, both
  roles.
- `GET /trainers/deleted` — soft-deleted trainers listing (separated from the
  active view), `trainers.manage`, ADMIN_1 only.
- `POST /trainers` — creates a user via the existing registration flow and its
  trainer profile; on trainer-creation failure the just-created user is
  soft-deleted as a compensation so no active account is ever left without a
  trainer profile, `trainers.manage`, ADMIN_1 only.
- `PATCH /trainers/{id}` — updates the whitelisted safe user-profile fields
  (`email`, `first_name`, `last_name`) through the existing admin user service,
  `trainers.manage`, ADMIN_1 only.
- `PATCH /trainers/{id}/disable` — soft-deletes the trainer only; the linked
  user account is never touched, `trainers.manage`, ADMIN_1 only.
- `POST /trainers/{id}/reactivate` — clears `deleted_at` only (same trainer
  UUID, user link and `created_at` preserved); fails with `USER_DISABLED` when
  the linked user is soft-deleted (never reactivates the user) and with
  `TRAINER_ALREADY_LINKED` when the user already owns another active trainer,
  `trainers.manage`, ADMIN_1 only.

Trainer and user lifecycles are intentionally independent: deleting a trainer
never affects the user account, and trainer operations never reactivate a
soft-deleted user. There is no permanent-deletion endpoint; reactivation
preserves the original trainer identity.

Verification: `go build ./...`, `go vet ./...` clean and `go test ./...`
`-count=1` passes (repository, service and handler/integration tests, including
both-role read access, ADMIN_2 mutation 403s, no-authorization/sensitive data
leaks, and internal-error handling). Documentation updated in `api.MD` (new
ADMIN TRAINER MANAGEMENT section) and `database.MD` (new `trainers` table
documentation).

Modified files:

- `main/src/backend/database/migrations/000003_create_trainers_table.up.sql`
- `main/src/backend/database/migrations/000003_create_trainers_table.down.sql`
- `main/src/backend/models/trainer.go`
- `main/src/backend/repositories/trainer_repository.go`
- `main/src/backend/repositories/trainer_repository_test.go`
- `main/src/backend/services/admin_trainers/admin_trainers.go`
- `main/src/backend/services/admin_trainers/admin_trainers_test.go`
- `main/src/backend/api/auth/admin_trainers_handler.go`
- `main/src/backend/api/auth/admin_trainers_handler_test.go`
- `main/src/backend/routes/routes.go`
- `docs/INSTRUCTIONS/api.MD`
- `docs/DATABASE/database.MD`

Reason:

Step 22 of the backend roadmap: establish the trainer entity and its
administrative management foundation with a clear `trainers.read` /
`trainers.manage` permission split, so future trainer features (trainer
profiles, RYZE Feed content ownership) can build on a verified, DB-enforced
one-to-one trainer/user model.

---

# Version 2.10.0

Title: Backend Step 21 Correction 2 — definitive admin user-management model

Date: 2026-08-11

Summary:

Transformed the Step 21 admin user-management foundation into the definitive
model. The shared `users` permission is now split into two finer-grained
permissions enforced by a new reusable `middleware.RequireAdminPermission(...)`
authorization middleware:

- `users.read` (list users, view one user) — granted to both administrator
  roles.
- `users.manage` (create, update, soft-delete, reactivate, administrative
  password reset, deleted-user listing) — granted only to the Technical
  Administrator (ADMIN_1).

ADMIN_1 owns the full user lifecycle; ADMIN_2 is strictly read-only. Every
ADMIN_2 mutation attempt (including the deleted-user listing) returns HTTP 403
`FORBIDDEN`, enforced server-side by `RequireAdminPermission` reading the
authenticated identity from the request context. The coarse `PermissionUsers`
remains granted to both roles and the role/permission mapping stays in
`middleware/adminroles` as the single reusable authorization source.

Permanent routes under `/api/v1/admin`:

- `GET /users`, `GET /users/{id}` — read-only, `users.read`, both roles.
- `GET /users/deleted` — soft-deleted users listing (separated from the active
  view), `users.manage`, ADMIN_1 only.
- `POST /users` — create user, `users.manage`, ADMIN_1 only.
- `PATCH /users/{id}` — update whitelisted safe fields (`email`,
  `first_name`, `last_name`), `users.manage`, ADMIN_1 only.
- `PATCH /users/{id}/disable` — soft-delete (row/UUID/`created_at` preserved,
  `deleted_at` set and `session_version` incremented atomically),
  `users.manage`, ADMIN_1 only.
- `POST /users/{id}/reactivate` — clears `deleted_at` only (same UUID, email,
  `created_at`, hash and session version preserved; pre-deletion sessions stay
  invalid), `users.manage`, ADMIN_1 only.
- `POST /users/{id}/password` — administrative password reset (Argon2id hash
  replacement + `session_version` increment in one statement, no current
  password required, nothing sensitive returned), `users.manage`, ADMIN_1
  only.

There is intentionally no permanent-deletion endpoint. Admin user creation
delegates to the existing registration service so the identity rule is shared
with public registration: an email belonging to a soft-deleted account
reactivates the original row (same UUID and `created_at`, exactly one row, old
password fails, new password works) instead of creating a duplicate; an email
belonging to an active account returns `EMAIL_ALREADY_REGISTERED`. Update
operations reject emails owned by any other identity (active or soft-deleted)
and the caller can never set `id`, `password_hash`, `deleted_at` or
`session_version`. Administrative password resets never return the plaintext,
the hash, a token or `session_version`.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes. Service tests cover list/deleted-list/get/
create/update/soft-delete/reactivate/reset-password success and error paths
(invalid input, duplicate email, not found, already active, empty password,
wrapped repository/hasher/registrar failures). Handler tests cover both roles
for the read routes, every mutation allowed for ADMIN_1 and HTTP 403 for
ADMIN_2 with no authorization/internal detail leakage, unauthenticated (401),
regular-user and stage/invalid-token rejection, create/update/reactivate/
password-reset semantics (including reactivation preserving the UUID and
invalidating pre-deletion sessions, reset invalidating old sessions and old
password, and duplicate/not-found/validation error mapping), deleted-only
listing and no sensitive leaks. Middleware tests cover the permission gates
(both roles read, ADMIN_2 denied on manage, fail-closed without
`AdminAuthenticate`) and the updated permission matrix. E2E against the running
server verified the full ADMIN_1 lifecycle (register → login → /me → disable →
old session 401 / login 401 / row preserved with `deleted_at` → reactivate →
same UUID / one row / login works → admin password reset → old password 401 /
new password 200 / old session 401 → disable → public re-registration → same
UUID / one row / `created_at` unchanged / `deleted_at` NULL / old password 401
/ new password 200) and ADMIN_2 read-only enforcement (list/get 200, every
mutation and deleted listing 403). All temporary rows were removed afterwards.

Files modified:

- `main/src/backend/middleware/adminroles/adminroles.go` — added
  `PermissionUsersRead`/`PermissionUsersManage`, updated the role permission
  map.
- `main/src/backend/middleware/admin_authorization.go` — added
  `RequireAdminPermission`.
- `main/src/backend/repositories/user_repository.go` — added
  `FindByIDIncludingDeleted`, `ListDeleted`, `ClearDeletedAt` to interface and
  implementation.
- `main/src/backend/services/admin_users/admin_users.go` and
  `main/src/backend/services/admin_users/admin_users_test.go` — full lifecycle
  service (constructor now takes the registrar and hasher), rewritten tests.
- `main/src/backend/api/auth/admin_users_handler.go` and
  `main/src/backend/api/auth/admin_users_handler_test.go` — create/update/
  reactivate/password-reset/deleted-list handlers and DTOs, read/mutate route
  groups, expanded tests.
- `main/src/backend/routes/routes.go` — read (`users.read`) and mutate
  (`users.manage`) admin route groups.
- `main/src/backend/middleware/admin_authorization_test.go` — permission-gate
  and permission-matrix tests.
- Test fakes extended with the three new repository methods:
  `main/src/backend/services/login/login_test.go`,
  `main/src/backend/services/delete_account/delete_account_test.go`,
  `main/src/backend/services/change_password/change_password_test.go`,
  `main/src/backend/services/registration/registration_test.go`,
  `main/src/backend/api/auth/register_handler_test.go`,
  `main/src/backend/api/auth/login_handler_test.go`
- `docs/INSTRUCTIONS/api.MD` (definitive ADMIN USER MANAGEMENT section with the
  eight endpoints and the read/mutate permission model),
  `docs/INSTRUCTIONS/security.MD` (permission split, ADMIN_2 read-only, admin
  reset/reactivation session semantics, no permanent deletion),
  `docs/INSTRUCTIONS/arquitecture.MD` (permission middleware and lifecycle
  layering), `docs/DATABASE/database.MD` (new repository queries and soft-delete
  lifecycle), `docs/INSTRUCTIONS/project.MD` (ADMIN_1 lifecycle vs ADMIN_2
  read-only), `docs/how_to_run.MD` (endpoint list and authorization model),
  `docs/REGISTRY/update_logs.MD` (this entry).

Reason:

The Step 21 foundation left both roles able to disable accounts, which does not
match the product intent: the Technical Administrator owns the technical user
lifecycle while the Management Administrator only needs visibility. The
correction makes the authorization model definitive (ADMIN_1 = full lifecycle,
ADMIN_2 = read-only), guarantees that a hard delete never exists, keeps the
soft-delete/reactivation lifecycle recoverable and shares the identity rule
between admin creation and public re-registration, all while reusing the
existing authorization, token, session-version, registration and hashing
infrastructure.

---

# Version 2.9.0

Title: Backend Step 21 — admin user management foundation

Date: 2026-08-11

Summary:

Implemented the first feature on top of the Step 20 authorization layer: a
backend foundation for administrators to manage regular user accounts under
the shared `users` permission. Both administrator roles (Technical
Administrator ADMIN_1 and Management Administrator ADMIN_2) may list users,
view one user and soft-disable a user, enforced by
`RequireAdminRole(TECHNICAL_ADMINISTRATOR, MANAGEMENT_ADMINISTRATOR)` after the
Step 19 `AdminAuthenticate` middleware. The scope is deliberately minimal: no
dashboard, no trainer management, no plans/finance/marketing/system endpoints,
no admin user creation, no permanent deletion, no bulk operations and no
frontend changes.

Three permanent routes were added under `/api/v1/admin`:
`GET /admin/users` (active users, newest first, paginated), `GET
/admin/users/{id}` (safe representation of one active user) and
`PATCH /admin/users/{id}/disable` (soft-disable). The chain is
`AdminUserHandler` (thin, in `api/auth`) → `admin_users` service (in
`services/admin_users`) → `UserRepository` (in `repositories`). The repository
gains a paginated active-user query (`ListActive`, ordered newest first,
soft-delete scope applied, returns the active total) and the admin service
reuses the existing atomic soft-delete primitive, which sets `deleted_at` and
increments `session_version` in one statement. Disabling therefore preserves
the row/UUID/email, immediately revokes every active session and blocks further
logins, while leaving the data recoverable.

The service validates identifiers (malformed UUIDs are rejected before any
database access), normalizes pagination (page ≥ 1, limit clamped to the
platform maximum of 100) and maps `repositories.ErrUserNotFound` to a
`USER_NOT_FOUND` error. Responses reuse the existing safe user representation
and never include password hashes, session versions or deletion markers; the
list endpoint reports `data.pagination` with `page`, `limit`, `total` and
`total_pages`.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes. New service tests cover pagination
normalization, UUID validation, not-found mapping and internal-failure
mapping; handler tests cover both roles for list/get/disable, unauthenticated,
regular-user and stage-token rejection, invalid/garbage cookies, active-only
listing, pagination and oversized-limit clamping, safe-representation and
no-sensitive-leak checks, unknown/malformed/already-deleted identifiers,
session revocation with the row preserved and `session_version` incremented,
and a 500 internal-error path that leaks no details. E2E against the running
server verified the full flow for both admins: register a temporary user,
login, admin two-stage login, list, get, disable, old user session rejected
(401), subsequent login rejected (401), and the database row preserved with
`deleted_at` set and `session_version` incremented. All temporary users were
removed from the database afterwards and the production API surface now only
contains the three documented endpoints.

Files modified:

- `main/src/backend/repositories/user_repository.go` (added `ListActive`
  paginated active-user query to interface and implementation)
- `main/src/backend/services/admin_users/admin_users.go` and
  `main/src/backend/services/admin_users/admin_users_test.go` (new admin
  user-management service)
- `main/src/backend/api/auth/admin_users_handler.go` and
  `main/src/backend/api/auth/admin_users_handler_test.go` (new handlers and
  tests)
- `main/src/backend/routes/routes.go` (wired the permanent `/admin/users`
  routes with `AdminAuthenticate` + `RequireAdminRole` for both roles)
- Test fakes extended with `ListActive`:
  `main/src/backend/services/login/login_test.go`,
  `main/src/backend/services/delete_account/delete_account_test.go`,
  `main/src/backend/services/change_password/change_password_test.go`,
  `main/src/backend/services/registration/registration_test.go`,
  `main/src/backend/api/auth/register_handler_test.go`,
  `main/src/backend/api/auth/login_handler_test.go`
- `docs/INSTRUCTIONS/api.MD` (new ADMIN USER MANAGEMENT section with the three
  implemented endpoints; technical/management sections marked as planned),
  `docs/INSTRUCTIONS/security.MD` (session revocation via admin disable,
  user-management authorization rules),
  `docs/INSTRUCTIONS/arquitecture.MD` (user-management layering),
  `docs/DATABASE/database.MD` (`session_version` triggers, `ListActive`),
  `docs/INSTRUCTIONS/project.MD` (user management shared by both roles),
  `docs/how_to_run.MD` (new endpoints), `docs/REGISTRY/update_logs.MD`

Reason:

Administrators need a safe, server-side foundation to view and soft-disable
regular user accounts before any dashboard exists. The implementation reuses
the existing authorization, token, session-version and soft-delete
infrastructure instead of building parallel systems, and keeps controllers
thin, business logic in a dedicated service and data access behind the
existing repository, so future admin features can follow the same pattern.

---

# Version 2.8.0

Title: Backend Step 20 — reusable admin authorization layer (roles and permissions)

Date: 2026-08-11

Summary:

Added a reusable backend authorization layer that maps the two configured
administrator identities to two roles operating strictly after the Step 19
`AdminAuthenticate` middleware. The role model lives in a new
`middleware/adminroles` package: ADMIN_1 is the **Technical Administrator**
(`TECHNICAL_ADMINISTRATOR`) and ADMIN_2 is the **Management Administrator**
(`MANAGEMENT_ADMINISTRATOR`). Users, trainers and statistics are shared by both
roles; the Technical Administrator additionally holds system, infrastructure,
technical configuration, security and development; the Management Administrator
additionally holds plans, finance and marketing. Neither role is a superset of
the other: the Technical Administrator is intentionally **not** a super
administrator and never automatically receives management business permissions.

A single reusable middleware (`middleware.RequireAdminRole`) enforces
authorization on protected admin routes. It reads the authenticated admin
identity from the context via
`adminauthcontext.AdminIdentityFromContext`, resolves the role through the
identity (never from client-supplied roles, permissions, headers or query
parameters) and allows the request only when the required role matches. Denials
return a uniform HTTP 403 `FORBIDDEN` response that never reveals the required
roles, the caller's role or any internal detail. The middleware fails closed:
mounted without `AdminAuthenticate` it still denies the request. Authentication
(401 `AUTHENTICATION_REQUIRED`, owned by `AdminAuthenticate`) and authorization
(403 `FORBIDDEN`) remain separate concerns.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes. New middleware tests cover the required
scenarios: ADMIN_1 on a technical resource allowed, ADMIN_2 on a technical
resource rejected, ADMIN_2 on a management resource allowed, ADMIN_1 on a
management resource rejected, both roles on a shared resource allowed, missing
cookie, invalid/expired token, user token, stage token and unknown identity all
rejected with 401, authorization failures exposing no internal details, the
middleware failing closed without authentication, the identity→role mapping,
and the permission sets of both roles. The full DB-backed suite (user
login/register/me/logout/change-password/delete-account) stays green, so
existing user flows are unaffected. E2E via temporary protected routes on the
running server verified ADMIN_1 200 / ADMIN_2 403 on the technical route,
ADMIN_1 403 / ADMIN_2 200 on the management route and both 200 on the shared
route, plus user session, stage-only cookie, missing cookie and garbage cookie
all 401; the temporary routes were removed afterwards and return 404, so the
production API surface is unchanged. No admin dashboards, business features or
new endpoints were implemented. The `project.MD` Technical Administrator
description was reconciled: "unrestricted access" applies to the technical side
only, and the role never receives management business permissions.

Files modified:

- `main/src/backend/middleware/adminroles/adminroles.go` (new role/permission
  model and identity mapping)
- `main/src/backend/middleware/admin_authorization.go` (new
  `RequireAdminRole` middleware)
- `main/src/backend/middleware/admin_authorization_test.go` (new authorization
  tests)
- `docs/INSTRUCTIONS/security.MD`, `docs/INSTRUCTIONS/api.MD`,
  `docs/INSTRUCTIONS/arquitecture.MD`, `docs/INSTRUCTIONS/project.MD`,
  `docs/REGISTRY/update_logs.MD`

Reason:

Future admin routes need a single, reusable authorization gate derived
exclusively from the authenticated admin identity, separating authorization
from the Step 19 authentication middleware and enforcing the two distinct
administrator responsibilities without treating either administrator as a
super admin.

---

# Version 2.7.0

Title: Backend Step 19 — admin authentication middleware

Date: 2026-08-11

Summary:

Added a reusable admin authentication middleware
(`middleware.AdminAuthenticate`) that protects future admin routes. It reads
only the final admin session cookie (`ryze_admin_access_token`), validates it
exclusively through the existing Token Service (`ValidateAdminToken`, which
enforces the `admin` token kind) and then checks the resolved identity against
the configured administrator identities (`ADMIN_1`/`ADMIN_2`). The middleware
never parses JWTs manually, never touches the JWT secret, the admin credentials
or GORM/MariaDB, and stores the validated admin identity in the request
context.

The middleware accepts only a fully authenticated admin session. Stage tokens
(`ryze_admin_stage_token`), normal user tokens (`ryze_access_token`), malformed,
expired, tampered and wrong-secret tokens, tokens with a wrong `kind`, and
tokens with a missing or non-configured admin identity are all rejected with
the same indistinguishable HTTP 401 `AUTHENTICATION_REQUIRED` error; the
response never reveals why authentication failed.

A dedicated `middleware/adminauthcontext` package mirrors the existing user
`authcontext` pattern: an unexported context key type, `AdminIdentityContextKey`
and `AdminIdentityFromContext`, which rejects missing values, wrong types,
empty strings and any value that is not a configured admin identity.
`config` now defines the canonical admin identities (`Admin1ID`/`Admin2ID`
constants and `IsValidAdminIdentity`) used by `LoadAdmin`, the middleware and
the context helper, so the allow-list cannot drift.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes. Middleware tests cover both valid admins
continuing with the identity stored in context, missing/empty cookie, malformed,
expired, tampered, wrong-secret tokens, user and stage tokens rejected, missing
and invalid admin identity rejected, internal token-validation failure returning
401 without leaking details, and the context helper's valid and invalid cases.
A temporary test-only protected route was used for an E2E check against the
running server (ADMIN_1 and ADMIN_2 full two-stage login reaching the protected
route; user session, stage cookie, missing cookie and invalid admin cookie all
401) and was removed afterwards, so the production API surface is unchanged and
the `/admin/auth/self` route returns 404. No admin authorization, permissions,
dashboard or CRUD were added.

Files modified:

- `main/src/backend/middleware/admin_auth.go` (new AdminAuthenticate middleware)
- `main/src/backend/middleware/adminauthcontext/adminauthcontext.go` (new
  context/helper package)
- `main/src/backend/middleware/admin_auth_test.go` (new middleware + context
  tests)
- `main/src/backend/config/config.go` (Admin1ID/Admin2ID constants,
  IsValidAdminIdentity)
- `main/src/backend/config/env.go` (LoadAdmin uses the canonical identity
  constants)
- `docs/INSTRUCTIONS/security.MD`, `docs/INSTRUCTIONS/api.MD`,
  `docs/INSTRUCTIONS/arquitecture.MD`, `docs/REGISTRY/update_logs.MD`

Reason:

Future admin endpoints need a single, reusable authentication gate that
accepts only a fully authenticated admin session and rejects everything else,
mirroring the existing user middleware while keeping the token kind separation
authoritative.

---

# Version 2.6.1

Title: Backend Step 18 correction — admin second-factor authentication (access code)

Date: 2026-08-11

Summary:

Admin authentication is now two-stage. `POST /api/v1/admin/auth/login` only
verifies the configured username/password and, on success, issues a short-lived
temporary stage cookie `ryze_admin_stage_token` (5 minutes, `kind: admin-stage`).
It never issues an admin session. `POST /api/v1/admin/auth/verify` then requires
that stage cookie and an `access_code`; on success it clears the stage cookie and
sets the admin session cookie `ryze_admin_access_token`. Every failure — wrong
password, unknown username, missing/expired/invalid stage state, wrong or
cross-admin access code — returns the same HTTP 401 `INVALID_CREDENTIALS`, and
malformed JSON or empty fields return HTTP 400 `VALIDATION_ERROR`, so callers
never learn which stage failed or whether an administrator exists.

Each administrator now defines a second secret in `.env`:
`ADMIN_1_ACCESS_CODE` and `ADMIN_2_ACCESS_CODE` (minimum 8 characters, validated
at startup like the username/password; invalid values fail with a clear
`configuration error`). Access codes are configuration-only, never stored in the
database, embedded in tokens, logged or returned by the API; changing them in
`.env` takes effect on restart. `config.Admin` gained `AccessCode`,
`LoadAdmin` validates it, the `admin_login` service exposes
`VerifyAccessCode` (constant-time comparison) and the wiring in `routes.go`
now copies the access code into the service credentials.

The token service gained a third `kind` (`admin-stage`) with
`GenerateAdminStageToken`/`ValidateAdminStageToken` and a 5-minute
`AdminStageTokenTTL`. Stage tokens are rejected by the admin and user
validators, and admin/user tokens are rejected by the stage validator, so a
leaked stage cookie can never be promoted to a session without the access code.

A minimal future-ready two-stage admin login page was added at `/admin/login`
(credentials step, then access-code step, then a placeholder "authenticated"
state) reusing the existing login design tokens and the shared HTTP client. No
admin dashboard, middleware, role mapping or authorization was added; the
`users` table is untouched.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes (new token, config, service, handler and
routes-wiring tests). Manual E2E against the running server: stage 1 sets only
the stage cookie (Max-Age=300, no final cookie); verify with the correct code
sets `ryze_admin_access_token` (Max-Age=900, HttpOnly, SameSite=Lax) and clears
the stage cookie; wrong password, unknown username, verify without stage state,
wrong and cross-admin codes are 401; missing fields 400; a final admin token
reused as a stage state and a user token reused as a stage state are both 401;
`/api/v1/me` rejects both the stage cookie and the admin cookie; changing the
access codes in `.env` and restarting rejects the old codes and accepts the new
ones. `npm run typecheck` and `npm run build` pass.

Files modified:

- `main/src/backend/config/config.go` (Admin.AccessCode)
- `main/src/backend/config/env.go` (LoadAdmin access-code validation,
  `minAdminAccessCodeLen`)
- `main/src/backend/config/env_test.go` (access-code tests)
- `main/src/backend/services/token/token.go` (admin-stage kind, 5m TTL,
  Generate/ValidateAdminStageToken, mutual kind rejection)
- `main/src/backend/services/token/token_test.go` (stage token tests)
- `main/src/backend/services/admin_login/admin_login.go` (VerifyInput,
  VerifyAccessCode, AdminCredential.AccessCode)
- `main/src/backend/services/admin_login/admin_login_test.go` (access-code tests)
- `main/src/backend/api/auth/cookies.go` (ryze_admin_stage_token cookie)
- `main/src/backend/api/auth/admin_login_handler.go` (two-stage flow)
- `main/src/backend/api/auth/admin_login_handler_test.go` (rewritten)
- `main/src/backend/middleware/auth_test.go` (failingTokenService implements
  the expanded token service)
- `main/src/backend/routes/routes.go` (verify route + access-code wiring)
- `main/src/backend/routes/admin_credentials_test.go` (new)
- `main/src/frontend/src/pages/admin_login_page/admin_login_page.tsx` (new)
- `main/src/frontend/src/pages/admin_login_page/admin_login_page.module.css` (new)
- `main/src/frontend/src/routes/app_routes.tsx` (`/admin/login`)
- `.env` (dev access-code secrets, git-ignored) and `.env.example` (placeholders)
- `docs/INSTRUCTIONS/api.MD`, `docs/INSTRUCTIONS/security.MD`,
  `docs/INSTRUCTIONS/arquitecture.MD`, `docs/how_to_run.MD`,
  `docs/REGISTRY/update_logs.MD`

Reason:

The Step 18 spec required a second authentication factor for administrators,
but Step 18 shipped only a single-stage password login. This correction adds
the access-code factor with a strictly ordered two-stage flow so the admin
session is never issued without both factors, while keeping the failure
semantics opaque and all secrets configuration-only.

---

# Version 2.6.0

Title: Backend Step 18 — configured administrator authentication (admin login)

Date: 2026-08-11

Summary:

`POST /api/v1/admin/auth/login` authenticates platform administrators using
credentials defined exclusively in configuration (`.env`), never in the
database. Two administrators are supported: `ADMIN_1`
(`ADMIN_1_USERNAME` / `ADMIN_1_PASSWORD`) and `ADMIN_2`
(`ADMIN_2_USERNAME` / `ADMIN_2_PASSWORD`). Missing, too-short or duplicated
credentials fail at startup with a clear `configuration error`, so a broken
configuration never silently disables an administrator. Changing the values in
`.env` takes effect on the next server restart.

The request body accepts only `username` and `password`. Unknown usernames and
wrong passwords map to the same HTTP 401 `INVALID_CREDENTIALS`, so callers
cannot distinguish which admin failed or whether a username exists. Empty
username/password and malformed JSON return HTTP 400 `VALIDATION_ERROR`. On
success the endpoint returns HTTP 200 with an empty `data` object and sets a
dedicated HttpOnly cookie `ryze_admin_access_token` (`SameSite=Lax`, `Secure`
per `COOKIE_SECURE`, same TTL as the client token). The cookie name
intentionally differs from the client `ryze_access_token` so an admin session
can never be confused with a regular user session. No password, username, JWT,
JWT secret or internal details are ever returned in the response.

Administrator identities (`ADMIN_1`/`ADMIN_2`) are carried in the admin JWT
subject. The JWT service now issues two token kinds distinguished by a `kind`
claim (`user` and `admin`): admin tokens are rejected by the user access-token
validator (the user middleware, including `/api/v1/me`, returns 401 for them)
and user tokens are rejected by the admin validator. This is the authentication
foundation for the future authorization layer; no admin endpoints, admin
middleware, role mapping or admin UI are added in this step. Administrators
remain configuration-only accounts and the `users` table is untouched.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passes, including new token, config, service and
handler tests. Manual verification against the running server: both admin
logins return 200 with the dedicated HttpOnly cookie, wrong password, unknown
username and empty fields are rejected (401/400), the admin cookie is rejected
by `/api/v1/me`, changing `ADMIN_2_PASSWORD` in `.env` and restarting
invalidates the old password and accepts the new one, and an invalid
configuration fails at startup.

Files modified:

- `main/src/backend/services/token/token.go` (kind claim, GenerateAdminToken,
  ValidateAdminToken, shared token parsing)
- `main/src/backend/services/token/token_test.go` (admin token tests)
- `main/src/backend/config/config.go` (Admin and AdminConfig types)
- `main/src/backend/config/env.go` (LoadAdmin validation)
- `main/src/backend/config/env_test.go` (LoadAdmin tests)
- `main/src/backend/services/admin_login/admin_login.go` (new service with
  constant-time password comparison)
- `main/src/backend/services/admin_login/admin_login_test.go` (service tests)
- `main/src/backend/api/auth/cookies.go` (ryze_admin_access_token cookie)
- `main/src/backend/api/auth/admin_login_handler.go` (new handler)
- `main/src/backend/api/auth/admin_login_handler_test.go` (handler tests)
- `main/src/backend/routes/routes.go` (admin login route + wiring)
- `main/src/backend/main.go` (admin config load with startup fail-fast)

---

# Version 2.5.0

Title: Frontend — profile account actions (change password and delete account UI)

Date: 2026-08-11

Summary:

The `/profile` page now exposes the two existing backend account actions,
`POST /api/v1/auth/change-password` and `POST /api/v1/auth/delete-account`,
through clean inline panels inside the existing profile card. The page design
(avatar, navbar, glass card, typography, spacing, logout behaviour) is
unchanged.

A "Change password" button opens an inline panel with current password, new
password and confirmation fields, client-side validation (all fields required,
new password must match the confirmation) and Confirm/Cancel buttons. On
success the user is redirected to `/login` because the backend invalidates the
session and clears the cookie; the user must log in with the new password.

A "Delete account" button, visually separated as a destructive action through a
new `danger` Button variant (red outline, consistent with the existing error
styling), opens a confirmation panel that requires the current password before
submitting. The account is never deleted from the first click; the existing
backend performs the soft delete and clears the session, and the user is
redirected to `/login`. Re-registering with the same email reactivates the
original account.

Both panels reuse the existing API client (`apiPost`, `credentials: include`),
the existing password input pattern (visibility toggle) and the existing
design tokens. Errors are mapped to clean user-facing messages; backend or
internal details, JWTs and plaintext passwords are never exposed or logged.

Verification: `npm run typecheck` and `npm run build` pass. Manual flow
verified end-to-end against the running backend: login -> /profile -> change
password -> session ends -> old password rejected -> new password works ->
login again -> delete account -> account soft-deleted -> re-register with the
same email reactivates the same UUID.

Files modified:

- `main/src/frontend/src/pages/profile_page/profile_page.tsx` (account action
  buttons, ChangePasswordPanel, DeleteAccountPanel, PasswordField, error
  mapping, success redirect to /login)
- `main/src/frontend/src/pages/profile_page/profile_page.module.css` (panel,
  field, eye-toggle, error and danger-panel styles)
- `main/src/frontend/src/components/button/button.tsx` (new `danger` variant)
- `main/src/frontend/src/components/button/button.module.css` (danger styles)

---

# Version 2.4.0

Title: Backend Step 17 — authenticated account deletion (soft delete)

Date: 2026-08-11

Summary:

`POST /api/v1/auth/delete-account` now lets an authenticated user soft-delete
their own account after verifying their current password. The request body
accepts only `password`; the user identity is resolved exclusively from the
authenticated session (`authcontext.UserIDFromContext`), never from
client-provided identifiers. Wrong password, unknown or already-deleted user
and unverifiable stored hash all map to the same HTTP 401
`INVALID_CREDENTIALS`; an empty password or malformed JSON returns HTTP 400
`VALIDATION_ERROR`. On success the endpoint returns HTTP 200 with an empty
`data` object and clears the `ryze_access_token` HttpOnly cookie. No password,
hash, JWT, JWT secret or internal details are ever returned.

The account is never physically removed. Following the soft-delete philosophy,
the database row is preserved with its UUID, email and `created_at` unchanged;
`deleted_at` and `updated_at` are set and `session_version` is incremented.
Deletion and session revocation happen in a single repository statement, so
every previously issued access token is immediately invalid server-side (the
authenticated user can no longer reach any protected endpoint, and login with
the old password fails). A soft-deleted account re-registering with the same
email is reactivated as the original row with the same UUID, per the existing
soft-delete/reactivation lifecycle.

The change follows the standard HTTP Handler -> Service -> Repository split:
a new `delete_account` service validates the input, resolves the user,
verifies the current password with the existing `password.Verifier` and
delegates the deletion to the repository, which gained a GORM-only
`DeleteAccount` operation (`deleted_at` + `session_version` bump in one
`Updates` statement, guarded by `deleted_at IS NULL`). No new dependency,
database connection or schema change was introduced.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passing. New tests cover: the delete-account service
(success with row preserved, id/email/created_at/password_hash unchanged,
session version bumped atomically, wrong password/unknown/already-deleted/
unverifiable-hash -> `ErrInvalidCredentials`, empty input, repository failure,
re-registration reactivates the same UUID and logs in with the new password)
and the HTTP layer (success envelope, cookie cleared with Max-Age=0, old token
rejected on /me, old login 401, empty password 400, invalid JSON 400,
unauthenticated 401, soft-deleted user blocked by middleware, no
password/JWT/JWT-secret/deleted_at exposure, internal error 500). A MariaDB
end-to-end flow confirmed register -> login -> /me 200 -> delete-account 200
with clearing Set-Cookie -> old cookie /me 401 -> login old password 401 ->
database row still present with `deleted_at` set, `session_version` bumped and
exactly one row -> re-register same email -> same UUID, `deleted_at` cleared,
login new password 200 -> /me 200; the manual test user was soft-deleted
afterwards.

Files modified:

- `main/src/backend/repositories/user_repository.go` (added atomic
  `DeleteAccount`: soft-delete + `session_version` bump in one statement)
- `main/src/backend/services/delete_account/delete_account.go` (new service)
- `main/src/backend/api/auth/delete_account_handler.go` (new handler)
- `main/src/backend/routes/routes.go` (wired
  `POST /api/v1/auth/delete-account` behind `Authenticate`)
- tests: `services/delete_account/delete_account_test.go` (new),
  `api/auth/delete_account_handler_test.go` (new), and the
  `UserRepository`-implementing mocks in `services/login/login_test.go`,
  `services/registration/registration_test.go`,
  `services/change_password/change_password_test.go`,
  `api/auth/login_handler_test.go`, `api/auth/register_handler_test.go`
- `docs/INSTRUCTIONS/api.MD` (added Delete Account endpoint),
  `docs/INSTRUCTIONS/security.MD` (Session Revocation covers account deletion)

---

# Version 2.3.0

Title: Backend Step 16 — authenticated change password with session revocation

Date: 2026-08-10

Summary:

`POST /api/v1/auth/change-password` now lets an authenticated user replace
their password after verifying the current one. The request body accepts only
`current_password` and `new_password`; the user identity is resolved
exclusively from the authenticated session (`authcontext.UserIDFromContext`),
never from client-provided identifiers. Wrong current password, unknown or
soft-deleted user and unverifiable stored hash all map to the same HTTP 401
`INVALID_CREDENTIALS`; empty passwords and malformed JSON return HTTP 400
`VALIDATION_ERROR`. On success the endpoint returns HTTP 200 with an empty
`data` object, clears the `ryze_access_token` HttpOnly cookie and the user must
authenticate again. No password, hash, JWT or JWT secret is ever returned.

The change is implemented through the standard HTTP Handler -> Service ->
Repository split: a new `change_password` service verifies the current password
with the existing `password.Verifier`, hashes the new one with the existing
`password.Hasher` and delegates the update to the repository. The User
Repository gained two GORM-only operations: `ChangePassword` (replaces the hash
and increments `session_version` in the same statement) and `GetSessionVersion`
(used by the authentication middleware). Migration 000002 adds the
`session_version INT NOT NULL DEFAULT 0` column to `users`.

Session revocation is backend-side, not just cookie clearing: access tokens now
carry a `ver` claim with the session version they were issued for and the
`Authenticate` middleware compares it against the current `session_version`
resolved through a small `SessionProvider` dependency (the repository). A token
issued for an older session is rejected with HTTP 401, so after a password
change every previously issued access token is immediately invalid. The token
service interface changed accordingly
(`GenerateAccessToken(userID, sessionVersion)` and `ValidateAccessToken`
returning `*token.Claims`), so login now embeds the user's session version.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passing. New tests cover: token `ver` claim
round-trip and default-to-zero for tokens without it; middleware rejection of
revoked-session tokens, unknown users and session-lookup failures; the
change-password service (success, hash replaced, old password rejected, new
password authenticates, session version bumped, wrong current/unknown/deleted/
unverifiable-hash -> `ErrInvalidCredentials`, empty input, repository failure);
and the HTTP layer (success envelope, cookie cleared with Max-Age=0, old token
rejected on /me, old login 401 / new login 200 with identity preserved, wrong
current 401, empty current/new 400, invalid JSON 400, unauthenticated 401, no
hash/plaintext/JWT/JWT-secret exposure, internal error 500). A MariaDB
end-to-end flow confirmed register -> login -> /me 200 -> change-password 200
-> old cookie /me 401 -> login old password 401 -> login new password 200 ->
/me 200 with the same identity, the clearing Set-Cookie and the
`session_version` increment; the manual test user was soft-deleted afterwards.

Files modified:

- `main/src/backend/database/migrations/000002_add_session_version_to_users.up.sql`
  (new)
- `main/src/backend/database/migrations/000002_add_session_version_to_users.down.sql`
  (new)
- `main/src/backend/models/user.go` (added `session_version` field)
- `main/src/backend/services/token/token.go` (session version `ver` claim,
  `GenerateAccessToken`/`ValidateAccessToken` signatures, `token.Claims`)
- `main/src/backend/repositories/user_repository.go` (added `GetSessionVersion`
  + `ChangePassword`)
- `main/src/backend/middleware/auth.go` (`Authenticate` now verifies the session
  version through a `SessionProvider`)
- `main/src/backend/services/change_password/change_password.go` (new service)
- `main/src/backend/api/auth/change_password_handler.go` (new handler)
- `main/src/backend/api/auth/login_handler.go` (embeds the session version in
  generated tokens)
- `main/src/backend/routes/routes.go` (wired `POST /api/v1/auth/change-password`)
- tests: `services/token/token_test.go`, `middleware/auth_test.go`,
  `api/auth/login_handler_test.go`, `api/auth/me_handler_test.go`,
  `api/auth/logout_handler_test.go`, `api/auth/register_handler_test.go`,
  `services/login/login_test.go`, `services/registration/registration_test.go`,
  `services/change_password/change_password_test.go` (new),
  `api/auth/change_password_handler_test.go` (new)
- `docs/DATABASE/database.MD`, `docs/INSTRUCTIONS/api.MD`,
  `docs/INSTRUCTIONS/security.MD`

Reason: RYZE requires the backend to be capable of invalidating sessions
(security.MD, Session Revocation), with password change explicitly listed as a
case. A stateless JWT with no `jti` cannot revoke a specific token on its own,
so the session-version mechanism was chosen as the minimal DB-backed marker
that also generalizes to future revocation cases (account disabled, security
incident).

---

# Version 2.2.99

Title: Backend Step 15 — reactivate soft-deleted users on re-registration

Date: 2026-08-10

Summary:

When a user whose account was soft-deleted re-registers with the same email,
`POST /api/v1/auth/register` now restores the existing row instead of creating
a duplicate account. The original UUID and `created_at` are preserved, the
first/last names are updated, `deleted_at` is cleared and the `password_hash`
is replaced with a freshly Argon2id-hashed password — the old password never
verifies again and the old hash is never restored. Registering an active
email still returns `ErrDuplicateEmail` (HTTP 409) and a brand new email still
creates a normal account (HTTP 201); the endpoint, response structure and
status codes are unchanged.

The orchestration lives in the `registration` service (lookup -> decide ->
create/reactivate). The User Repository gained two GORM-only operations:
`FindByEmailIncludingDeleted` (`Unscoped` lookup to detect whether a previous
account owns the email) and `Reactivate` (`Unscoped` update clearing
`deleted_at` and replacing names + password hash). No new endpoint, no second
database connection and no new library were introduced; `created_at`,
`updated_at`, soft-delete and the generated `active_email` uniqueness are all
enforced by the existing table.

Verification: `go build ./...`, `go vet ./...`, `gofmt -l .` clean and
`go test ./... -count=1` passing. New tests cover: repository lookup
including deleted rows + reactivation (same UUID, restored active lookups,
uniqueness re-enforced); registration reactivation (same UUID, updated names,
preserved created_at, no second row, only the new password verifies);
reactivation repository failure propagation; login after reactivation (new
password authenticates, old password rejected); and the HTTP layer
(re-registering a soft-deleted email returns 201 with the same user id and no
password_hash/plaintext exposure). A MariaDB end-to-end flow confirmed the
full cycle (register -> soft delete -> re-register same email -> same UUID,
single row, login with new password 200, old password 401) and the manual test
user was removed afterwards.

Files modified:

- `main/src/backend/repositories/user_repository.go` (added
  `FindByEmailIncludingDeleted` + `Reactivate`)
- `main/src/backend/services/registration/registration.go` (reactivation flow)
- `main/src/backend/services/registration/registration_test.go`
- `main/src/backend/services/login/login_test.go` (login after reactivation)
- `main/src/backend/repositories/user_repository_test.go`
- `main/src/backend/api/auth/register_handler_test.go`
- `main/src/backend/api/auth/login_handler_test.go` (mock updated)
- `docs/INSTRUCTIONS/api.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

A soft-deleted account should be recoverable: re-registering with the same
email should restore the person's original account (same identity) rather than
splitting their history across a second row, matching the project's soft-delete
and data-preservation philosophy.

---

# Version 2.2.98

Title: Step 14 — simple authenticated profile page

Date: 2026-08-10

Summary:

Implemented the first authenticated Profile page at `/profile`, replacing the
under-development placeholder. The page is intentionally small: it proves the
user is authenticated and shows their personal area. No dashboard.

The page calls `GET /api/v1/me` with credentials on load and renders only real
data (`id`, `email`, `first_name`, `last_name`, `created_at`) — nothing is
hardcoded. A 401 redirects to `/login`; loading uses the shared RYZE
`LoadingScreen`; unexpected errors show a generic error state with a "Try
again" action. The HttpOnly cookie remains browser-managed: the JWT is never
read from JavaScript and no localStorage/sessionStorage is used for
authentication.

The page renders its own minimal navbar (brand left linking home, reserved
empty center, first name + circular avatar right) so it is chrome-free in the
global layout (`/profile` added to `CHROME_FREE_PATHNAMES`). A reusable
`ProfileAvatar` renders the default person icon and accepts an `imageUrl` so a
real profile image can be added later without redesigning the page.

Content is a centered glass card consistent with the auth card: large circular
avatar, full name, email, "Member since <Month Year>" derived from
`created_at`, a divider and a secondary "Log out" button. Logout calls the
existing `POST /api/v1/auth/logout` (no new backend functionality) and
redirects to `/login` on success. No fake buttons for Change Password or
Delete Account (those backend endpoints do not exist yet).

The HTTP client gained `apiGet` alongside the existing `apiPost`
(`main/src/utils/http_client.ts`).

Files modified:

- `main/src/utils/http_client.ts` (added `apiGet`)
- `main/src/frontend/src/pages/profile_page/profile_page.tsx` (new)
- `main/src/frontend/src/pages/profile_page/profile_page.module.css` (new)
- `main/src/frontend/src/routes/app_routes.tsx` (`/profile` → `ProfilePage`)
- `main/src/frontend/src/app.tsx` (`/profile` added to chrome-free paths)
- `docs/PAGES/design.MD` (new Profile section)
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

Turn the now-working authentication flow into a visible, minimal personal area
while keeping scope tight and avoiding any new backend endpoint, migration or
fake functionality.

---

# Version 2.2.97

Title: Step 13 — connect Login/Register to the backend

Date: 2026-08-10

Summary:

Connected the existing `/login` and `/register` pages to the backend
authentication endpoints. The register form now sends `first_name`, `last_name`,
`email` and `password` to `POST /api/v1/auth/register`; the login form sends
`email` and `password` to `POST /api/v1/auth/login`. The register form field
"Full name" was split into "First name" and "Last name" to match the backend
contract (both are required). Successful login/register redirects to `/profile`
(the existing placeholder). Backend and network errors are displayed inside the
card as a compact inline alert using the design tokens; submit buttons disable
while a request is pending.

No JWT ever touches JavaScript: the backend keeps delivering the access token
as the HttpOnly `ryze_access_token` cookie and every request is sent with
`credentials: "include"`.

A small reusable HTTP client was created at `main/src/utils/http_client.ts`
(`apiPost`, `ApiError`) following the API envelope, with the base URL read from
`VITE_API_BASE_URL` (default `http://localhost:8080/api/v1`). No new frontend
dependency was added.

Because the frontend runs on a different origin during development, a minimal
CORS middleware was added to the backend (`middleware/cors.go`, no new
library). It only serves headers to an explicit origin allow-list (never `*`)
because the auth cookie requires credentials. The allow-list is configured with
`CORS_ALLOWED_ORIGINS` (comma-separated, defaults to
`http://localhost:5173,http://127.0.0.1:5173`) and loaded through the existing
config package. Preflight `OPTIONS` requests return 204 with the allowed
methods and headers; unknown origins receive no CORS headers.

Files modified:

- `main/src/backend/middleware/cors.go` (new)
- `main/src/backend/middleware/cors_test.go` (new)
- `main/src/backend/config/config.go` (added `CORSConfig`)
- `main/src/backend/config/env.go` (added `LoadCORS`)
- `main/src/backend/routes/routes.go` (CORS middleware + new `Setup` signature)
- `main/src/backend/main.go` (loads CORS config)
- `main/src/utils/http_client.ts` (new)
- `main/src/frontend/src/pages/login_page/login_page.tsx` (forms connected,
  first/last name fields, error alert, disabled submit)
- `main/src/frontend/src/pages/login_page/login_page.module.css` (error alert)
- `docs/PAGES/design.MD`
- `docs/how_to_run.MD`
- `.env.example`

Reason:

Make the authentication flow functional end to end while keeping the existing
visual design, keeping the JWT away from JavaScript and reusing the backend
authentication architecture already built.

---

# Version 2.2.96

Title: Backend Step 12 — logout endpoint

Date: 2026-08-10

Summary:

Implemented `POST /api/v1/auth/logout`. The endpoint clears the browser
authentication cookie (`ryze_access_token`) with the same attributes used by
login (HttpOnly, Path=/, SameSite=Lax, Secure from `COOKIE_SECURE`) and a
negative Max-Age so the browser removes it (`Max-Age=0`). It deliberately does
not validate the JWT, does not require the authentication middleware and does
not touch the database — logout works even when the cookie is missing, expired
or malformed, and the user row is never deleted or soft-deleted.

Cookie configuration was extracted into a shared helper
(`api/auth/cookies.go`: `accessTokenCookie`, `accessTokenLifetime`, plus the
`AccessTokenCookieName` constant) used by both login and logout, so the two
endpoints cannot drift. `LogoutHandler` receives only the Secure flag from the
existing `COOKIE_SECURE` configuration; no repository, database connection or
token service is involved.

Response: HTTP 200 with the envelope `{ success: true, message: "Logout
successful.", data: {} }`. Never returns the JWT, JWT_SECRET, password,
password_hash or internal details.

Verification: HTTP-level tests cover success 200, clearing Set-Cookie with
empty value, HttpOnly, Path=/, SameSite=Lax, Max-Age=0, Secure following the
`COOKIE_SECURE` configuration (enabled by default, disabled for local HTTP),
logout without a cookie, logout with invalid/expired cookies, and responses
never containing a JWT, JWT_SECRET or password_hash. Existing login, register
and `/me` tests continue passing. An end-to-end `curl` check (register ->
login -> `/me` 200 -> logout -> `/me` 401) confirmed the cookie is cleared
while the user row remains in the database (`deleted_at` NULL); the manual
test user was removed afterwards.

Files modified:

- `main/src/backend/api/auth/logout_handler.go` (new)
- `main/src/backend/api/auth/logout_handler_test.go` (new)
- `main/src/backend/api/auth/cookies.go` (new, shared cookie helper)
- `main/src/backend/api/auth/login_handler.go` (uses shared cookie helper)
- `main/src/backend/routes/routes.go`

Reason:

Complete the authentication lifecycle by allowing the browser to clear its
authentication state without server-side token revocation.

---

# Version 2.2.95

Title: Backend Step 11 — authenticated user endpoint

Date: 2026-08-10

Summary:

Implemented `GET /api/v1/me`, a protected endpoint returning the authenticated
user's safe account information. The route is guarded by the existing
authentication middleware (`middleware.Authenticate(tokenService)`) and the
handler receives the existing User Repository through dependency injection, so
it never touches GORM or MariaDB directly and never re-implements
authentication. Middleware logic was not modified.

After the middleware resolves the authenticated UUID, `MeHandler.GetMe`
retrieves the user via `repo.FindByID` (which already excludes soft-deleted
users) and returns HTTP 200 with the envelope: `data { id, email,
first_name, last_name, created_at, updated_at }`. `password_hash`,
`deleted_at`, the JWT and the JWT secret are never returned. An authenticated
UUID that no longer exists (or a soft-deleted user) maps to HTTP 401; missing
or invalid cookies are handled by the middleware as 401
`AUTHENTICATION_REQUIRED`; unexpected repository failures map to HTTP 500
without leaking details.

To avoid an import cycle (`api/auth` needs the context helper while the
middleware uses `api/auth`'s error envelope and cookie name), the
authentication context key and `UserIDFromContext` helper were extracted into
a new `middleware/authcontext` package; `middleware/auth.go` now imports it
and its logic is unchanged. The safe user response DTO was renamed
`registerResponse`/`newRegisterResponse` -> `userResponse`/`newUserResponse`
and is now shared by register and `/me`.

Verification: HTTP-level tests cover success 200 with the correct id, email,
first_name, last_name and non-empty created_at/updated_at, no
password_hash/deleted_at/JWT/JWT-secret exposure, missing cookie 401, invalid
JWT 401, expired JWT 401, unknown authenticated UUID 401, soft-deleted user
401, and repository failure 500 without leaking details. Database-backed cases
run inside rolled-back transactions, leaving no permanent users. An
end-to-end `curl` check (register -> login -> `/me` with the HttpOnly cookie)
returned the correct user with 200 and 401 without a cookie; the manual test
user was removed afterwards.

Files modified:

- `main/src/backend/api/auth/me_handler.go` (new)
- `main/src/backend/api/auth/me_handler_test.go` (new)
- `main/src/backend/middleware/authcontext/authcontext.go` (new)
- `main/src/backend/middleware/auth.go`
- `main/src/backend/middleware/auth_test.go`
- `main/src/backend/api/auth/register_handler.go` (renamed shared response DTO)
- `main/src/backend/routes/routes.go`

Reason:

Expose the authenticated user profile as the first protected route built on
the authentication foundation.

---

# Version 2.2.94

Title: Backend Step 10 — authentication middleware

Date: 2026-08-10

Summary:

Implemented Gin authentication middleware in `middleware/auth.go`.
`middleware.Authenticate(tokens)` receives the existing Token Service through
dependency injection, reads the `ryze_access_token` HttpOnly cookie, validates
the JWT, extracts the authenticated user UUID and stores it in the Gin
context before calling `c.Next()`. It never touches GORM, MariaDB or the JWT
secret directly and never logs or returns the token.

All authentication failures (missing cookie, empty cookie, malformed/expired/
tampered JWT, wrong secret, invalid or missing subject) map to the same
indistinguishable HTTP 401 `AUTHENTICATION_REQUIRED` error using the project
envelope, so validation internals are never exposed. The middleware is not
applied globally and does not protect registration or login; it is ready to be
used explicitly on protected routes (no protected endpoint was added).

The authenticated user ID uses a dedicated unexported context-key type
(`UserIDContextKey`) and a helper `UserIDFromContext(c *gin.Context)` that
safely rejects missing values, wrong types, empty strings and non-UUID values
instead of silently returning a zero UUID. `RespondError` in `api/auth` was
exported so the middleware reuses the single error envelope implementation
instead of duplicating it.

Verification: middleware tests (httptest, no MariaDB) cover valid cookie
continuation, correct UUID stored in context, missing/empty cookie 401,
malformed JWT 401, expired JWT 401, wrong-secret JWT 401, invalid UUID subject
401, missing subject 401, internal validation failure 401 without leaking
details, and the context helper's valid and invalid retrieval cases.

Files modified:

- `main/src/backend/middleware/auth.go` (new)
- `main/src/backend/middleware/auth_test.go` (new)
- `main/src/backend/api/auth/register_handler.go` (exported `RespondError`)
- `main/src/backend/api/auth/login_handler.go` (uses exported helper)

Reason:

Provide reusable request authentication for future protected routes while
keeping the public registration and login endpoints untouched.

---

# Version 2.2.93

Title: Backend Step 9 correction — HttpOnly auth cookie

Date: 2026-08-10

Summary:

Changed the login authentication transport from a JSON-returned token to an
HttpOnly cookie. `POST /api/v1/auth/login` still authenticates through the
Login Service and generates the JWT through the Token Service, but now sets
the token as an HttpOnly cookie `ryze_access_token` via `http.SetCookie` and
returns only `{ success, message, data: {} }` — the JWT never appears in the
JSON response.

Cookie attributes: HttpOnly, SameSite=Lax, Path=/, Max-Age matching the
configured access-token TTL (no hardcoded lifetime), and Secure by default.
`config.LoadJWT()` now reads `COOKIE_SECURE` (boolean, default `true`) so
production stays secure over HTTPS while local HTTP development can disable
the flag in the gitignored `.env`. `.env.example` and `how_to_run.MD` document
the variable. The Login Service, User Repository and JWT Service were not
modified.

Verification: login HTTP tests updated — success 200 without a token in the
JSON body, Set-Cookie present, cookie name `ryze_access_token`, cookie value
validated through the Token Service with subject matching the authenticated
user UUID, HttpOnly enabled, SameSite=Lax, Path=/, Max-Age 900 matching the
TTL, Secure enabled by default and disabled for local HTTP, plus the existing
401/400/500 and secret-exposure tests unchanged. Database-backed cases still
run inside rolled-back transactions, leaving no permanent users.

Files modified:

- `main/src/backend/api/auth/login_handler.go`
- `main/src/backend/api/auth/login_handler_test.go`
- `main/src/backend/routes/routes.go`
- `main/src/backend/config/config.go`
- `main/src/backend/config/env.go`
- `.env.example`

Reason:

Adopt the definitive browser authentication transport (HttpOnly cookie) while
keeping credential verification, token generation and the API surface
unchanged.

---

# Version 2.2.92

Title: Backend Step 9 — login HTTP endpoint

Date: 2026-08-10

Summary:

Exposed authentication through `POST /api/v1/auth/login`. The new
`api/auth` `LoginHandler` receives the existing Login Service, the JWT Token
Service and the access-token lifetime via dependency injection (wired once in
`routes.Setup(db, jwtCfg)`), so it never touches GORM, MariaDB, password
verification or JWT generation directly.

The handler parses a dedicated request DTO (`email`, `password`), delegates
credential verification to the Login Service, then issues an access token
through the Token Service. Success returns HTTP 200 with the project envelope:
`data { access_token, token_type: "Bearer", expires_in }`. All authentication
failures (unknown email, wrong password, soft-deleted user, invalid stored
hash) map indistinguishably to HTTP 401 `INVALID_CREDENTIALS`; malformed JSON
and missing fields map to 400 `VALIDATION_ERROR`; unexpected failures map to
500 `INTERNAL_ERROR` without leaking details. `password_hash`, plaintext
passwords and the JWT secret are never included in responses.

For this step only, the token is returned in the JSON response to allow
end-to-end testing; cookies, localStorage, refresh tokens and the final
browser transport are deliberately left for a future decision.

Verification: HTTP-level tests cover success 200, Bearer token in the
response, token validation through the Token Service, token subject matching
the authenticated user UUID, wrong password 401, non-existent email 401,
soft-deleted user 401, invalid JSON 400, missing fields 400, unexpected
repository failure 500 (without leaking details), and responses never
containing `password_hash`, plaintext passwords or the JWT secret.
Database-backed cases run inside rolled-back transactions, leaving no
permanent users. An end-to-end `curl` check (register -> login) confirmed
201/200 with a token whose `sub` matches the registered user UUID; the manual
test user was removed afterwards.

Files modified:

- `main/src/backend/api/auth/login_handler.go` (new)
- `main/src/backend/api/auth/login_handler_test.go` (new)
- `main/src/backend/routes/routes.go`
- `main/src/backend/main.go`

Reason:

Bring credential-based login to the API surface using the tested Login and
Token services, while deferring middleware, protected routes and the browser
token transport to later steps.

---

# Version 2.2.91

Title: Backend Step 8 — JWT token service

Date: 2026-08-10

Summary:

Implemented a dedicated JWT service in `services/token` using
`github.com/golang-jwt/jwt/v5` with HMAC-SHA256 (HS256). `NewService(secret,
ttl)` exposes `GenerateAccessToken(userID)`, which signs a token carrying only
`sub` (user UUID), `iat` and `exp`, and `ValidateAccessToken(token)`, which
verifies the signature, rejects any signing algorithm other than HS256 (via
`jwt.WithValidMethods`, never trusting the algorithm declared by the token),
enforces an expiration claim (`jwt.WithExpirationRequired`), checks the token
is not expired, parses the subject and requires it to be a valid UUID. No
email or personal information is embedded in tokens.

Safe typed errors are returned: `token.ErrInvalidToken` for malformed tokens,
wrong-secret signatures, unexpected algorithms, missing/invalid subjects, and
`token.ErrExpiredToken` for expired tokens. The secret is never logged and
never included in errors.

Configuration: `config.LoadJWT()` reads `JWT_SECRET` (required, minimum 32
characters — startup fails clearly if missing or weak, no insecure default)
and `JWT_ACCESS_TOKEN_TTL` (Go duration, defaults to 15 minutes). `main.go`
validates the JWT configuration at startup; the service itself is not wired to
the Login Service or any HTTP endpoint yet. `.env.example` gained placeholder
entries and `how_to_run.MD` documents the variables; the local development
secret was generated and added only to the gitignored `.env`.

Verification: isolated unit tests (no MariaDB) cover valid token generation,
successful validation, correct UUID extraction, expired token rejection,
malformed token rejection, wrong-secret rejection, unexpected-algorithm
rejection, missing subject rejection, invalid UUID subject rejection, missing
expiration rejection, and missing/short/invalid JWT configuration handling.

Files modified:

- `main/src/backend/services/token/token.go` (new)
- `main/src/backend/services/token/token_test.go` (new)
- `main/src/backend/config/config.go`
- `main/src/backend/config/env.go`
- `main/src/backend/main.go`
- `main/src/backend/go.mod`, `main/src/backend/go.sum` (added golang-jwt/jwt/v5)
- `.env.example`

Reason:

Provide signed access-token generation and validation as the next step toward
authentication, kept isolated from login, middleware and HTTP.

---

# Version 2.2.90

Title: Backend Step 7 — login service

Date: 2026-08-10

Summary:

Implemented the login use-case service in `services/login`. `NewLoginService(
repo, verifier)` combines the existing User Repository and Password Service
through dependency injection (a `Verifier` type exposing `VerifyPassword` as a
method was added to `services/password`, mirroring the existing `Hasher`
pattern). The service never touches GORM or creates database connections.

The `Login` flow validates input (email required, empty passwords rejected with
`password.ErrEmptyPassword`), resolves the active user through
`repo.FindByEmail` (soft-deleted users are never returned by the repository),
verifies the supplied password with the constant-time `VerifyPassword`, and
returns the authenticated user with `password_hash` cleared.

Authentication behaviour is deliberately opaque: unknown emails, wrong
passwords, soft-deleted users and unverifiable stored hashes all return the
single `login.ErrInvalidCredentials`, so callers cannot tell which case
occurred. Unexpected repository failures propagate as wrapped internal errors
(never reported as invalid credentials); plaintext passwords are never logged,
stored, returned or embedded in errors.

Verification: isolated tests cover success (stored hash differs from plaintext
and verifies), wrong password, non-existent email, indistinguishability
between wrong password and unknown email, soft-deleted users rejected, empty
email/password rejected, repository failure propagation, and the result never
exposing `password_hash`. Database-backed cases run inside a rolled-back
transaction, leaving no permanent users.

Files modified:

- `main/src/backend/services/login/login.go` (new)
- `main/src/backend/services/login/login_test.go` (new)
- `main/src/backend/services/password/password.go`

Reason:

Provide credential verification as the next step toward authentication,
keeping the service isolated from HTTP, JWT and tokens.

---

# Version 2.2.89

Title: Backend Step 6 — registration HTTP endpoint

Date: 2026-08-10

Summary:

Exposed the existing Registration Service through a single public endpoint
`POST /api/v1/auth/register` using Gin (already part of the documented backend
stack). Dependencies are wired once at startup in `routes.Setup(db)`:
repository -> registration service -> handler, so the handler never hashes
passwords, touches GORM or creates database connections.

The HTTP layer only parses JSON into a request DTO and maps results. It does
not re-implement semantic validation: the service still owns required-field and
format checks, so handler code does not duplicate service validation. Error
responses follow the project's API envelope
`{success, message, error:{code, details}}` and map to: 201 with safe public
user data (id, email, first_name, last_name, created_at, updated_at) on
success; 409 `EMAIL_ALREADY_REGISTERED` for duplicate email; 400
`VALIDATION_ERROR` for malformed JSON or invalid input; 500 `INTERNAL_ERROR`
for unexpected failures. `password_hash` is never returned and plaintext
passwords never leak (an explicit response DTO, not the model, is serialized).

`main.go` now starts the HTTP server after the database connection, honoring a
`PORT` environment variable (default 8080); the `migrate` CLI is unchanged.

Verification: HTTP-level tests (httptest) cover success 201, duplicate email
409, invalid JSON 400, missing required fields 400, no `password_hash` in the
response, no plaintext password in the response, and an unexpected repository
failure mapping to 500 without leaking internal details. Database-backed cases
run inside rolled-back transactions, leaving no permanent users. End-to-end
manual check via `curl` against a running server confirmed 201/409/400/400; the
single manual test user was removed afterwards.

Files modified:

- `main/src/backend/api/auth/register_handler.go` (new)
- `main/src/backend/api/auth/register_handler_test.go` (new)
- `main/src/backend/routes/routes.go`
- `main/src/backend/main.go`
- `main/src/backend/go.mod`, `main/src/backend/go.sum` (added Gin)

Reason:

Bring the registration flow to the API surface as the next step toward
authentication, keeping the HTTP layer thin and the service layer reusable.

---

# Version 2.2.88

Title: Backend Step 5 — user registration service

Date: 2026-08-10

Summary:

Implemented the registration use-case service in `services/registration`. It
combines the existing Password Hashing Service and User Repository through
dependency injection: `NewRegistrationService(repo, hasher)` accepts the
repository and a `PasswordHasher` (implemented by `password.Hasher`), and the
service never touches GORM or creates database connections directly.

The `Register` flow validates input (required email/first_name/last_name,
basic email format, empty passwords rejected with `password.ErrEmptyPassword`),
hashes the plaintext password with Argon2id, stores the hash via the User
Repository, propagates `ErrDuplicateEmail` unchanged, and returns the created
user with `password_hash` cleared so it is never exposed to callers. Plaintext
passwords are never logged, stored, embedded in errors or returned. UUID
generation remains handled by the model/repository.

A `Hasher` type was added to `services/password` so the hashing component can
be injected as a dependency.

Verification: isolated tests cover successful registration (persisted hash
differs from plaintext and verifies with the Password Service; a wrong
password fails), duplicate email returning `ErrDuplicateEmail`, empty password
rejected, invalid input rejected, and repository failures propagated without
being mistaken for `ErrDuplicateEmail`. Database-backed cases run inside a
rolled-back transaction, leaving no permanent users.

Files modified:

- `main/src/backend/services/password/password.go`
- `main/src/backend/services/registration/registration.go` (new)
- `main/src/backend/services/registration/registration_test.go` (new)

Reason:

Combine the tested components (hashing + repository) into the registration
use-case as the next step toward authentication, kept isolated from HTTP and
login.

---

# Version 2.2.87

Title: Backend Step 4 — password hashing service (Argon2id)

Date: 2026-08-10

Summary:

Implemented a small, isolated password hashing component in
`services/password`. It provides `HashPassword` and `VerifyPassword` using
Argon2id (`golang.org/x/crypto/argon2`) with explicit, documented parameters:
time cost 3, memory 64 MiB (65536 KiB), parallelism 4, 16-byte random salt and
32-byte derived key. The hash is returned as a self-contained PHC string
(`$argon2id$v=19$m=...,t=...,p=...$<salt>$<key>`), so it carries everything
required for verification. Unique random salts ensure hashing the same
password twice produces different hashes.

Plaintext passwords are never logged, stored, embedded in errors or returned.
Behaviour is explicit: empty passwords return `ErrEmptyPassword`; malformed,
empty or out-of-range hashes return `ErrInvalidHash` and never panic.
Verification applies constant-time comparison and rejects hashes declaring
excessive parameters (memory above 512 MiB, time above 32, threads above 32)
as a safety guard against hostile input. The component is not connected to
the User Repository yet, and no database or migration was touched.

Verification: isolated unit tests (no database) cover hashing a valid
password, hash differing from the plaintext, successful and failed
verification, unique salts per hash, malformed hashes handled safely, and
empty-input behaviour.

Files modified:

- `main/src/backend/go.mod`
- `main/src/backend/go.sum`
- `main/src/backend/services/password/password.go` (new)
- `main/src/backend/services/password/password_test.go` (new)

Reason:

Add the secure password hashing primitive required by the upcoming
registration and authentication features, kept isolated per the step scope.

---

# Version 2.2.86

Title: Backend Step 3 — user repository and active-email uniqueness fix

Date: 2026-08-10

Summary:

Implemented the User repository layer over the existing `users` table,
following the documented GORM stack and dependency-injection conventions.
A `models.User` entity matches the `users` table (UUID id auto-generated
before insert, GORM soft delete via `deleted_at`). A `UserRepository`
interface with a GORM-backed implementation provides Create, FindByID,
FindByEmail, Update (first_name, last_name and email only) and SoftDelete
(setting `deleted_at` without physically removing the row). Lookups never
return soft-deleted users. Errors are typed sentinels (`ErrUserNotFound`,
`ErrDuplicateEmail`) plus wrapped database errors, and database credentials
are never exposed.

Schema correction: the repository test exposed that `UNIQUE (email,
deleted_at)` does not prevent duplicate emails among active users, because
MariaDB treats NULL values as distinct in unique indexes, so two active rows
(`deleted_at = NULL`) could share an email. Migration 000001 was edited to
add a stored generated column `active_email` (`IF(deleted_at IS NULL, email,
NULL)`) with `UNIQUE (active_email)`, enforcing email uniqueness for active
users while keeping emails reusable after soft delete. The local dev database
was reconciled with `go run . migrate down` and `go run . migrate up`.

Verification: an isolated repository test runs inside a rolled-back
transaction (no permanent data). It creates a user, rejects a duplicate
email, finds by ID and by email, updates first/last name and email, soft
deletes the user, confirms the user is no longer returned by normal lookups,
and confirms the physical row still exists with `deleted_at` populated.

Files modified:

- `main/src/backend/go.mod`
- `main/src/backend/go.sum`
- `main/src/backend/models/user.go` (new)
- `main/src/backend/repositories/user_repository.go` (new)
- `main/src/backend/repositories/user_repository_test.go` (new)
- `main/src/backend/database/migrations/000001_create_users_table.up.sql`

Reason:

Add the data-access layer for users as the next foundation step and fix the
email uniqueness constraint that did not enforce uniqueness for active users.

---

# Version 2.2.85

Title: Backend Step 2 — golang-migrate workflow and initial users table

Date: 2026-08-10

Summary:

Introduced the database migration workflow using `golang-migrate` and created
the first migration, which defines the initial normalized `users` table. The
backend now exposes three migration commands:

- `go run . migrate up` — applies all pending migrations
- `go run . migrate down` — rolls back all applied migrations
- `go run . migrate version` — shows the current migration version and dirty state

The migration workflow is embedded in the backend binary; migration files
live in `main/src/backend/database/migrations/`. Database schema changes must
be made through versioned migrations, never manually.

The `users` table contains the identity/account foundation only: `id`
(UUID, `CHAR(36)` primary key), `email`, `password_hash`, `first_name`,
`last_name`, `created_at`, `updated_at` and `deleted_at` (soft delete).
Email uniqueness applies to active users through a composite
`UNIQUE (email, deleted_at)` key, which keeps soft-deleted rows recoverable
while preventing duplicate active emails.

The migration was executed against the local `ryze` database: UP created the
table, DOWN removed it, and UP recreated it successfully. Final state is
migration version 1, clean.

Files modified:

- `main/src/backend/go.mod`
- `main/src/backend/go.sum`
- `main/src/backend/main.go`
- `main/src/backend/database/database.go`
- `main/src/backend/database/migrate.go`
- `main/src/backend/database/migrations/000001_create_users_table.up.sql` (new)
- `main/src/backend/database/migrations/000001_create_users_table.down.sql` (new)
- `docs/how_to_run.MD`

Reason:

Establish the reproducible, versioned migration workflow and the initial
users table as the next foundation step before authentication features.

---

# Version 2.2.84

Title: Local database setup — system MariaDB and dedicated ryze_admin user

Date: 2026-08-10

Summary:

The local database configuration now targets the normal system MariaDB
instance on port 3306 instead of the temporary isolated instance on port 3307.
A dedicated development administrator user `ryze_admin` is provisioned with
full privileges only on the `ryze` database (`ryze.*`); it receives no
privileges over unrelated databases. The root `.env` was updated to use
`DB_HOST=127.0.0.1`, `DB_PORT=3306`, `DB_NAME=ryze`, `DB_USER=ryze_admin`
and a local development password (kept out of Git). The temporary port 3307
MariaDB instance was stopped and removed. Provisioning the database and user
on the system instance requires root privileges, so the exact `sudo mariadb`
commands are provided in `docs/how_to_run.MD`.

Files modified:

- `.env` (local, not committed)
- `.env.example`
- `docs/how_to_run.MD`

Reason:

Use the normal system MariaDB for local development with a dedicated,
least-privilege RYZE database user.

---

# Version 2.2.83

Title: Backend foundation — MariaDB/MySQL database connection layer

Date: 2026-08-10

Summary:

Initial Go backend foundation: the backend now loads its database
configuration from environment variables and establishes a verified
MariaDB/MySQL connection using the documented GORM stack. A small `main.go`
entrypoint was added purely to verify the connection, reporting success or a
clear failure (non-zero exit) when the database is unreachable, misconfigured
or the credentials are rejected. No authentication, API endpoints, migrations
or business logic were introduced.

Database configuration is read from `DB_HOST`, `DB_PORT`, `DB_NAME`,
`DB_USER` and `DB_PASSWORD` (defaults: `DB_HOST=127.0.0.1`, `DB_PORT=3306`;
`DB_NAME` and `DB_USER` are required). The `.env` file at the repository root
is loaded automatically when the backend runs. Connection attempts use a 5s
timeout and fail fast, and credentials are never exposed in errors or logs.

Files modified:

- `main/src/backend/go.mod` (new)
- `main/src/backend/go.sum` (new)
- `main/src/backend/main.go` (new)
- `main/src/backend/config/config.go` (new)
- `main/src/backend/config/env.go` (new)
- `main/src/backend/database/database.go` (new)
- `main/src/backend/database/migrate.go` (package declaration only)
- `main/src/backend/logs/logger.go` (package declaration only)
- `main/src/backend/routes/routes.go` (package declaration only)
- `.env.example` (new)
- `docs/how_to_run.MD`

Reason:

First backend foundation step: establish a reliable, isolated database
connection layer before authentication, users, API endpoints and migrations
are implemented in later tasks.

---

# Version 2.2.82

Title: Feedback hero — bottom-left copy, RYZE logo on the right, solid CTA and animations

Date: 2026-08-07

Summary:

Feedback hero layout was reworked: the copy (eyebrow, headline, description)
moved to the bottom-left of the hero, the RYZE slogan logo
(`BRAND_ASSETS.slogan`) sits on the right side, aligned near the title. The
CTA became a solid futuristic cyan/blue button (gradient #00C6FF → #087BFF,
dark navy text/arrow, 11px radius, soft cyan outer glow, hover brightens with
translateY(-2px) and a stronger glow), with text "START YOUR TRANSFORMATION".
Animations were added using the shared `Reveal` component (staggered fade-up
for eyebrow, title, description, logo, cards and CTA) plus subtle continuous
glow pulses on the accent headline and the RYZE logo, and a lift + glow on
card hover. All CSS animations respect `prefers-reduced-motion`.

Files modified:

- `main/src/frontend/src/pages/feedback_page/feedback_page.tsx`
- `main/src/frontend/src/pages/feedback_page/feedback_page.module.css`
- `docs/PAGES/design.MD`

Reason:

Match the updated reference: left-aligned copy, footer logo on the right and
a solid blue CTA, with the RYZE motion language.

---

# Version 2.2.81

Title: Feedback hero — single-line title and neon quote mark above the stars

Date: 2026-08-07

Summary:

The Feedback hero headline now fits on a single line ("Real people. Real
results."). In each testimonial card the quotation mark was moved out of the
testimonial text and rendered as a single larger neon-style blue quote glyph
above the star row; the testimonial copy is now plain text.

Files modified:

- `main/src/frontend/src/pages/feedback_page/feedback_page.tsx`
- `main/src/frontend/src/pages/feedback_page/feedback_page.module.css`

Reason:

Visual feedback from the user on the hero composition.

---

# Version 2.2.80

Title: Build the Feedback page hero section

Date: 2026-08-07

Summary:

Replaced the `/feedback` under-development placeholder with a new Feedback
page containing only the hero. Solid near-black background (`#000001`) with a
very subtle radial blue atmosphere and no background artwork. Centered
composition: blue "FEEDBACK" eyebrow, large white/blue headline ("Real
people." / "Real results." with a restrained blue glow), gray description, and
exactly three premium testimonial cards in a single row using the supplied
assets (`main/src/resources/img/feedbacks/f1.jpeg`, `f2.jpeg`, `f3.jpeg`).
Each card: 4:5 image with rounded top corners and `object-fit: cover`, thin
blue border, subtle edge glow, five blue stars, "Anonymous" label, blue
quotation mark and short muted testimonial copy. Centered outline CTA "START
MY TRANSFORMATION" (with arrow) linking to `/register`. Cards stack on small
screens.

Files modified:

- `main/src/frontend/src/pages/feedback_page/feedback_page.tsx` (new)
- `main/src/frontend/src/pages/feedback_page/feedback_page.module.css` (new)
- `main/src/frontend/src/routes/app_routes.tsx`
- `docs/PAGES/design.MD`

Reason:

Provide the approved Feedback hero reference as the first stage of the
Feedback page.

---

# Version 2.2.79

Title: Update home hero headline to "The Future of Fitness"

Date: 2026-08-07

Summary:

Replaced the home hero headline ("Train smarter. Fuel better. Evolve always.")
with "The Future of Fitness", highlighting "Future" with the existing accent
highlight style, split across two lines: "The Future of" / "Fitness".

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`

Reason:

Brand copy update requested by the user.

---

# Version 2.2.78

Title: Translate the auth page forms to English

Date: 2026-08-07

Summary:

All login and register form copy on the auth page was translated to English,
matching the rest of the site: "Welcome back", "Log in to continue your
journey.", "Email or username", "Password", "Remember me", "Forgot your
password?", "Log in", "or continue with", "Don't have an account? Create one
here", "Create account", "Join RYZE and start your evolution.", "Full name",
"Email", "Confirm password", the terms checkbox and "Already have an account?
Log in". ARIA labels were translated as well.

Files modified:

- `main/src/frontend/src/pages/login_page/login_page.tsx`
- `docs/PAGES/design.MD`

Reason:

The auth page should be fully in English like the rest of the product.

---

# Version 2.2.77

Title: Keep the auth card within the viewport and make it scrollable

Date: 2026-08-07

Summary:

The register form filled almost the entire screen height, was pushed up by the
card's upward offset and got clipped by the section's `overflow: hidden`,
leaving it glued to the top edge without any way to scroll. The card now has
`max-height: calc(100svh - 5rem)` and scrolls its own content (`overflow-y:
auto`, thin cyan scrollbar), so it always stays centered and fully reachable.
The form was also tightened (padding, gap and field height reduced) to lower
the vertical footprint.

Files modified:

- `main/src/frontend/src/pages/login_page/login_page.module.css`

Reason:

On shorter viewports the register form overflowed the screen and could not be
scrolled.

---

# Version 2.2.76

Title: Restructure the authentication page as a standalone premium experience

Date: 2026-08-07

Summary:

The login page was completely restructured. The background image is no longer
an `<img>` element — it is now a CSS `background-image` on the full-viewport
section using `background-size: cover`, `background-position: center center`
and `background-repeat: no-repeat`, so the original aspect ratio is always
preserved and never stretched. The site chrome (navbar, footer and animated
background) is hidden on `/login` and `/register` via a new `showChrome` prop
on `Layout`. Only a small RYZE logo (mark + wordmark) remains in the top-left
corner. The oversized glass panel was replaced by a compact, content-sized
floating card (~420px, dark/translucent, subtle cyan border, `border-radius:
20px`, very discreet glow) centered slightly above the vertical center, with
the space background clearly visible around it. A discreet legal line
`© 2026 RYZE · Privacy Policy · Terms of Service` is pinned to the bottom.

Files modified:

- `main/src/frontend/src/components/layout/layout.tsx`
- `main/src/frontend/src/app.tsx`
- `main/src/frontend/src/pages/login_page/login_page.tsx`
- `main/src/frontend/src/pages/login_page/login_page.module.css`
- `docs/PAGES/design.MD`

Reason:

The auth page should feel like a premium, minimalist RYZE experience
independent from the main website layout, with the untouched space background
as the only backdrop.

---

# Version 2.2.75

Title: Remove empty space at the bottom of the login form

Date: 2026-08-07

Summary:

The login form (shorter than the register form) left a large empty area at
the bottom of the fixed-height panel. The panel min-height was reduced
(690px to 680px) and the form now distributes the free vertical space evenly
between its blocks (`justify-content: space-between`) instead of pushing only
the footer to the bottom. The register form already fills the panel, so its
layout is unchanged. Panel size still stays constant between modes.

Files modified:

- `main/src/frontend/src/pages/login_page/login_page.module.css`

Reason:

The login form displayed an oversized empty space beneath its content.

---

# Version 2.2.74

Title: Stretch login background and swap forms via LoadingScreen

Date: 2026-08-07

Summary:

The login background image now stretches to fill the screen height
(`object-fit: fill` instead of `cover`). The energy-wave curtain transition
was removed; switching between the login and register forms now shows the
shared RYZE `LoadingScreen` overlay for ~1 second, then swaps the content and
lets the new form stagger in. Still respects `prefers-reduced-motion`.

Files modified:

- `main/src/frontend/src/pages/login_page/login_page.tsx`
- `main/src/frontend/src/pages/login_page/login_page.module.css`
- `docs/PAGES/design.MD`

Reason:

Product feedback: stretch the background further in height and replace the
wave animation with the existing loading screen during form swaps.

---

# Version 2.2.73

Title: Create Login / Register page with energy-wave form transition

Date: 2026-08-07

Summary:

New `LoginPage` replaces the under-development placeholders for `/login` and
`/register`. A single glass panel over the full-bleed
`main/src/resources/img/login_create/background.png` hosts two interchangeable
forms (LOGIN "Bem-vindo de volta" and CREATE ACCOUNT "Criar conta") with the
specified fields, options row, Google circle, dividers and switch links. The
panel keeps a fixed position/size and the background never changes; switching
forms runs an energy-wave curtain (thin dark/electric/cyan sine lines sliding
across the clipped panel) while the outgoing content fades out and the new
form staggers in (~50ms per block). Respects `prefers-reduced-motion`.

Files modified:

- `main/src/frontend/src/pages/login_page/login_page.tsx` (new)
- `main/src/frontend/src/pages/login_page/login_page.module.css` (new)
- `main/src/frontend/src/routes/app_routes.tsx`
- `docs/PAGES/design.MD`

Reason:

Feature request: visual login and account creation screen reusing the provided
`background.png`, with a premium RYZE-styled transition between the two forms.

---

# Version 2.2.72

Title: Drop feature numbering and slim the join feature bar

Date: 2026-08-07

Summary:

The "01"–"05" numbering was removed from the items of the "Join the
movement" feature bar, and the bar is slightly shorter (panel padding
reduced; the number/header wrappers and their CSS were removed).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: numbering should be dropped and the bar shorter.

---

# Version 2.2.71

Title: Make join feature bar pill-shaped and enlarge its icons

Date: 2026-08-07

Summary:

The feature bar of the "Join the movement" section now uses a fully rounded
pill shape (border-radius 999px) with a slightly tighter padding, and the
feature icons grew (well 2.2rem to 2.7rem, glyph 1.15rem to 1.5rem).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: feature bar should look like a pill and its icons larger.

---

# Version 2.2.70

Title: Move join CTA slightly lower

Date: 2026-08-07

Summary:

The "JOIN RYZE" button of the "Join the movement" section gained a small
top margin (0.6rem) so it sits slightly lower under the description.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: button should sit a bit lower.

---

# Version 2.2.69

Title: Lower join intro text further

Date: 2026-08-07

Summary:

The eyebrow, title, description and CTA of the "Join the movement" section
moved further down (top padding 3.5rem to 5.5rem).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: top text should be even lower.

---

# Version 2.2.68

Title: Lower join intro text and nudge feature bar down

Date: 2026-08-07

Summary:

In the "Join the movement" section the eyebrow, title, description and CTA
moved lower (top padding 2rem to 3.5rem) and the section is slightly taller
(min-height 135svh to 140svh), which also lowers the feature bar a bit while
keeping it pinned to the bottom above the closing lines.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: top text should sit lower and the feature bar a touch lower.

---

# Version 2.2.67

Title: Join CTA links to WhatsApp; pin feature bar and closing lines to bottom

Date: 2026-08-07

Summary:

The "JOIN RYZE" button of the "Join the movement" section now opens the
WhatsApp community link (`CONTACT_WHATSAPP_URL`) in a new tab instead of
navigating to `/register`. The join layout now fills the full section height
(135svh) so the feature bar and the two closing lines sit at the very bottom
of the section, no longer floating in the middle.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: CTA should point to the WhatsApp community; feature bar and
closing statements must be pinned to the bottom of the section.

---

# Version 2.2.66

Title: Move join feature bar below the RYZE logo and horizon

Date: 2026-08-07

Summary:

The feature bar of the "Join the movement" section no longer overlaps the
RYZE logo or cuts the planet horizon. The section minimum height grew from
100svh to 135svh so the logo/horizon band sits clearly above the bar. The bar
is now 90% wide (max 1450px), horizontally centered, slightly shorter, and
keeps its glass/dark styling, 5 equal grid columns, icon-on-the-left layout,
single-line titles (flex-wrap allowed as fallback) and identical content.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: feature bar crossed the horizon and covered the R symbol;
the desired vertical order is headline, description, CTA, RYZE logo +
horizon, feature bar, statement, movement.

---

# Version 2.2.65

Title: Tune join section background, feature panel, CTA and footer position

Date: 2026-08-07

Summary:

In the "Join the movement" section the background is no longer cut at the
bottom on wide screens (object-position biased to `center 85%` so the
illuminated platform stays visible). The feature panel is wider (92rem) and
much thinner in height, with larger, more legible icons and text now laid out
as icon + number/title inline + subtitle, and sits lower. The CTA is rectangular with rounded corners, close
to the text, with a stronger cyan glow (brighter border, outer and inner
glows) that intensifies on hover. The title moved higher (2rem top padding) and the two closing lines moved closer to the
bottom (1.25rem bottom padding).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`

Reason:

Product feedback: background cropped at bottom, title too low, CTA too big
and pill-shaped, feature panel too tall and narrow.

---

# Version 2.2.64

Title: Compact join feature panel and lower the bottom statement

Date: 2026-08-07

Summary:

In the "Join the movement" section the feature panel is now much more compact
(smaller padding, icons, numbers and text) and pinned just above the closing
statement via `margin-top: auto`. The two closing lines were wrapped in a
`.joinFooter` group, brightened (larger, `--text-secondary` / accent glow) and
anchored to the bottom of the viewport.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: smaller feature bar right above a more visible, lower closing
statement.

---

# Version 2.2.63

Title: Move join section text higher

Date: 2026-08-07

Summary:

The "Join the movement" section was vertically centered; the eyebrow, headline
and lead sat too low. The section is now top-aligned with reduced top padding,
while the closing statement stays anchored to the bottom of the viewport via
`margin-top: auto` on a flex column content wrapper.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

Product feedback: the top centered text must sit higher.

---

# Version 2.2.62

Title: Remove logo from the Our Vision join section

Date: 2026-08-07

Summary:

Removed the glowing RYZE R mark that floated between the CTA and the feature
panel in the new "Join the movement" section. Deleted the element, its
`ryzeIcon` import and the orphaned `.joinLogo` CSS. The background remains
unchanged.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`

Reason:

Product feedback: the logo does not belong in the composition.

---

# Version 2.2.61

Title: Add "Join the movement" closing section to Our Vision

Date: 2026-08-07

Summary:

Added a new final section to the Our Vision page using the supplied
`join_bg.png` background (full-bleed, unchanged). Fully centered composition:
"The journey continues" eyebrow, two-line headline with cyan gradient accent on
"join us now.", short description, neon outline "JOIN RYZE →" CTA to
`/register`, the glowing RYZE R mark floating above the platform, a wide glass
panel with five numbered feature items, and the closing "RYZE is more than a
platform." / "It's a movement." statement. Reveal animations match the rest of
the page. Updated the Our Vision section list in the page design docs.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`

Reason:

The "Technology first" section was replaced by a closing platform section per
product decision.

---

# Version 2.2.60

Title: Remove Technology first section from Our Vision page

Date: 2026-08-07

Summary:

Removed the "Technology first" section completely from the Our Vision page,
including its `techFeatures` data, its JSX block, the now-unused
`Boxes`/`Sparkles`/`technologyBackground` imports and the orphaned `tech*` CSS
rules. The section remains on the home page.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`

Reason:

The section only belongs on the home page.

---

# Version 2.2.59

Title: Match home page tech section to Our Vision

Date: 2026-08-07

Summary:

The second section of the home page (Technology first) was rendering unstyled
because its CSS module classes were missing from `home_page.module.css`. Copied
the exact `.section`, `.sectionTech`, `.tech*` and text styles from the Our
Vision page so both sections are identical.

Files modified:

- `main/src/frontend/src/pages/home_page/home_page.module.css`

Reason:

Home page tech section appeared broken ("loose") while the Our Vision section
rendered correctly.

---

# Version 2.2.58

Title: Fix broken lucide-react imports causing blank page

Date: 2026-08-07

Summary:

Fixed two frontend import defects that broke the build and rendered a blank
page: the duplicated `ChevronRight` import in the home page, and the missing
`Boxes`/`Sparkles`/`Zap`/`technologyBackground` imports in the Our Vision page
(which threw a `ReferenceError` at module evaluation because all pages are
statically imported through the route table, crashing the whole app).

Files modified:

- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`

Reason:

Duplicate identifier broke the Vite/OXC parse; missing identifiers produced a
blank page at runtime.

---

# Version 2.2.57

Title: Rename product language from Plan to Program

Date: 2026-08-07

Summary:

Replaced product terminology across the frontend and documentation so RYZE
sells Programs, not Plans. Updated copy, route slugs, constants
(`ServiceProgram` / `SERVICE_PROGRAMS`), asset folder `program_cards`, and
related docs.

Files modified:

- `main/src/frontend/src/constants/services.ts`
- `main/src/frontend/src/components/service_card/service_card.tsx`
- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/nutrition_section/nutrition_section.tsx`
- `main/src/frontend/src/components/training_section/training_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_preview/contact_preview.tsx`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/services_page/services_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/routes/app_routes.tsx`
- `main/src/resources/img/program_cards/**` (renamed from `plan_cards`)
- `docs/INSTRUCTIONS/project.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/INSTRUCTIONS/api.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/INSTRUCTIONS/conventions.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/DATABASE/database.MD`
- `docs/PAGES/design.MD`
- `docs/FAQ/perguntas.MD`
- `docs/FAQ/respostas.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User clarified that RYZE sells programs, not plans.

---

# Version 2.2.56

Title: Add Feed nav item and under-development page

Date: 2026-08-07

Summary:

Added a public “Feed” navigation item and `/feed` under-development route,
matching the existing placeholder pattern used by Feedback and auth pages.

Files modified:

- `main/src/frontend/src/constants/navigation.ts`
- `main/src/frontend/src/components/navbar/navbar.tsx`
- `main/src/frontend/src/routes/app_routes.tsx`
- `docs/PAGES/design.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to add Feed to the navbar and create the not-yet-implemented page.

---

# Version 2.2.55

Title: Add services background and plan card logos

Date: 2026-08-05

Summary:

Applied `bg_services.png` as the full-bleed background on `/services` and the
homepage plans section. Switched plan cards to independent logos
(`logo_card1`–`3`) matching the Catalog mock, with glass cards over the shared
background.

Files modified:

- `main/src/frontend/src/constants/services.ts`
- `main/src/frontend/src/components/service_card/service_card.tsx`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/pages/services_page/services_page.tsx`
- `main/src/frontend/src/pages/services_page/services_page.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/home_page/home_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for `bg_services.png` behind services/plans and the three
independent plan logos on the cards.

---

# Version 2.2.54

Title: Modernize services catalog and plan cards

Date: 2026-08-05

Summary:

Rebuilt `/services` into a Catalog layout (highlights, three plan cards, trust
row) matching the reference mock without a background image yet. Redesigned
shared `ServiceCard` with compact logo mark, badges, stronger glass treatment
and featured Premium Level 1 CTA — also used on the homepage plans preview.

Files modified:

- `main/src/frontend/src/constants/services.ts`
- `main/src/frontend/src/components/service_card/service_card.tsx`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/pages/services_page/services_page.tsx`
- `main/src/frontend/src/pages/services_page/services_page.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/home_page/home_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to renew the services section and modernize homepage plan cards to
match the Catalog mock.

---

# Version 2.2.53

Title: Rebuild Fitness OS section with evo_man background

Date: 2026-08-05

Summary:

Rebuilt the Our Vision “Why RYZE” block into a Fitness Operating System
surface over `evo_man_bg.png`: left thesis copy + Explore RYZE CTA, and a
compact 2×4 feature panel with updated titles, descriptions and staggered
reveal animations.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to match the Fitness Operating System mock, use `evo_man_bg.png`,
and tighten the feature boxes into one composition.

---

# Version 2.2.52

Title: Remove people section and add roadmap future background

Date: 2026-08-05

Summary:

Removed the “The people behind RYZE” section from Our Vision for a later
release, and applied full-bleed `future_bg.png` behind the “Where RYZE is
going” roadmap section.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to drop the founders section for now and use `future_bg.png` as the
roadmap background.

---

# Version 2.2.51

Title: Modernize navbar icons and compact scroll mode

Date: 2026-08-05

Summary:

Restyled the public navbar toward a more rectangular cyan-edged glass bar with
icon+label nav items, richer color, and auth CTAs with icons. On scroll, nav
labels collapse so only icons remain while keeping the same destinations and
brand.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.tsx`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to modernize navbar appearance to match the reference and collapse
link text to icons while scrolling.

---

# Version 2.2.50

Title: Center navbar links and add auth CTAs

Date: 2026-08-05

Summary:

Centered public nav links in the floating navbar and replaced the profile icon
with outline “Log in” and solid cyan “Start free” pills. Added `/login` and
`/register` under-development routes for the new CTAs.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.tsx`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/routes/app_routes.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for centered navigation items and LOG IN / START FREE buttons
matching the provided mock instead of the profile icon.

---

# Version 2.2.49

Title: Match How RYZE Works panel to mock colors

Date: 2026-08-05

Summary:

Restyled the Get Started “How RYZE works” panel to match the reference:
solid dark navy frame, cyan border/glow, cyan step numbers, neon circular
icons, white titles, muted gray descriptions, and a connecting rail with
glowing midpoint dots.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User reported the previous panel styling did not match the provided mock.

---

# Version 2.2.48

Title: Revamp Our Mission with How RYZE Works panel

Date: 2026-08-05

Summary:

Rebuilt the Our Vision “Our Mission” section as a Get Started surface over
`how_bg.png`, with exact mock copy and a real HTML “How RYZE works” five-step
panel (bordered frame, titled rail, numbered icon steps).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to match the Get Started mock and build the bottom How RYZE Works
box in markup instead of relying on baked-in image UI.

---

# Version 2.2.47

Title: Restore full-bleed tech bg and clean home hero

Date: 2026-08-05

Summary:

Restored Technology First `tech_bg.png` as a full-section cover background.
Reverted the home hero to the simple full-bleed image pattern with a light
bottom fade only, removed the hero clip shelf, and anchored Start Faster
background to `center top` so the phone is not cropped at the top.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/home_page/home_page.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Previous “fixes” over-faded the hero and shrank the tech background; user asked
to restore full-bleed section art and undo the broken homepage treatment.

---

# Version 2.2.46

Title: Hard-clip home hero + tighten Technology First

Date: 2026-08-05

Summary:

Clipped the home hero inside a fixed `100svh` shelf, removed PageWrapper Y
transform (stacking culprit), and strengthened the hero bottom fade. Technology
First: removed bottom pillars, pulled copy up, and zoomed `tech_bg` out further.

Files modified:

- `main/src/frontend/src/components/page_wrapper/page_wrapper.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/home_page/home_page.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Hero still bled into Start Faster; Technology First needed no bottom cards,
higher copy and less image zoom.

---

# Version 2.2.45

Title: Clip home hero overlap + remodel Technology First

Date: 2026-08-05

Summary:

Fixed the home hero image bleeding over Start Faster by locking the hero to
`100svh` with `overflow: clip` / `contain: paint`, and raising the next section
stacking. Remodeled Our Vision Technology First around `tech_bg.png` with left
copy, feature list, CTA, and four bottom pillar cards.

Files modified:

- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Hero image still overlapped section 2 after the z-index attempt; Technology
First needed to match the provided mock with a background asset.

---

# Version 2.2.44

Title: Stretch Our Vision hero + remodel Continuous Evolution

Date: 2026-08-05

Summary:

Stretched the hero `our_view.png` to the top of the page. Remodeled Continuous
Evolution to match the mock: `conti_bg.png` full-bleed background, left intro,
and a 2×3 grid of glowing cards with icon / title / copy / chevron.

Files modified:

- `main/src/resources/img/hero/conti_bg.png` (placeholder until real asset)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for hero top coverage and a Continuous Evolution section matching
the attached mock with `conti_bg.png`.

---

# Version 2.2.43

Title: Liven Our Vision page — quieter chips, no orbit rings

Date: 2026-08-05

Summary:

Made hero chips smaller with soft static shadows (no pulse). Removed planet
orbit lines from the roadmap. Boosted ambient light, card color, shadows and
hover motion across the page.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to stop the blinking chips, drop the orbit lines, and make the page
feel less dead.

---

# Version 2.2.42

Title: Restyle Our Vision hero capability chips

Date: 2026-08-05

Summary:

Rebuilt hero chips to match the neon glass mock: white labels, glowing icon
wells, cyan border bloom, staggered entrance and ambient pulse animation.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Previous chips were low-contrast and static; user asked for the mock style with
light and motion.

---

# Version 2.2.41

Title: Restore Our Vision hero capability chips

Date: 2026-08-05

Summary:

Added AI / Training / Nutrition / Recovery / Progress chips above the Our
Vision button in the hero, matching the provided block mock.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to put the capability blocks back above the hero CTA.

---

# Version 2.2.40

Title: Fix Our Vision hero crop, timeline nodes and lighting

Date: 2026-08-05

Summary:

Stopped cropping the bottom of `our_view.png` in the hero. Aligned roadmap
timeline dots to the rail. Increased page ambient light and enlarged / enriched
the bottom-left orbital planet graphic.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User reported clipped hero artwork, misaligned timeline nodes, and asked for
more light plus a stronger orbit visual.

---

# Version 2.2.39

Title: Redesign Our Vision roadmap to match mock

Date: 2026-08-05

Summary:

Rebuilt the “Where RYZE is going” section as a two-column layout with left
intro + orbital planet glow and a right vertical timeline of milestone cards
(icon, index, phase, title, chevron).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Match the provided roadmap mock exactly.

---

# Version 2.2.38

Title: Simplify Our Vision hero copy

Date: 2026-08-05

Summary:

Removed the five capability badges under the hero lead. Kept only the Our
Vision jump link. Slightly reduced the hero section height so the background
image takes less vertical space.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for a cleaner hero: text + CTA only, with a slightly smaller image.

---

# Version 2.2.37

Title: Replace 3D planet with our_view.png hero background

Date: 2026-08-05

Summary:

Removed the Three.js RYZE Planet component and all related dependencies.
Our Vision hero now uses `our_view.png` as a full-bleed background image,
matching Training / Start Faster section pattern, with left copy only.

Files modified:

- `main/src/frontend/src/components/ryze_planet/**` (removed)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The 3D planet path was unstable; ship a clean image-backed hero instead.

---

# Version 2.2.36

Title: Fit full planet in canvas, remove sparks, blink RYZE, shooting stars

Date: 2026-08-05

Summary:

Enlarged the planet canvas (~52rem) and pulled the camera back so the whole
sphere fits without clipping. Removed blinking surface squares. Added gold
shooting-star orbits and a 1s / 0.3s-fade RYZE wordmark blink. Violet rim
palette to match the reference.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Planet was clipped by a too-small frustum/canvas; sparks and missing stars
did not match the brief.

---

# Version 2.2.35

Title: Restore original Three.js rotating RYZE Planet (v1)

Date: 2026-08-05

Summary:

Restored the first-version 3D planet (commit `planet` / 91cd3b8): shader
sphere, slow rotation, atmosphere, hologram mark + RYZE wordmark, particles.
Removed EffectComposer bloom (it caused the opaque canvas square). Slightly
boosted lighting/surface so the ball stays readable.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to bring back the original rotating 3D planet from the first
working version.

---

# Version 2.2.34

Title: Fix invisible planet — opaque SVG sphere

Date: 2026-08-05

Summary:

Rebuilt the planet as an opaque SVG ball with solid purple fill, rim light,
gold circuits and a CSS fallback background so the sphere cannot blend into
the page. Previous CSS gradients faded to page-black and made the ball
invisible.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Screenshot showed only RYZE text + haze because the sphere color matched the
background.

---

# Version 2.2.33

Title: Replace broken WebGL planet with CSS/SVG RYZE Planet

Date: 2026-08-05

Summary:

Removed the failing Three.js canvas (opaque/blurred square, invisible sphere)
and rebuilt the Our Vision planet as a reliable CSS/SVG composition matching
the reference: large violet sphere, gold circuit lines, RYZE blink, shooting
stars, nebula glow, and entrance animation.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx` (removed)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

WebGL path kept failing visually; the hero needs a guaranteed circular planet
aligned to the attached concept.

---

# Version 2.2.32

Title: Restore visible RYZE Planet sphere with reliable materials

Date: 2026-08-05

Summary:

Rebuilt the planet core with MeshStandardMaterial + panel texture so the
sphere is actually visible (custom shader was rendering near-black against
the page). Kept cyan/blue/violet site palette, orbital trails, shooting
stars, RYZE blink, and entrance animation. Removed broken bloom/shader path.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts` (removed)
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Users only saw a rectangular canvas with faint orbits because the sphere
silhouette matched the page background.

---

# Version 2.2.31

Title: Fix planet canvas box, site palette, RYZE blink and entrance

Date: 2026-08-05

Summary:

Removed EffectComposer bloom (it painted an opaque purple rectangle over the
transparent canvas and hid the sphere). Restored the visible planet with
site cyan/blue/violet colors, RYZE blink (1s cycle, 0.3s fade-out), and a
CSS entrance fade/scale for the planet.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The hero visual was broken by postprocessing alpha; palette and motion needed
to match the RYZE site language.

---

# Version 2.2.30

Title: Scale up RYZE Planet hero presence

Date: 2026-08-05

Summary:

Enlarged the Our Vision planet (container ~46rem, larger radius, closer
camera) and gave the visual column more grid space so the sphere dominates
the first viewport.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Planet still read too small against the reference; it needs hero-level scale.

---

# Version 2.2.29

Title: Refine planet size, shooting-star gold and RYZE wordmark

Date: 2026-08-05

Summary:

Enlarged and nudged the planet up/right; replaced gold burst lines with a
single shooting-star meteor; removed blinking surface sparks; restored a
clear intermittent gold “RYZE” wordmark via Troika Text.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Match the intended meteor motion and make the gold RYZE wordmark actually
visible.

---

# Version 2.2.28

Title: Tune RYZE Planet size, violet glow, gold streaks and wordmark

Date: 2026-08-05

Summary:

Made the Our Vision planet smaller and shifted left; switched glow to ultra
neon violet; slowed rotation to ~95s/turn; added occasional high-speed gold
surface streaks and a fluorescent gold “RYZE” wordmark that fades in and out.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Refine the planet signature toward the requested neon-violet / gold pulse feel.

---

# Version 2.2.27

Title: Add real-time RYZE Planet 3D hero to Our Vision

Date: 2026-08-05

Summary:

Replaced the empty Our Vision hero visual with a lazy-loaded Three.js /
React Three Fiber planet: dark metallic tech sphere, ~50s rotation, Fresnel
cyan rim, subtle particles, soft bloom, and a centered holographic RYZE mark
+ wordmark. No orbiting modules in this phase.

Files modified:

- `main/src/frontend/src/components/ryze_planet/**` (new)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Our Vision needed a premium tech-first signature visual that communicates
RYZE as future fitness software, not a traditional gym landing page.

---

# Version 2.2.26

Title: Start Our Vision hero rebuild — new copy, clear right visual

Date: 2026-08-05

Summary:

Rewrote the Our Vision first section to match the reference: “THE FUTURE OF
FITNESS” eyebrow, accent “software.” headline, adaptive-training lead,
horizontal AI / Training / Nutrition / Recovery / Progress badges, and an Our
Vision jump link. Removed the orbit / node hero visual so the right side can
receive a new composition next.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Full Our Vision page redo starts with locking the hero copy and clearing the
outdated right-side system diagram.

---

# Version 2.2.25

Title: Rebuild Progress section with analytics card and feature badges

Date: 2026-08-04

Summary:

Replaced the Progress showcase/demo with a dedicated section: left accent copy
and vertical badges, right glass progress card matching the attached chart /
summary / metric tiles design, with reveal and float animations.

Files modified:

- `main/src/frontend/src/components/progress_section/progress_section.tsx`
- `main/src/frontend/src/components/progress_section/progress_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Progress needed the same complete treatment as Training and Nutrition.

---

# Version 2.2.24

Title: Rebuild Nutrition section with accent copy, badges and glass card

Date: 2026-08-04

Summary:

Replaced the Nutrition showcase/demo with a dedicated section: left copy with
accent highlight and vertical feature badges (Training-style), right glass
nutrition card matching the attached kcal/macros/recommendation design, plus
reveal and float animations.

Files modified:

- `main/src/frontend/src/components/nutrition_section/nutrition_section.tsx`
- `main/src/frontend/src/components/nutrition_section/nutrition_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Nutrition needed the same richer left-column treatment as Training and a card
matching the provided reference.

---

# Version 2.2.23

Title: Rebuild Training section with background artwork and feature badges

Date: 2026-08-04

Summary:

Replaced the Training showcase/demo block with a full-bleed section using
`training_sec_bg.png`. Left column now matches the reference copy hierarchy and
vertical feature badges, with reveal animations kept.

Files modified:

- `main/src/frontend/src/components/training_section/training_section.tsx`
- `main/src/frontend/src/components/training_section/training_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Training needed the same artwork-driven treatment as Start Faster, with left
content matching the attached reference.

---

# Version 2.2.22

Title: Start Faster background image test without phone

Date: 2026-08-04

Summary:

Temporarily removed the phone mockup from Start Faster. Section now uses
`start_sec_bg.png` like the Hero full-bleed background. Kept copy, CTAs,
process badges and reveal animations. Shifted left content inward from the edge.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Layout/phone implementation was not matching the reference; this is a visual
test with the provided background asset.

---

# Version 2.2.21

Title: Fix Start Faster spacing, phone atmosphere and section palette continuity

Date: 2026-08-04

Summary:

Increased page gutters so Start Faster copy and phone are no longer glued to the
edges. Centered the phone in its column with cinematic glow, stars and planetary
arc matching the phone reference. Enriched the phone status/nav chrome. Removed
hard section tone jumps so the site scrolls as one continuous palette.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/*`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- Shared cards/CTA/contact/showcase CSS modules
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Layout spacing and inter-section contrast still broke visual continuity.

---

# Version 2.2.20

Title: Pixel-match Start Faster + migrate official RYZE palette

Date: 2026-08-04

Summary:

Rebuilt Start Faster to match the attached reference (~100vh, 45/55 grid,
gradient headline, process steps, hero-like atmosphere). Restored the detailed
phone screen UI at a correct non-oversized scale with premium glow and
perspective. Migrated the global token system to the new Hero palette across the
site (footer preserved).

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/*`
- Shared component/page CSS modules (palette migration)
- `docs/INSTRUCTIONS/design.MD`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The section diverged from the approved mockup, the phone was oversized, and the
site still needed the final official palette applied consistently.

---

# Version 2.2.19

Title: Match Start Faster section to reference mockup

Date: 2026-08-04

Summary:

Rebuilt the Start Faster / phone section to match the attached reference:
accent headline phrase, process steps, thin connecting arc, and a larger
front-facing premium phone with circular ready gauge, glass reflection and
soft blue ambient glow.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The live phone section diverged from the approved reference composition.

---

# Version 2.2.18

Title: Official Hero token system + restore premium Platform phone

Date: 2026-08-04

Summary:

Restored the Start Faster phone as a large premium product mockup and removed
the random decorative graphics. Introduced the official global CSS token system
from the Hero palette and migrated shared UI surfaces to consume it. Updated
design, project and procedure documentation so future work follows one palette.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- Shared component/page CSS modules (token migration)
- `docs/INSTRUCTIONS/design.MD`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform pass was a visual regression, and the site still felt like
multiple palettes. Hero is now the only official design reference.

---

# Version 2.2.17

Title: Unify global palette to Hero design system

Date: 2026-08-04

Summary:

Replaced the legacy teal/blue-grey token set with the Hero-derived futuristic
palette across global CSS and shared UI surfaces. Updated design documentation
as the new visual source of truth. Footer styling was intentionally preserved.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- Multiple component/page CSS modules (palette sync)
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The Hero already defined the premium futuristic identity, but later sections
still used the old palette and broke visual continuity.

---

# Version 2.2.16

Title: Rebuild Start Faster around phone atmosphere + page reveals

Date: 2026-08-04

Summary:

Removed floating Platform cards and made the phone the sole hero, surrounded by
atmospheric grid/glow/trails and tiny decorative fragments. Shortened Start
Faster copy, added a secondary link, and introduced shared scroll reveals across
Home and Services.

Files modified:

- `main/src/frontend/src/components/platform_section/**`
- `main/src/frontend/src/components/reveal/reveal.tsx` (new)
- `main/src/frontend/src/components/showcase_section/showcase_section.tsx`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/services_page/services_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Floating cards still felt AI-generated and competed with the phone; Home and
Services needed the same premium scroll motion language as Our Vision.

---

# Version 2.2.15

Title: Replace About Us with Our Vision page

Date: 2026-08-04

Summary:

Replaced the About Us placeholder with a premium `/our-vision` page focused on
platform vision, mission, roadmap and technology. Updated navigation/footer
labels, redirected `/about-us`, and synchronized design/project/run docs.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/**` (new)
- `main/src/frontend/src/routes/app_routes.tsx`
- `main/src/frontend/src/constants/navigation.ts`
- `main/src/frontend/src/components/footer/footer.tsx`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The old About Us framing no longer matched RYZE as a technology company
reinventing fitness; the public page needed to sell vision, not founders.

---

# Version 2.2.14

Title: Fix Start Faster card orbit layout around tilted phone

Date: 2026-08-04

Summary:

Rebuilt the Platform visual composition as a 3-column CSS grid so Nutrition,
This week and Recovery stay anchored beside the phone instead of drifting with
absolute edge positioning. Increased phone tilt to `rotateY(-40deg)`.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Absolute left/right card placement made the panels look scattered and poorly
aligned relative to the centered phone.

---

# Version 2.2.13

Title: Rebalance Start Faster phone and floating cards

Date: 2026-08-04

Summary:

Increased the phone `rotateY` to `-30deg`, moved Nutrition to the lower-left of
the device, kept This week and Recovery on the right, and removed all connector
lines. Mobile stacks or hides the cards so they never overlay the phone.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The visual needed a stronger 3D phone angle and more balanced card weight on
both sides of the device, without connector lines.

---

# Version 2.2.12

Title: Rebuild Start Faster visual around a 3D phone

Date: 2026-08-04

Summary:

Replaced the crowded Platform visual with a tilted almost-3D phone (realistic
proportions and RYZE UI inside) and three emerging glass panels: This week,
Nutrition and Recovery. Soft dashed connectors and slow float motion remain.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous right-side composition still felt busy; the section needed a
clearer product focal point with only the strongest supporting panels.

---

# Version 2.2.11

Title: Enable WhatsApp community contact link

Date: 2026-08-04

Summary:

Activated WhatsApp across contact surfaces with the official community invite
URL. Updated homepage preview chips, Contact page channel row and footer
Resources link. Removed disabled/coming-soon states.

Files modified:

- `main/src/frontend/src/constants/contact.ts`
- `main/src/frontend/src/components/contact_preview/contact_preview.tsx`
- `main/src/frontend/src/components/contact_preview/contact_preview.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

WhatsApp community is live and should be reachable from every contact entry
point.

---

# Version 2.2.10

Title: Show plan price above service card CTA

Date: 2026-08-04

Summary:

Added a compact right-aligned price above each service card CTA, using the
existing `ServicePlan.price` field. Applies to both homepage plans preview and
`/services`.

Files modified:

- `main/src/frontend/src/components/service_card/service_card.tsx`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Plan pricing needed to be visible on cards without competing with the CTA.

---

# Version 2.2.9

Title: Remove homepage Athletes testimonials section

Date: 2026-08-04

Summary:

Removed the Athletes/testimonials section from the homepage and deleted the
unused `testimonials_section` component. Updated Home page section order in
design docs.

Files modified:

- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/components/testimonials_section/**` (removed)
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The testimonials block was no longer wanted on the homepage product tour.

---

# Version 2.2.8

Title: Harmonize homepage palette after Platform redesign

Date: 2026-08-04

Summary:

Aligned the remaining homepage surfaces and interactive components with the new
blue/cyan brand palette and deep background continuity. Kept footer structure
and styling unchanged while improving cross-section visual cohesion.

Files modified:

- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/showcase_section/showcase_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/contact_preview/contact_preview.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

After the Platform and global palette overhaul, several homepage blocks still
used legacy hardcoded tones, creating visible inconsistency between sections.

---

# Version 2.2.7

Title: Rebuild Platform into a hierarchical mobile-first system

Date: 2026-08-04

Summary:

Reworked the homepage Platform section away from the decorative orbit/mind-map
layout into a clearer mobile-first composition: goals bar, primary phone
surface, and grouped support panels for training, nutrition, recovery,
progress, PRs and coach feedback.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform section still behaved like a decorative mental map with
weak hierarchy, fragile layout rules and poor mobile behavior.

---

# Version 2.2.6

Title: Platform ecosystem visualization redesign

Date: 2026-08-04

Summary:

Replaced the generic Platform dashboard demo with a premium ecosystem
visualization: floating phone, glass orbit cards, connection lines, parallax and
slow motion. Updated Platform messaging and removed DashboardDemo.

Files modified:

- `main/src/frontend/src/components/platform_section/**` (new)
- `main/src/frontend/src/components/product_demos/dashboard_demo.tsx` (removed)
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform section read like an AI-generated admin dashboard and
failed to communicate that RYZE is a connected fitness operating system.

---

# Version 2.2.5

Title: Point image imports to hero and plan_cards

Date: 2026-08-04

Summary:

Updated hero and service plan image imports after removing `img/testing` and
moving assets to `img/hero` and `img/plan_cards`.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/constants/services.ts`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Asset paths broke after the testing folder was deleted and images were renamed
and relocated.

---

# Version 2.2.4

Title: Premium embedded Contact experience

Date: 2026-08-04

Summary:

Rebuilt Contact as a dense two-column SaaS layout with compact channel rows and
an embedded glass form (no modal). Home now uses a compact Get in touch preview.
Fixed the mismatched top strip on `/contact`.

Files modified:

- `main/src/frontend/src/components/contact_section/**`
- `main/src/frontend/src/components/contact_form/**` (new)
- `main/src/frontend/src/components/contact_preview/**` (new)
- `main/src/frontend/src/components/email_contact_panel/**` (removed)
- `main/src/frontend/src/pages/contact_page/**`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous contact cards + modal felt like a generic template and created a
visible top color seam on the Contact page.

---

# Version 2.2.3

Title: Living dark atmospheric background

Date: 2026-08-04

Summary:

Added stronger cyan nebulae, soft vignette shadows, breathing glow and translucent
section washes so the dark UI feels alive instead of flat/static. Footer unchanged.

Files modified:

- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Solid opaque section fills were covering the animated atmosphere and making the
page feel dead.

---

# Version 2.2.2

Title: Rounded pill navbar and larger brand mark

Date: 2026-08-04

Summary:

Changed the navbar from a rectangular strip to a floating pill bar and enlarged
the navigation brand mark.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The rectangular navbar felt too rigid; a rounded capsule and a stronger icon
match the premium SaaS navigation pattern.

---

# Version 2.2.1

Title: Full visual consistency pass

Date: 2026-08-04

Summary:

Finished the SaaS consistency pass: thinner navbar, soft section tone blends,
premium service cards, and purged remaining teal/grey-blue accents from demos,
panels and secondary pages. Footer left unchanged.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/section_wrapper/**`
- `main/src/frontend/src/components/showcase_section/**`
- `main/src/frontend/src/components/service_card/**`
- `main/src/frontend/src/components/product_demos/product_demos.module.css`
- `main/src/frontend/src/components/email_contact_panel/email_contact_panel.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/pages/home_page/**`
- `main/src/frontend/src/pages/services_page/services_page.module.css`
- `main/src/frontend/src/pages/under_development_page/under_development_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

After adopting the official dark cyan identity, leftover teal accents and
inconsistent section/card treatments still broke the Linear/Apple/Vercel feel.

---

# Version 2.2.0

Title: Official dark cyan SaaS visual identity

Date: 2026-08-03

Summary:

Fixed the clipped navbar, reduced the brand mark size, and adopted the official
near-black / electric cyan design system across the product UI while preserving
the existing footer.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/footer/footer.module.css` (restored)
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero established the official RYZE SaaS look. The rest of the product needed
to match that darker high-contrast cyan system, and the navbar needed room so it
is no longer cut off at the top.

---

# Version 2.1.9

Title: Dark contrast palette and navbar polish

Date: 2026-08-03

Summary:

Aligned the global palette to a deeper navy / electric cyan system and cleaned
the navbar into a transparent integrated bar with stronger text and button
contrast.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/animated_background/animated_background.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The reference hero look depends on near-black surfaces, white text and cyan CTAs.
The navbar also needed to feel integrated instead of heavily framed.

---

# Version 2.1.8

Title: Rebuild hero as background artwork

Date: 2026-08-03

Summary:

Rebuilt the hero from scratch with a single container, left-side content only,
and an absolute oversized artwork background layer starting mid-viewport with a
left fade mask.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Previous approaches kept treating the illustration as layout content. The
intended design is immersive background artwork like Linear / Apple / Vercel.

---

# Version 2.1.7

Title: Oversized hero background artwork

Date: 2026-08-03

Summary:

Reworked the hero illustration into an oversized CSS background layer using the
premium SaaS pattern: absolute right-anchored artwork with `background-size:
cover`, intentional overflow and left-edge masking.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Fitting the full image inside a container made the artwork look embedded. The
hero needs immersive oversized background artwork like Linear / Raycast /
Vercel landing pages.

---

# Version 2.1.6

Title: Atmospheric hero illustration blend

Date: 2026-08-03

Summary:

Converted the hero artwork into an absolute right-side atmospheric layer with
left-edge CSS masking so it blends into the page instead of appearing as an
embedded image.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

The illustration was still reading as a separate image block. It needs to feel
like integrated SaaS landing artwork that never pushes the text column.

---

# Version 2.1.5

Title: Hero split layout for illustration

Date: 2026-08-03

Summary:

Rebuilt the hero as a balanced 45/55 content/illustration grid so the artwork
stays on the right, scales with `object-fit: contain`, and never overlaps text.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Treating the illustration as a full-bleed cover background caused cropping and
overlap. The artwork needs a dedicated right column that grows wider on large
screens while preserving the full composition.

---

# Version 2.1.4

Title: Right-aligned contained hero artwork

Date: 2026-08-03

Summary:

Scaled the hero artwork down, locked it to the right side with full visibility,
kept the left side as atmosphere only, and cleared the navbar overlap zone.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Cover-cropping was hiding the right composition and pushing important artwork
under the navbar. The visual needs to stay fully visible, upright and smaller.

---

# Version 2.1.3

Title: Smaller hero image and seamless page blend

Date: 2026-08-03

Summary:

Reduced the hero image scale/height and faded it into a deeper navy page
atmosphere so the cut into the sections below is no longer abrupt.

Files modified:

- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/animated_background.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero artwork ended in a hard horizontal seam against the rest of the site.
The page needed a smaller hero treatment and a continuous galactic surface.

---

# Version 2.1.2

Title: Full-bleed hero image layout

Date: 2026-08-03

Summary:

Corrected the hero so `test4.png` spans the full hero width, with copy on the
left and the image subject visible on the right, and removed the floating cards.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero image already contains the composed product visual. It should act as a
full-bleed canvas, not as a cropped card with extra floating UI overlays.

---

# Version 2.1.1

Title: Galactic hero composition and richer atmosphere

Date: 2026-08-03

Summary:

Rebuilt the hero into a free-floating brand composition with orbiting glass
cards, enriched the galactic background atmosphere and boxed the athlete
testimonials.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/hero/**`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Removed the rectangular frame around the hero image and treated `test4.png`
  as a free-floating brand centerpiece.
- Added floating glass product cards for training, volume, readiness, nutrition
  and recovery around the hero mark.
- Updated hero copy, CTA labels and trust indicators to match the product-led
  galactic direction.
- Increased cyan/blue atmospheric depth in the global palette and background.
- Added boxed glass testimonial cards.
- Updated design documentation for the hero composition and richer atmosphere.

Reason:

The previous hero still looked boxed and flat. The site needed more life,
color and a technological / galactic product presence without leaving the RYZE
blue palette.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.

---

# Version 2.1.0

Title: Darker product-led homepage redesign

Date: 2026-08-03

Summary:

Moved the public visual identity to a darker premium technology language,
redesigned navbar behaviour and replaced the generic "Why RYZE" feature cards
with product-demo showcase sections.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/navbar/**`
- `main/src/frontend/src/components/hero/**`
- `main/src/frontend/src/components/showcase_section/**`
- `main/src/frontend/src/components/product_demos/**`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/components/section_title/section_title.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/testimonials_section/**`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/REGISTRY/update_logs.MD`

Removed:

- `main/src/frontend/src/components/why_choose_ryze/**`
- `main/src/frontend/src/components/coaches_section/**`
- `main/src/frontend/src/components/faq_section/**`
- empty `main/src/frontend/src/components/hero_scene/`
- empty `main/src/core/`

Changes made:

- Updated global tokens to a darker surface system with blue as the primary
  accent.
- Softened the animated background and reduced flashy gradient treatment.
- Redesigned the navbar so it is nearly transparent at the top and gradually
  gains opacity, blur and darker glass while scrolling.
- Replaced boxed active navigation states with an animated accent underline.
- Rebuilt the homepage flow as a product tour: dashboard, training, nutrition,
  progress, plans, testimonials, CTA and contact.
- Added reusable showcase and product-demo components that visually demonstrate
  the platform instead of explaining it with feature cards.
- Tightened hero copy, buttons, cards, glass panels and testimonials for a
  calmer premium look.
- Synchronized design, architecture and project status documentation.
- Aligned package metadata with version `2.1.0`.

Reason:

The previous landing still felt like a generic AI marketing page. RYZE needed a
recognizable darker product identity and a homepage that sells through the
interface itself.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- Temporary hero testing imagery remains until final brand assets are chosen.

---

# Version 1.1.4

Title: Footer slogan and email contact panel

Date: 2026-08-03

Summary:

Refined footer branding, enlarged the loading screen logo and added a visual
email contact panel.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/constants/brand_assets.ts`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added the existing `ryze_slogan.png` asset to the shared brand asset registry.
- Changed the footer brand image from the icon mark to the RYZE slogan logo.
- Kept footer social actions as three compact circular icon-only controls.
- Increased the loading screen brand mark and adjusted the rotating arrows to
  match the larger scale.
- Changed the email contact card to open a centered overlay panel with blurred
  backdrop, close controls and a structured visual form.
- Added Escape-key and backdrop-close behavior for the email panel.
- Updated design documentation to match the footer and contact interaction.
- Aligned package metadata with patch version `1.1.4`.

Reason:

The footer needed to use the full slogan asset, the loading screen logo was
still too small, and the contact email action needed to behave like a polished
frontend form interaction without introducing backend logic.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.3

Title: Logo rendering and footer simplification

Date: 2026-08-03

Summary:

Adjusted the brand logo rendering and simplified the footer layout according to
the latest visual direction.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Removed the visual wrapper, background, border, shadow and crop from the RYZE
  logo component.
- Rendered the original logo asset directly in the navbar, footer and loading
  screen.
- Simplified the footer to a logo plus three small circular social/contact icon
  buttons.
- Removed the footer contact cards and future social links label.
- Updated the design documentation footer specification.
- Aligned package metadata with patch version `1.1.3`.

Reason:

The RYZE logo asset should be displayed directly without an artificial container,
and the footer needed a cleaner minimal structure with only the brand and social
icons.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.2

Title: Navbar and footer visual correction

Date: 2026-08-03

Summary:

Corrected the navbar, hero first viewport and footer layout after visual review,
with a much larger rendered RYZE logo and a cleaner footer structure.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_mark/**`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.tsx`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/navbar/navbar.tsx`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added a reusable `BrandMark` component that crops and zooms the original
  `ryze_icon.png` asset so the logo is visibly large instead of appearing tiny.
- Applied the enlarged brand mark to the navbar, footer and loading screen.
- Rebuilt the navbar layout with a cleaner full-width header, larger brand
  area, stronger navigation pill and larger profile button.
- Refined the hero first viewport spacing and visual balance below the navbar.
- Rebuilt the footer into a two-column brand/contact layout with structured
  contact rows and subtle future social placeholders.
- Increased footer contact icon sizes and improved footer alignment on desktop
  and mobile.
- Aligned package metadata with patch version `1.1.2`.

Reason:

The previous navbar and footer still looked visually weak because the logo asset
was rendered with too much surrounding image area and the footer contact/social
layout felt unbalanced. This change improves brand visibility, hierarchy and
overall polish while preserving the documented RYZE visual identity.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.1

Title: Frontend visual refinement

Date: 2026-08-03

Summary:

Improved the initial frontend visual presentation based on design review
feedback, with stronger brand scale, better navigation polish, higher button
contrast and correct branded contact icons.

Files modified:

- `.gitignore`
- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_icon/**`
- `main/src/frontend/src/components/button/button.tsx`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.tsx`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/styles/global.css`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Increased the RYZE brand icon size in the navbar, footer and loading screen.
- Increased service and contact icon sizing for stronger visual hierarchy.
- Removed the large top gap above the hero by tightening hero and navbar
  spacing.
- Improved the navbar with a stronger branded active state, larger profile and
  menu icons, and a more polished glass surface.
- Added reusable branded SVG icons for Gmail, Instagram and WhatsApp.
- Replaced generic contact icons in the contact section and footer with the
  correct branded icons.
- Improved dark CTA button contrast by adding light and light-outline button
  variants.
- Refined the page background, footer and CTA band styling for a cleaner,
  premium RYZE aesthetic.
- Narrowed the root `how_to_run.MD` ignore rule so `docs/how_to_run.MD` can be
  tracked.
- Aligned package metadata with the project patch version.

Reason:

The first visual pass had weak brand scale, low-contrast CTA links on dark
sections, generic social icons and excessive spacing above the hero. These
adjustments improve clarity, accessibility and visual consistency while keeping
the documented RYZE color palette and frontend architecture.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.0

Title: Initial frontend implementation

Date: 2026-08-03

Summary:

Implemented the initial visual frontend for the RYZE platform using the
documented React, TypeScript, Vite, React Router and CSS Modules stack.

Files modified:

- `.gitignore`
- `index.html`
- `package.json`
- `package-lock.json`
- `tsconfig.json`
- `vite.config.ts`
- `main/src/frontend/src/**`
- `main/src/utils/class_names.ts`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added the Vite React application entry point.
- Implemented global styling tokens based on the documented RYZE palette.
- Added reusable frontend components for buttons, cards, layout, navbar,
  footer, loading screen, section titles, hero, service cards, CTA bands and
  contact content.
- Implemented the Home page with navbar, hero, services preview, CTA sections,
  contact section and footer.
- Implemented React Router routes for Home, Services, Contact, Feedback,
  About Us, Profile and service detail paths.
- Added professional under development screens for routes not fully implemented
  yet.
- Added the reusable global loading screen with the RYZE logo and two rotating
  arrows.
- Used existing logo assets from `main/src/resources/img/logo/` without moving
  or duplicating them.
- Added developer run documentation.
- Documented the current frontend folder structure.

Reason:

The platform needed a production-quality initial frontend foundation before
backend integration begins, while preserving the documented visual identity,
responsive behavior and reusable component architecture.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---
