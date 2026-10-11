# Version 2.85.0

Title: Payment reconciliation, single pending intent and webhook settlement durability — late verified payments are never discarded

Date: 2026-10-11

Summary:

v2.85.0 closes the settlement window where a provider-verified payment could be
silently lost. One direct migration (`000027`) and one webhook-bound work
surface were touched; the database moves to migration version `27`. No refund,
regeneration, Premium Level 2 or new payment provider surface was introduced and
the proven PayPal/Stripe return-context handling is unchanged; Test Mode,
Premium Level 1 privacy and the Generic checkout are unaffected.

Late settlements are reconciled, never discarded. Previously both webhooks
answered "purchase not pending" with 200 whenever a verified payment arrived
after the purchase left `pending`, so a payment captured on the browser return
leg while a supersede was landing could grant nothing and be silently dropped.
Now a `failed` purchase is the recovery state: `ReconcilePurchase`
(service) / `ReconcileFailedPurchase` (repository) atomically returns it to
`completed` — superseding any stale pending sibling, running the guarded
failed→completed transition and restoring-or-creating the entitlement in one
transaction. The two database uniqueness guards
(`uq_purchases_completed_purchase`, `uq_entitlements_active_entitlement`) make a
second sale impossible: a conflict rolls everything back and the failed record
stays as the persisted, diagnosable recovery state. A webhook answers 200
(reconciled) on success, 200 with a diagnostic on a conflict or purchase-that-moved-on,
and 500 only on an integrity violation or transient failure so the provider
redelivers. `ReconcilePurchase` is internal-only: no route accepts a purchase id
without an authenticated owner, so a browser can never trigger it.

Single pending intent at the database level. Migration `000027` adds the
`pending_purchase` generated column with a unique index, so the engine itself
enforces at most one active pending intent per `(user, program)` pair. The
repository maps a violation to `ErrPurchaseAlreadyExists` and intent creation
re-evaluates the resolved ownership (`DUPLICATE_ENTITLEMENT` or
`DUPLICATE_PURCHASE`) instead of leaking a raw driver error.

PayPal capture settlements ordered by proof, not by notification. The handler
now processes `PAYMENT.CAPTURE.COMPLETED`: the order is loaded from the PayPal
API (never trusted from the event), its single purchase-unit reference recovers
the RYZE purchase, the captured amount and currency are verified against the
snapshot, and the purchase is completed (pending) or reconciled (failed). A late
`CHECKOUT.ORDER.APPROVED` for a failed purchase is consent, not money — it is
never captured and is acknowledged with a diagnostic, letting the order lapse at
PayPal; when funds actually moved, the capture event reconciles.

Client recovery. `useProgramCheckout` now polls ownership briefly (5 × 1 s)
after a 409 `PURCHASE_NOT_PENDING` instead of immediately declaring failure,
and the fallback message no longer claims "no money was taken" when the payment
may already have moved.

Main changes:

- Migration: `000027_enforce_single_pending_purchase` (applied, DB at version
  `27`, `dirty=false`) adds `uq_purchases_pending_purchase`; down migration drops
  it. Repository `Create` maps the violation to `ErrPurchaseAlreadyExists`.
- Repository: `repositories/purchase_repository.go` adds
  `ReconcileFailedPurchase` (one transaction: supersede stale pending siblings,
  guarded failed→completed with `ErrCompletedPurchaseExists` /
  `ErrPurchaseNotFound` verdicts, restore-or-create entitlement with
  `ErrEntitlementAlreadyExists`).
- Service: `services/purchases/purchases.go` adds the `ReconcilePurchase` service
  method and `ErrReconciliationConflict`; an unknown purchase is a settled state
  (`ErrPurchaseNotCompleted`), a completed purchase is idempotent, and intent
  creation handles the concurrent-writer duplicate.
- Webhooks: shared `reconcileLateSettlement` helper; Stripe routes a `failed`
  purchase into reconciliation; PayPal gains the combined verifier surface
  (`PayPalWebhookVerifier`), a `PAYMENT.CAPTURE.COMPLETED` handler backed by a
  server-side order read, and the late-approval diagnostic (never captured).
- Tests: repository reconcile suite (success, restore, conflicts, sibling
  supersede, single-pending rejection), service reconcile suite, and webhook
  late-settlement suites for both providers including the no-refund sweep.
- Frontend: `use_program_checkout.ts` polls ownership after 409
  `PURCHASE_NOT_PENDING` and uses an honest confirmation message.
