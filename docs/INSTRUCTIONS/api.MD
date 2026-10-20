# RYZE FITNESS
# API.MD

Version: 2.85.0

---

# PURPOSE

This document defines the complete REST API architecture used by the RYZE platform.

It is the single source of truth regarding:

- API architecture
- Endpoints
- Authentication
- Authorization
- Request structure
- Response structure
- Error handling
- Versioning
- Permissions
- Validation
- Uploads
- WebSockets

Every endpoint implemented in the backend must follow this document.

The API is expected to evolve during development.

Whenever a better architecture is identified, this document should be updated before implementation.

---

# API PHILOSOPHY

The API is designed with the following priorities:

- Simplicity
- Consistency
- Security
- Performance
- Scalability
- Maintainability

The API should always remain predictable.

Similar operations must behave similarly.

Responses should always follow the same structure.

Breaking changes should be avoided whenever possible.

When breaking changes become necessary, a new API version should be created.

---

# API STYLE

RYZE uses a RESTful architecture.

Resources should be represented by nouns rather than verbs.

Examples:

Good

```
GET /packs

GET /users

GET /notifications
```

Bad

```
GET /getPacks

POST /createPack

GET /retrieveNotifications
```

HTTP methods must represent the intended action.

GET

Read data.

POST

Create resources.

PUT

Replace a complete resource.

PATCH

Update part of a resource.

DELETE

Soft delete whenever applicable.

---

# API VERSIONING

Every endpoint is versioned.

Current version:

```
/api/v1
```

Examples

```
GET /api/v1/packs

POST /api/v1/auth/login

PATCH /api/v1/users/me
```

Future breaking changes should create:

```
/api/v2
```

instead of modifying v1 behavior.

---

# DATA FORMAT

All communication uses:

```
application/json
```

except:

- File uploads
- Image uploads
- PDF uploads

Those use:

```
multipart/form-data
```

Character encoding:

```
UTF-8
```

---

# RESPONSE FORMAT

Every endpoint must return the same response structure.

Successful response

```json
{
    "success": true,
    "message": "Operation completed successfully.",
    "data": {}
}
```

Failed response

```json
{
    "success": false,
    "message": "Validation failed.",
    "error": {
        "code": "VALIDATION_ERROR",
        "details": []
    }
}
```

Responses should always be easy to parse.

---

# HTTP STATUS CODES

The API should use proper HTTP status codes.

200

Successful request.

201

Resource created successfully.

204

Successful request without response body.

400

Invalid request.

401

Authentication required.

403

Permission denied. A browser-originated state-changing request whose origin is
neither the API's own host nor an allow-listed origin (`CORS_ALLOWED_ORIGINS`)
is rejected before the handler runs with `CROSS_ORIGIN_FORBIDDEN`; requests
with no browser origin context (curl, webhooks, server-to-server) are not
affected.

404

Resource not found.

409

Conflict.

413

Request body too large. Returned before any handler runs when a request body
exceeds the configured limit.

422

Validation error.

429

Too many requests. Returned as `RATE_LIMITED` by the authentication rate
limiter with a `Retry-After` header naming the whole seconds left in the
window.

500

Internal server error.

---

# AUTHENTICATION

Authentication uses:

JWT (JSON Web Tokens)

Authorization header:

```
Authorization: Bearer <token>
```

The backend is responsible for:

- Token generation
- Token validation
- Token expiration
- Refresh tokens
- Logout invalidation

Passwords are never stored in plain text.

Passwords must always be hashed using a secure algorithm.

Google OAuth authentication is also supported.

---

# AUTHORIZATION

Every authenticated user has exactly one primary role.

Supported roles:

- Client
- Trainer
- Technical Administrator
- Management Administrator

Permissions are determined exclusively by the backend.

Frontend restrictions are visual only and must never replace backend authorization.

Administrator authorization is enforced by reusable backend middlewares
(`middleware.RequireAdminRole` and `middleware.RequireAdminPermission`) mounted
after the admin authentication middleware. The role and its permissions are
always derived from the authenticated admin identity (ADMIN_1 → Technical
Administrator, ADMIN_2 → Management Administrator); roles or permissions
supplied by the client are never accepted.

Authentication failures return HTTP 401 `AUTHENTICATION_REQUIRED`.
Authorization failures return HTTP 403 `FORBIDDEN` with a generic message.

---

# ROLE PERMISSIONS

## Public

Accessible without authentication.

Examples:

- Landing page
- Public trainer profiles
- Generic packs
- Public program catalog
- Authentication endpoints

---

 ## Authenticated User

Accessible after login.

Examples:

- Personal profile
- Purchases
- Notifications
- Messages
- Current pack

### Client Program Read

`GET /api/v1/me/program` returns the full active structure of the program
currently assigned to the authenticated user, read-only.

Identity comes exclusively from the authentication context. A client-supplied
`user_id`, `client_id` or `trainer_id` (query, body or header) is always ignored
and can never influence which program is returned. The endpoint is read-only: a
client can never create, modify or delete programs, weeks, workouts, workout
exercises, exercises or assignments.

The owning `trainer_id` and every parent identifier (`program_id`,
`program_week_id`, `program_workout_id`) are never exposed. Weeks, workouts,
workout exercises and exercises are returned in display order
(`week_number` / `position` ascending). Soft-deleted programs, weeks, workouts,
workout exercises, exercises and assignments are excluded. The program `status`
carries no access semantics: a draft program is as readable as a published one.
A user holding several active assignments (one per trainer) reads the most
recently created one.

---

#### `GET /api/v1/me/program`

Returns the program of the authenticated user's most recent active assignment
with its full active structure (weeks, workouts, workout exercises and their
catalog exercises).

Response `data`:

- `id` / `name` / `description` / `type` / `status` — public program metadata.
- `created_at` / `updated_at` — program timestamps.
- `weeks` — array of week objects, each with:
  - `id` / `week_number` / `created_at` / `updated_at`.
  - `workouts` — array of workout objects, each with:
    - `id` / `position` / `created_at` / `updated_at`.
    - `exercises` — array of workout exercise objects, each with:
      - `id` / `position` / `created_at` / `updated_at`.
      - `exercise` — safe catalog summary: `id`, `name`, `description`,
        `target_muscles`, `equipment`, `difficulty`, `video_url`, `image_url`.

Responses:

- `200 OK` — the assigned program with its full structure; programs without
  weeks, workouts or workout exercises yet return empty arrays.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.
- `404 PROGRAM_NOT_FOUND` — no active assignment or the assigned program is not
  readable; both cases are indistinguishable and never reveal why.

---

### Client Workout History

`POST /api/v1/me/workouts/{workoutID}/complete` records that the authenticated
user completed one workout of their currently assigned program.
`GET /api/v1/me/workouts/history` returns the user's own completed workouts,
newest first, with pagination.

Identity comes exclusively from the authentication context. A client-supplied
`user_id`, `client_id` or `trainer_id` (query, body or header) is always ignored
and can never influence which workout is completed or which history is returned.
A workout is only executable when it belongs to the active structure of the
user's most recent active assignment; a foreign, soft-deleted or unknown workout
and a missing assignment are all indistinguishable and map to a single
`404 WORKOUT_NOT_FOUND`. The owning user id, trainer id and every parent
identifier (`program_id`, `program_week_id`) are never exposed.

Repeated executions are preserved: completing the same workout more than once
creates a new history entry. Completion is recorded server-side; no
client-supplied timestamp or identity is ever accepted. This foundation records
only the completion itself — no per-exercise data (sets, reps, rest, weights,
duration) is stored yet.

---

#### `POST /api/v1/me/workouts/{workoutID}/complete`

Marks a workout of the authenticated user's currently assigned program as
completed. The `workoutID` path parameter identifies the requested resource and
never proves access: only workouts belonging to the active structure of the
user's most recent active assignment are executable.

Response `data`:

- `id` — the generated history entry id.
- `program_workout_id` — the completed workout id.
- `completed_at` — server-side completion timestamp.
- `created_at` / `updated_at` — entry timestamps.

Responses:

- `201 Created` — the history entry was created.
- `400 VALIDATION_ERROR` — malformed workout id.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.
- `404 WORKOUT_NOT_FOUND` — no active assignment, or the workout does not belong
  to the assigned program's active structure; both cases are indistinguishable
  and never reveal why.

---

#### `GET /api/v1/me/workouts/history`

Returns one page of the authenticated user's own completed workouts, newest
first.

Query parameters:

- `page` — page number, defaults to `1`.
- `limit` — page size, defaults to `20`, capped at `100`.

Response `data`:

- `entries` — array of history entry objects, each with `id`,
  `program_workout_id`, `completed_at`, `created_at`, `updated_at`.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

Responses:

- `200 OK` — the user's own history; a user without entries gets an empty array.
- `400 VALIDATION_ERROR` — invalid pagination values.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.

---

### Client Entitlements

#### `GET /api/v1/me/entitlements`

Returns the safe entitlement metadata for every active entitlement held by the
authenticated user. Each entitlement carries only its own id, the associated
program id and a safe nested Program object. Entitlements grant access to the
entitlement-protected program read `GET /api/v1/me/programs/:programID`; they do
NOT grant access to `GET /me/program`, which remains assignment-based.

Response `data`:

- array of entitlement objects, each with:
  - `id` — entitlement UUID.
  - `program_id` — the entitlement program reference.
  - `program` — safe Program object: `id`, `name`, `description`, `type`,
    `status`, `level`, `training_type`, `duration_weeks`,
    `frequency_per_week`, `price_minor_units`, `currency`,
    `created_at`, `updated_at`. The four calendar metadata fields can be
    `null` when the program does not define them.
  - `created_at`, `updated_at` — entitlement lifecycle timestamps.

Responses:

- `200 OK` — the user's entitlements; a user without entitlements gets an
  empty array.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.

#### `GET /api/v1/me/programs/:programID`

Returns the client-safe structure of one program the authenticated user is
entitled to access. Access is granted only when BOTH conditions hold: the
authenticated user holds an active entitlement for the program AND the program
itself is still published (matching the marketplace contract). A missing
entitlement, a missing program, a soft-deleted program and an unpublished
program are all indistinguishable and return the same generic `404`. The user
identity comes exclusively from the authentication context; the path parameter
only identifies the requested resource.

Response `data` — the safe published program detail, identical in shape to the
public marketplace detail:

- `id`, `trainer_id` (NULL for platform-owned programs), `name`, `description`,
  `type`, `status`, `level`, `duration_weeks`, `frequency_per_week`,
  `training_type`, `price_minor_units`, `currency`, `created_at`, `updated_at`.
- `weeks` — array of weeks, each with `week_number` and `workouts`; each
  workout has `position` and `exercises`; each exercise has `name`,
  `description`, `instructions`, `target_muscles`, `equipment`, `difficulty`,
  `video_url`, `image_url`, `position`, `notes` and `sets`; each set has
  `set_number`, `set_type`, `reps`, `weight_kg`, `rir`, `rpe`,
  `rest_seconds`, `tempo`.

Responses:

- `200 OK` — the entitled, published program with its full structure.
- `400 VALIDATION_ERROR` — malformed program id.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.
- `404 PROGRAM_NOT_FOUND` — no active entitlement, program unpublished,
  soft-deleted, or nonexistent. The cases are indistinguishable by design.

---

### Client Purchase

**RYZE purchases are final.** A customer purchases a plan once and the purchase
is final: the product does not expose or support a refund workflow. There are no
refund endpoints, no refund status, and no refund management. The purchase
lifecycle moves through `pending` → `completed` (or `failed`, when payment does
not complete, and `failed` → `completed` only through the internal
reconciliation path described below, exclusively for provider-verified late
settlements).

#### `POST /api/v1/me/programs/:programID/purchase`

Creates a pending purchase transaction for the authenticated user. The endpoint
validates the program, snapshots the current price and currency, resolves the
applicable commission, calculates the deterministic split, and persists a
pending purchase record. The purchase status is always "pending"; no payment is
processed and no entitlement is created. Completion happens only through a
server-verified capture or a verified payment provider event, which transitions
the purchase to "completed" and creates the entitlement atomically.

Identity comes exclusively from the authentication context. A client-supplied
`user_id` or `program_id` in the body is always ignored.

Request: no body required.

Response `data`:

- `id` — purchase UUID.
- `program_id` — the purchased program reference.
- `price_minor_units` — snapshot of the program price at creation time.
- `currency` — snapshot of the program currency at creation time.
- `status` — always `"pending"`.
- `payment_method` — present only after a payment method has been initiated for
  the purchase (see the payment endpoint below); omitted at creation time and
  never included for Test Mode / zero-price purchases.

The internal trainer economics — `commission_bps`, `platform_amount` and
`trainer_amount` — are deliberately **not** part of any client-facing purchase
response, including this one. They are not needed by the client and remain
internal to RYZE, so a purchase has the same public shape whichever endpoint the
browser calls. The commission split is still snapshotted on the purchase record
at creation time; it is simply never serialized to a client.

Responses:

- `201 CREATED` — the pending purchase snapshot.
- `400 VALIDATION_ERROR` — invalid program id.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.
- `404 PROGRAM_NOT_FOUND` — program does not exist, is soft-deleted or is
  not published.
- `409 PROGRAM_NOT_PURCHASABLE` — the program is free and cannot be purchased.
- `409 DUPLICATE_ENTITLEMENT` — the user already owns this program.
- `409 DUPLICATE_PURCHASE` — a non-expired purchase for this program is already
  in progress.

Expiry: a pending purchase is only payable within its payment window (24 hours
by default, derived from its creation timestamp — it is never persisted). Once
the window lapses, the next intent creation for the same program supersedes the
expired purchase: it is atomically marked `"failed"` (`purchases.status`) so the
pending slot is closed and a fresh intent is returned. An expired purchase can
never be resumed; a failed purchase is terminal, cannot be paid or completed and
never grants access. If a concurrent completion wins the race during a
supersede, ownership is re-evaluated and the request refuses with
`409 DUPLICATE_ENTITLEMENT` instead of reopening the slot.

A `failed` purchase can only come back through the internal reconciliation
path: a provider-verified late settlement (`ReconcilePurchase`, below) returns
it to `"completed"` with its entitlement when no other sale of the same pair
already resolved. A client can never trigger this — no route accepts a purchase
id without an authenticated owner on the reconciliation surface.

Commission exemption: platform-owned generic programs (`trainer_id` NULL) have
no owning trainer, so no commission rule is resolved and no commission is
applied. For these purchases `commission_bps` is always `0`, `trainer_amount`
is `0` and the platform keeps the full amount (`platform_amount` equals
`price_minor_units`).

#### `GET /api/v1/me/purchases`

Returns every active purchase transaction belonging to the authenticated user,
most recently created first. Purchases carry the full commercial snapshot
captured at creation time, so historical prices remain accurate even after a
program is re-priced.

This is the client-facing purchase **history** endpoint and it intentionally
exposes no internal payout metadata: commission basis points, platform amount
and trainer amount are never included. The create and capture responses carry
exactly the same public purchase shape, so those fields are never exposed to a
client by any endpoint. Each entry carries a safe program
summary (see `program` below) and two computed flags: `access`, which reflects
whether the user can currently open the program (completed purchase of a
published, active program), and `test`, which marks zero-price Test Mode
purchases.

Identity comes exclusively from the authentication context. A cross-user lookup
is impossible: the endpoint only ever returns the authenticated user's purchases
and never exposes the internal `user_id` in the response.

Authentication: required (client session).

Response `data` (array of purchase history objects, newest first):

- `id` — purchase UUID.
- `program_id` — the purchased program reference.
- `price_minor_units` — snapshot of the program price at creation time.
- `currency` — snapshot of the program currency at creation time.
- `status` — `"pending"`, `"completed"` or `"failed"`. A `failed` purchase is
  terminal: it was superseded (for example an expired pending purchase reopened
  by a fresh intent) or abandoned, and it never grants access.
- `test` — boolean; `true` for zero-price Test Mode purchases.
- `access` — boolean; `true` only when the purchase is `completed` and the
  program is published and not deleted. A completed purchase of a retired or
  deleted program keeps its history but loses `access`.
- `payment_method` — the method selected at payment initiation (`"card"`,
  `"paypal"`); omitted when the purchase never reached a payment
  provider (Test Mode purchases, legacy pending purchases).
- `created_at`, `updated_at` — purchase lifecycle timestamps.
- `program` — safe program summary: `id`, `name`, `description`, `type`,
  `status`, `level`, `training_type`, `duration_weeks`, `frequency_per_week`,
  `price_minor_units`, `currency`, `created_at`, `updated_at`. The summary
  mirrors the entitlement program summary and never carries trainer or payout
  metadata.

Responses:

- `200 OK` — the purchase history for the authenticated user.
- `400 VALIDATION_ERROR` — invalid internal state.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.

Duplicate protection: a user can complete at most one purchase per program. A
pending purchase in progress blocks a second purchase through the service layer,
and the database enforces the completed-purchase invariant with a unique
constraint over the active `(user_id, program_id)` pair (see migration 000022).

#### Internal: `CompletePurchase`

The `CompletePurchase` service method is an internal-only operation with no HTTP
endpoint. It is invoked by the capture flow and the payment-provider
webhook/adapter handlers once external payment confirmation is received.

The method accepts a purchase id, loads the purchase, and atomically transitions
it to "completed" while provisioning the entitlement. It is idempotent:
calling it on an already-completed purchase with an existing entitlement
returns a success; if the entitlement is missing, an integrity error is raised.

Completion deliberately ignores purchase expiry: a provider-verified payment
that settles after the payment window still completes instead of being
discarded. When the buyer already owns a soft-deleted entitlement for the
program (for example a previous purchase that was later reversed), completion
restores that entitlement and completes the purchase inside a single database
transaction instead of inserting a duplicate row; if the purchase is not
pending by the time the payment is verified, the restore is rolled back and the
completion refuses, never leaving a restored entitlement behind a non-completed
purchase.

Error codes (service-level, not HTTP):

- `ErrPurchaseNotFound` — purchase does not exist, is soft-deleted or not in a
  completable state.
- `ErrPurchaseNotCompleted` — purchase is in a terminal state (failed) that
  prevents completion.
- `ErrEntitlementIntegrity` — purchase is completed but entitlement is missing;
  a data consistency violation.
- `ErrInvalidInput` — empty purchase id.

#### Internal: `ReconcilePurchase`

The `ReconcilePurchase` service method is an internal-only operation with no
HTTP endpoint, intended exclusively for the verified webhook layer: no route
accepts a purchase id without an authenticated owner on it, so a browser can
never trigger it.

