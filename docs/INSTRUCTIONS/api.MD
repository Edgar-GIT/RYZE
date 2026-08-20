# RYZE FITNESS
# API.MD

Version: 1.0.0

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

Permission denied.

404

Resource not found.

409

Conflict.

422

Validation error.

429

Too many requests.

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
program id and a safe nested Program object. Entitlements do NOT grant access
to `GET /me/program` — assignment remains the sole access mechanism.

Response `data`:

- array of entitlement objects, each with:
  - `id` — entitlement UUID.
  - `program_id` — the entitlement program reference.
  - `program` — safe Program object: `id`, `name`, `description`, `type`,
    `status`, `created_at`, `updated_at`.
  - `created_at`, `updated_at` — entitlement lifecycle timestamps.

Responses:

- `200 OK` — the user's entitlements; a user without entitlements gets an
  empty array.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.

---

### Client Purchase

#### `POST /api/v1/me/programs/:programID/purchase`

Creates a pending purchase transaction for the authenticated user. The endpoint
validates the program, snapshots the current price and currency, resolves the
applicable commission, calculates the deterministic split, and persists a
pending purchase record. The purchase status is always "pending"; no payment is
processed and no entitlement is created. A future payment provider will
transition the purchase to "completed" and create the entitlement atomically.

Identity comes exclusively from the authentication context. A client-supplied
`user_id` or `program_id` in the body is always ignored.

Request: no body required.

Response `data`:

- `id` — purchase UUID.
- `program_id` — the purchased program reference.
- `price_minor_units` — snapshot of the program price at creation time.
- `currency` — snapshot of the program currency at creation time.
- `commission_bps` — the applicable commission in basis points.
- `platform_amount` — the platform's share in minor currency units.
- `trainer_amount` — the trainer's share in minor currency units.
- `status` — always `"pending"`.

Responses:

- `201 CREATED` — the pending purchase snapshot.
- `400 VALIDATION_ERROR` — invalid program id.
- `401 AUTHENTICATION_REQUIRED` — missing or invalid authentication.
- `404 PROGRAM_NOT_FOUND` — program does not exist, is soft-deleted or is
  not published.
- `409 PROGRAM_NOT_PURCHASABLE` — the program is free and cannot be purchased.
- `409 DUPLICATE_ENTITLEMENT` — the user already owns this program.
- `409 DUPLICATE_PURCHASE` — a purchase for this program is already in
  progress.

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
- `400 VALIDATION_ERROR` — missing/malformed identifiers, self-relationship,
  invalid body.
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
- Financial management
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
the admin token validator.

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
details are ever returned.

Public endpoint.

Routes that require administrator access are protected by the admin
authentication middleware, which accepts only the `ryze_admin_access_token`
cookie validated as an `admin`-kind token carrying a configured admin identity
(`ADMIN_1`/`ADMIN_2`). Stage tokens, user tokens and any invalid token are
rejected with HTTP 401 `AUTHENTICATION_REQUIRED`.

---

## Register

```
POST /api/v1/auth/register
```

Creates a new client account.

Public endpoint.

Accounts are inactive until email verification is completed.

If the email belongs to a soft-deleted (previously deleted) account, the same
registration restores that account instead of creating a duplicate: the
original UUID and creation date are kept, the names are updated and the
password is replaced by the new password hash. The endpoint still returns
HTTP 201 and the same response structure in both cases. Registering an email
that belongs to an active account returns HTTP 409
`EMAIL_ALREADY_REGISTERED`.

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
HTTP 401 `INVALID_CREDENTIALS`. The new password replaces the stored
`password_hash` and the account's `session_version` is incremented, revoking
every previously issued access token. On success the response clears the
`ryze_access_token` cookie (HTTP 200, empty `data`) and the user must
authenticate again. Empty passwords return HTTP 400 `VALIDATION_ERROR`.

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

Returns one page of published programs from all trainers, newest first, with
pagination metadata.

Public endpoint. No authentication required.

Query parameters:

- `page` — page number, 1-based, default `1`.
- `limit` — page size, default `20`, clamped to `100`.

Response `data`:

- `programs` — array of program objects: `id`, `trainer_id`, `name`,
  `description`, `type`, `status`, `price_minor_units`, `currency`,
  `created_at`, `updated_at`.