- Docs: `api.MD`, `security.MD`, `project.MD` and `database.MD` document the
  reconciliation lifecycle, the single-pending guarantee and migration `000027`.

---

# Version 2.84.0

Title: Purchase integrity, payment recovery and transaction safety — expired-purchase supersede, published-program revalidation at initiation, and atomic entitlement restoration

Date: 2026-10-09

Summary:

v2.84.0 closes the purchase-integrity audit of the payment lifecycle with three
production fixes and one removal. No schema change was made: the database stays
at migration version `26` with `dirty=false`, and no migration file was added,
modified or removed. Expiry is derived and never stored, so superseding an
expired purchase needs no new column. No refund, regeneration, Premium Level 2
or new payment provider surface was introduced; the proven PayPal/Stripe
return-context handling was re-verified and left untouched.

Second charge protection for abandoned checkouts. A pending purchase could be
paid at the provider without any server-side time limit, so a buyer who created
an intent and abandoned it could be charged by a provider session opened long
after the checkout was meant to end. A pending purchase is now payable only
within a payment window measured from its creation timestamp (24 hours by
default, configurable server-side via `WithPurchaseExpiryWindow`): once the
window lapses, `InitiatePayment` refuses with `409 PURCHASE_EXPIRED` and never
opens a new provider session. The buyer recovers by starting a fresh intent,
which supersedes the expired purchase — it is marked `failed`, closing the
pending slot — and returns a new pending purchase. The supersede only ever acts
on the pending row and is guarded at the database level (`WHERE status =
'pending'`), so a concurrently completed purchase or a competing supersede is
left untouched and ownership is re-evaluated before a fresh intent is allowed.
A `failed` purchase is terminal: it cannot be initiated, completed or reach
access. Completion paths deliberately ignore expiry: a provider-verified
payment that settles after the window still completes instead of being
discarded, so a buyer who genuinely paid is never stranded.

No provider payments for retired programs. `InitiatePayment` revalidated
nothing about the program after the intent was created, so a pending purchase
for a program that was unpublished or soft-deleted in the meantime could still
open a new provider session. Initiation now reloads the program through the
published-only lookup and refuses new provider contact with
`404 PROGRAM_NOT_FOUND` when it is no longer published. Completion is
deliberately unaffected: money already captured at the provider still
completes, and access rendering already derives from publication.

Atomic entitlement restoration. When a buyer completed a repurchase over a
soft-deleted entitlement, the restore and the completion were separate
database operations: a crash or a racing completion between them could leave an
active entitlement with a pending (or missing) purchase. The purchase
repository now provides `CompleteWithRestoredEntitlement`, which restores the
soft-deleted entitlement and completes the purchase inside a single transaction
and rolls everything back if the purchase is not pending. The service drops its
two-step restore-then-complete path and the dead `payment payments.Provider`
constructor argument from `purchases.NewService`. Error mapping for the new
expiry refusal was added to the handler surface.

Frontend recovery. The shared checkout hook recovers from `PURCHASE_EXPIRED`
transparently: it creates a fresh intent once (the backend supersedes the
expired purchase) and continues the checkout instead of showing a dead end. The
guard is bounded — a checkout freshens its intent at most once.

Main changes:

- Expiry: `services/purchases/purchases.go` gains `ErrPurchaseExpired`,
  `defaultPurchaseExpiryWindow` (24h) and `WithPurchaseExpiryWindow`;
  `CreatePurchaseIntent` supersedes an expired pending purchase (`Fail`) and
  re-checks ownership; `InitiatePayment` refuses expired purchases.
- Published revalidation: `InitiatePayment` reloads the program via
  `FindPublishedByID` and refuses (`ProgramNotFound`) before contacting any
  provider; completion paths are untouched.
- Atomic restoration: `repositories/purchase_repository.go` adds
  `CompleteWithRestoredEntitlement` (single tx, restore rolled back on
  non-pending) and `Fail` (pending-guarded); the service completion path uses
  the atomic method and falls back to a fresh entitlement only when nothing is
  left to restore; the service interface drops `Complete` and
  `RestoreByUserAndProgram`.
- Removal: `payment payments.Provider` parameter removed from
  `purchases.NewService` (dead argument); `routes/routes.go` and all test call
  sites updated.
- Handler: `api/auth/purchase_handler.go` maps `ErrPurchaseExpired` to
  `409 PURCHASE_EXPIRED`.
- Tests: service tests cover expiry boundaries, supersede vs completion race,
  unpublished-program refusal, failed-purchase refusal and the not-expired
  happy path; repository tests cover `Fail` guards, restore+complete success,
  replay fail-closed and rollback on non-pending purchases; handler table adds
  the `PURCHASE_EXPIRED` code.