Every call site records its outcome in `purchase_reconciliation_events`, keyed
by provider event id (see [ADMIN COMMERCE: Reconciliation Events](#reconciliation-events)).

It completes a purchase whose provider-verified, contract-matched payment
arrived after the purchase left `pending`. A `failed` purchase (superseded or
expired intent) is atomically returned to `completed` — any other pending intent
for the same `(user, program)` pair is superseded, the guarded failed→completed
transition runs, and the entitlement is restored or created, all in one
transaction. The database uniqueness guards (`uq_purchases_completed_purchase`
and `uq_entitlements_active_entitlement`) make double-granting impossible: if a
completed purchase or an active entitlement already exists for the pair, the
whole transaction rolls back and the purchase stays `failed` as the persisted,
diagnosable recovery state.

- An already-completed purchase is a safe idempotent success and still runs the
  delivery hook; a completed purchase without an entitlement is an integrity
  error.
- A pending purchase surfaced with `ErrPurchaseNotCompleted`, as is an unknown
  purchase id (a settled state, never a retryable fault).

Error codes (service-level, not HTTP):

- `ErrReconciliationConflict` — the pair already resolved (a completed purchase
  or an active entitlement exists), so the late payment cannot define a second
  sale; the failed record is kept untouched.
- `ErrPurchaseNotCompleted` — purchase unknown or not in a failed or completed
  state.
- `ErrEntitlementIntegrity` — completed purchase without its entitlement.
- `ErrInvalidInput` — empty purchase id.

#### `POST /api/v1/me/purchases/:purchaseID/payment`

Initiates a payment for an existing pending purchase via the configured payment
provider. The purchase must belong to the authenticated user and be in "pending"
status. The client selects the payment method (`card` or `paypal`, as
advertised by `GET /api/v1/payments/methods`); the server validates it and routes
it to the correct provider via a centralized `MethodProviderMap`. On a
configured provider, the validated method is recorded on the purchase
(`purchases.payment_method`) only after that provider has accepted the
initiation, so it becomes immutable for that purchase and the later capture
flow can always resolve the correct provider server-side without trusting any
client input. A provider rejection therefore leaves the method unbound: no
provider payment exists, and the buyer may retry with a different method.
The immutable purchase snapshot is used to construct the provider request; no
client-supplied commercial values are accepted. The purchase status is NOT
modified during initiation — it remains "pending" until a server-verified
capture or a verified provider event flows through the completion path.

Authentication: required (client session).

Parameters:

- `purchaseID` (path) — purchase UUID.

Request body:

- `payment_method` (required) — one of `"card"`, `"mbway"` or `"paypal"`,
  matching an advertised method.

Validation:

- `payment_method` is required and must be one of the allowed values. Any other
  value or a missing field returns `400 VALIDATION_ERROR`.
- A valid method that is not currently configured — for instance `"mbway"` while
  Stripe is unconfigured — returns `502 PAYMENT_PROVIDER_ERROR`: the method is
  never silently substituted and no purchase is completed.
- Re-initiating the **same** method is the supported retry path: it is safe to
  repeat and, for Stripe, replays the same Checkout Session instead of creating
  a second payable one.
- Re-initiating a **different** method than the one already recorded on the
  purchase returns `409 PAYMENT_METHOD_MISMATCH`. The recorded method is
  immutable: the outstanding provider payment belongs to it, so the purchase is
  never rebound and no provider is contacted. The client must resume the
  purchase with the method reported in `payment_method`.
- A pending purchase that outlived its payment window (24 hours by default)
  returns `409 PURCHASE_EXPIRED` and never opens a new provider payment. The
  buyer recovers by calling `POST /api/v1/me/purchases` again, which supersedes
  (marks `failed`) the expired purchase and returns a fresh intent.
- The program must still be published when the payment is initiated: a pending
  purchase for a program that was retired after the intent was created returns
  `404 PROGRAM_NOT_FOUND` and no provider session is opened. Completion paths
  are unaffected by both checks — a payment already captured at the provider
  still completes.
- A pending purchase whose (user, program) pair already resolved another way —
  a sibling purchase completed or was reconciled first — returns
  `409 DUPLICATE_ENTITLEMENT` and never opens a provider payment: the money
  could never grant access. The client recovers by checking its ownership
  state (an active entitlement answers the same way).

Response `200 OK`:

- `payment_id` — provider-specific payment identifier.
- `checkout_url` — optional redirect URL for the user to complete payment
  (omitted when the provider does not use redirect-based flows).
- `status` — provider-independent payment status (`pending`,
  `requires_action`, `failed`).
- `purchase_id` — RYZE purchase identifier.

Error codes:

- `400 VALIDATION_ERROR` — empty purchase id, missing or invalid
  `payment_method`.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid session.
- `404 PURCHASE_NOT_FOUND` — purchase does not exist, is soft-deleted or does
  not belong to the authenticated user.
- `404 PROGRAM_NOT_FOUND` — the program is no longer published, so a new
  provider payment cannot be opened for it.
- `409 PURCHASE_NOT_PENDING` — the purchase is not in "pending" status.
- `409 PURCHASE_EXPIRED` — the pending purchase outlived its payment window and
  can no longer be paid; start a fresh intent.
- `409 PAYMENT_METHOD_MISMATCH` — the requested method differs from the method
  already recorded on the pending purchase.
- `409 DUPLICATE_ENTITLEMENT` — the (user, program) pair is already resolved
  (an active entitlement or a completed sibling purchase), so the pending
  purchase can never complete and no provider session is opened.
- `502 PAYMENT_PROVIDER_ERROR` — the payment provider could not initiate the
  payment. Internal provider details are never exposed.

#### `POST /api/v1/me/purchases/:purchaseID/capture`

Verifies an approved provider payment for an existing pending purchase and
completes it. This is the endpoint invoked by the browser return callback after
the buyer completes the checkout on the provider page. The purchase must belong
to the authenticated user and be in "pending" status.

The provider payment identifier (`provider_payment_id`) is treated as untrusted
input: the backend reloads the real payment from the provider API server-side
and binds it to the purchase before anything is completed.

- **PayPal** — the identifier is the Order ID; the backend reloads the order and
  captures it, verifying that the purchase is recorded with the `paypal` method,
  the purchase-unit reference id, the amount and the currency against the
  immutable purchase snapshot.
- **Stripe** — the identifier is the Checkout Session ID; the backend reloads the
  session and verifies that it is a fully paid one-time payment
  (`payment_status == "paid"`, `mode == "payment"`) whose `client_reference_id`
  is this purchase, whose `amount_total` and `currency` match the immutable
  purchase snapshot, and whose `payment_method_types` contain the payment method
  type of the purchase's recorded `payment_method`. This covers both the `card`
  and the `mbway` methods, which share the same provider.

Because the capture provider is resolved from the recorded method and each
provider re-checks that same method, a payment made with one method can never
complete a purchase bound to another — a Stripe session id presented for a PayPal
purchase, or a PayPal order id presented for a card or MB WAY purchase, is
rejected before any external call.

A browser redirect alone never completes a purchase — the provider must confirm
the payment before the purchase transitions to "completed" and the entitlement
is provisioned atomically.

The capture provider is never chosen by the client: it is resolved server-side
from the payment method recorded on the purchase at `InitiatePayment`. A capture
attempt on a purchase with no recorded method fails closed with
`502 PAYMENT_PROVIDER_ERROR` and leaves the purchase pending — legacy pending
purchases created before the method column existed must be re-initiated
(which records the method) before they can be captured.

The operation is idempotent: an already-completed purchase (e.g. the webhook
racing the browser return) returns the completed snapshot safely without
re-capturing or duplicating the entitlement.

The capture also fails closed when the pair already resolved another way: a
purchase whose (user, program) pair already owns an active entitlement (a
sibling purchase completed mid-flight) is never presented to the provider —
the money could not grant access — and returns `409 DUPLICATE_ENTITLEMENT`.
When the purchase moves on while the capture is in flight (a concurrent
verified delivery completed it, or it was superseded), the outcome is read
back from the durable state instead of surfacing a stale failure: a completed
purchase returns the completed snapshot, and a superseded purchase returns
`409 PURCHASE_NOT_PENDING` so the client re-resolves the buyer's real access
instead of retrying blindly.

Authentication: required (client session).

Parameters:

- `purchaseID` (path) — purchase UUID.

Request body:

- `provider_payment_id` (required) — the provider's identifier for the payment
  returned by the checkout flow: a PayPal Order ID or a Stripe Checkout Session
  ID. It is only a correlation hint; the backend re-verifies the payment
  provider-side.

Response `200 OK`: the completed purchase snapshot:

- `id` — purchase UUID.
- `program_id` — the purchased program reference.
- `price_minor_units` / `currency` — immutable commercial snapshot.
- `status` — `"completed"`.
- `payment_method` — the method recorded on the purchase, omitted when none was
  ever recorded.

The commission split is never part of this response; see the create endpoint for
the full client-facing purchase shape.

Error codes:

- `400 VALIDATION_ERROR` — missing purchase id, missing or blank
  `provider_payment_id`, malformed body.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid session.
- `404 PURCHASE_NOT_FOUND` — purchase does not exist, is soft-deleted or does
  not belong to the authenticated user.
- `409 PURCHASE_NOT_PENDING` — the purchase is not in "pending" status (e.g.
  it already failed, or it was superseded while the payment was being
  verified).
- `409 DUPLICATE_ENTITLEMENT` — the (user, program) pair is already resolved,
  so the capture is refused and the provider is not contacted.
- `502 PAYMENT_PROVIDER_ERROR` — the provider could not be reached, the payment
  could not be verified/captured (including an unpaid, expired, foreign or
  mismatched Stripe Checkout Session), or the purchase has no recorded payment
  method so no capture provider can be resolved. No money was taken at this
  point; the purchase remains "pending" so the buyer can retry.

---

### Payment Methods

#### `GET /api/v1/payments/methods`

Public, unauthenticated endpoint exposing the payment methods the backend
currently accepts for checkout. The list is derived server-side from the
configured payment providers (`MethodProviderMap.AvailableMethods`): a method is
advertised only when a provider is actually configured for it. The checkout UI
consumes this endpoint and never hard-codes a payment method.

When no provider is configured at all the endpoint returns an empty list and
the frontend renders checkout as unavailable.

`card` and `mbway` both require Stripe to be completely configured: both
`STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`. A secret key without a signing
secret leaves both methods unadvertised, because without webhooks a payment could
never be reliably confirmed. The same pair gates
`POST /api/v1/webhooks/stripe`, so the webhook endpoint only exists when Stripe
payments are actually possible. MB WAY is served by the same Stripe provider
instance as card, so it is always advertised exactly together with card.

`mbway` additionally depends on the `mb_way_payments` capability being enabled
on the Stripe account (a per-account Stripe Dashboard setting, not a RYZE
environment variable). RYZE cannot read that setting, so enabling it is a
deployment step; without it Stripe rejects the Checkout Session and the purchase
fails closed without ever completing.

`paypal` requires the PayPal credentials to be configured.

Response `data` (array, stable order `card`, `mbway`, `paypal`, restricted to
configured methods):

- `method` — normalized method identifier (`"card"`, `"mbway"`, `"paypal"`).
- `label` — display name for the customer ("Card", "MB WAY", "PayPal").

Responses:

- `200 OK` — the currently available payment methods.

---

## Test Mode

Test Mode lets the Technical Administrator exercise the customer commerce
flows without real payments. A predefined persona (Test Client or Test Trainer)
is fixed by the backend; the client never chooses or provides a user identity.
Inside Test Mode every purchasable Generic Program renders as Free and the
test purchase completes server-side with no payment provider. Test Mode is
globally disabled via the `TEST_MODE_ENABLED` environment variable; when
disabled every Test Mode action returns `403 TEST_MODE_DISABLED`.

### `POST /api/v1/admin/auth/test-mode`

Authenticated Test Mode endpoint (admin). Enters the Test Mode session for
one of the predefined personas. Protected by the admin authentication
middleware and the `RequireAdminRole` guard; **only the Technical
Administrator (ADMIN_1) may enter Test Mode**. The backend mints the persona
travel session: it clears the admin access cookie and sets a
persona-authenticated session (`ryze_test_session`) whose identity is the
predefined persona user. The persona session is short-lived: it carries the
same server-side lifetime as the access token (the configured
`ACCESS_TOKEN_TTL`), after which the persona is treated as having no active
session. Exiting always restores the stored administrator identity regardless
of expiry, so an admin can never be stranded in persona.

Request body:

- `persona` (required) — one of the whitelisted personas: `"client"`
  (Test Client) or `"trainer"` (Test Trainer).
- `return_path` (required) — validated client-side safe path the session will
  return to on exit.

Responses:

- `200 OK` — `data`: `{ persona, return_path }`. A persona already in an
  active Test Mode session is reused (the same persona user continues).
- `400 VALIDATION_ERROR` — missing/unknown `persona`, invalid `return_path`.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid admin session.
- `403 FORBIDDEN` — authenticated but not the Technical Administrator.
- `403 TEST_MODE_DISABLED` — Test Mode not enabled on the server.

### `POST /api/v1/admin/auth/test-mode/exit`

Authenticated Test Mode endpoint (admin). Exits the active Test Mode session,
restores the stored administrator identity (ADMIN_1) and redirects the client
to the stored `return_path`. Deliberately NOT guarded by the admin role: exit
must be reachable from the impersonated client context so the persona session
can always be ended.

Responses:

- `200 OK` — `data`: `{ return_path }`. The admin session is restored on
  exit.
- `403 TEST_MODE_DISABLED` — Test Mode not enabled on the server.
- `403 TEST_MODE_INACTIVE` — no active Test Mode session (including a missing
  persona cookie).

### `GET /api/v1/auth/test-mode`

Unauthenticated Test Mode endpoint. Returns whether a Test Mode session is
currently active for the persona cookie. Used by the client shell to render
the Test Mode banner. Always returns `200`.

Response `data`:

- `active` — boolean; true when a Test Mode session is active.
- `persona` — `"client"` / `"trainer"` when active, otherwise `null`.

### `POST /api/v1/auth/test-mode/programs/:programID/purchase`

Authenticated Test Mode purchase endpoint (persona). Completes the program
purchase server-side for the active persona at zero cost, creating a real
`completed` purchase record and an active entitlement. No payment provider is
involved; the amount never leaves the backend and the provider code path is
never reached. A free program is rejected (free grants have nothing to obtain).

Request path:

- `programID` (required) — the target Generic Program UUID. Its type must be
  `premium` (purchasable); free programs are not purchasable.

Responses:

- `200 OK` — `data`: the completed purchase snapshot (same shape as the normal
  capture response), `status` set to `"completed"`, `price_minor_units` `0`.
- `400 VALIDATION_ERROR` — malformed program id.
- `401 AUTHENTICATION_REQUIRED` — no active persona session.
- `403 FORBIDDEN` — the authenticated identity is not an active Test Mode
  persona (a normal user cannot complete Test Mode purchases).
- `403 TEST_MODE_DISABLED` — Test Mode not enabled on the server.
- `404 PROGRAM_NOT_FOUND` — program does not exist, is soft-deleted,
  unpublished, or not a purchasable premium Generic Program.
- `409 PROGRAM_NOT_PURCHASABLE` — the program is a free program (nothing to
  purchase).
- `409 DUPLICATE_ENTITLEMENT` — the persona already holds an active entitlement
  for the program (idempotent signal; the client resolves ownership).

---

### Webhook Endpoints

#### `POST /api/v1/webhooks/stripe`

Stripe webhook endpoint. Receives verified Stripe events and triggers purchase
completion when appropriate.

No authentication middleware — verification is performed via Stripe-Signature
cryptographic verification.

Request:

- Header: `Stripe-Signature` (required) — Stripe webhook signature for
  verification
- Body: raw Stripe event payload

Supported event types:

- `checkout.session.completed` — primary payment success event
- `checkout.session.async_payment_succeeded` — delayed payment methods settling
  later

A completion event is only honored when the session is a fully paid one-time
payment (`payment_status == "paid"`, `mode == "payment"`) whose
`client_reference_id` matches the `purchase_id` metadata, whose `amount_total`
and `currency` match the purchase snapshot, and whose `payment_method_types`
contain the payment method type of the purchase's recorded `payment_method`
(recorded method must be non-empty). Delayed methods emit
`checkout.session.completed` while still unpaid; those events are acknowledged
and ignored until `checkout.session.async_payment_succeeded` arrives.

MB WAY is served by the same Checkout Session as card, so it completes through
the same two events. The buyer approves the payment in the MB WAY app, and the
session is only honored once Stripe reports it as paid.

**Late settlements are reconciled, never discarded.** A verified, fully paid
event for a purchase that is no longer pending is routed by its state: a
`failed` purchase is reconciled through `ReconcilePurchase` (reconciled → 200),
while a reconciliation conflict, a purchase that moved on, or a completion
that finds the (user, program) pair already resolved (a duplicate sale racing
a sibling purchase) is acknowledged with 200 and a diagnostic outcome
(`conflict_pair_resolved` / `purchase_moved_on`) is recorded — a redelivery
cannot change the durable state, so it is never retried forever. Only an
integrity violation or transient failure returns 500 so Stripe retries. A
completed purchase stays idempotently acknowledged with `already_completed`. No
event type — including refunds and disputes — can revert a completed purchase:
RYZE purchases are final, there is no refund workflow, and a refund event is
acknowledged and ignored.

**Every verified delivery is recorded.** Each verified event writes one durable
record to `purchase_reconciliation_events` keyed by the provider event id,
stamped with a fixed outcome code (`completed`, `already_completed`,
`reconciled`, `conflict_pair_resolved`, `purchase_moved_on`,
`entitlement_integrity`, `transient_failure`, `amount_mismatch`,
`currency_mismatch`, `method_mismatch`, `reference_mismatch`,
`provider_uncertain`, `late_approval`, `permanent_invalid`,
`test_purchase_event`). The outcome is recorded best-effort — a persistence
failure never changes the HTTP verdict — and is immutable once resolved: only
the provisional outcomes (`transient_failure`, `provider_uncertain`) may be
overwritten by a later delivery of the same event. Permanent mismatches (amount,
currency, method, reference, malformed but verified event) and Test Mode
purchases are acknowledged with 200 and recorded, so the provider stops
redelivering and the diagnostic is preserved for the administrative
reconciliation view instead of being lost in a 500.

All other event types are safely acknowledged with 200 and ignored, including
refunds and disputes: RYZE purchases are final, there is no refund workflow, and
no event type can revert a completed purchase.

Responses:

- `200` — event processed (completed, reconciled, already completed, not a
  successful payment, unsupported mode, unsupported event type, unknown purchase,
  purchase not pending, late settlement acknowledged, permanent mismatch
  recorded, duplicate sale already resolved by a sibling purchase, Test Mode
  purchase guarded, purchase without a recorded payment method)
- `400` — invalid signature, missing header, malformed payload, unverifiable
  mode or payment status
- `500` — reconciliation failure (integrity or transient, retried), internal
  completion failure

**Checkout initiation does not complete a purchase. Browser redirects do not
complete a purchase. Completion is reachable only through two server-side
paths: the verified capture endpoint (`POST /me/purchases/:purchaseID/capture`)
and verified provider events (`CompletePurchase()`).**

#### `POST /api/v1/webhooks/paypal`

PayPal webhook endpoint. Receives verified PayPal events and triggers purchase
completion when appropriate.

No authentication middleware — verification is performed via PayPal's
VerifyWebhookSignature API.

Request:

- Body: PayPal webhook event payload (with PayPal verification headers)

Supported event types:

- `CHECKOUT.ORDER.APPROVED` — buyer approved the order. The handler reads the
  `reference_id` (RYZE purchase id) and the `resource.id` (PayPal order id)
  from the event payload, verifies the amount and currency against the purchase
  snapshot, captures the approved order server-side (`CompletePurchaseWithCapture`)
  and completes the purchase atomically. A verified approval for a purchase that
  is no longer pending is **not** captured: approval is consent, not a settled
  payment, so a late approval for a failed purchase is acknowledged with 200 and
  a diagnostic log, leaving the order to lapse at PayPal with no funds moved —
  when those funds actually did move, `PAYMENT.CAPTURE.COMPLETED` reconciles.
  An approval whose (user, program) pair already resolved another way is
  likewise **not** captured — the money could never grant access — and is
  acknowledged with 200 with a `conflict_pair_resolved` outcome.
- `PAYMENT.CAPTURE.COMPLETED` — an order was captured and the funds settled
  (this proves the money moved, unlike approval). The related `order_id` is read
  from `supplementary_data`, the RYZE purchase is recovered by loading that
  order from the PayPal API and reading its single purchase-unit reference
  (never trusted from the event), and the captured amount and currency are
  verified against the purchase snapshot. The purchase is then routed by state:
  a still-pending purchase is completed through the pure completion path (the
  funds are already captured, so nothing is re-captured), a failed purchase is
  reconciled through `ReconcilePurchase`, a completed purchase is an
  idempotent acknowledgement, and a completion that finds the (user, program)
  pair already resolved is acknowledged with a `conflict_pair_resolved`
  outcome. This closes the race where the browser return leg
  captured the order while a supersede was landing on the purchase.

All other event types are safely acknowledged with 200.

Every verified delivery is recorded with the same durable outcome vocabulary
described for the Stripe webhook, keyed by the PayPal `event.id`; permanent
mismatches and Test Mode bounds are acknowledged with 200 plus the recorded
outcome so PayPal stops redelivering while the administrative reconciliation
view keeps the diagnostic.

Responses:

- `200` — event processed (completed, reconciled, idempotent acknowledgement,
  late settlement acknowledged, permanent mismatch recorded, Test Mode guarded,
  unsupported or non-completed event type)
- `400` — invalid verification, malformed payload
- `500` — missing order id, order resolution failure, capture failure, internal
  completion or reconciliation failure. The event is NOT acknowledged so PayPal
  retries it.

---

## Client

Additional permissions:

- Purchase packs
- Complete questionnaires
- Upload measurements
- View progress
- Share profile
- Leave feedback

---

## Trainer

Additional permissions:

- Create packs
- Edit packs
- Manage own packs
- View associated clients
- Access trainer dashboard

The trainer area is protected by the trainer permission model
(`middleware/trainerroles`), built separately from trainer authentication
(`middleware/trainercontext` + `middleware.TrainerAuthenticate`). Four base
permissions exist today:

- `trainer.profile` — the trainer's own profile and dashboard area.
- `trainer.clients` — the trainer's client management area.
- `trainer.programs` — the trainer's programs area.
- `trainer.statistics` — the trainer's statistics area.

Every active trainer is granted every base permission. Route authorization is
enforced by `middleware.RequireTrainerPermission`, which runs strictly after
`TrainerAuthenticate`, reads the trainer identity exclusively from
`trainercontext` and never trusts client-supplied user IDs, trainer IDs,
permissions, roles, headers, query or body values. Missing or invalid trainer
identity returns HTTP 401 `AUTHENTICATION_REQUIRED`; unknown or unheld
required permissions return HTTP 403 `FORBIDDEN` with the generic envelope.
Protected trainer routes mount `RequireTrainerPermission` behind
`TrainerAuthenticate`.

### `GET /api/v1/trainer/profile`

Returns the authenticated trainer's own profile with the nested user details.
Identity comes exclusively from the authenticated trainer context; client
supplied `user_id` / `trainer_id` query or body values are ignored.
Response `data`:

- `id` — trainer UUID.
- `user_id` — owner user UUID.
- `created_at` / `updated_at` — ISO 8601 timestamps.
- `user` — nested safe user object: `id`, `email`, `first_name`,
  `last_name`, `created_at`, `updated_at`. Password hashes, session versions
  and soft-delete flags are never exposed.

Requires `trainer.profile`.

---

### `GET /api/v1/trainer/clients`

Returns one page of the authenticated trainer's active clients, newest first,
with pagination metadata. Identity comes exclusively from the authenticated
trainer context; a client-supplied `trainer_id` query value is ignored and can
never change which clients are listed.

Query parameters:

- `page` — page number, 1-based, default `1`.
- `limit` — page size, default `20`, clamped to `100`.

Response `data`:

- `clients` — array of client relationship objects:
  - `id` — relationship UUID.
  - `trainer_id` — owner trainer UUID (always the authenticated trainer).
  - `user_id` — client user UUID.
  - `created_at` / `updated_at` — relationship timestamps.
  - `user` — nested safe user object: `id`, `email`, `first_name`,
    `last_name`, `created_at`, `updated_at`. Password hashes, session versions
    and soft-delete flags are never exposed.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

Requires `trainer.clients`.

---

### `GET /api/v1/trainer/clients/:userID`

Returns the safe profile of one of the authenticated trainer's active clients.
Identity comes exclusively from the authenticated trainer context; the `userID`
path parameter only identifies the requested resource and never proves access.
Only an active `trainer_clients` relationship (same trainer and user,
`deleted_at IS NULL`) grants access, so the profile of a user that is not the
trainer's client, does not exist or is soft-deleted is never revealed.

Response `data` uses the same shape as a list item: the relationship identity,
`trainer_id` (always the authenticated trainer), `user_id` and a nested safe
user object (`id`, `email`, `first_name`, `last_name`, `created_at`,
`updated_at`). Password hashes, session versions and soft-delete flags are never
exposed. No trainer-specific fitness data (weight, goals, diet, progress, PRs,
programs, statistics or payments) is returned.

Responses:

- `200 OK` — the client profile.
- `400 VALIDATION_ERROR` — malformed identifiers.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication or trainer
  context.
- `403 FORBIDDEN` — non-trainer or missing `trainer.clients` permission.
- `404 CLIENT_RELATION_NOT_FOUND` — the user does not exist, is soft-deleted,
  is not an active client of this trainer or the relationship was soft-deleted;
  the response never reveals which case occurred.

Requires `trainer.clients`.

---

### `GET /api/v1/trainer/clients/lookup`

Resolves an active user by **exact** email so the trainer can confirm who will
be added before creating the relationship. Identity comes exclusively from the
authenticated trainer context. The lookup is exact-match only: it never searches
or lists users, and it returns the minimal identity needed to confirm the match.
The trainer's own email resolves to their own account only as a candidate; the
add step rejects it.

Query parameters:

- `email` — required; trimmed server-side; empty values return
  `400 VALIDATION_ERROR`.

Response `data`:

- `user_id` — the matched user UUID.
- `email`, `first_name`, `last_name` — safe identity fields.

Responses:

- `200 OK` — exact match found.
- `400 VALIDATION_ERROR` — missing or blank email.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication or trainer
  context.
- `403 FORBIDDEN` — non-trainer or missing `trainer.clients` permission.
- `404 CLIENT_NOT_FOUND` — no active user has that email (a soft-deleted user
  is reported identically).

Requires `trainer.clients`.

---

### `POST /api/v1/trainer/clients`

Creates the active relationship between the authenticated trainer and an
existing active user. The trainer identity never comes from the request body:
a client-supplied `trainer_id` value is ignored and the relationship is always
created for the authenticated trainer.

Request body:

- `user_id` — the client user UUID.

Responses:

- `201 Created` — relationship created; response `data` uses the same shape as
  a list item.
- `400 VALIDATION_ERROR` — missing/malformed identifiers, self-relationship
  (the trainer's own user account is resolved from the trainer profile, so the
  two different identifiers for the same person are still rejected), invalid
  body.
- `404 CLIENT_NOT_FOUND` — the client user does not exist or is soft-deleted
  (a soft-deleted user can never be added).
- `409 CLIENT_ALREADY_ADDED` — an active relationship already exists.

Requires `trainer.clients`.

---

### `DELETE /api/v1/trainer/clients/:userID`

Soft-deletes ONLY the relationship between the authenticated trainer and the
user identified by the path parameter. The user account and the trainer profile
are never deleted and remain active; their lifecycles are independent of the
relationship.

Responses:

- `200 OK` — relationship soft-deleted.
- `400 VALIDATION_ERROR` — malformed identifiers.
- `404 CLIENT_RELATION_NOT_FOUND` — no relationship exists for this pair.

Requires `trainer.clients`.

---

### `POST /api/v1/trainer/clients/:userID/reactivate`

Restores the exact same previously removed relationship row between the
authenticated trainer and the user identified by the path parameter. The client
must still exist and be active, and an already-active relationship can never be
reactivated.

Responses:

- `200 OK` — relationship restored; response `data` uses the same shape as a
  list item.
- `400 VALIDATION_ERROR` — malformed identifiers.
- `404 CLIENT_NOT_FOUND` — the client user does not exist or is soft-deleted.
- `404 CLIENT_RELATION_NOT_FOUND` — no previously removed relationship exists.
- `409 CLIENT_ALREADY_ADDED` — the relationship is already active.

Requires `trainer.clients`.

---

### Trainer Client Programs

Program assignment endpoints are grouped under
`/api/v1/trainer/clients/:userID/programs` and require `trainer.clients` (the
same permission as the client relationship itself; no new permission is
invented). A program assignment links one of the authenticated trainer's own
programs to one of the authenticated trainer's active clients. Identity comes
exclusively from the authenticated trainer context; a client-supplied
`trainer_id` (body or query) is always ignored and can never influence which
assignment is created, listed or removed. A client that does not exist, is not
an active client of the trainer or whose relationship was soft-deleted is
indistinguishable from a client without assignments; a program or assignment
that is missing, soft-deleted or owned by another trainer is indistinguishable
and returns `NOT_FOUND`.

---

#### `POST /api/v1/trainer/clients/:userID/programs`

Assigns one of the authenticated trainer's own active programs to one of the
authenticated trainer's active clients. The trainer identity never comes from
the request body: a client-supplied `trainer_id` value is ignored and the
assignment is always created for the authenticated trainer.

Request body:

- `program_id` — the program UUID; the program must be active and owned by the
  trainer.

Response `data`:

- `id` — assignment UUID.
- `trainer_id` — the authenticated trainer UUID.
- `program_id` / `user_id` — the assigned program and client user UUIDs.
- `program` — nested safe program object: `id`, `trainer_id`, `name`,
  `description`, `type`, `status`, `created_at`, `updated_at`. Deletion
  markers and internal data are never exposed.
- `created_at` / `updated_at` — assignment timestamps.

Responses:

- `201 Created` — assignment created.
- `400 VALIDATION_ERROR` — malformed/missing identifiers or invalid body.
- `404 CLIENT_RELATION_NOT_FOUND` — the client user does not exist, is
  soft-deleted or is not an active client of this trainer.
- `404 PROGRAM_NOT_FOUND` — the program does not exist, is soft-deleted or is
  not owned by the trainer.
- `409 ASSIGNMENT_ALREADY_ACTIVE` — the client already holds an active
  assignment.

Requires `trainer.clients`.

---

#### `GET /api/v1/trainer/clients/:userID/programs`

Returns every active program assignment of one of the authenticated trainer's
active clients, in assignment order, each with its safe program data. A
client-supplied `trainer_id` query value is ignored; a client that does not
exist, is not active or was removed is indistinguishable from a client without
assignments and yields an empty list.

Response `data`:

- `programs` — array of assignment objects with the same shape as the `POST`
  response.

Responses:

- `200 OK` — the assignments (empty array for a client without assignments).
- `400 VALIDATION_ERROR` — malformed identifiers.

Requires `trainer.clients`.

---

#### `DELETE /api/v1/trainer/clients/:userID/programs/:assignmentID`

Soft-deletes ONLY the program assignment between the authenticated trainer and
the client identified by the path parameters. The program and the
trainer-client relationship are never deleted; only the assignment row is
touched and never removed.

Responses:

- `200 OK` — assignment soft-deleted.
- `400 VALIDATION_ERROR` — malformed identifiers.
- `404 ASSIGNMENT_NOT_FOUND` — the assignment does not exist, is soft-deleted,
  belongs to another client or the owning relationship is not active.

Requires `trainer.clients`.

---

### Trainer Programs

Program management endpoints are grouped under
`/api/v1/trainer/programs` and require `trainer.programs`. Identity comes
exclusively from the authenticated trainer context; a client-supplied
`trainer_id` (body, query, header) is always ignored and can never influence
which program is created, listed, read, updated, deleted or restructured. A
program, week or workout that is missing, soft-deleted or owned by another
trainer is indistinguishable and returns `NOT_FOUND`.

#### `POST /api/v1/trainer/programs`

Creates a new program owned by the authenticated trainer. The `trainer_id` in
the body is deliberately ignored; the owner is always the authenticated
trainer.

Request body:

- `name` — required, trimmed, max 255 characters.
- `description` — optional, max 5000 characters.
- `type` — one of `free`, `premium`, `personalized`.
- `status` — one of `draft`, `published`; defaults to `draft` when omitted.
- `price_minor_units` — integer, price in minor currency units (cents for EUR).
  Free programs must have 0; paid programs must meet the configured minimum
  (env `MIN_PROGRAM_PRICE_MINOR_UNITS`, default 100 = €1.00). Defaults to 0.
- `currency` — ISO 4217 code, currently only `EUR`. Defaults to `EUR`.

Response:

- `201 Created` — response `data`: `id`, `trainer_id`, `name`,
  `description`, `type`, `status`, `price_minor_units`, `currency`,
  `created_at`, `updated_at`.
- `400 VALIDATION_ERROR` — invalid or incomplete input.
- `500 INTERNAL_ERROR`.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs`

Returns one page of the authenticated trainer's active programs, newest first,
with pagination metadata.

Query parameters:

- `page` — page number, 1-based, default `1`.
- `limit` — page size, default `20`, clamped to `100`.

Response `data`:

- `programs` — array of program objects (same shape as a create response).
- `pagination` — `page`, `limit`, `total`, `total_pages`.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID`

Returns one of the authenticated trainer's own active programs.

Responses:

- `200 OK` — program object.
- `400 VALIDATION_ERROR` — malformed program id.
- `404 PROGRAM_NOT_FOUND` — missing, soft-deleted or foreign program.

Requires `trainer.programs`.

#### `PATCH /api/v1/trainer/programs/:programID`

Updates the whitelisted fields of one of the authenticated trainer's own
programs. `trainer_id` is immutable and never accepted. Nil fields are left
unchanged; at least one field must be provided.

Request body (all optional):

- `name`, `description`, `type`, `status`, `price_minor_units`, `currency`.

Responses:

- `200 OK` — updated program object.
- `400 VALIDATION_ERROR` — invalid input or empty update.
- `404 PROGRAM_NOT_FOUND`.

Requires `trainer.programs`.

#### `DELETE /api/v1/trainer/programs/:programID`

Soft-deletes one of the authenticated trainer's own programs. Only the program
row is soft-deleted; it is never removed from the database. Weeks and workouts
remain stored but become unreachable.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 PROGRAM_NOT_FOUND`.

Requires `trainer.programs`.

---

### Trainer Program Structure

Program structure endpoints manage the ordered week and workout slots of a
trainer's own programs. They live under
`/api/v1/trainer/programs/:programID/weeks` and require `trainer.programs`.
Weeks are ordered by `week_number`, workouts by `position` (both 1-based).
A week or workout carries no name or content yet: in this foundation the
position is its identity. Creating always appends to the end; reordering
replaces the full order and requires the exact active set.

Workout exercises live under
`/api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises`.
They assign global catalog exercises to a workout, ordered by `position`
(1-based). The same exercise may appear more than once in the same workout
(each assignment is a distinct position). The API only accepts the catalog
`exercise_id`; the trainer ownership and the target workout always come from
the authenticated trainer context and the URL path respectively — a client can
never select another trainer's workout or embed an ownership id in the body.

#### `POST /api/v1/trainer/programs/:programID/weeks`

Appends a new empty week to the end of one of the authenticated trainer's own
programs.

Responses:

- `201 Created` — response `data`: `id`, `program_id`, `week_number`,
  `workouts` (empty array), `created_at`, `updated_at`.
- `400 VALIDATION_ERROR` — malformed identifiers.
- `404 PROGRAM_NOT_FOUND` — missing, soft-deleted or foreign program.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks`

Returns every active week of one of the authenticated trainer's own programs,
in week order, with each week's active workouts in position order.

Response `data`:

- `weeks` — array of week objects; each `workouts` array holds workout objects
  (`id`, `program_week_id`, `position`, `created_at`, `updated_at`).

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.

Requires `trainer.programs`.

#### `PATCH /api/v1/trainer/programs/:programID/weeks/order`

Replaces the order of every active week of one of the authenticated trainer's
own programs.

Request body:

- `ids` — array of every active week id in the new order, each exactly once.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR` — empty, malformed or duplicated id list.
- `404 PROGRAM_NOT_FOUND`.
- `409 REORDER_CONFLICT` — the list does not match the active weeks exactly.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks/:weekID`

Returns one active week of one of the authenticated trainer's own programs,
with its active workouts in position order.

Responses:

- `200 OK` — week object.
- `400 VALIDATION_ERROR`.
- `404 WEEK_NOT_FOUND` — missing, soft-deleted or foreign week.

Requires `trainer.programs`.

#### `DELETE /api/v1/trainer/programs/:programID/weeks/:weekID`

Soft-deletes one of the authenticated trainer's own program weeks. Only the
week row is soft-deleted; its workouts are preserved but become unreachable.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 WEEK_NOT_FOUND`.

Requires `trainer.programs`.

#### `POST /api/v1/trainer/programs/:programID/weeks/:weekID/workouts`

Appends a new empty workout to the end of one of the authenticated trainer's
own program weeks.

Responses:

- `201 Created` — response `data`: `id`, `program_week_id`, `position`,
  `created_at`, `updated_at`.
- `400 VALIDATION_ERROR`.
- `404 WEEK_NOT_FOUND` — missing, soft-deleted or foreign week.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks/:weekID/workouts`

Returns every active workout of one of the authenticated trainer's own program
weeks, in position order.

Response `data`:

- `workouts` — array of workout objects.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.

Requires `trainer.programs`.

#### `PATCH /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/order`

Replaces the order of every active workout of one of the authenticated
trainer's own program weeks.

Request body:

- `ids` — array of every active workout id in the new order, each exactly
  once.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 WEEK_NOT_FOUND`.
- `409 REORDER_CONFLICT` — the list does not match the active workouts exactly.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID`

Returns one active workout of one of the authenticated trainer's own program
weeks.

Responses:

- `200 OK` — workout object.
- `400 VALIDATION_ERROR`.
- `404 WORKOUT_NOT_FOUND` — missing, soft-deleted or foreign workout.

Requires `trainer.programs`.

#### `DELETE /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID`

Soft-deletes one of the authenticated trainer's own program workouts.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 WORKOUT_NOT_FOUND`.

Requires `trainer.programs`.

#### `POST /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises`

Appends one active global catalog exercise to the end of one of the
authenticated trainer's own program workouts. The exercise is selected only by
its catalog id; no other per-assignment data exists in this foundation.

Request body:

- `exercise_id` — the global catalog exercise id (required, must be a valid
  UUID).

Responses:

- `201 Created` — response `data`: `id`, `program_workout_id`, `position`,
  `exercise` (safe catalog data: `id`, `name`, `description`,
  `target_muscles`, `equipment`, `difficulty`, `video_url`, `image_url`),
  `created_at`, `updated_at`.
- `400 VALIDATION_ERROR` — malformed identifiers, missing or invalid body.
- `404 WORKOUT_NOT_FOUND` — missing, soft-deleted or foreign workout.
- `404 EXERCISE_NOT_FOUND` — missing or soft-deleted catalog exercise.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises`

Returns every active workout exercise of one of the authenticated trainer's own
program workouts, in position order, each with its safe exercise data.

Response `data`:

- `exercises` — array of workout exercise objects.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.

Requires `trainer.programs`.

#### `PATCH /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/order`

Replaces the order of every active workout exercise of one of the
authenticated trainer's own program workouts.

Request body:

- `ids` — array of every active workout exercise id in the new order, each
  exactly once.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 WORKOUT_NOT_FOUND`.
- `409 REORDER_CONFLICT` — the list does not match the active workout exercises
  exactly.

Requires `trainer.programs`.

#### `GET /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID`

Returns one active workout exercise of one of the authenticated trainer's own
program workouts, with its safe exercise data.

Responses:

- `200 OK` — workout exercise object.
- `400 VALIDATION_ERROR`.
- `404 WORKOUT_EXERCISE_NOT_FOUND` — missing, soft-deleted or foreign workout
  exercise (a workout exercise of another workout, another program or another
  trainer is indistinguishable from a non-existent one).

Requires `trainer.programs`.

#### `DELETE /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID`

Soft-deletes one of the authenticated trainer's own workout exercises. Only the
workout exercise row is soft-deleted; the workout, the week, the program and
the catalog exercise are never touched.

Responses:

- `200 OK`.
- `400 VALIDATION_ERROR`.
- `404 WORKOUT_EXERCISE_NOT_FOUND`.

Requires `trainer.programs`.

---

## Technical Administrator

The configured ADMIN_1 identity. Responsible for the platform's technical
side.

Permissions include:

- User management
- Trainer management
- Trainer application review (`trainer_applications.read` /
  `trainer_applications.manage`, shared with the Management Administrator)
- Statistics
- System management
- Infrastructure
- Technical configuration
- Security / authentication administration
- Development
- Test Mode administration (see the [Test Mode](#test-mode) section; entering
  Test Mode is exclusive to the Technical Administrator)

Technical administrators do not manage business content unless explicitly
permitted. The Technical Administrator is not a super administrator: plans,
finance and marketing are reserved for the Management Administrator.

---

## Management Administrator

The configured ADMIN_2 identity. Responsible for business management.

Permissions include:

- User management
- Trainer management
- Trainer application review (`trainer_applications.read` /
  `trainer_applications.manage`, shared with the Technical Administrator)
- Statistics
- Plans management and creation
- Generic program management (platform-owned programs, `plans` permission)
- Financial management
- Commerce management (admin sales view over purchases, `commerce` permission —
  see the [ADMIN COMMERCE](#admin-commerce) section)
- Marketing (promotions)

---

# REQUEST VALIDATION

Every request must be validated.

Validation occurs in the backend.

Never trust client-side validation.

Invalid requests must return descriptive validation errors.

---

# PAGINATION

Collection endpoints support pagination.

Example:

```
GET /packs?page=1&limit=20
```

Default page size:

20

Maximum page size:

100

The backend may adjust these limits in future versions.

---

# SORTING

Resources support sorting whenever applicable.

Examples:

```
?sort=name

?sort=created_at

?sort=price
```

Order:

```
?order=asc

?order=desc
```

---

# SEARCH

Endpoints may provide full-text search.

Example:

```
GET /packs?q=hypertrophy
```

Search behavior depends on the endpoint.

---

# FILTERING

Endpoints may expose resource-specific filters.

Examples:

```
GET /packs?difficulty=beginner

GET /packs?category=strength

GET /packs?trainer=sandro

GET /packs?equipment=home
```

Filters may be combined.

---

# FILE UPLOADS

Uploads use:

```
multipart/form-data
```

Supported upload types include:

- Profile pictures
- Trainer certificates
- Feedback images
- Pack resources
- Temporary PDFs

Binary files are stored on disk.

The database stores metadata only.

Uploads must be validated before storage.

Validation includes:

- MIME type
- File extension
- Maximum size
- Virus scanning (future)
- Image optimization

Temporary files should be removed automatically.

---

# WEBSOCKETS

Real-time communication uses Gorilla WebSocket.

WebSockets are reserved for events requiring immediate updates.

Examples:

- Chat messages
- Notifications
- Typing indicators
- System announcements
- Live dashboard updates

The API should continue functioning normally if WebSockets are unavailable.

---

# RATE LIMITING

Rate limiting protects the API against abuse.

Limits may differ depending on the endpoint.

Examples:

Authentication endpoints

More restrictive.

Read-only endpoints

Less restrictive.

Authenticated users may receive higher limits than anonymous visitors.

## Authentication rate limiting (v2.90.0)

Rate limiting is currently applied to the credential and account endpoints:
`POST /api/v1/auth/register`, `POST /api/v1/auth/login`,
`POST /api/v1/admin/auth/login`, `POST /api/v1/admin/auth/verify`,
`POST /api/v1/auth/change-password` and `POST /api/v1/auth/delete-account`.
Each endpoint is limited independently, by client IP and — where the body
carries an account identifier — by the normalised (trimmed, lowercased) `email`
(user endpoints) or `username` (admin login).

An exhausted limit returns:

```
HTTP/1.1 429 Too Many Requests
Retry-After: <seconds>
```

```json
{
  "success": false,
  "message": "Too many requests. Please try again later.",
  "error": {
    "code": "RATE_LIMITED",
    "details": null
  }
}
```

`Retry-After` is a whole number of seconds and is never below 1. Counters are
in memory and per instance, bounded by least-recently-used eviction; no body is
altered and no identifier is stored in clear. Webhooks and non-authentication
endpoints are never rate limited.

---

# CACHING

Caching should be implemented whenever beneficial.

Typical cache candidates include:

- Public packs
- Public trainer profiles
- Categories
- Static resources

Sensitive user information must never be cached publicly.

---

# ERROR HANDLING

Errors should always be consistent.

Error responses must:

- Clearly describe the problem.
- Avoid exposing internal implementation.
- Be easy to understand.
- Include machine-readable error codes whenever appropriate.

Unexpected server errors should always be logged.

---

# SECURITY

Every endpoint must follow secure development practices.

Including:

- JWT validation
- Permission verification
- SQL Injection prevention
- XSS prevention
- Input validation
- Output sanitization
- Rate limiting
- Secure headers

Security checks should never rely on the frontend.

---

# API EVOLUTION

The API is expected to evolve throughout the project's lifetime.

AI assistants are encouraged to improve:

- Endpoint organization
- Response structure
- Validation
- Security
- Performance
- Naming
- Documentation

...whenever those improvements are justified.

Breaking changes should be avoided whenever possible.

Whenever API behavior changes, both this document and `update_logs.MD` must be updated before implementation.

The API should remain predictable, RESTful and consistent across every version.

# AUTHENTICATION ENDPOINTS

Authentication is handled using JWT tokens.

Users may authenticate using:

- Email and password
- Google OAuth

The backend is responsible for validating credentials and generating access tokens.

---

## Login

```
POST /api/v1/auth/login
```

Authenticates a user using email and password.

Request body:

```json
{
  "email": "...",
  "password": "..."
}
```

Only these two fields are used. Empty email or password, or malformed JSON,
returns HTTP 400 `VALIDATION_ERROR`. An unknown email and a wrong password both
return the same HTTP 401 `INVALID_CREDENTIALS`; the unknown-email path still
performs a dummy password verification so the response time does not reveal
whether the account exists. The endpoint is rate limited per IP and per
normalised email; an exhausted limit returns HTTP 429 `RATE_LIMITED` with a
`Retry-After` header.

Public endpoint.

---

## Admin Authentication

Administrator access uses two sequential steps. The first authenticates the
credentials, the second completes the second factor (the admin access code).
Only the second step issues the admin session cookie.

```
POST /api/v1/admin/auth/login
```

Step 1 — authenticates a platform administrator using a configured username
and password.

Request body:

```json
{
  "username": "...",
  "password": "..."
}
```

Only these two fields are accepted. Administrators are configuration-only
accounts defined by `ADMIN_1_USERNAME`/`ADMIN_1_PASSWORD` and
`ADMIN_2_USERNAME`/`ADMIN_2_PASSWORD` in `.env`; they are never stored in the
database. Missing, too-short or duplicated admin credentials fail at server
startup. Credential changes take effect on the next restart.

An unknown username or a wrong password fails with the same HTTP 401
`INVALID_CREDENTIALS`; callers cannot distinguish which case occurred. Empty
username/password or malformed JSON returns HTTP 400 `VALIDATION_ERROR`. On
success the endpoint returns HTTP 200 with an empty `data` object and sets a
short-lived temporary HttpOnly cookie `ryze_admin_stage_token` (`SameSite=Lax`,
`Secure` per `COOKIE_SECURE`, lifetime 5 minutes). This cookie only proves that
step 1 succeeded; it is not an authenticated admin session and is rejected by
the admin token validator. The endpoint is rate limited per IP and per
normalised username; an exhausted limit returns HTTP 429 `RATE_LIMITED` with a
`Retry-After` header.

Public endpoint.

---

```
POST /api/v1/admin/auth/verify
```

Step 2 — completes admin authentication with the access code. Requires the
`ryze_admin_stage_token` cookie issued by step 1.

Request body:

```json
{
  "access_code": "..."
}
```

Only `access_code` is accepted. The access code is a configuration-only secret
(`ADMIN_1_ACCESS_CODE`/`ADMIN_2_ACCESS_CODE` in `.env`, minimum 8 characters),
never stored in the database. Missing, too-short or invalid access codes fail
at server startup.

A missing, invalid or expired stage cookie, an unknown administrator or a wrong
access code all fail with the same HTTP 401 `INVALID_CREDENTIALS`; callers
cannot distinguish which case occurred. Empty `access_code` or malformed JSON
returns HTTP 400 `VALIDATION_ERROR`. On success the endpoint returns HTTP 200
with an empty `data` object, clears the `ryze_admin_stage_token` cookie and
sets the admin session HttpOnly cookie `ryze_admin_access_token`
(`SameSite=Lax`, `Secure` per `COOKIE_SECURE`, same lifetime as the user access
token). The admin cookie name differs from the user `ryze_access_token` cookie
so the two session kinds cannot be confused. The admin JWT carries the
administrator identity (`ADMIN_1`/`ADMIN_2`) as its subject and is only
accepted by admin token validation; user, admin and stage tokens are mutually
rejected. No password, access code, username, JWT, JWT secret or internal
details are ever returned. The endpoint is rate limited per IP; an exhausted
limit returns HTTP 429 `RATE_LIMITED` with a `Retry-After` header.

Public endpoint.

Routes that require administrator access are protected by the admin
authentication middleware, which accepts only the `ryze_admin_access_token`
cookie validated as an `admin`-kind token carrying a configured admin identity
(`ADMIN_1`/`ADMIN_2`). Stage tokens, user tokens and any invalid token are
rejected with HTTP 401 `AUTHENTICATION_REQUIRED`.

```
GET /api/v1/admin/auth/me
```

Returns the identity of the authenticated administrator, used by the admin
console to resolve which sections to display. Requires the
`ryze_admin_access_token` cookie.

Response `data`:

```json
{
  "id": "ADMIN_1",
  "role": "TECHNICAL_ADMINISTRATOR"
}
```

`id` is the configured administrator identity (`ADMIN_1`/`ADMIN_2`); `role` is
the mapped role (`TECHNICAL_ADMINISTRATOR` for ADMIN_1,
`MANAGEMENT_ADMINISTRATOR` for ADMIN_2). The role is derived server-side from
the identity; the client must never interpret the id itself. A missing,
invalid or expired admin cookie returns HTTP 401 `AUTHENTICATION_REQUIRED`.
User tokens and stage tokens are rejected. No secrets are ever returned.

Authenticated endpoint (admin session required).

```
POST /api/v1/admin/auth/logout
```

Ends the current administrator session. Clears both admin cookies
(`ryze_admin_access_token` and `ryze_admin_stage_token`) and returns HTTP 200
with an empty `data` object. It intentionally does NOT require an active admin
session: an expired or already-ended session can still log out cleanly. A user
session (the `ryze_access_token` cookie) is never affected.

Public endpoint.

---

## Register

```
POST /api/v1/auth/register
```

Creates a new client account.

Public endpoint.

Accounts are inactive until email verification is completed.

The `password` must satisfy the shared password policy (12 to 128 characters);
a shorter or longer password returns HTTP 400 `VALIDATION_ERROR`.

If the email belongs to a soft-deleted (previously deleted) account, the same
registration restores that account instead of creating a duplicate: the
original UUID and creation date are kept, the names are updated and the
password is replaced by the new password hash. The endpoint still returns
HTTP 201 and the same response structure in both cases. Registering an email
that belongs to an active account returns HTTP 409
`EMAIL_ALREADY_REGISTERED`.

The endpoint is rate limited per IP and per normalised email; an exhausted limit
returns HTTP 429 `RATE_LIMITED` with a `Retry-After` header.

---

## Logout

```
POST /api/v1/auth/logout
```

Invalidates the current session.

Authentication required.

---

## Change Password

```
POST /api/v1/auth/change-password
```

Replaces the password of the authenticated user after verifying the current
password.

Request body:

```json
{
  "current_password": "...",
  "new_password": "..."
}
```

Only these two fields are accepted. The user identity is resolved exclusively
from the authenticated session; client-provided identifiers are never accepted.

The current password must be verified, otherwise the request fails with
HTTP 401 `INVALID_CREDENTIALS`. The new password must satisfy the shared
password policy (12 to 128 characters); an out-of-policy password returns
HTTP 400 `VALIDATION_ERROR`. The new password replaces the stored
`password_hash` and the account's `session_version` is incremented, revoking
every previously issued access token. On success the response clears the
`ryze_access_token` cookie (HTTP 200, empty `data`) and the user must
authenticate again. Empty passwords return HTTP 400 `VALIDATION_ERROR`. The
endpoint is rate limited per IP; an exhausted limit returns HTTP 429
`RATE_LIMITED` with a `Retry-After` header.

Authentication required.

---

## Delete Account

```
POST /api/v1/auth/delete-account
```

Soft-deletes the authenticated user's account after verifying the current
password.

Request body:

```json
{
  "password": "..."
}
```

Only this single field is accepted. The user identity is resolved exclusively
from the authenticated session; client-provided identifiers are never accepted.

The account is never physically removed: the row is preserved with its UUID,
email and `created_at` unchanged and `deleted_at` populated. The account's
`session_version` is incremented in the same operation, revoking every
previously issued access token, and the `ryze_access_token` cookie is cleared.
A soft-deleted account can no longer log in; re-registering with the same
email reactivates the original row (same UUID) instead of creating a duplicate.

A wrong password, an unknown or already-deleted user and an unverifiable stored
hash all fail with the same HTTP 401 `INVALID_CREDENTIALS`. An empty password
or malformed JSON returns HTTP 400 `VALIDATION_ERROR`. On success the response
returns HTTP 200 with an empty `data` object. No password, hash, JWT, JWT
secret or internal details (`deleted_at`, database columns) are ever returned.
The endpoint is rate limited per IP; an exhausted limit returns HTTP 429
`RATE_LIMITED` with a `Retry-After` header.

Authentication required.

---

## Refresh Token

```
POST /api/v1/auth/refresh
```

Generates a new JWT access token.

Authentication required.

---

## Forgot Password

```
POST /api/v1/auth/forgot-password
```

Sends a password recovery email.

Public endpoint.

---

## Reset Password

```
POST /api/v1/auth/reset-password
```

Updates the user's password using a recovery token.

Public endpoint.

---

## Verify Email

```
POST /api/v1/auth/verify-email
```

Verifies the user's email address.

Public endpoint.

---

## Resend Verification

```
POST /api/v1/auth/resend-verification
```

Resends the email verification link.

Public endpoint.

---

## Google Authentication

```
GET /api/v1/auth/google
```

Redirects the user to Google's authentication page.

---

## Google Callback

```
GET /api/v1/auth/google/callback
```

Processes Google's authentication response.

---

# USER ENDPOINTS

Endpoints related to user accounts.

Authentication required unless otherwise stated.

---

## Current User

```
GET /api/v1/users/me
```

Returns the authenticated user's information.

---

## Update Current User

```
PATCH /api/v1/users/me
```

Updates editable account information.

Examples:

- Display name
- Biography
- Location
- Language
- Preferences

---

## Delete Account

```
DELETE /api/v1/users/me
```

Performs a soft delete.

The account remains stored but becomes inactive.

---

## Change Password

```
PATCH /api/v1/users/me/password
```

Changes the current password.

Requires the current password.

---

## Upload Avatar

```
POST /api/v1/users/me/avatar
```

Uploads a profile picture.

Image optimization is performed automatically.

---

## Remove Avatar

```
DELETE /api/v1/users/me/avatar
```

Removes the current avatar.

---

## Upload Banner

```
POST /api/v1/users/me/banner
```

Uploads a profile banner.

---

## Remove Banner

```
DELETE /api/v1/users/me/banner
```

Removes the profile banner.

---

## Public Profile

```
GET /api/v1/users/{username}
```

Returns the public profile of any user.

Visibility depends on privacy settings.

---

## Public Packs

```
GET /api/v1/users/{username}/packs
```

Returns packs published by a trainer.

Public endpoint.

---

## Public Programs

### `GET /api/v1/programs`

Returns one page of published programs from all owners (trainers and the
platform), newest first, with pagination metadata. The catalog is global: no
authentication or ownership filter is applied. `scope=generic` restricts the
catalog to platform-owned generic programs (the marketplace foundation).

Public endpoint. No authentication required.

Query parameters:

- `page` — page number, 1-based, default `1`.
- `limit` — page size, default `20`, clamped to `100`.
- `scope` — `generic` restricts the catalog to platform-owned programs
  (`trainer_id` is null) whose `product_type` is `generic`.
  `premium_level_1` restricts the catalog to the Premium Level 1 product family,
  regardless of owner. Any other value is ignored.
- `q` — name query (LIKE, wildcards escaped), optional.
- `type` — program type filter (`free`, `premium`, `personalized`), optional.
- `training_type` — exact training type filter (`Hypertrophy`, `Strength`,
  `HYROX`, `CrossFit`, `Fat Loss`, `At Home`), optional.
- `level` — exact audience level filter (`Beginner`, `Intermediate`,
  `Advanced`), optional.
- `frequency` — exact days-per-week filter (1–7), optional.
- `duration_min`, `duration_max` — inclusive duration range in weeks, optional.
- `sort` — `created_at` or `name` (any other value falls back to `created_at`),
  optional.
- `order` — `asc` or `desc` (default `desc`), optional.

When none of the search/filter/scope/sort parameters are present the endpoint
returns the default catalog (newest first); otherwise it delegates to the
search path.

Response `data`:

- `programs` — array of program objects: `id`, `trainer_id` (null for generic
  programs), `name`, `description`, `type`, `status`, `level`,
  `duration_weeks`, `frequency_per_week`, `training_type`,
  `price_minor_units`, `currency`, `created_at`, `updated_at`.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

### `GET /api/v1/programs/:programID`

Returns one published program with its complete client-safe structure. Draft,
soft-deleted and unknown programs all return the same generic 404 to prevent
information leakage.

Public endpoint. No authentication required.

Response `data`:

- `id`, `trainer_id` (null for generic programs), `name`, `description`,
  `type`, `status`, `level`, `duration_weeks`, `frequency_per_week`,
  `training_type`, `price_minor_units`, `currency`, `created_at`,
  `updated_at`.
- `weeks` — array of `{ week_number, workouts }`; each workout is
  `{ position, exercises }`; each exercise carries the safe catalog summary
  (`name`, `description`, `instructions`, `target_muscles`, `equipment`,
  `difficulty`, `video_url`, `image_url`, `position`, `notes`) plus its
  `sets` (`set_number`, `set_type`, `reps`, `weight_kg`, `rir`, `rpe`,
  `rest_seconds`, `tempo`).

---

## Search Users

```
GET /api/v1/users/search
```

Searches public users.

Supports:

- Name
- Username
- Trainer
- Country

---

# USER SETTINGS

User preferences.

Authentication required.

---

## Get Settings

```
GET /api/v1/settings
```

Returns every configurable preference.

---

## Update Settings

```
PATCH /api/v1/settings
```

Updates preferences.

Examples:

- Language
- Notifications
- Privacy
- Units
- Public profile

---

## Notification Settings

```
PATCH /api/v1/settings/notifications
```

Updates notification preferences.

---

## Privacy Settings

```
PATCH /api/v1/settings/privacy
```

Updates profile visibility.

---

## Connected Accounts

```
GET /api/v1/settings/accounts
```

Returns connected authentication providers.

Example:

- Google

---

## Link Google Account

```
POST /api/v1/settings/accounts/google
```

Associates a Google account.

---

## Unlink Google Account

```
DELETE /api/v1/settings/accounts/google
```

Removes the Google association.

---

# NOTIFICATION ENDPOINTS

Notifications generated by the platform.

Authentication required.

---

## List Notifications

```
GET /api/v1/notifications
```

Returns recent notifications.

Supports pagination.

---

## Notification Count

```
GET /api/v1/notifications/count
```

Returns the number of unread notifications.

---

## Mark as Read

```
PATCH /api/v1/notifications/{id}/read
```

Marks a notification as read.

---

## Mark All as Read

```
PATCH /api/v1/notifications/read-all
```

Marks every notification as read.

---

## Delete Notification

```
DELETE /api/v1/notifications/{id}
```

Removes a notification from the interface.

---

# FILE UPLOAD ENDPOINTS

> **Status: not implemented.** The endpoints in this section are the planned
> upload contract; no upload route, handler, storage or media-processing code
> exists in the backend yet. They are documented so the future implementation
> has an agreed contract, and none of them can currently be reached.

Handles media uploaded by users.

Authentication required.

---

## Upload Image

```
POST /api/v1/uploads/image
```

Uploads an image.

Supported uses:

- Avatar
- Banner
- Feedback
- Trainer resources

---

## Upload Temporary File

```
POST /api/v1/uploads/temp
```

Uploads a temporary file.

Files are automatically removed after expiration.

---

## Delete Temporary File

```
DELETE /api/v1/uploads/temp/{id}
```

Deletes a temporary upload.

---

## Upload Trainer Certificate

```
POST /api/v1/uploads/certificate
```

Uploads professional certification during trainer application.

---

# LOCALIZATION

---

## Supported Languages

```
GET /api/v1/localization/languages
```

Returns every supported language.

Current languages:

- English
- Portuguese

---

## Change Language

```
PATCH /api/v1/localization/language
```

Updates the user's preferred language.

If no preference exists, the backend automatically selects the language based on the user's approximate location.

---

# HEALTH CHECK

Public endpoints used for monitoring.

---

## API Status

```
GET /api/v1/health
```

Returns the current API status.

- `200 OK` — `data`: `{ status: "ok" }`. The database is reachable.
- `503 SERVICE_UNAVAILABLE` — the database is unreachable. Used by load
  balancers and orchestrators so traffic is not routed to an instance that
  cannot serve requests.

---

## API Version

```
GET /api/v1/version
```

Returns the running API version.

- `200 OK` — `data`: `{ version }`. The value the binary was built with
  (`dev` for local builds, the released version in production).

---

# NOTES

All endpoints described in this section represent the base user management layer of the RYZE platform.

Additional modules such as Packs, Purchases, Payments, Workouts, Nutrition, Feedback, Chat, Trainer and Administration are documented in the following sections.

# PACK ENDPOINTS

Packs represent the commercial products available on the RYZE platform.

Every pack belongs to one of the supported categories:

- Generic
- Premium Level 1
- Premium Level 2

Only trainers and management administrators can create or modify packs.

---

## List Packs

```
GET /api/v1/packs
```

Returns every available pack.

Supports:

- Pagination
- Search
- Sorting
- Filtering

---

## Get Pack

```
GET /api/v1/packs/{slug}
```

Returns detailed information about a specific pack.

Public endpoint.

---

## Search Packs

```
GET /api/v1/packs/search
```

Searches available packs.

Supports:

- Name
- Difficulty
- Trainer
- Equipment
- Goal
- Category

---

## Featured Packs

```
GET /api/v1/packs/featured
```

Returns featured packs displayed on the homepage.

---

## Create Pack

```
POST /api/v1/packs
```

Creates a new pack.

Trainer or Management Administrator only.

---

## Update Pack

```
PATCH /api/v1/packs/{id}
```

Updates an existing pack.

Changes create a new version whenever necessary.

---

## Publish Pack

```
PATCH /api/v1/packs/{id}/publish
```

Publishes a draft pack.

---

## Unpublish Pack

```
PATCH /api/v1/packs/{id}/unpublish
```

Temporarily removes a pack from the marketplace.

---

## Delete Pack

```
DELETE /api/v1/packs/{id}
```

Soft deletes a pack.

Historical purchases remain untouched.

---

# PURCHASE ENDPOINTS

Handles every purchase performed on the platform.

---

## Checkout

```
POST /api/v1/purchases/checkout
```

Starts the checkout process.

---

## Purchase History

```
GET /api/v1/purchases
```

Returns every purchase performed by the authenticated user.

---

## Purchase Details

```
GET /api/v1/purchases/{id}
```

Returns detailed purchase information.

---

## Active Pack

```
GET /api/v1/purchases/current
```

Returns the currently active pack.

---

## Expired Packs

```
GET /api/v1/purchases/history
```

Returns previous packs already completed.

---

## Cancel Purchase

```
POST /api/v1/purchases/{id}/cancel
```

Cancels a purchase when allowed.

---

# PAYMENT ENDPOINTS

Responsible for payment processing.

---

## Payment Methods

```
GET /api/v1/payments/methods
```

Returns supported payment providers.

---

## Create Payment

```
POST /api/v1/payments
```

Creates a payment request.

---

## Payment Status

```
GET /api/v1/payments/{id}
```

Returns payment status.

---

## Retry Payment

```
POST /api/v1/payments/{id}/retry
```

Attempts payment again.

---

## Invoice

```
GET /api/v1/payments/{id}/invoice
```

Downloads the generated invoice.

---

# PREMIUM LEVEL 1 ENDPOINTS

These endpoints implement the Premium Level 1 journey: collect a nutrition
intake before checkout, then serve the server-owned nutrition programme once the
purchase is verified.

All five endpoints require authentication. The acting user is always taken from
the authentication context; no endpoint accepts a user id from the client.

## Ownership model

Every endpoint is scoped to `(authenticated user, program_id)`. A client may only
act on its own intake and its own nutrition assignment.

A program with no record for the calling user is indistinguishable from one the
caller does not own: both answer `404 NUTRITION_NOT_FOUND`. This is deliberate,
because a `403` would confirm that another client's record exists.

Intake answers are never returned by any endpoint. The requirement endpoint
reports only that an intake exists and its revision, never its contents.

---

## Questionnaire Question Catalog

```
GET /api/v1/me/questionnaire/questions
```

Returns the authoritative field contract for the intake: field name, type
(`text` | `textarea` | `number` | `select` | `multi_select` | `list`), label,
help text, whether the field is required or sensitive, allowed options, numeric
bounds and entry caps.

`list` fields hold an ordered array of free-choice strings entered one per
line; `multi_select` fields pick from the declared options (`multi_select` is a
future capability the server currently contracts but never emits). Numeric
fields such as `training_days_per_week` are declared as `number` with server
bounds, never as `select` option lists.

The catalog is public product metadata with no user data attached, so it is
resolved from the authentication context alone and does not require a program id.

The client renders the form from this response rather than from a hardcoded
field list, so adding a question does not require a client release. Validation
rules are also declared server-side and are enforced again on submission; the
catalog is a convenience, never the authority.

Success: `200 OK` with the catalog under the response `data` wrapper.

---

## Questionnaire Requirement

```
GET /api/v1/me/programs/{programID}/questionnaire
```

Returns the state of the calling user's intake for this program.

Fields:

- `program_id` — the program the state belongs to.
- `submitted` — whether an intake exists.
- `locked` — whether the intake is immutable, that is, whether an active
  entitlement exists for this program.
- `version` — the current intake revision; `0` when nothing is stored.
- `submitted_at` — first submission time, nullable.

`locked` is the server's authoritative answer to "may this intake still be
edited". The client must never infer it from `submitted`, from purchase state, or
from anything the user did. If the server cannot verify the lock it still answers
`200` with `locked: true`, so the client can render the package it already owns
instead of failing the page.

Success: `200 OK` with the fields above under the response `data` wrapper
(`{ success, message, data: { program_id, submitted, locked, version,
submitted_at } }`).

Failure:

- `401 UNAUTHORIZED` — no valid session.
- `404 PROGRAM_NOT_FOUND` — the program does not exist or is not available.

---

## Submit Questionnaire

```
POST /api/v1/me/programs/{programID}/questionnaire
```

Request body:

```json
{ "answers": { "goal": "fat_loss", "height_cm": 180, "dietary_restrictions": ["lactose"] } }
```

Creates or replaces the calling user's intake for this program.

Before a purchase, resubmission is allowed and increments `version`; the intake is
an editable draft and the nutrition programme has not been built from it yet.

Once an active entitlement exists, the intake is immutable and every mutation is
refused with `409 QUESTIONNAIRE_LOCKED`. This holds for regular purchases and for
Test Mode purchases, because both create the same entitlement. There is no flag
that reopens editing: the intake is the basis of a delivered package, and
silently changing it would change the plan the client already received. Because
the intake cannot move for the life of an entitlement, a delivered plan is always
derived from the current intake; `out_of_date` on the assignment is a holding
safety invariant for that guarantee, not a routine client path.

A refused mutation changes nothing: the stored revision is not incremented and
no field is written.

Answers are validated and normalized server-side. Unknown fields, unsupported
option values, out-of-range numbers, over-long text and exceeded entry caps are
all rejected.

Success: `200 OK` with the requirement fields (`submitted`, `locked`, `version`,
`submitted_at`) under the response `data` wrapper.

Failure:

- `400 VALIDATION_ERROR` — rejected answers. The response carries per-field
  reasons as short stable codes (`required`, `unsupported value`, `out of range`,
  `too many entries`, `entry too long`, `too long`). The client owns the
  presentation wording for these codes.
- `401 UNAUTHORIZED` — no valid session.
- `404 PROGRAM_NOT_FOUND` — the program does not exist, is not a Premium Level 1
  program, or is not available.
- `409 QUESTIONNAIRE_LOCKED` — an active entitlement exists, so the intake is
  immutable.

---

## Nutrition Assignment Status

```
GET /api/v1/me/programs/{programID}/nutrition
```

Returns the calling user's nutrition assignment state and, only when it is
`completed`, the generated plan.

Both endpoints that return assignment state (`GET` here and `POST` on
`/nutrition/generate`) answer with the same envelope:

```json
{
  "success": true,
  "message": "Nutrition status retrieved successfully.",
  "data": {
    "program_id": "8e3a…",
    "status": "completed",
    "version": 2,
    "questionnaire_version": 1,
    "out_of_date": false,
    "plan": { "…": "the nutrition plan document" }
  }
}
```

Fields in `data`:

- `program_id` — the program the assignment belongs to.
- `status` — `pending` | `processing` | `completed` | `failed`. `completed` is
  the only status that may be rendered as a plan; `failed` is a retryable state,
  and `pending`/`processing` mean the run has not finished.
- `version` — the assignment revision. It equals the version of the active plan
  and is incremented on every completed generation.
- `questionnaire_version` — the intake revision the plan was derived from.
- `out_of_date` — whether a newer intake revision exists than the one this plan
  was built from. Because the intake is immutable once an entitlement exists, it
  stays `false` in practice and exists to keep the guarantee honest rather than
  to power a routine user flow.
- `plan` — the generated programme, or `null` for every non-completed status.

`plan` is `null` unless `status` is `completed`. A completed checkout attempts
plan delivery as part of the purchase itself, but delivery can fail and the
ambit is swallowed by the purchase so a payment is never undone by a plan
problem; no endpoint ever reports a plan that has not been generated, and only
`completed` proves one exists.

This endpoint requires all three of: the program being `premium_level_1`, an
active entitlement, and a submitted intake. Any missing condition answers
`404 NUTRITION_NOT_FOUND`, which is the same response an unowned assignment
returns. Reading someone's nutrition programme must not be distinguishable from
reading nothing.

Success: `200 OK`.

Failure:

- `401 UNAUTHORIZED` — no valid session.
- `404 NUTRITION_NOT_FOUND` — not entitled, not Premium Level 1, no submitted
  intake, or no assignment for this user.

---

## Generate Nutrition Assignment

```
POST /api/v1/me/programs/{programID}/nutrition/generate
```

Runs the deterministic, server-side generation step over the stored intake.

The same ownership, entitlement and intake preconditions as the status endpoint
apply, and each missing condition answers `404 NUTRITION_NOT_FOUND`.

Generation is idempotent. An assignment that already has the plan its intake
revision requires is returned as-is, so retrying after a network failure
converges on the same single plan and never produces a duplicate. A `failed`
assignment is retried in place. If the intake revision ever moved on after a plan
was delivered, an explicit run inserts the next plan version and marks the
previous one superseded rather than rewriting it, so a delivered plan is never
silently changed and the history of what a client was shown is retained. The
`version` returned then matches the new active plan.

Two generation attempts can never fight: a run claims the assignment at the
revision it read, and a claim that races one already in flight or that targets a
revision the assignment has moved past is refused with `409 NUTRITION_NOT_READY`
instead of generating a second plan.

Success: `200 OK` with the assignment envelope.

Failure:

- `401 UNAUTHORIZED` — no valid session.
- `404 NUTRITION_NOT_FOUND` — not entitled, not Premium Level 1, or no submitted
  intake.
- `409 NUTRITION_NOT_READY` — the assignment changed during generation, is
  already processing, or is not in a state this run can move forward. Retrying
  later is the correct client response.
- `422 NUTRITION_RESTRICTIONS_UNSATISFIABLE` — the stored dietary restrictions
  cannot be satisfied by the plan catalog. Nothing was altered: the intake, the
  assignment and any previously delivered plan are untouched, and the message
  never echoes a health answer.

---

## Nutrition Plan Document

The `plan` field of a `completed` assignment is the server-owned nutrition
programme. It is generated deterministically from the normalized intake, so the
same intake always produces the same plan, and it is written as a normalised,
versioned record rather than an opaque blob.

The document shape:

```json
{
  "version": 2,
  "engine_version": 2,
  "fingerprint": "9f8b1c…",
  "dietary_pattern": "omnivore",
  "maintenance_calories": 2400,
  "target_calories": 2400,
  "daily": { "calories": 2400, "protein_grams": 150, "carbs_grams": 260, "fat_grams": 75, "fiber_grams": 30 },
  "protein_percent": 25,
  "carbohydrate_percent": 45,
  "fat_percent": 30,
  "meals_per_day": 4,
  "snacks_per_day": 1,
  "meals": [
    {
      "position": 1,
      "label": "Breakfast",
      "kind": "meal",
      "percent_of_daily": 25,
      "macros": { "calories": 600, "protein_grams": 40, "carbs_grams": 65, "fat_grams": 20, "fiber_grams": 8 },
      "notes": "Cook from scratch where you can.",
      "items": [
        {
          "position": 1,
          "food_name": "Rolled oats",
          "category": "grains",
          "quantity": 80,
          "unit": "g",
          "macros": { "calories": 300, "protein_grams": 11, "carbs_grams": 54, "fat_grams": 6, "fiber_grams": 8 },
          "substitution_note": "Swap for 1 slice of wholemeal bread"
        }
      ]
    }
  ],
  "exclusions": [
    { "reason_code": "allergy", "token": "peanut" },
    { "reason_code": "diet", "token": "vegetarian" }
  ],
  "hydration": { "daily_litres": 2.5, "note": "Drink steadily through the day." },
  "cautions": ["This plan is general guidance, not medical advice."],
  "prep_guidance": "Batch the grains once and portion them cold.",
  "summary": "…"
}
```

Key points:

- `daily` is the energy and macro contract for the whole day, with
  `target_calories` as the day's energy and the three `*_percent` fields as the
  macro split.
- `meals` carries every eating occasion in order. `kind` is `meal` or `snack`;
  the main meals carry the majority of the day. Each meal's `macros` is the sum
  of its `items`.
- Each `item` is an immutable snapshot of a food resolved from the server-owned
  catalog: its name, category, quantity, unit and macros are exactly what was
  generated, and their totals can never drift from a catalog row that later
  changes.
- `substitution_note` appears only where the engine had to swap the food; the
  note names the equivalent the client may use instead.
- `exclusions` is the audit of what the plan removed and why. `reason_code` is a
  controlled token (`allergy`, `intolerance`, `diet`, `excluded_food`,
  `disliked_food`, `insufficient_alternatives`) and `token` is a normalised
  vocabulary entry, never raw questionnaire text, so no health answer is echoed
  back through the plan.
- `cautions` are the non-prescriptive safety notes shown with the plan. They are
  a small controlled list, never questionnaire content.
- `engine_version` and `fingerprint` make a delivered plan auditable: the engine
  that produced it and the deterministic digest of the intake and plan body. An
  unchanged intake is provably still current by reproducing the fingerprint.

---

## Purchase Prerequisite

Premium Level 1 participates in the existing purchase flow. Checkout against a
`premium_level_1` program is refused until the calling user has a submitted
intake for that exact program.

Failure:

- `409 PURCHASE_PREREQUISITE_NOT_MET` — a Premium Level 1 program was purchased
  without a submitted intake.

This applies identically to the regular purchase path and to the Test Mode
purchase path; both verify the same prerequisite through the same code path.
Programs of any other `product_type` are unaffected and continue to purchase
without an intake.

---

## Product Type in Program Responses

`product_type` is the product family of a program (`generic` or
`premium_level_1`). It is public product metadata, not client data, and it is
included in the public catalog responses, in the public program detail response
and in the authenticated entitlement and program access responses.

It lets the client render a Premium Level 1 package as training plus nutrition
without a second request, and it lets the public catalog be scoped to a single
family.

The public catalog accepts a `scope` query parameter:

- `scope=generic` — platform-owned (`trainer_id IS NULL`) programs with
  `product_type = 'generic'`.
- `scope=premium_level_1` — programs with `product_type = 'premium_level_1'`,
  regardless of owner. A trainer cannot publish into this family.

The two scopes are independent predicates. A platform-owned Premium Level 1
program never appears in the generic marketplace, where the questionnaire
prerequisite would not be enforced.


# WORKOUT ENDPOINTS

Training management.

---

## Current Workout

```
GET /api/v1/workouts/current
```

Returns today's workout.

---

## Workout History

Implemented. `GET /api/v1/me/workouts/history` returns the authenticated user's
completed workouts, newest first, with pagination. See "Client Workout History"
under "Authenticated User" for the full contract.

---

## Workout Details

```
GET /api/v1/workouts/{id}
```

Returns a completed workout.

---

## Complete Workout

Implemented. `POST /api/v1/me/workouts/{workoutID}/complete` marks a workout of
the authenticated user's currently assigned program as completed. See "Client
Workout History" under "Authenticated User" for the full contract.

---

## Download Workout PDF

```
GET /api/v1/workouts/pdf
```

Downloads the workout program.

---
# EXERCISE ENDPOINTS

Implemented platform exercise library. An exercise is a global,
platform-owned catalog entry that exists only once and is reused across
programs and packs. The catalog is persisted in the database and populated by
an idempotent dataset import plus an administrative write surface (see
**Admin Exercise Catalogue** below). The three client endpoints below are
**public and read-only**: they require no authentication, are identical for
every caller and only ever return published, active entries. Soft-deleted or
non-published exercises are never returned.

Every exercise object now carries the structured training description:

- `instructions` — step-by-step movement text;
- `instruction_steps` — ordered array of individual execution steps (may be
  empty);
- `body_part` — the dataset body-part classification (`chest`, `back`,
  `shoulders`, `upper arms`, `lower arms`, `upper legs`, `lower legs`, `waist`,
  `neck`, `cardio`) or an empty string when unclassified;
- `primary_muscle_group` — the target muscle group (controlled vocabulary);
- `secondary_muscle_groups` — comma-separated secondary groups (optional);
- `movement_category` — movement classification (`Compound`, `Isolation`,
  `Core`, `Plyometric`, `Cardio`, `Mobility`, `Stretching`);
- `difficulty` — `Beginner`, `Intermediate` or `Advanced`;
- media references: `image_url`, `animation_url` (animated demonstration) and
  `video_url`, plus `media_attribution` for the source credit;
- `media_available` — `true` only when media exposure is enabled on the server
  and the entry has a stored media reference. When it is `false`, the media
  fields are empty and the client must render its unavailable state;
- `three_d_key` — the 3D demonstration scene implemented by the client viewer
  (`squat`, `push_up`, `biceps_curl`, `lunge`), or an empty string when the
  entry has no 3D demonstration;
- plus the original fields: `name`, `description`, `target_muscles`,
  `equipment`, `created_at`, `updated_at`.

## Exercise Library

```
GET /api/v1/exercises?page=1&limit=20
```

Returns one page of the exercise catalog in alphabetical order by name. Public
endpoint, no authentication required.

Optional filters, combinable together and with `q` on search:

- `muscle` — matches the primary or secondary muscle group (substring on the
  controlled-group values);
- `equipment` — substring match on the equipment value;
- `difficulty` — `Beginner`, `Intermediate`, `Advanced`; any other value
  returns `400 VALIDATION_ERROR`;
- `category` — one of the controlled movement categories; any other value
  returns `400 VALIDATION_ERROR`;
- `body_part` — one of the controlled body-part values listed above; any other
  value returns `400 VALIDATION_ERROR`.

Unknown parameter values never error except for `difficulty`, `category` and
`body_part`, which are validated against the controlled vocabulary so typos
cannot silently empty the catalog.

Pagination follows the platform rules: page ≥ 1, limit clamped to a maximum of
100, `data.pagination` reports `page`, `limit`, `total` and `total_pages`.
`data.exercises` is an array of exercise objects.

```
GET /api/v1/exercises?muscle=Quads&difficulty=Intermediate&limit=2
```

Response:

```json
{
  "success": true,
  "message": "Exercises retrieved successfully.",
  "data": {
    "exercises": [
      {
        "id": "…",
        "name": "Barbell Squat",
        "body_part": "upper legs",
        "description": "…",
        "instructions": "…",
        "instruction_steps": ["Stand with the bar on your upper back.", "Squat down."],
        "target_muscles": "…",
        "primary_muscle_group": "upper legs",
        "secondary_muscle_groups": "Glutes, Core, Hamstrings",
        "equipment": "Barbell",
        "difficulty": "Intermediate",
        "movement_category": "Compound",
        "video_url": "",
        "image_url": "…",
        "animation_url": "…",
        "three_d_key": "squat",
        "media_attribution": "© Gym visual — https://gymvisual.com/",
        "media_available": false,
        "created_at": "…",
        "updated_at": "…"
      }
    ],
    "pagination": { "page": 1, "limit": 2, "total": 48, "total_pages": 24 }
  }
}
```

## Exercise Details

```
GET /api/v1/exercises/{id}
```

Returns complete exercise information for one active catalog entry, including
its linked alternatives.

Includes:

- Description
- Instructions and ordered `instruction_steps`
- Body part, target muscles and primary / secondary muscle groups
- Movement category, difficulty and equipment
- Images (`image_url`), animated demonstrations (`animation_url`) and videos
  (`video_url`), with `media_available` and `media_attribution`
- `three_d_key` when the entry has a client-rendered 3D demonstration
- Alternatives — `alternatives` is an array of `{ id, exercise_id,
  alternative_exercise_id, alternative_name }`; `alternative_name` is the
  resolved name of the alternative catalog entry. Alternatives belong only to
  active exercises and reference active entries.

A malformed `{id}` returns `400 VALIDATION_ERROR`. An unknown or soft-deleted
exercise returns `404 EXERCISE_NOT_FOUND`.

```
GET /api/v1/exercises/51f8b06a-…-a6f1f4b6e257
```

Response:

```json
{
  "success": true,
  "message": "Exercise retrieved successfully.",
  "data": {
    "exercise": {
      "id": "51f8b06a-…-a6f1f4b6e257",
      "name": "Barbell Squat",
      "body_part": "upper legs",
      "description": "…",
      "instructions": "…",
      "instruction_steps": ["Stand with the bar on your upper back.", "Squat down."],
      "target_muscles": "…",
      "primary_muscle_group": "upper legs",
      "secondary_muscle_groups": "Glutes, Core, Hamstrings",
      "equipment": "Barbell",
      "difficulty": "Intermediate",
      "movement_category": "Compound",
      "video_url": "",
      "image_url": "…",
      "animation_url": "…",
      "three_d_key": "squat",
      "media_attribution": "© Gym visual — https://gymvisual.com/",
      "media_available": false,
      "created_at": "…",
      "updated_at": "…"
    },
    "alternatives": [
      {
        "id": "…",
        "exercise_id": "51f8b06a-…-a6f1f4b6e257",
        "alternative_exercise_id": "…",
        "alternative_name": "Leg Press Machine"
      }
    ]
  }
}
```

## Exercise Search

```
GET /api/v1/exercises/search?q={query}&page=1&limit=20
```

Searches exercises by name, case-insensitively. The `q` parameter is required;
an empty or blank query returns `400 VALIDATION_ERROR`. LIKE wildcards in the
query are treated literally. The same muscle / equipment / difficulty /
category / body-part filters as the library endpoint can be combined with the
query. The response shape and pagination are identical to
`GET /api/v1/exercises`.

```
GET /api/v1/exercises/search?q=bench&category=Compound
```

## Admin Exercise Catalogue

The administrative catalogue surface is protected by the `exercises`
permission group, which belongs to the **Management Administrator** (ADMIN_2)
only. Technical Administrators receive `403 FORBIDDEN`. These endpoints return
draft, published and archived entries and additionally expose provenance
(`source`, `source_id`) plus the raw stored media references
(`image_url`, `animation_url`, `source_media_id`, `media_attribution`).

Statuses form an explicit publication lifecycle:

- `draft` — created but not public;
- `published` — visible through the public exercise endpoints;
- `archived` — retained but no longer public.

Names are immutable after creation because program references and public links
may already point at the entry. New entries are always created as drafts;
publication is a separate status update.

### `GET /api/v1/admin/exercises`

Returns one page of catalogue entries (default 20, limit clamped to 100),
optionally filtered by `q`, `status`, `body_part` and `source`. Unlike the
public endpoint, this list includes non-published entries. Requires
`exercises.read`.

Response `data`: `exercises` (administrative exercise objects) and
`pagination` (`page`, `limit`, `total`, `total_pages`).

### `GET /api/v1/admin/exercises/{exerciseID}`

Returns one entry regardless of publication status. Malformed ids return
`400 VALIDATION_ERROR`; unknown or soft-deleted entries return
`404 EXERCISE_NOT_FOUND`. Requires `exercises.read`.

### `POST /api/v1/admin/exercises`

Creates a catalogue entry as a **draft**. Requires `exercises.manage`.

Request body:

- `name` — required, unique among active entries (`VARCHAR(255)`);
- `description`, `instructions`, `instruction_steps`;
- `body_part`, `target_muscles`, `primary_muscle_group`,
  `secondary_muscle_groups`, `equipment`, `difficulty`, `movement_category`;
- `image_url`, `animation_url`, `source_media_id`, `media_attribution`.

`body_part`, `difficulty` and `movement_category` are validated against their
controlled vocabularies; media references must be relative paths (no scheme,
no leading slash, no `..`). Returns `201` with `data.exercise`.

Errors: `400 VALIDATION_ERROR`, `409 EXERCISE_NAME_TAKEN`.

### `PATCH /api/v1/admin/exercises/{exerciseID}`

Updates one entry. Requires `exercises.manage`. Every field is optional:
absent fields stay untouched, while a present empty value clears the field.
`name` is immutable (it is not part of the update payload);
`source`/`source_id` are provenance and are not editable. `status` accepts
`draft`, `published` or `archived`. Returns `200` with `data.exercise`.

Errors: `400 VALIDATION_ERROR`, `404 EXERCISE_NOT_FOUND`,
`409 EXERCISE_NAME_TAKEN`.

---

# MEASUREMENT ENDPOINTS

Body measurements.

---

## Current Measurements

```
GET /api/v1/measurements
```

Returns current measurements.

---

## Add Measurement

```
POST /api/v1/measurements
```

Stores a new measurement.

---

## Update Measurement

```
PATCH /api/v1/measurements/{id}
```

Updates a measurement.

---

## Measurement History

```
GET /api/v1/measurements/history
```

Returns historical evolution.

---

# ACHIEVEMENT ENDPOINTS

Gamification.

---

## Achievements

```
GET /api/v1/achievements
```

Returns unlocked achievements.

---

## Badges

```
GET /api/v1/badges
```

Returns earned badges.

---

## Missions

```
GET /api/v1/missions
```

Returns active missions.

---

## Events

```
GET /api/v1/events
```

Returns active events.

---

## Streak

```
GET /api/v1/streak
```

Returns the user's current workout streak.

---

# FEEDBACK ENDPOINTS

Community feedback.

---

## Feed

```
GET /api/v1/feedback
```

Returns the public feed.

---

## Create Post

```
POST /api/v1/feedback
```

Creates a new post.

---

## Update Post

```
PATCH /api/v1/feedback/{id}
```

Updates a post.

---

## Delete Post

```
DELETE /api/v1/feedback/{id}
```

Soft deletes a post.

---

## Like Post

```
POST /api/v1/feedback/{id}/like
```

Likes a post.

---

## Remove Like

```
DELETE /api/v1/feedback/{id}/like
```

Removes a like.

---

## Comments

```
GET /api/v1/feedback/{id}/comments
```

Returns every comment.

---

## Create Comment

```
POST /api/v1/feedback/{id}/comments
```

Creates a comment.

---

## Delete Comment

```
DELETE /api/v1/comments/{id}
```

Removes a comment.

---

# CHAT ENDPOINTS

Messaging between clients and trainers.

---

## Conversations

```
GET /api/v1/chat
```

Returns every conversation.

---

## Conversation

```
GET /api/v1/chat/{id}
```

Returns a conversation.

---

## Messages

```
GET /api/v1/chat/{id}/messages
```

Returns conversation messages.

---

## Send Message

```
POST /api/v1/chat/{id}/messages
```

Sends a new message.

---

## Delete Message

```
DELETE /api/v1/chat/messages/{id}
```

Soft deletes a message.

---

## Upload Chat Image

```
POST /api/v1/chat/upload
```

Uploads an image inside a conversation.

---

This section defines every endpoint related to the platform's core functionality, including packs, purchases, workouts, nutrition, feedback, messaging and user progression.

Administrative endpoints, trainer management and system operations are documented in the final section.

# TRAINER ENDPOINTS

Endpoints available only to approved trainers.

---

## Trainer Dashboard

```
GET /api/v1/trainer/dashboard
```

Returns the trainer's dashboard.

Includes:

- Revenue
- Clients
- Published packs
- Statistics
- Ratings

---

## My Packs

```
GET /api/v1/trainer/packs
```

Returns every pack created by the trainer.

---

## Create Pack

```
POST /api/v1/trainer/packs
```

Creates a new training pack.

Initially created as a draft.

---

## Update Pack

```
PATCH /api/v1/trainer/packs/{id}
```

Updates one of the trainer's packs.

---

## Delete Pack

```
DELETE /api/v1/trainer/packs/{id}
```

Soft deletes one of the trainer's packs.

---

## Published Clients

```
GET /api/v1/trainer/clients
```

Returns every client associated with the trainer.

---

## Client Details

```
GET /api/v1/trainer/clients/{id}
```

Returns detailed client information available to the trainer.

---

## Trainer Statistics

```
GET /api/v1/trainer/statistics
```

Returns statistics including:

- Revenue
- Active clients
- Sales
- Pack popularity
- Reviews

---

## Trainer Revenue

```
GET /api/v1/trainer/revenue
```

Returns financial statistics.

---

# TRAINER APPLICATION

Any authenticated user may apply to become a trainer. The trainer is an
extension of the existing user identity: approving an application creates a
trainer profile linked to the **same** user account (same UUID, email and
authentication) — it never creates a second account.

The user endpoint is:

```
POST /api/v1/trainer/apply
```

Authentication required (user cookie `ryze_access_token`).

Submits a trainer application for the authenticated user. The request body is
intentionally not trusted: the applicant is resolved exclusively from the
authenticated session, so no `user_id` and no `status` are ever accepted from
the client. The application is created with status `PENDING`.

Success returns HTTP 201 with the safe application representation:

```json
{
  "success": true,
  "message": "Trainer application submitted successfully.",
  "data": {
    "id": "...",
    "status": "PENDING",
    "user_id": "...",
    "user": { "id": "...", "email": "...", "first_name": "...", "last_name": "..." },
    "created_at": "...",
    "updated_at": "..."
  }
}
```

Errors:

- HTTP 401 `AUTHENTICATION_REQUIRED` — no valid user session (including a
  soft-deleted user).
- HTTP 409 `ALREADY_TRAINER` — the user already owns a trainer profile.
- HTTP 409 `APPLICATION_ALREADY_EXISTS` — the user already has an active
  (PENDING or APPROVED) application.

A rejected application stays in history; the user can apply again. An approved
application is final.

## Access Status

```
GET /api/v1/trainer/access
```

Authentication required (user cookie `ryze_access_token`). Returns the
authenticated user's trainer access state, resolved exclusively from the
session: it is the single source of truth the frontend uses to decide whether
to show the apply screen, wait for review or open the trainer area. Nothing is
inferred client-side.

Response `data`:

- `is_trainer` — `true` when an active trainer profile exists for the user.
- `application` — `null` when the user has never applied, otherwise the latest
  application with `id`, `status` (`PENDING`, `APPROVED` or `REJECTED`) and
  `created_at` / `updated_at`. The applicant's own view omits the nested user
  object.

Errors:

- HTTP 401 `AUTHENTICATION_REQUIRED` — no valid user session (including a
  soft-deleted user).
- HTTP 500 `INTERNAL_ERROR` — unexpected failure.

Future application data (curriculum vitae, personal information) extends this
endpoint without changing the identity rule.

## Application Documents (Applicant)

Evidence documents are **optional** and can only be attached while the latest
application is `PENDING`. After review the submission is locked: uploads and
deletions answer `409 APPLICATION_STATE_CONFLICT`. The document ownership is
always resolved from the authenticated session; a document id can never select
another user's evidence.

Allowed kinds (`kind`): `certification`, `identification`, `insurance`,
`other`. Allowed file types are PDF, JPEG and PNG, verified server-side by
magic bytes; the client-provided filename and content type are never trusted.
The maximum size is bounded by `TRAINER_EVIDENCE_MAX_BYTES` (default 10 MiB,
ceiling 25 MiB). Files are stored in a private directory outside any web root;
the storage key, hash and internal path are never exposed.

### `POST /api/v1/trainer/application/documents`

Multipart upload with fields `kind` and `file`. Authentication required (user
cookie `ryze_access_token`). Returns `201` with the document metadata as
`data`:

```json
{
  "id": "…",
  "kind": "certification",
  "original_filename": "certification.pdf",
  "content_type": "application/pdf",
  "byte_size": 2048,
  "created_at": "…"
}
```

Errors:

- HTTP 400 `VALIDATION_ERROR` — missing fields, invalid kind or malformed id.
- HTTP 400 `DOCUMENT_TYPE_NOT_ALLOWED` — content is not PDF, JPEG or PNG.
- HTTP 413 `DOCUMENT_TOO_LARGE` — file exceeds the configured bound.
- HTTP 404 `APPLICATION_NOT_FOUND` — no application exists for the caller.
- HTTP 409 `APPLICATION_STATE_CONFLICT` — the application is no longer
  `PENDING`.

### `GET /api/v1/trainer/application/documents`

Returns the caller's active documents for their latest application as
`data.documents`. Same error mapping as above.

### `GET /api/v1/trainer/application/documents/{documentID}/download`

Streams one of the caller's own documents as a private attachment
(`Content-Disposition: attachment`, `X-Content-Type-Options: nosniff`,
`Cache-Control: private, no-store`). A document that does not belong to the
caller answers `404 DOCUMENT_NOT_FOUND`.

### `DELETE /api/v1/trainer/application/documents/{documentID}`

Soft-deletes one of the caller's own documents while the application is
`PENDING` and returns `data: {}`. A missing or already withdrawn document
answers `404 DOCUMENT_NOT_FOUND`; a reviewed application answers
`409 APPLICATION_STATE_CONFLICT`.

---

# ADMIN TRAINER APPLICATIONS

Implemented backend foundation for reviewing trainer applications. A trainer is
an extension of an existing user account: approval creates a trainer profile
linked to the applicant's **same** `user_id`, keeping the user identity,
authentication and password untouched.

The application permission is split into two finer-grained permissions:

- `trainer_applications.read` — list applications, view one application.
  Granted to both administrator roles.
- `trainer_applications.manage` — approve and reject applications. Granted to
  both administrator roles.

Unlike `users.manage` and `trainers.manage`, the trainer-application review is
shared by both administrators: Technical (ADMIN_1) and Management (ADMIN_2)
administrators can both list, view, approve and reject applications.

The flow is:

```
Admin Login
    ↓
Admin Verify (access code)
    ↓
RequireAdminPermission(trainer_applications.read | trainer_applications.manage)
    ↓
AdminTrainerApplicationHandler → TrainerApplicationsService → Repository → MariaDB
```

Applications are never deleted: rejected rows stay in history so the applicant
can re-apply.

## List Applications

```
GET /api/v1/admin/trainer-applications?page=1&limit=20&status=PENDING
```

Returns one page of trainer applications, newest first, each with its linked
user profile. `status` optionally filters by `PENDING`, `APPROVED` or
`REJECTED`. Accessible by both roles (`trainer_applications.read`).

Pagination follows the platform rules: page ≥ 1, limit clamped to a maximum of
100, `data.pagination` reports `page`, `limit`, `total` and `total_pages`.

## Get Application

```
GET /api/v1/admin/trainer-applications/{id}
```

Returns the safe representation of one application with its linked user.
Malformed UUIDs return HTTP 400 `VALIDATION_ERROR`; unknown applications return
HTTP 404 `APPLICATION_NOT_FOUND`. Accessible by both roles
(`trainer_applications.read`).

## Approve Application

```
POST /api/v1/admin/trainer-applications/{id}/approve
```

Approves a `PENDING` application and creates the trainer profile for the
applicant's user account in the **same** database transaction: the application
becomes `APPROVED` and the trainer row is inserted atomically. On success
returns HTTP 200 with the application (`status: APPROVED`) and the same
`user_id` — the user identity is never replaced.

Request body is optional: `{"reason": "..."}` records an optional reviewer
reason (at most 500 characters, whitespace/control characters normalized).
Every decision is appended to the decision history with the reviewing
administrator identity and the status transition.

Errors:

- HTTP 404 `APPLICATION_NOT_FOUND` — unknown application.
- HTTP 400 `VALIDATION_ERROR` — malformed application id.
- HTTP 409 `APPLICATION_STATE_CONFLICT` — the application is not pending
  (already approved, rejected).
- HTTP 409 `USER_ALREADY_TRAINER` — the applicant already owns a trainer
  profile (the transaction rolls back: the application stays `PENDING`).
- HTTP 409 `USER_NOT_ACTIVE` — the linked user account is soft-deleted.

Accessible by both roles (`trainer_applications.manage`).

## Reject Application

```
POST /api/v1/admin/trainer-applications/{id}/reject
```

Rejects a `PENDING` application. The row is kept in history with status
`REJECTED` and no trainer profile is created; the applicant can apply again.
The optional `{"reason": "..."}` body is recorded in the decision history under
the same rules as approval. Errors mirror the approve endpoint
(`APPLICATION_NOT_FOUND`, `APPLICATION_STATE_CONFLICT`,
`VALIDATION_ERROR`). Accessible by both roles (`trainer_applications.manage`).

## Reviewer Evidence and Decision History

Reviewers can inspect the evidence attached to any application and the full,
append-only decision history. Both surfaces require
`trainer_applications.read` (shared by both roles).

### `GET /api/v1/admin/trainer-applications/{id}/documents`

Returns the active evidence documents of the application as
`data.documents`. Each document exposes only `id`, `kind`,
`original_filename`, `content_type`, `byte_size` and `created_at`; the storage
key and hash are never projected. Malformed ids return
`400 VALIDATION_ERROR`; unknown applications return
`404 APPLICATION_NOT_FOUND`.

### `GET /api/v1/admin/trainer-applications/{id}/documents/{documentID}/download`

Streams one evidence document as a private attachment with the same response
headers as the applicant download. A document that does not belong to the
application answers `404 DOCUMENT_NOT_FOUND`.

### `GET /api/v1/admin/trainer-applications/{id}/decisions`

Returns the application's decision history, in chronological order (oldest
first), as `data.decisions`. Each decision exposes `id`, `application_id`,
`admin_id`, `from_status`, `to_status`, `reason` and `created_at`. Decisions
are audit facts: they are never updated or deleted.

---

## Admin Program Pricing

Program pricing endpoints allow the Management Administrator to view and update
the selling price of any active program. These endpoints are protected by the
`plans` permission, which is exclusive to the Management Administrator (ADMIN_2).

Both the program owner (trainer via existing PATCH) and the Management
Administrator can change prices. The pricing model uses integer minor-units
(cents for EUR) to avoid floating-point rounding issues. Free programs must
always have a price of 0; paid programs must meet the configurable minimum.

```
GET /api/v1/admin/programs/:programID
PATCH /api/v1/admin/programs/:programID/pricing
```

### `GET /api/v1/admin/programs/:programID`

Returns one active program by its id (no ownership scoping). Protected by
`plans` permission.

Response `data`:

- `id`, `trainer_id`, `name`, `description`, `type`, `status`,
  `price_minor_units`, `currency`, `created_at`, `updated_at`.

Responses:

- `200 OK` — program object.
- `400 VALIDATION_ERROR` — malformed program id.
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — missing or soft-deleted program.

### `PATCH /api/v1/admin/programs/:programID/pricing`

Updates the price of one active program. Protected by `plans` permission.

Request body:

- `price_minor_units` — integer, price in minor currency units. Free programs
  must have 0; paid programs must meet the configured minimum.
- `currency` — ISO 4217 code, currently only `EUR`.

Responses:

- `200 OK` — updated program object.
- `400 VALIDATION_ERROR` — invalid pricing input.
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — missing or soft-deleted program.

# GENERIC PROGRAMS (PLATFORM-OWNED)

Generic programs are pre-built, platform-owned programs created by the RYZE
team. They reuse the same Program → Week → Workout → Exercise structure as
trainer programs, but their `trainer_id` is always `NULL` and they can never be
converted into trainer-owned content. They hold no commission: a generic program
is never involved in trainer revenue.

## Ownership & Authorization

- Protected by `AdminAuthenticate` and `RequireAdminPermission(plans)` — the
  `plans` permission is exclusive to the Management Administrator (ADMIN_2).
- The platform is the owner. There is no trainer ownership to verify; client
  identity is never trusted — the server derives everything from the session.

## Endpoints

```
GET    /api/v1/programs/generic
POST   /api/v1/programs/generic
GET    /api/v1/programs/generic/:programID
PATCH  /api/v1/programs/generic/:programID
POST   /api/v1/programs/generic/:programID/publish
DELETE /api/v1/programs/generic/:programID
```

## Cap Limits

Every create and update is validated against these hard caps; exceeding any of
them returns `400 VALIDATION_ERROR`:

- Maximum 52 weeks per program (`duration_weeks` ≤ 52).
- Maximum 14 workouts per week.
- Maximum 50 exercises per workout.
- Maximum 50 sets per exercise.
- `tempo` ≤ 20 characters.
- `rpe` between 1 and 10.
- `name` ≤ 255 characters; `description` ≤ 5000 characters.
- `search` ≤ 100 characters; `duration` filter ≤ 52; `frequency` filter ≤ 7.

## Structure & Ordering

Within one program the server derives ordering and uniqueness server-side:

- `week_number`, workout `position`, exercise `position` and `set_number` are
  always the sequence `1..n` in body order (never client-chosen).
- Structural consistency (draft/published) uses: weeks → workouts → exercises →
  sets → catalog exercise lookup.
- At most one assignment of a given catalog exercise per workout; the same
  exercise may appear in different workouts.

## Status & Pricing

- `status`: `draft` | `published`. Only a `draft` generic program can be
  published; publishing is irreversible without a new update cycle.
- `type`: `free` | `premium` | `personalized`.
  - `free` → `price_minor_units` must be 0.
  - `premium`/`personalized` → must meet the configured minimum price.
  - `currency` is always `EUR` (ISO 4217, integer minor units, no floats).
- Downwards pricing changes are not restricted for platform-owned programs.
- `training_type`: controlled vocabulary stored verbatim, `Hypertrophy` |
  `Strength` | `HYROX` | `CrossFit` | `Fat Loss` | `At Home` (nullable). It is
  the marketplace category facet consumed by the public catalog.

## `GET /api/v1/programs/generic`

Paginated list with optional search and filters. Protected by `plans`.

Query parameters:

- `page` — page number, default 1.
- `limit` — page size, default 20.
- `search` — substring match on program name.
- `type` — `free` | `premium` | `personalized`.
- `level` — `Beginner` | `Intermediate` | `Advanced`.
- `training_type` — `Hypertrophy` | `Strength` | `HYROX` | `CrossFit` |
  `Fat Loss` | `At Home`.
- `duration` — exact `duration_weeks`.
- `frequency` — exact `frequency_per_week`.

Response `data`:

- `page`, `limit`, `total`, `total_pages`, `programs` — array of program
  summaries (no nested structure).

Responses:

- `200 OK` — paginated summaries.
- `400 VALIDATION_ERROR` — malformed query parameters.
- `403 FORBIDDEN` — insufficient permissions (ADMIN_1).

## `POST /api/v1/programs/generic`

Creates a generic program (always `draft`, `trainer_id` NULL) and its full
week/workout/exercise/set tree in one transaction. Protected by `plans`.

Request body:

- `name` — string ≤ 255.
- `description` — string ≤ 5000.
- `type` — `free` | `premium` | `personalized`.
- `level` — `Beginner` | `Intermediate` | `Advanced` (nullable).
- `training_type` — controlled vocabulary value (nullable).
- `duration_weeks` — integer, ≤ 52 (nullable).
- `frequency_per_week` — integer, ≤ 7 (nullable).
- `price_minor_units` — integer minor units; 0 for `free`, minimum for paid.
- `currency` — `EUR`.
- `weeks` — array of week objects:
  - `week_number` (ignored, derived server-side), `workouts`:
    - `name`, `description`, `exercises`:
      - `exercise_id` — catalog exercise id.
      - `instructions`, `notes` (optional per-assignment copy).
      - `sets` — array of: `set_type` (`warmup`|`working`|`drop`|`backoff`|
        `failure`), `reps`, `weight_kg`, `rir`, `rpe` (1–10), `rest_seconds`,
        `tempo` (≤ 20).

Responses:

- `201 CREATED` — full generic program object.
- `400 VALIDATION_ERROR` — invalid body or any cap exceeded.
- `403 FORBIDDEN` — insufficient permissions.
- `422 EXERCISE_NOT_FOUND` — unknown catalog `exercise_id`.
- `409 DUPLICATE_EXERCISE` — same catalog exercise twice in one workout.

## `GET /api/v1/programs/generic/:programID`

Returns one generic program with its full nested structure (weeks, workouts,
exercises and sets). Protected by `plans`.

Response `data`:

- Program fields (`id`, `name`, `description`, `type`, `status`, `level`,
  `training_type`, `duration_weeks`, `frequency_per_week`, `price_minor_units`,
  `currency`, `created_at`, `updated_at`) plus `weeks` (each with ordered
  `workouts`); each exercise exposes `catalog_instructions`, `target_muscles`,
  `equipment`, `difficulty`, `video_url`, `image_url`, per-assignment
  `instructions` and `notes`, and its `sets`. Only `draft`/`published` programs
  are returned; the `trainer_id` field is never exposed.

Responses:

- `200 OK` — full program object.
- `400 VALIDATION_ERROR` — malformed program id.
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — missing, soft-deleted generic program, or a
  trainer-owned program id addressed through this endpoint.

## `PATCH /api/v1/programs/generic/:programID`

Full hierarchy update, applied transactionally. Weeks/workouts/exercises/sets
already present keep their ids; structure in the body is matched by position;
missing children are soft-deleted; new ones are created. Protected by `plans`.

Request body: same shape as `POST /api/v1/programs/generic`.

Responses:

- `200 OK` — updated full program object.
- `400 VALIDATION_ERROR` — invalid body, cap exceeded, or structural
  inconsistency (referencing non-existing week/workout/exercise).
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — program not found.
- `422 EXERCISE_NOT_FOUND` — unknown catalog `exercise_id`.
- `409 DUPLICATE_EXERCISE` — same catalog exercise twice in one workout.

## `POST /api/v1/programs/generic/:programID/publish`

Transitions a `draft` generic program to `published` via a guarded conditional
update. Protected by `plans`.

Because publishing exposes the program as a purchasable marketplace entry, the
commercial gate is re-checked server-side before the transition: free programs
must carry a price of `0` minor units and paid programs must carry at least the
configured minimum (`MIN_PROGRAM_PRICE_MINOR_UNITS`, default 100 = €1.00). An
invalid price rejects the publish even though create/update already validate
prices.

Responses:

- `200 OK` — `{ "status": "published" }`.
- `400 VALIDATION_ERROR` — the program price is invalid for its type.
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — program missing, not `draft`, trainer-owned, or
  already published.

## `DELETE /api/v1/programs/generic/:programID`

Soft-deletes a generic program (`deleted_at` set; structure preserved).
Protected by `plans`.

Responses:

- `200 OK` — deletion successful.
- `403 FORBIDDEN` — insufficient permissions.
- `404 PROGRAM_NOT_FOUND` — missing or already deleted program.

# ADMIN COMMISSION RULES

Trainer-specific commission overrides for the commercial rules system. The
Management Administrator (ADMIN_2) manages commission rules through the
`plans.commission.manage` permission.

Every commission rule expresses the platform commission in basis points (1 bps =
0.01%). When a trainer-specific override is active it replaces the global
default commission (env var `DEFAULT_PLATFORM_COMMISSION_BPS`, default 2000 =
20%). The rule resolution is deterministic: trainer-specific active override →
global default.

Commission calculation uses floor division so the platform amount is never
larger than the actual proportional share:

```
platform_amount = floor(price * commission_bps / 10000)
trainer_amount  = price - platform_amount
```

The flow is:

```
Admin Login
    ↓
Admin Verify (access code)
    ↓
RequireAdminPermission(plans.commission.manage)
    ↓
AdminCommissionHandler → CommissionRulesService → CommissionRuleRepository → MariaDB
```

The endpoints are:

```
GET    /api/v1/admin/trainers/:id/commission
PATCH  /api/v1/admin/trainers/:id/commission
DELETE /api/v1/admin/trainers/:id/commission
GET    /api/v1/admin/trainers/:id/commission/resolve
```

### `GET /api/v1/admin/trainers/:id/commission`

Returns the active commission rule for a given trainer. Protected by
`plans.commission.manage` permission.

Response `data`:

- `id`, `trainer_id`, `commission_bps`, `valid_from`, `valid_until`,
  `created_at`, `updated_at`.

Responses:

- `200 OK` — commission rule object.
- `403 FORBIDDEN` — insufficient permissions.
- `404 TRAINER_NOT_FOUND` — trainer not found or soft-deleted.
- `404 COMMISSION_RULE_NOT_FOUND` — no active rule for this trainer.

### `PATCH /api/v1/admin/trainers/:id/commission`

Creates or replaces the active commission rule for a given trainer. Any
previous active rule for the same trainer is soft-deleted. Protected by
`plans.commission.manage` permission.

Request body:

- `commission_bps` — unsigned integer, platform commission in basis points.
  Maximum 10000 (100%). A value of 0 means the trainer keeps the full price.

Response `data`:

- `id`, `trainer_id`, `commission_bps`, `valid_from`, `valid_until`,
  `created_at`, `updated_at`.

Responses:

- `200 OK` — upserted commission rule object.
- `400 VALIDATION_ERROR` — invalid commission input or malformed request body.
- `403 FORBIDDEN` — insufficient permissions.
- `404 TRAINER_NOT_FOUND` — trainer not found or soft-deleted.

### `DELETE /api/v1/admin/trainers/:id/commission`

Soft-deletes the active commission rule for a given trainer. The trainer falls
back to the global default commission after deletion. Protected by
`plans.commission.manage` permission.

Responses:

- `200 OK` — deletion successful.
- `403 FORBIDDEN` — insufficient permissions.
- `404 COMMISSION_RULE_NOT_FOUND` — no active rule for this trainer.

### `GET /api/v1/admin/trainers/:id/commission/resolve`

Returns the resolved commission for a given trainer, whether from a
trainer-specific override or the global default. Protected by
`plans.commission.manage` permission.

Response `data`:

- `commission_bps` — the applicable commission in basis points.
- `is_override` — `true` when the resolution comes from a trainer-specific
  override, `false` when the global default is used.

Responses:

- `200 OK` — commission resolution object.
- `403 FORBIDDEN` — insufficient permissions.

---

# ADMIN COMMERCE

Implemented backend foundation for the administrator sales view over customer
purchases. Accessible only by the Management Administrator (ADMIN_2) through the
`commerce` permission. The endpoints are read-only: no endpoint can create,
modify or refund a purchase. RYZE purchases are final — there is no refund
workflow.

Authorization model:

- A missing or invalid admin session returns HTTP 401 `AUTHENTICATION_REQUIRED`.
- The Technical Administrator (ADMIN_1) returns HTTP 403 `FORBIDDEN` for every
  commerce endpoint.
- The Management Administrator (ADMIN_2) is the sole granted role.

The customer and program objects are preloaded unscoped: soft-deleted buyer
accounts and retired programs keep resolving so the view stays historically
accurate. The `access` flag re-computes the current entitlement state
(completed purchase of a published, non-deleted program) server-side on every
response.

Test Mode purchases are always reported separately and never contribute to real
revenue.

## List Purchases

```
GET /api/v1/admin/purchases
```

Returns one page of purchases matching the optional filters, newest first
(`created_at` desc, id desc for stable ordering), together with the filtered
sales summary and pagination metadata. The static `/purchases/programs` route is
registered before `/purchases/:id` so the router resolves both without
ambiguity.

Query parameters:

- `page` — page number, default `1`.
- `limit` — page size, default `20`, clamped to a maximum of `100`.
- `program_id` — optional UUID; restricts to a single program.
- `status` — optional `pending` | `completed` | `failed`.
- `test` — optional `true` | `false`; absent means both test and real.
- `from` / `to` — optional `YYYY-MM-DD` bounds over `created_at`. The range is
  half-open: `created_at >= from` and `created_at < to + 1 day` (UTC), so `to`
  includes its whole day.

Response `data`:

- `purchases` — array of safe purchase objects:
  - `id` / `program_id` — purchase and program identifiers.
  - `price_minor_units` / `currency` — the amount paid at purchase time.
  - `status` — `pending` | `completed` | `failed`.
  - `test` — `true` for a Test Mode purchase (also visible in the history API).
  - `access` — `true` when the buyer currently holds access (completed purchase
    of a published, non-deleted program).
  - `created_at` / `updated_at` — timestamps.
  - `customer` — `{ name, email }`; `name` is the trimmed display name (may be
    empty) and `email` is the contact address. Never includes user ids or
    session data.
  - `program` — safe program summary: `id`, `name`, `type`, `status`, `level`,
    `duration_weeks`, `frequency_per_week`, `training_type`,
    `price_minor_units`, `currency`, `deleted` (true when the program is
    soft-deleted) and the program timestamps. `trainer_id`, `deleted_at` and
    internal content fields are never exposed.
- `summary` — the same metrics shown on the dashboard:
  - `completed_count` — purchases with status `completed`, test purchases
    included.
  - `pending_count` / `failed_count` — per-status counts.
  - `real_revenue_minor_units` — sum of `price_minor_units` over completed
    non-test purchases.
  - `test_purchase_count` — completed Test Mode purchases, never part of
    `real_revenue_minor_units`.
  - `programs_with_sales` — distinct programs with at least one completed
    non-test purchase.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

Responses:

- `200 OK` — purchases, summary and pagination.
- `400 VALIDATION_ERROR` — malformed `page`/`limit`, non-UUID `program_id`,
  unknown `status`, invalid `test` value, malformed date or a `from` bound later
  than `to`.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid admin session.
- `403 FORBIDDEN` — authenticated admin without the `commerce` permission.

## Purchase Details

```
GET /api/v1/admin/purchases/:purchaseID
```

Returns the safe representation of one purchase (same shape as a list entry)
including the buyer identity and the program summary.

Responses:

- `200 OK` — the purchase object under `data.purchase`.
- `400 VALIDATION_ERROR` — malformed purchase id.
- `401 AUTHENTICATION_REQUIRED` / `403 FORBIDDEN` — as above.
- `404 PURCHASE_NOT_FOUND` — unknown purchase id.

## Program Sales

```
GET /api/v1/admin/purchases/programs
```

Aggregates purchases per program over the same filters, most completed sales
first (`completed` desc, then revenue desc, then program id). Feeds the dashboard
program filter and the per-program section. Also supports a batch mode used by
the program-management area.

Query parameters:

- All the List Purchases parameters describe the aggregation scope. `status`,
  `test`, `from` and `to` compose in batch mode too.
- `program_ids` — optional comma-separated UUIDs. When set, batch mode returns
  exactly one row per requested program, in request order, zero-filled for any
  program without matching purchases; `revenue`, `completed_sales` and
  `real_sales` then count only the matching purchases over the auxiliary
  filters. Ids are deduplicated, validated as UUIDs and capped at 100
  (`MaxPageSize`). `program_id` and `program_ids` are mutually exclusive
  (`400 VALIDATION_ERROR` when both are set).
- In batch mode the repository skips the count query and `LIMIT/OFFSET`; the
  response `pagination` is always `total = len(ids)`, `page = 1`,
  `limit = len(ids)`, `total_pages = 1`.

Response `data`:

- `sales` — array of program rows:
  - `program` — safe program summary (see List Purchases); soft-deleted
    programs keep their historical rows with `deleted: true`.
  - `completed_sales` — completed purchases, test purchases included
    (v2.75 counting, kept for backward compatibility).
  - `real_sales` — completed non-test purchases. This is the authoritative
    "sales" figure used across the admin surfaces: the program area, the
    dashboard and the plan detail page all read the same number from this
    field.
  - `revenue_minor_units` — completed non-test revenue.
  - `test_purchases` — completed Test Mode purchases.
  - `pending_purchases` / `failed_purchases` — per-status counts.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

Responses mirrors List Purchases.

## Reconciliation Events

```
GET /api/v1/admin/purchases/reconciliation-events
```

Returns one page of purchase reconciliation events, newest first. The route is
published before `/purchases/:id` and the surface is read-only: it exists only
to give the Management Administrator visibility over provider-settlement
diagnostics. It can never change a purchase, an entitlement or an event, and it
exposes no decisions users can take from the browser: recovery is always
server-side and idempotent.

Query parameters:

- `page` — page number, default `1`.
- `limit` — page size, default `20`, clamped to a maximum of `100`.
- `scope` — optional `open` | `all`, default `open`. `open` (the default) only
  returns events that still need attention (provisional, misdelivered or
  conflicting outcomes): resolved settlements (`completed`,
  `already_completed`, `reconciled`) are hidden from an open view. `all` returns
  the complete audit trail.
- `outcome` — optional exact outcome code to filter on (see the vocabulary
  below). Takes precedence over `scope`: when set, the event must match that
  outcome.
- `provider` — optional `stripe` | `paypal` to restrict the source provider.

Response `data`:

- `events` — array of safe event objects, newest first (`created_at` desc, id
  desc for stable ordering):
  - `id` — event row identifier.
  - `purchase` — the safe purchase object (same shape as List Purchases,
    including `customer` and `program`). The customer and program are preloaded
    unscoped so soft-deleted buyers and retired programs keep resolving.
  - `provider` — `stripe` | `paypal`.
  - `event_id` — the provider event identifier (Stripe `id` / PayPal `id`).
  - `outcome` — one code from the vocabulary below.
  - `created_at` — record timestamp.
  - The response never includes user ids, secrets, signatures, raw provider
    payloads, `failure_reason`, trainer identifiers or any revenue/commission
    breakdown.

Outcome vocabulary:

- `completed` — a verified delivery completed a pending purchase.
- `reconciled` — a late verified delivery reconciled a failed purchase.
- `already_completed` — the purchase was already completed when the delivery
  arrived (idempotent acknowledgement).
- `conflict_pair_resolved` — a verified settlement found the (user, program)
  pair already resolved another way (a sibling purchase completed first), so
  the payment could not define a second sale.
- `purchase_moved_on` — a verified delivery arrived but the purchase had already
  left the recoverable state.
- `entitlement_integrity` — a settlement could not restore the entitlement
  (recoverable, provider should retry).
- `transient_failure` — a transient/internal failure; the provider is retrying.
  A later delivery of the same event may overwrite this provisional outcome.
- `provider_uncertain` — provider acknowledgement uncertain; the event may be
  redelivered.
- `amount_mismatch`, `currency_mismatch`, `method_mismatch`,
  `reference_mismatch` — the verified event did not match the purchase snapshot
  (recorded and acknowledged with 200).
- `permanent_invalid` — a verified event that can never settle the purchase.
- `late_approval` — PayPal approval arrived for a purchase already failed
  (consent, not money); the order was left to lapse.
- `test_purchase_event` — a verified delivery for a Test Mode purchase, guarded
  and acknowledged with 200.

Only `transient_failure` and `provider_uncertain` are provisional: a later
redelivery overwrites them. Every other outcome is immutable; an idempotent
duplicate delivery leaves the resolved outcome untouched.

Responses:

- `200 OK` — events and pagination.
- `400 VALIDATION_ERROR` — malformed `page`/`limit`, unknown `outcome`, unknown
  `scope`, unknown `provider`.
- `401 AUTHENTICATION_REQUIRED` / `403 FORBIDDEN` — missing or invalid admin
  session / admin without the `commerce` permission.

---

Implemented backend foundation for managing regular client accounts. The user
permission is split into two finer-grained permissions:

- `users.read` — read-only view (list users, view one user). Granted to both
  administrator roles.
- `users.manage` — full lifecycle (create, update, soft-delete, reactivate,
  administrative password reset, deleted-user listing). Granted only to the
  Technical Administrator (ADMIN_1).

The Management Administrator (ADMIN_2) holds `users.read` only: every lifecycle
endpoint returns HTTP 403 `FORBIDDEN` for ADMIN_2, enforced server-side by
`RequireAdminPermission(PermissionUsersManage)`.

Only regular client accounts are managed here. Trainer content management,
plans, finance and marketing are separate permissions.

The flow is:

```
Admin Login
    ↓
Admin Verify (access code)
    ↓
RequireAdminPermission(users.read | users.manage)
    ↓
AdminUserHandler → AdminUserService → UserRepository → MariaDB
```

There is intentionally **no permanent-deletion endpoint**. Account removal is
always a soft delete; rows, UUIDs and `created_at` are preserved and the account
can be reactivated by an administrator or re-registered publicly.

## List Users

```
GET /api/v1/admin/users?page=1&limit=20
```

Returns active registered users, newest first. Soft-deleted users are never
returned. Accessible by both roles (`users.read`).

Pagination:

- `page` defaults to 1 (must be ≥ 1).
- `limit` defaults to 20; values above the platform maximum (100) are clamped.
- Invalid numeric values return HTTP 400 `VALIDATION_ERROR`.

Response body:

```
data.users[]          safe user representation (id, email, first_name,
                      last_name, created_at, updated_at)
data.pagination       { page, limit, total, total_pages }
```

---

## User Details

```
GET /api/v1/admin/users/{id}
```

Returns the safe representation of one active user. Soft-deleted users are
never returned. Accessible by both roles (`users.read`).

- Unknown or soft-deleted users: HTTP 404 `USER_NOT_FOUND`.
- Malformed identifiers: HTTP 400 `VALIDATION_ERROR`.

The safe representation never includes password hashes, session versions or
deletion markers.

---

## List Deleted Users

```
GET /api/v1/admin/users/deleted?page=1&limit=20
```

Returns soft-deleted users, most recently deleted first. Active and deleted
users are never mixed in a single list. Accessible only by ADMIN_1
(`users.manage`); ADMIN_2 receives HTTP 403 `FORBIDDEN`.

Same pagination and safe-representation rules as List Users.

---

## Create User

```
POST /api/v1/admin/users
```

Creates a new active user (HTTP 201, message "User created successfully.").
Accessible only by ADMIN_1 (`users.manage`).

Request body:

```
{
  "email": "user@ryze.com",
  "password": "plaintext-password",
  "first_name": "User",
  "last_name": "Name"
}
```

Rules:

- The caller can never set `id`, `deleted_at`, `session_version` or
  `password_hash`: those are controlled exclusively by the backend.
- The password is always hashed with Argon2id before storage and never returned
  or logged.
- If the email belongs to an active account: HTTP 409
  `EMAIL_ALREADY_REGISTERED`.
- If the email belongs to a soft-deleted account: the existing identity is
  reactivated (same UUID and `created_at` are preserved, `deleted_at` is
  cleared and the password is replaced). No second row is created.
- Missing or malformed fields return HTTP 400 `VALIDATION_ERROR`.

---

## Update User

```
PATCH /api/v1/admin/users/{id}
```

Updates the whitelisted safe fields of an active user (HTTP 200, message "User
updated successfully."). Accessible only by ADMIN_1 (`users.manage`).

Request body (at least one field required; `null` means unchanged):

```
{
  "email": "new@ryze.com",
  "first_name": "New",
  "last_name": "Name"
}
```

Rules:

- Only `email`, `first_name` and `last_name` can be modified. `id`,
  `password_hash`, `deleted_at` and `session_version` can never be changed here
  (password changes use the dedicated administrative reset).
- The email must not be owned by any other existing identity (active or
  soft-deleted): HTTP 409 `EMAIL_ALREADY_REGISTERED`.
- Unknown or soft-deleted users: HTTP 404 `USER_NOT_FOUND`.
- Empty body or malformed fields: HTTP 400 `VALIDATION_ERROR`.

---

## Disable User

```
PATCH /api/v1/admin/users/{id}/disable
```

Soft-disables a regular user account (HTTP 200, message "User disabled
successfully."). Accessible only by ADMIN_1 (`users.manage`).

Effects:

- Sets `deleted_at` (soft delete): the row, UUID and email are preserved.
- Increments the account `session_version` in the same operation, immediately
  revoking every active session.
- The user can no longer log in.
- The account data remains intact and recoverable.

Unknown, already-disabled or malformed identifiers follow the same error
contract as User Details.

---

## Reactivate User

```
POST /api/v1/admin/users/{id}/reactivate
```

Restores a soft-deleted user (HTTP 200, message "User reactivated
successfully."). Accessible only by ADMIN_1 (`users.manage`).

Effects:

- Clears `deleted_at` only: the same UUID, email, `created_at`, password hash
  and session version are preserved.
- The account becomes active again and can authenticate again.
- Because the soft delete incremented `session_version`, every pre-deletion
  access token remains invalid.
- The password is not changed.

- Already-active users: HTTP 409 `USER_ALREADY_ACTIVE`.
- Unknown users: HTTP 404 `USER_NOT_FOUND`.

---

## Reset User Password

```
POST /api/v1/admin/users/{id}/password
```

Replaces an active user's password without requiring the current password
(HTTP 200, message "Password reset successfully."). Accessible only by ADMIN_1
(`users.manage`).

Request body:

```
{
  "new_password": "new-plaintext-password"
}
```

Effects:

- The password is hashed with Argon2id and stored as `password_hash`.
- `session_version` is incremented in the same statement, invalidating every
  previously issued session.
- The plaintext password, the hash, the token and `session_version` are never
  returned.

- Empty `new_password`: HTTP 400 `VALIDATION_ERROR`.
- Unknown or soft-deleted users: HTTP 404 `USER_NOT_FOUND`.

---

## Public Re-registration after Disable

When a disabled user's email is registered again through the public endpoint,
the original soft-deleted row is reactivated: the same UUID and `created_at`
are preserved, `deleted_at` is cleared, the password is replaced, and the
previous password stops working. Exactly one row exists for the email at all
times.

---

# ADMIN TRAINER MANAGEMENT

Implemented backend foundation for managing trainer profiles. A trainer is a
distinct entity from the user account: the `trainers` table stores only the
trainer identity (`id`, `user_id`, timestamps and soft-delete state) while the
linked `users` row owns authentication, contact and profile data. One user can
own at most one **active** trainer profile.

The trainer permission is split into two finer-grained permissions:

- `trainers.read` — read-only view (list trainers, view one trainer). Granted
  to both administrator roles.
- `trainers.manage` — full lifecycle (create, update, soft-delete, reactivate,
  deleted-trainer listing). Granted only to the Technical Administrator
  (ADMIN_1).

The Management Administrator (ADMIN_2) holds `trainers.read` only: every
lifecycle endpoint returns HTTP 403 `FORBIDDEN` for ADMIN_2, enforced
server-side by `RequireAdminPermission(PermissionTrainersManage)`.

The flow is:

```
Admin Login
    ↓
Admin Verify (access code)
    ↓
RequireAdminPermission(trainers.read | trainers.manage)
    ↓
AdminTrainerHandler → AdminTrainerService → TrainerRepository → MariaDB
```

Trainer and user lifecycles are fully independent:

- Soft-deleting a trainer **never touches** the linked user account.
- Reactivating a trainer **never reactivates** a soft-deleted user: the
  operation fails with HTTP 409 `USER_DISABLED` instead.
- There is intentionally **no permanent-deletion endpoint**. Trainer removal is
  always a soft delete; rows, UUIDs, user links and `created_at` are preserved.

## List Trainers

```
GET /api/v1/admin/trainers?page=1&limit=20
```

Returns active trainers, newest first, each with its linked user profile.
Soft-deleted trainers are never returned. Accessible by both roles
(`trainers.read`).

Pagination:

- `page` defaults to 1 (must be ≥ 1).
- `limit` defaults to 20; values above the platform maximum (100) are clamped.
- Invalid numeric values return HTTP 400 `VALIDATION_ERROR`.

Response body:

```
data.trainers[]       safe trainer representation (id, user_id, email,
                      first_name, last_name, status, created_at, updated_at)
data.pagination       { page, limit, total, total_pages }
```

## Trainer Details

```
GET /api/v1/admin/trainers/{id}
```

Returns one active trainer with its linked user profile. Soft-deleted trainers
are never returned. Accessible by both roles (`trainers.read`).

- Malformed or missing `{id}`: HTTP 400 `VALIDATION_ERROR`.
- Unknown or soft-deleted trainer: HTTP 404 `TRAINER_NOT_FOUND`.

## Deleted Trainers

```
GET /api/v1/admin/trainers/deleted?page=1&limit=20
```

Returns soft-deleted trainers, newest deletion first, as a clearly separated
view for lifecycle management. The same safe representation is used. Only
ADMIN_1 (`trainers.manage`); ADMIN_2 receives HTTP 403 `FORBIDDEN`.

## Create Trainer

```
POST /api/v1/admin/trainers
```

Creates a new user account and its trainer profile in one operation. The user
is created through the registration flow, so the identity rule is shared with
public registration (email validation, Argon2id hashing, soft-deleted-email
reactivation). If the trainer profile cannot be created, the just-created user
is soft-deleted as a compensation: there are never partial accounts. Only
ADMIN_1 (`trainers.manage`).

Request body:

```
{
  "email": "trainer@ryze.local",
  "password": "plaintext-password",
  "first_name": "First",
  "last_name": "Last"
}
```

Responses:

- HTTP 201 `Trainer created successfully.` with the safe trainer
  representation.
- HTTP 400 `VALIDATION_ERROR` for malformed input or an empty password.
- HTTP 409 `EMAIL_ALREADY_REGISTERED` when the email belongs to an active
  identity.
- HTTP 409 `TRAINER_ALREADY_LINKED` when the (reactivated) user already owns an
  active trainer profile.

The caller can never set the trainer id, the user id or `deleted_at`.

## Update Trainer

```
PATCH /api/v1/admin/trainers/{id}
```

Updates the whitelisted safe profile fields of the user linked to an active
trainer. The trainer profile itself has no editable fields. Only ADMIN_1
(`trainers.manage`).

Request body (all fields optional, at least one required):

```
{
  "email": "new@ryze.local",
  "first_name": "Renamed",
  "last_name": "Surname"
}
```

Responses:

- HTTP 200 `Trainer updated successfully.` with the safe trainer
  representation.
- HTTP 400 `VALIDATION_ERROR` when no field is provided or a field is invalid.
- HTTP 404 `TRAINER_NOT_FOUND` for an unknown or soft-deleted trainer.
- HTTP 409 `USER_DISABLED` when the linked user account is soft-deleted.
- HTTP 409 `EMAIL_ALREADY_REGISTERED` when the email belongs to another
  identity.

## Disable Trainer

```
PATCH /api/v1/admin/trainers/{id}/disable
```

Soft-deletes an active trainer. The row is preserved and the linked user
account is **never touched**. Only ADMIN_1 (`trainers.manage`).

- HTTP 200 `Trainer disabled successfully.`
- HTTP 400 `VALIDATION_ERROR` for a malformed `{id}`.
- HTTP 404 `TRAINER_NOT_FOUND` for an unknown or already-deleted trainer.

## Reactivate Trainer

```
POST /api/v1/admin/trainers/{id}/reactivate
```

Restores a soft-deleted trainer: the same trainer UUID, user link and
`created_at` are preserved and `deleted_at` is cleared. Reactivation never
alters the linked user lifecycle. Only ADMIN_1 (`trainers.manage`).

- HTTP 200 `Trainer reactivated successfully.`
- HTTP 400 `VALIDATION_ERROR` for a malformed `{id}`.
- HTTP 404 `TRAINER_NOT_FOUND` for an unknown trainer.
- HTTP 409 `TRAINER_ALREADY_ACTIVE` for an already-active trainer.
- HTTP 409 `USER_DISABLED` when the linked user account is soft-deleted.
- HTTP 409 `TRAINER_ALREADY_LINKED` when the user already owns another active
  trainer profile.

---

# TECHNICAL ADMINISTRATION

Planned endpoints. Accessible only by Technical Administrators.

---

## Dashboard

```
GET /api/v1/admin/dashboard
```

Returns system statistics.

---

## Audit Logs

```
GET /api/v1/admin/logs
```

Returns administrative audit logs.

---

## System Settings

```
GET /api/v1/admin/settings
```

Returns global platform settings.

---

## Update System Settings

```
PATCH /api/v1/admin/settings
```

Updates global configuration.

---

## Maintenance Mode

```
PATCH /api/v1/admin/maintenance
```

Enables or disables maintenance mode.

---

# MANAGEMENT ADMINISTRATION

Planned endpoints. Accessible only by Management Administrators.

---

## Dashboard

```
GET /api/v1/management/dashboard
```

Returns business statistics.

---

## Marketplace Statistics

```
GET /api/v1/management/statistics
```

Returns marketplace analytics.

---

## Trainers

```
GET /api/v1/management/trainers
```

Returns every registered trainer.

---

## Trainer Applications

Trainer application review is implemented and shared by both administrator
roles under `GET/POST /api/v1/admin/trainer-applications` (see the ADMIN TRAINER
APPLICATIONS section). There are no separate `/api/v1/management/...`
application endpoints.

## Platform Revenue

```
GET /api/v1/management/revenue
```

Returns platform-wide revenue.

---

## Sales

The admin sales view is implemented and Management-Administrator-only under the
`GET /api/v1/admin/purchases*` endpoints (see the [ADMIN COMMERCE](#admin-commerce)
section; the `commerce` permission). There are no separate
`/api/v1/management/...` sales endpoints.

---

## Promotions

```
GET /api/v1/management/promotions
```

Returns active promotions.

---

## Create Promotion

```
POST /api/v1/management/promotions
```

Creates a promotion.

---

## Update Promotion

```
PATCH /api/v1/management/promotions/{id}
```

Updates a promotion.

---

## Delete Promotion

```
DELETE /api/v1/management/promotions/{id}
```

Disables a promotion.

---

# SCHEDULER ENDPOINTS

Reserved for internal system operations.

These endpoints are not publicly accessible.

Examples:

- Notification cleanup
- Temporary upload cleanup
- Scheduled emails
- Cache refresh
- Statistics generation
- Ranking updates
- Event activation
- Event expiration
- Pack expiration
- Backup generation

---

# WEBSOCKET EVENTS

The platform exposes several real-time events.

Examples include:

```
notification:new

notification:read

chat:new_message

chat:typing

chat:message_deleted

trainer:application_status

purchase:completed

pack:published

mission:completed

achievement:unlocked

event:started

event:ended

system:announcement
```

Additional events may be introduced as the platform evolves.

---

# API DOCUMENTATION

Every endpoint should include:

- Description
- Authentication requirements
- Required permissions
- Request body
- Response body
- Possible errors
- Validation rules

API documentation should remain synchronized with the implementation.

---

# DEPRECATION POLICY

Deprecated endpoints should not be removed immediately.

Instead:

- Mark the endpoint as deprecated.
- Document the replacement endpoint.
- Maintain compatibility until the next major API version.

Breaking changes should only occur between major API versions.

---

## RYZE Feed API

The RYZE Feed API provides endpoints for consuming Feed content, managing Likes and Follows, and allowing authorized trainers to publish and manage their own content.

All endpoints must follow the existing RYZE API conventions, authentication requirements, versioning strategy, validation rules, and error-handling standards.

### Feed Consumption

#### Get Feed

```http
GET /api/v1/feed

Returns a paginated set of Feed content personalized for the authenticated user.

The response should provide the information required to render the Feed without requiring unnecessary additional requests.

Conceptual response data:

feed item
├── id
├── video
├── thumbnail
├── title
├── description
├── trainer
│   ├── id
│   ├── name
│   └── profile_image
├── likes_count
└── user_liked

The endpoint should support incremental loading suitable for infinite scrolling.

The pagination mechanism must be stable enough to prevent excessive duplication or skipping of content while the user is consuming the Feed.

Feed Content
Get Feed Item
GET /api/v1/feed/:id

Returns a single published Feed item and its associated creator information.

Unpublished, hidden, deleted, or otherwise inaccessible content must not be returned to normal users.

Likes
Like Feed Item
POST /api/v1/feed/:id/like

Creates a Like for the authenticated user.

The operation must be idempotent from the client's perspective.

A user must not be able to create duplicate Likes for the same Feed item.

Unlike Feed Item
DELETE /api/v1/feed/:id/like

Removes the authenticated user's Like from the Feed item.

The operation should remain safe if the user has not previously Liked the content.

Trainer Following
Follow Trainer
POST /api/v1/trainers/:id/follow

Creates a Follow relationship between the authenticated user and the specified trainer.

The operation must prevent duplicate Follow relationships.

Unfollow Trainer
DELETE /api/v1/trainers/:id/follow

Removes the authenticated user's Follow relationship with the specified trainer.

Get Trainer Follow State

The API should provide the current Follow state where required by Feed and Trainer interfaces.

The exact endpoint may be:

GET /api/v1/trainers/:id/follow

or incorporated into the Trainer response.

The implementation should avoid unnecessary requests when the state can already be returned as part of another response.

Trainer Content Management

Only authorized trainers may create and manage their own RYZE Feed content.

Create Feed Content
POST /api/v1/trainer/feed

Creates a new Feed content record.

The initial content should not automatically become publicly available if video processing, validation, moderation, or other required processing has not completed.

Upload Video
POST /api/v1/trainer/feed/:id/video

Uploads or associates the video media with the Feed content.

The upload must pass all RYZE media security and validation requirements before becoming available for playback.

Update Feed Content
PATCH /api/v1/trainer/feed/:id

Allows the owning trainer to modify editable metadata such as:

Title
Description
Category/tags when supported
Visibility/status when permitted

A trainer must not be able to modify content belonging to another trainer.

Delete Feed Content
DELETE /api/v1/trainer/feed/:id

Removes the trainer's Feed content according to the project's established deletion strategy.

Deleted content must immediately stop appearing in normal Feed results.

Authorization

Feed permissions are separated into two primary groups.

Authenticated Users

Users may:

View Feed content
Like content
Remove Likes
Follow trainers
Unfollow trainers
View public trainer information
Authorized Trainers

Trainers may additionally:

Create Feed content
Upload videos
Edit their own content
Publish content when permitted
Hide their own content
Remove their own content

A trainer must never be able to modify another trainer's content.

Content Visibility

The Feed API must distinguish between content states.

Normal Feed requests should only return content that is:

Published
Valid
Available for the current user
Not deleted
Not hidden

Administrative or trainer-management endpoints may access additional states according to their authorization level.

Personalization

Feed retrieval may use:

Trainer follows
Video Likes
Content relevance
Content freshness
Popularity
Future recommendation signals

The API contract should not expose or depend on the internal ranking formula.

The ranking system may evolve independently while maintaining the same Feed API contract.

Error Handling

The API must use the standard RYZE error response format.

Examples of relevant errors include:

401 Unauthorized
403 Forbidden
404 Feed content not found
409 Relationship already exists
422 Invalid content

The exact status codes must follow the global RYZE API conventions.

Security

All Feed API endpoints must follow the security requirements defined in security.md.

Particular attention must be given to:

Authentication
Authorization
Trainer ownership checks
Rate limiting
Input validation
Upload validation
File type validation
File size limits
Malicious media detection
Abuse prevention
Resource enumeration protection

Feed APIs must never trust client-provided trainer IDs, user IDs, ownership information, file metadata, or permission claims.

---

# FUTURE EXPANSION

The API has been designed to support future features without major architectural changes.

Possible future additions include:

- Native mobile applications
- Public developer API
- GraphQL gateway
- AI endpoints
- Advanced analytics
- Additional payment providers
- Multi-currency support
- Wearable integrations
- Health platform integrations
- External trainer APIs

The existing architecture should be extended whenever possible instead of being replaced.

---

# FINAL NOTES

This document defines the official API specification of the RYZE platform.

The API is expected to evolve throughout development.

AI assistants may introduce:

- New endpoints
- New parameters
- Additional response fields
- Better naming
- Improved security
- Improved validation
- Better endpoint organization

...provided that these changes improve the platform's architecture and remain consistent with the project's documentation.

Whenever the API changes:

- Update `api.MD`.
- Update `update_logs.MD`.
- Maintain backward compatibility whenever possible.
- Avoid breaking changes unless a new API version is introduced.

The RYZE API should always prioritize consistency, simplicity, security, scalability and long-term maintainability.