- `pagination` — `page`, `limit`, `total`, `total_pages`.

### `GET /api/v1/programs/:programID`

Returns one published program. Draft, soft-deleted and unknown programs all
return the same generic 404 to prevent information leakage.

Public endpoint. No authentication required.

Response `data`:

- `id`, `trainer_id`, `name`, `description`, `type`, `status`,
  `price_minor_units`, `currency`, `created_at`, `updated_at`.

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

---

## API Version

```
GET /api/v1/version
```

Returns the running API version.

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

## Refund

```
POST /api/v1/payments/{id}/refund
```

Processes a refund when applicable.

---

# QUESTIONNAIRE ENDPOINTS

Premium programs require questionnaires.

---

## List Questionnaires

```
GET /api/v1/questionnaires
```

Returns available questionnaires.

---

## Questionnaire Details

```
GET /api/v1/questionnaires/{id}
```

Returns a questionnaire.

---

## Submit Questionnaire

```
POST /api/v1/questionnaires/{id}
```

Stores questionnaire answers.

---

## Questionnaire History

```
GET /api/v1/questionnaires/history
```

Returns previously submitted questionnaires.

---

# NUTRITION ENDPOINTS

Nutrition programs associated with purchased packs.

---

## Current Nutrition Program

```
GET /api/v1/nutrition/current
```

Returns the user's current nutrition program.

---

## Nutrition History

```
GET /api/v1/nutrition/history
```

Returns previous nutrition programs.

---

## Download Nutrition PDF

```
GET /api/v1/nutrition/pdf
```

Downloads the nutrition program.

---

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

Implemented backend foundation for the platform exercise library. An exercise
is a global, platform-owned catalog entry that exists only once and is reused
across programs and packs. The catalog is **public and read-only**: the three
endpoints below require no authentication and are identical for every caller.
There are no write endpoints in this foundation; the catalog is populated
outside the API (seed data / future admin tooling). Soft-deleted exercises are
never returned.

## Exercise Library

```
GET /api/v1/exercises?page=1&limit=20
```

Returns one page of the exercise catalog in alphabetical order by name. Public
endpoint, no authentication required.

Pagination follows the platform rules: page ≥ 1, limit clamped to a maximum of
100, `data.pagination` reports `page`, `limit`, `total` and `total_pages`.
`data.exercises` is an array of exercise objects.

```
GET /api/v1/exercises?limit=2
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
        "description": "…",
        "target_muscles": "…",
        "equipment": "…",
        "difficulty": "…",
        "video_url": "…",
        "image_url": "…",
        "created_at": "…",
        "updated_at": "…"
      }
    ],
    "pagination": { "page": 1, "limit": 2, "total": 40, "total_pages": 20 }
  }
}
```

## Exercise Details

```
GET /api/v1/exercises/{id}
```

Returns complete exercise information for one active catalog entry.

Includes:

- Description
- Images (`image_url`)
- Videos (`video_url`)
- Target muscles
- Equipment

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
    "id": "51f8b06a-…-a6f1f4b6e257",
    "name": "Barbell Squat",
    "description": "…",
    "target_muscles": "…",
    "equipment": "…",
    "difficulty": "…",
    "video_url": "…",
    "image_url": "…",
    "created_at": "…",
    "updated_at": "…"
  }
}
```

## Exercise Search

```
GET /api/v1/exercises/search?q={query}&page=1&limit=20
```

Searches exercises by name, case-insensitively. The `q` parameter is required;
an empty or blank query returns `400 VALIDATION_ERROR`. LIKE wildcards in the
query are treated literally. The response shape and pagination are identical to
`GET /api/v1/exercises`.

```
GET /api/v1/exercises/search?q=squat
```

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

Future application data (curriculum vitae, certificates, personal information)
extends this endpoint without changing the identity rule.

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
Errors mirror the approve endpoint (`APPLICATION_NOT_FOUND`,
`APPLICATION_STATE_CONFLICT`). Accessible by both roles
(`trainer_applications.manage`).

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

# ADMIN USER MANAGEMENT

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

```
GET /api/v1/management/sales
```

Returns sales statistics.

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