- Frontend: `use_program_checkout.ts` handles `PURCHASE_EXPIRED` with a
  one-shot fresh-intent recovery.
- Documentation: `api.MD` (initiation validity window + `409 PURCHASE_EXPIRED`
  + `404 PROGRAM_NOT_FOUND` at initiation, completion ignoring expiry, restore
  atomicity, failed status semantics), `security.MD` (expiry, published check,
  restore rollback), `DATABASE/database.MD` (derived expiry — no migration;
  `CompleteWithRestoredEntitlement`; failed status), `project.MD` (payment
  window and authoritative completion wording), and this update log. Versions
  bumped to 2.84.0 where the document changed.

# Version 2.83.0

Title: End-to-end Premium Level 1 integration audit and production hardening — envelope consistency, questionnaire contract agreement, bounded Test Mode sessions and safer payment method binding

Date: 2026-10-08

Summary:

v2.83.0 is the result of an end-to-end audit of the Premium Level 1 purchase
lifecycle (catalog → intake → purchase → payment → capture → entitlement → plan
delivery → access). The milestone found and fixed genuine integration defects
across the backend contract, the questionnaire payload agreement, the payment
flow and the frontend checkout recovery, and hardened two production
behaviours. No schema change was made in this build: the database stays at
migration version `26` with `dirty=false`, and no migration file was added,
modified or removed. No refund, regeneration or Premium Level 2 surface was
introduced; audit-only notes are recorded below and were deliberately left
out of scope.

Response envelope consistency. The three questionnaire endpoints
(`GET /me/questionnaire/questions`, `GET /me/programs/{id}/questionnaire`,
`POST /me/programs/{id}/questionnaire`) and the two nutrition assignment
endpoints returned their payloads flat while every other endpoint in the
platform returns `{ success, message, data }`. All five now wrap their payload
under the standard `data` envelope. The premium E2E test was updated to read
through the envelope, so the integration test asserts the real wire contract.

Questionnaire contract agreement. The intake submission endpoint binds on the
server side, so a catalog type that does not match the payload type silently
fails to save. `training_days_per_week` was declared as a `select` with string
options while the payload field is an integer; the catalog now declares it as
`number` with the server's bounds, matching the int field. The catalog also
declares several `list` questions (allergies, intolerances, excluded foods,
disliked foods, medical conditions, injuries, limitations) that the form could
not render — they fell through to the default text input and produced a string
that the `*[]string` server field rejects. The form now renders `list` fields as
one free-typed entry per line in a multi-line control (blank lines dropped,
entries trimmed), so every server-declared type has a faithful control.

Bound Test Mode sessions. The persona session had no server-side lifetime:
`GET /auth/test-mode` and persona purchases accepted any session whose cookie
had not expired, regardless of how old the entry was, because the server kept no
record of time alongside the session. Persona sessions are now bounded by the
same lifetime as the access token (`ACCESS_TOKEN_TTL`, configured server-side
via `WithSessionTTL`); an expired session has no active persona and an absent
persona answers `false`/`403` exactly as if no session existed. TTL zero means
unlimited (test-only defaults preserved). Exit is deliberately exempt from the
expiry check so the stored administrator identity can always be restored and an
admin is never stranded in an impersonated context. The existing disabled-path
behaviour is now explicitly tested (`TEST_MODE_DISABLED`), and the
socket/restore surfaces preserved the previous behaviour.

Safer payment method binding. `InitiatePayment` recorded the validated method
on the purchase as soon as the provider was resolved, so a provider rejection
left a method bound to a purchase that had no provider payment — the buyer could
not retry with another method without a mismatch refusal. The method is now
written only after the provider accepts the initiation; a rejection leaves the
method unbound and retryable with any other method. The repository
`SetPaymentMethod` gained a `WHERE payment_method IS NULL` single-writer guard
as defense in depth, and a same-method replay (the supported retry path) is no
longer re-persisted. Capture and webhook flows are unchanged: they still re-derive
the provider only from the recorded method, never from client input.

Frontend checkout recovery. Payment return URLs are configured for the Generic
Program page and are shared by both product families, but the frontend had no
route matching `/services/generic-program/{program_id}/purchase/success` or
`.../cancelled`, so a returning buyer landed on the 404/`NotFound` fallback
instead of the checkout detail page. Both exact routes now render
`GenericProgramDetailPage` and are declared before the `/services/:serviceSlug`
catch-all. The `useProgramCheckout` hook no longer resolves the retry method
from the currently selected control (which may not be the method the purchase is
bound to) — a retried payment now reads the purchase's recorded method from the
purchase history and resolves the provider from it. The mount-time capture
return leg runs before and independently of the product-detail load, so a buyer
who already paid at the provider is never left un-captured because the program
data failed to load.

The database `payment_method` column documentation is unchanged in structure
(it still exists, migration `000024`) but the documented write timing now
reflects "only after provider acceptance". Test session documentation was added
to `api.MD` and `security.MD`.

Main changes:

- Backend contract: `questionnaire_handler.go` and `nutrition_handler.go` wrap
  success payloads in `{ success, message, data }`; `premium_e2e_test.go`
  asserts through the envelope.
- Questionnaire: the catalog (`services/nutrition_questionnaire/questionnaire.go`)
  declares `training_days_per_week` as `number` (Min/Max from server bounds)
  instead of `select`; frontend `premium_questionnaire_form.tsx` added a `list`
  multi-line control.
- Test Mode: `services/test_mode/test_mode.go` gains `WithSessionTTL` +
  `Option` and an `enabled` gate that is now tested; `routes/routes.go` wires
  `WithSessionTTL(jwtCfg.AccessTokenTTL)`; new tests cover expiry, zero-TTL,
  disabled and exit-after-expiry.
- Payments: `services/purchases/purchases.go` records the method only after
  provider acceptance and only when unbound; `repositories/purchase_repository.go`
  guards with `WHERE payment_method IS NULL`; `purchases_test.go` covers the
  provider-rejection-releases-method path and the no-re-persist replay.
- Frontend: `app_routes.tsx` registers the two payment return routes before the
  slug catch-all; `use_program_checkout.ts` resolves the retry method from the
  purchase record and runs the return-leg capture independently of the detail
  load.
- Documentation: `api.MD` (envelope, catalog types incl. `list`, payment
  binding timing, Test Mode lifetime), `security.MD` (payment single-writer,
  persona session TTL), `DATABASE/database.MD` (payment_method write timing),
  `PAGES/design.MD` (list control), `how_to_run.MD` (return routes),
  `project.MD` (payment initiation wording), and this update log. Versions
  bumped to 2.83.0 where the document changed.
- Audit notes deliberately left out of scope (no change): an inactive pending
  purchase never expires (frontend recovery resumes it), the `failed` purchase
  status is currently unreachable, a dead `payments.Provider` constructor
  parameter, a non-transactional entitlement restore path, the exported
  `CreateEntitlement`/`RevokeEntitlement` surface, the published re-check at
  completion, PayPal return-context plumbing, and the hardcoded prices / Premium
  Level 2 card in frontend constants.

---

# Version 2.82.0

Title: Nutrition plans are now normalised, versioned and server-owned — generator v2 with an immutable plan lifecycle

Date: 2026-10-06

Summary:

v2.82.0 replaces the v2.81.0 single JSON plan snapshot with a normalised,
versioned plan domain and rewrites the generator around it. A delivered plan is
no longer an opaque document: the daily contract, every meal, every food and the
exclusions audit are persisted as rows, and each food is an immutable snapshot
taken from a server-owned food catalog that ships with the engine as code.
Because the plan is immutable and versioned, nothing can ever silently rewrite
what a client was shown, and the storage engine itself guarantees at most one
active plan per assignment.

The generator is upgraded to a fully deterministic search over that catalog. For
a given intake it always produces the same plan — the same fingerprint — which
makes regeneration idempotent and provably in place: a plan whose intake has not
changed is returned as it is, a failed run is retryable on the same revision, and
Plan delivery is now part of a completed purchase. The purchase service runs a
post-commit fulfilment hook that generates and stores the plan before the
checkout finishes; the hook is idempotent and its failures are logged and
deliberately swallowed, because no nutrition problem may ever unwind a verified
payment. A failure leaves the assignment retryable in place and never destroys a
previously delivered plan.

The API contract gains an explicit on-demand path,

```text
POST /api/v1/me/programs/{programID}/nutrition/generate
```

that is idempotent when the intake is unchanged and supersedes the delivered plan
when the intake revision has moved on. Supersession is atomic: the next plan
version is inserted, the previous active version is marked superseded and the
assignment completes at the new revision in one transaction with the assignment
row locked, so a concurrent run can neither observe a half-written version nor
race to deliver a competing plan. The claim itself is compare-and-swap, so two
simultaneous runs never both generate, and database constraints (
`uq_nutrition_plans_active`, `uq_nutrition_plans_version`) back the same
guarantee. A set of stored restrictions that cannot be satisfied now answers
`422 NUTRITION_RESTRICTIONS_UNSATISFIABLE` with a fixed sentence that names no
health data and alters nothing.

The plan document stays exclusively a read view. It exposes the version,
`engine_version`, `fingerprint`, `out_of_date` and the full plan body, with every
meal item as an immutable snapshot that shows the catalog code it came from, the
exact quantity and unit, its macros and any substitution note. The exclusions
audit is written from controlled server tokens (`reason_code`) and normalised
vocabulary entries (`token`), never from the free text a client typed, so no
sensitive health answer is ever duplicated outside the questionnaire row; free
text that no token resolves surfaces only as a generic caution and is never
echoed. The internal `failure_reason` of an assignment remains never-served.

On the frontend, the nutrition panel is a real renderer of the delivered
contract instead of summary tiles. It shows the daily targets, the day's meals as
a list of actual foods with quantity, unit and macros, substitution notes, the
exclusions audit, the server cautions, hydration and preparation guidance. A plan
the server marks `out_of_date` is still shown with the delivered content and a
stale banner that offers to update from the latest answers; a generation refusal
shows client-owned wording for the server reason code; and a failed refresh never
hides a delivered plan, because the page re-reads the authoritative state before
deciding what to render. A delivered plan is otherwise read-only: the only calls
to generation are the pending/failed build and retry actions and the defensive
stale-banner update.

The repository layer is now integration-tested against the real schema. New tests
cover the assignment state machine (idempotent pending, owner and program
scoping across owners, compare-and-swap claims rejecting in-flight and stale
runs, failure and retry) and the plan lifecycle (all rows persist and round-trip,
cross-owner reads fail closed, supersession retains the previous body, the
database rejects a second active plan, a stale questionnaire revision is refused,
and unclaimed or cross-owner completion is refused). The premium E2E test was
corrected to the real fulfilment contract: a completed purchase arrives with a
`completed` plan, regeneration is fingerprint-identical, and exactly one
assignment and one active plan row exist afterwards.

One defect was found and fixed during the work. Child-plan rows (meals, meal
items, exclusions) were linked and keyed inside the assignment service, splitting
ownership of a plan's identity across two layers; the repository now stamps
`plan_id` and `meal_id` and mints the IDs, so the service places formed plan
content and the persistence layer owns row identity. The retained
`nutrition_assignments.configuration` column is documented as deprecated leftover
from v2.81.0.

Main changes:

- Migration `000026_persist_nutrition_plans` adds `nutrition_plans`, with a
  `STORED` generated `active_plan` column enforcing one active plan per
  assignment (`uq_nutrition_plans_active`), `UNIQUE (assignment_id, version)`,
  `engine_version`, `fingerprint`, the normalised macro body, controlled
  `cautions` and soft deletes; `nutrition_plan_meals` with a `meal`/`snack`
  `CHECK` and unique positions; `nutrition_plan_meal_items` as immutable food
  snapshots with `catalog_code` provenance; and `nutrition_plan_exclusions` as
  the controlled-token audit trail. Four read views,
  `v_nutrition_plan`, `v_nutrition_plan_meal`, `v_nutrition_plan_meal_item` and
  `v_nutrition_plan_exclusion`, become the only read surface for a plan.
- The generator (`services/nutrition_assignment`, engine v2) is deterministic,
  catalog-backed, idempotent and supersede-aware, with compare-and-swap claims
  and a `NUTRITION_RESTRICTIONS_UNSATISFIABLE` path.
- Purchase fulfilment runs a post-commit hook that generates and persists the
  plan; failures are logged and swallowed so a payment is never undone.
- `POST /nutrition/generate` is the explicit on-demand endpoint (idempotent,
  supersede, `409 NUTRITION_NOT_READY`, `422` on unsatisfiable restrictions).
- The frontend nutrition panel (`premium_nutrition_plan.tsx`) renders the full
  delivered contract with a stale banner and client-owned failure wording, on
  both the Premium Level 1 page and Program Access.
- Repository integration tests for the assignment and plan lifecycle and a
  corrected premium E2E test.
- Documentation updated across `security.MD`, `arquitecture.MD`, `database.MD`,
  `design.MD`, `api.MD` and `how_to_run.MD`; `nutrition_assignments.configuration`
  is documented as deprecated, and the plan document replaces the old JSON
  snapshot contract.

---

# Version 2.81.0