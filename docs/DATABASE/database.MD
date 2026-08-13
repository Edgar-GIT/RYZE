# RYZE FITNESS
# DATABASE.MD

Version: 1.0.0

---

# PURPOSE

This document defines the complete database architecture of the RYZE platform.

It is the single source of truth regarding every entity, relationship,
business rule and persistence decision.

No table should be created without following this document.

The database is expected to evolve during development.

Whenever a better design is identified, this document should be updated before implementation.

# DATABASE PHILOSOPHY

The database is designed around long-term maintainability rather than short-term simplicity.

Historical data should be preserved whenever possible.

Data loss should be avoided.

Entities should remain extensible, allowing new functionality without requiring major structural changes.

Whenever there is uncertainty between deleting data or preserving it, preserving data should be preferred.

The database should be treated as one of the project's most valuable assets.

# FLEXIBILITY

This document defines the current architecture, not a limitation.

AI assistants are encouraged to improve:

- Table design
- Relationships
- Indexes
- Constraints
- Normalization
- Performance
- Security

...whenever those improvements are justified.

Additional tables, columns or relationships may be introduced whenever they improve the overall architecture.

Every modification must be documented inside update_logs.MD.

# COMMON COLUMNS

Whenever applicable, entities should include:

id

created_at

updated_at

deleted_at

is_active

created_by

updated_by

deleted_by

version

The exact set of columns depends on the entity.

Not every table requires every column.

The AI should determine the most appropriate structure for each case.

# FILE STORAGE

Binary files should never be stored inside the database.

The database stores only metadata.

Examples:

- File path
- File name
- MIME type
- Size
- Owner
- Upload date

Physical files remain inside the uploads directory.

# INDEXING

Indexes should be created whenever they improve performance.

Typical indexed columns include:

- email
- username
- slug
- created_at
- trainer_id
- pack_id
- is_active

Avoid unnecessary indexes.

Indexes should be evaluated according to real usage.

# CONSTRAINTS

Use database constraints whenever possible.

Examples:

- UNIQUE
- NOT NULL
- FOREIGN KEY
- CHECK
- DEFAULT

Business rules should not rely solely on backend validation.

# TRANSACTIONS

Critical operations should use database transactions.

Examples:

- Purchases
- Refunds
- Trainer approval
- Account deletion
- Payment confirmation

Transactions should guarantee data consistency.

# AI GENERATED DATA

Artificial Intelligence may generate:

- Training programs

- Nutrition programs

- Recommendations

- Notifications

- Suggestions

AI-generated content should remain identifiable inside the database whenever possible.

Future versions may store metadata such as:

- AI provider

- Model version

- Prompt version

- Generation timestamp

# DATA RETENTION

Historical information should be preserved whenever possible.

Temporary data may be removed automatically.

Examples:

- Temporary uploads

- Expired notifications

- Cache

Critical business information should never be deleted automatically.

# MIGRATIONS

Every database modification must be performed through migrations.

The schema should never be manually modified in production.

Migration history must remain consistent across environments.

---

# DESIGN PRINCIPLES

The database prioritizes:

- Scalability
- Maintainability
- Performance
- Security
- Auditability
- Future expansion

The database should always favor normalization unless a justified optimization exists.

Soft deletion is preferred over permanent deletion whenever possible.

Historical information should be preserved.

Whenever business data changes over time, versioning should be considered.

---

# GENERAL RULES

## IDs

Every table uses:

UUID

Never sequential IDs.

---

## Dates

Every table should contain, whenever applicable:

created_at

updated_at

deleted_at

---

## Soft Delete

User information is never permanently removed.

Records should be marked as inactive instead.

Example:

is_active

deleted_at

deleted_reason

---

## Auditability

Administrative actions should always be traceable.

Critical modifications must generate audit records.

---

## Naming

Tables:

snake_case

Columns:

snake_case

Foreign Keys:

entity_id

Example:

user_id

trainer_id

pack_id

---

## Relationships

Prefer explicit foreign keys.

Avoid duplicated information.

Whenever a many-to-many relationship exists,
create an intermediate table.

---

# MAIN ENTITIES

## Users

Stores authentication and account information.

Includes:

- Authentication
- Roles
- Profile
- Preferences
- Security
- Status

Security fields:

- password_hash
- session_version

`session_version` is incremented whenever the account password changes (by the
account holder or by an administrator) or the account is soft-deleted (by the
account holder or by an administrator). Access tokens embed the session version
they were issued for, so such operations revoke every previously issued token
(see security.MD, Session Revocation). Reactivation (administrative or public
re-registration) preserves the increment, so pre-deletion tokens stay invalid.

Soft delete is the only deletion mechanism: rows, UUIDs and `created_at` are
preserved and `deleted_at` is populated. There is no permanent-deletion
endpoint; reactivation and re-registration restore the original identity.

The repository provides the queries used by the admin user-management
foundation:

- `ListActive` — paginated active users (soft-delete scope applied, newest
  first, active total).
- `ListDeleted` — paginated soft-deleted users (`deleted_at IS NOT NULL`,
  newest first, deleted total), a clearly separated view from the active
  listing.
- `FindByIDIncludingDeleted` — user lookup without the soft-delete scope,
  used by the reactivation lifecycle.
- `ClearDeletedAt` — reactivates a row by clearing `deleted_at` only
  (UUID, email, `created_at`, password hash and session version preserved),
  guarded by `deleted_at IS NOT NULL` so an already-active user can never be
  reactivated.
- `DeleteAccount` — the atomic soft-delete primitive (`deleted_at` +
  `session_version` increment in one statement) shared by account deletion and
  the admin disable endpoint.

---

## User Profiles

Stores public profile information.

Examples:

- Name
- Biography
- Avatar
- Banner
- Links
- Visibility
- Location

---

## Roles

Supported roles:

- Client
- Trainer
- Technical Administrator
- Management Administrator

Each account has exactly one role.

Administrators and trainers can also purchase programs as regular users.

---
## Packs

Represents every commercial product available on the platform.

Each pack contains one or more training and nutrition resources depending on its type.

Packs are immutable once purchased.

Future updates create new versions instead of modifying historical purchases.

Types:

- Generic
- Premium Level 1
- Premium Level 2

Supports:

- Categories
- Tags
- Versioning
- Search
- Difficulty
- Equipment
- Duration

---

## Purchases

Stores every purchase.

Historical data never changes even if the pack changes later.

Includes:

- Purchase price
- Currency
- Discounts
- Payment status
- Invoice
- Refunds

---

## Exercises

Central exercise library.

Every exercise exists only once.

Stores:

- Name
- Description
- Video
- Images
- Target muscles
- Equipment
- Difficulty

Exercises are reused by multiple packs.

The implemented `exercises` table contains:

- `id` — UUID primary key (generated by the backend).
- `name` — exercise name (`VARCHAR(255)`), required.
- `description` — optional long text description.
- `target_muscles` — optional free text describing the muscles targeted.
- `equipment` — optional free text describing the required equipment.
- `difficulty` — optional free text describing the difficulty level.
- `video_url` — optional URL reference to a demonstration video.
- `image_url` — optional URL reference to a demonstration image.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_name` — MariaDB generated column `IF(deleted_at IS NULL, name,
  NULL)`, unique. The database itself guarantees every **active** exercise
  exists only once; soft-deleting an exercise frees its name for a new catalog
  entry.

The exercise catalog is a **platform-owned, read-only entity** in this
foundation: it is the same for every caller, there is no trainer or admin
ownership on it, and the API exposes no write operation. The repository is
read-only on purpose (`FindByID`, `List`, `Search`) so that no write path can
be invented. The catalog is populated outside the application (seed data /
future admin tooling).

---

## Nutrition

Stores nutrition programs associated with Premium programs.

Premium Level 1:

Generated automatically by AI.

Premium Level 2:

Created manually by trainers.

Future versions may also become AI assisted.

---

## Questionnaires

Stores every questionnaire submitted by users.

Questionnaires are versioned.

Responses should never be lost.

---

## Measurements

Stores body measurements.

Examples:

- Weight
- Body fat
- Waist
- Chest
- Arms
- Legs

Historical evolution is preserved.

---

## Workout History

Stores completed workouts.

Includes:

- Date
- Duration
- Exercises
- Weights
- Repetitions
- Completion

Used for statistics.

---

## Personal Records

Stores automatically detected PRs.

Examples:

- Bench Press
- Squat
- Deadlift

---

## Achievements

Stores:

- Badges
- Medals
- Missions
- Events
- Streaks

---

## Notifications

Stores notifications generated by the system.

Notifications disappear from the interface after 24 hours.

Database records remain available until scheduled cleanup.

Supported categories include:

- Pack
- Purchase
- Badge
- Mission
- Message
- System
- Account
- Administration

---

## Conversations

Stores messaging between:

- Client ↔ Trainer
- Client ↔ Support

Messages use soft deletion.

---

## Feedback

Stores:

- Ratings
- Reviews
- Images
- Likes
- Comments

---

## Trainers

A trainer is the professional profile linked to exactly one user account. The
`trainers` table stores only the trainer identity while the linked `users` row
owns authentication, contact and profile data.

The implemented table contains:

- `id` — UUID primary key (generated by the backend).
- `user_id` — the linked user account (`users.id`), foreign key.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_user_id` — MariaDB generated column `IF(deleted_at IS NULL,
  user_id, NULL)`, unique. It is a **real stored database constraint**, not an
  application rule: the database itself guarantees a user can own at most one
  **active** trainer profile, while historical soft-deleted trainer rows are
  preserved.

One user ↔ at most one active trainer. Deleting a trainer never touches the
linked user account and reactivating a trainer never reactivates a soft-deleted
user: the two lifecycles are independent. The repository provides the queries
used by the admin trainer-management foundation:

- `ListActive` — paginated active trainers (soft-delete scope applied, newest
  first, active total).
- `ListDeleted` — paginated soft-deleted trainers (`deleted_at IS NOT NULL`,
  newest deletion first, deleted total), a clearly separated view from the
  active listing.
- `FindByIDIncludingDeleted` — trainer lookup without the soft-delete scope,
  used by the reactivation lifecycle.
- `Reactivate` — clears `deleted_at` only (trainer UUID, user link and
  `created_at` preserved), guarded by `deleted_at IS NOT NULL` and by the
  active-user-id unique constraint.

Future trainer-specific information (biography, certificates, approval status,
revenue, statistics) extends this profile without touching the user table.

---

## Trainer Applications

A trainer application is a review request from a user who wants to become a
trainer. It is linked to exactly one user account; approving it creates the
trainer profile for the **same** user, so the trainer is always an extension of
an existing user identity and never a separate account.

The implemented `trainer_applications` table contains:

- `id` — UUID primary key (generated by the backend).
- `user_id` — the applicant (`users.id`), foreign key.
- `status` — `PENDING`, `APPROVED` or `REJECTED` (a VARCHAR, validated by the
  application layer).
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker (kept for consistency with the platform
  soft-delete philosophy; applications are never deleted by the API).
- `active_user_id` — MariaDB generated column `IF(deleted_at IS NULL,
  user_id, NULL)`, unique. It is a **real stored database constraint**, not an
  application rule: the database itself guarantees a user can hold at most one
  **active** (PENDING or APPROVED) application, while rejected and historical
  rows are preserved.

Lifecycle rules:

- A user applies → a `PENDING` application row is created.
- An administrator approves → the row becomes `APPROVED` and a trainer row for
  the same `user_id` is inserted in the same transaction (atomic; a duplicate
  trainer rolls the whole approval back).
- An administrator rejects → the row becomes `REJECTED` and no trainer is
  created; the user may apply again (creating a new active application row).
- An approved application is final; an already-reviewed application cannot be
  reviewed again.

The repository provides the queries used by the trainer-application foundation:

- `Create` — creates a PENDING application, enforcing the active-user-id unique
  constraint (duplicate → `ErrApplicationAlreadyActive`).
- `FindActiveByUserID` — finds the user's current PENDING or APPROVED
  application (used by the apply flow).
- `FindByID` — one application with its linked user preloaded.
- `List` — paginated applications, optional status filter, active total.
- `Approve` — transactional PENDING→APPROVED update plus trainer insert.
- `Reject` — PENDING→REJECTED update.

---

## Trainer Clients

A trainer-client relationship links a trainer profile to a user account (the
client). A trainer manages many clients and a user may be a client of many
trainers. The relationship is a first-class entity with its own lifecycle,
stored in the `trainer_clients` table; the client keeps being a regular `User`
— no second identity is ever created.

The implemented table contains:

- `id` — UUID primary key (generated by the backend).
- `trainer_id` — the trainer profile (`trainers.id`), foreign key.
- `user_id` — the client user account (`users.id`), foreign key.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_relation` — MariaDB generated column `IF(deleted_at IS NULL,
  CONCAT(trainer_id, ':', user_id), NULL)`, unique. It is a **real stored
  database constraint**, not an application rule: the database itself
  guarantees a trainer can have at most one **active** relationship with a
  given user, while removed and historical relationships are preserved.

Additional constraints:

- `chk_trainer_clients_not_self` — `CHECK (trainer_id <> user_id)`; a trainer
  can never be their own client (enforced by the database, not just the API).
- Indexes on `trainer_id` and `user_id` cover the main access patterns
  (list a trainer's clients, resolve a relationship).

Lifecycle rules:

- A trainer adds a client → an active relationship row is created. The client
  must exist and be active; a soft-deleted user can never be added.
- A trainer removes a client → only the relationship row is soft-deleted.
  The user account and the trainer profile are never touched: the two
  lifecycles are fully independent.
- A trainer reactivates a client → the exact same previously removed row is
  restored (`deleted_at` cleared, original UUID and timestamps preserved).
  The client must still exist and be active, and an already-active
  relationship can never be reactivated.

The repository provides the queries used by the trainer-client foundation:

- `Create` — creates an active relationship, enforcing the unique active
  relation constraint (duplicate → `ErrClientRelationAlreadyActive`).
- `FindActiveByTrainerAndUser` — the active relationship with its linked user
  preloaded.
- `FindIncludingDeletedByTrainerAndUser` — relationship lookup without the
  soft-delete scope.
- `ListActiveClients` — paginated active clients of one trainer, newest first,
  active total, linked user preloaded.
- `SoftDelete` — clears `deleted_at` only, guarded by `deleted_at IS NULL`
  (an already-removed relationship can never be removed again).
- `Reactivate` — clears `deleted_at` only, guarded by `deleted_at IS NOT NULL`
  and by the active-relation unique constraint.

---

## Programs

A program is a structured, trainer-owned collection of ordered workouts. The
program lifecycle is managed by trainers through the trainer API. Programs are
stored across three tables: `programs` (the program itself), `program_weeks`
(its week slots) and `program_workouts` (the workout slots inside each week).

The implemented `programs` table contains:

- `id` — UUID primary key (generated by the backend).
- `trainer_id` — the owning trainer profile (`trainers.id`), foreign key,
  nullable. A `NULL` owner is reserved for future platform-owned programs and
  can never be produced by the trainer API.
- `name` — program name (`VARCHAR(255)`), required.
- `description` — optional long text description.
- `type` — product category (`free`, `premium` or `personalized`),
  `CHECK`-constrained.
- `status` — `draft` or `published`, `CHECK`-constrained. Publishing is only a
  state and carries no purchase, assignment or access semantics.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.

The implemented `program_weeks` table contains:

- `id` — UUID primary key (generated by the backend).
- `program_id` — the owning program (`programs.id`), foreign key.
- `week_number` — the 1-based position of the week inside its program,
  `CHECK (week_number > 0)`.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_week` — MariaDB generated column `IF(deleted_at IS NULL,
  CONCAT(program_id, ':', week_number), NULL)`, unique. It is a **real stored
  database constraint**, not an application rule: the database itself
  guarantees a program can have at most one **active** week with a given
  `week_number`.

The implemented `program_workouts` table contains:

- `id` — UUID primary key (generated by the backend).
- `program_week_id` — the owning week (`program_weeks.id`), foreign key.
- `position` — the 1-based position of the workout inside its week,
  `CHECK (position > 0)`.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_workout` — MariaDB generated column `IF(deleted_at IS NULL,
  CONCAT(program_week_id, ':', position), NULL)`, unique. The database itself
  guarantees a week can have at most one **active** workout with a given
  `position`.

The implemented `workout_exercises` table (workout exercise foundation)
contains:

- `id` — UUID primary key (generated by the backend).
- `program_workout_id` — the owning workout (`program_workouts.id`), foreign
  key. The workout is the direct owner; the ownership of the program, the week
  and the trainer is derived through the existing tree
  (`programs → program_weeks → program_workouts → workout_exercises`).
- `exercise_id` — the referenced global catalog exercise (`exercises.id`),
  foreign key. There is **no cascade** from this side in either direction: the
  exercise catalog is independent and its soft delete never touches existing
  workout exercises (historical references stay coherent).
- `position` — the 1-based position of the exercise inside its workout,
  `CHECK (position > 0)`.
- `created_at`, `updated_at` — timestamps.
- `deleted_at` — soft-delete marker.
- `active_workout_exercise` — MariaDB generated column `IF(deleted_at IS NULL,
  CONCAT(program_workout_id, ':', position), NULL)`, unique. The database
  itself guarantees a workout can have at most one **active** exercise with a
  given `position`.

The same exercise may appear more than once in the same workout: the
`position` distinguishes each assignment and the unique constraint is scoped to
the pair `(program_workout_id, position)`, not to `(program_workout_id,
exercise_id)`. This is a documented decision: the documentation defines no
training parameters for this foundation, so the assignment entity stays minimal
(`id`, `program_workout_id`, `exercise_id`, `position` and lifecycle fields)
and later steps can add per-assignment training data (sets, reps, rest, tempo,
weights, notes, progression, order) without a schema redesign.

Lifecycle and ordering rules:

- Creating a program week appends it at the end (`week_number` =
  `max(active week_number) + 1`) inside a transaction that locks the owning
  program row, serializing concurrent creation.
- Creating a program workout appends it at the end (`position` =
  `max(active position) + 1`) inside a transaction that locks the owning week
  row.
- Adding a workout exercise appends it at the end (`position` =
  `max(active position) + 1`) inside a transaction that locks the owning
  workout row, so concurrent additions are serialized. The referenced exercise
  must exist and be active in the global catalog: a missing or soft-deleted
  exercise can never be added.
- Reordering replaces the `week_number`/`position` of every active row with the
  index of its id inside the submitted order. The reassignment runs in two
  phases inside one transaction: every active slot is first released to a value
  far outside the final range (`+1000000`), then each row is assigned its final
  slot. This keeps the unique generated column from ever observing a collision
  while never violating the `> 0` checks. The submitted list must match the
  active set exactly; anything else is rejected before any write. Workout
  exercises follow the same two-phase reorder pattern.
- Soft-deleting a week leaves its workouts preserved but unreachable, because
  every workout query is scoped through an active week. Soft-deleting a program
  leaves its weeks and workouts preserved but unreachable through the same
  mechanism. A workout soft delete behaves identically for its workout
  exercises: the rows are preserved but every workout exercise query is scoped
  through an active workout.
- All queries are scoped by the owning trainer; a week or workout that is
  missing, soft-deleted or under a foreign program is indistinguishable from a
  non-existent one. The same holds for workout exercises, which are scoped
  through the full owning chain (`trainer → program → week → workout →
  workout_exercise`).

The repositories provide the queries used by the program structure foundation:

- `ProgramRepository` — `Create`, `FindByIDAndTrainer`, `ListByTrainer`
  (paginated, newest first), `Update` (whitelisted fields only), `SoftDelete`.
- `ProgramWeekRepository` — `Create` (append), `ListByProgram` (weeks in order
  with active workouts preloaded), `FindByIDAndProgram`, `Reorder`,
  `SoftDelete`.
- `ProgramWorkoutRepository` — `Create` (append), `ListByWeek`,
  `FindByIDAndWeek`, `Reorder`, `SoftDelete`.
- `WorkoutExerciseRepository` — `AddExercise` (append, verifies the owning
  chain and an active catalog exercise), `ListByWorkout` (in position order,
  with the safe exercise data preloaded), `FindByIDAndWorkout`, `Reorder`,
  `SoftDelete`.

---

## Audit Logs

Stores every critical administrative action.

Examples:

- User modifications
- Pack changes
- Permission changes
- Trainer approvals

---

## Scheduler

Several automated background jobs interact with the database.

Examples:

- Expired notifications
- Temporary uploads
- Statistics
- Rankings
- Event activation
- Pack expiration
- Email queue
- Backups

---

## RYZE Feed Database

RYZE Feed requires persistent data for trainer content, user likes, and trainer follows.

The database design must remain normalized and follow the general RYZE database principles:

- UUID identifiers
- Normalized relational structure
- Foreign-key relationships
- `created_at` / `updated_at` timestamps where applicable
- Soft deletion where appropriate
- Proper indexing
- Unique constraints for relationships that must not be duplicated

### Feed Content

Represents a piece of content published to RYZE Feed.

Conceptual entity:

```text
feed_content

Fields:

id
trainer_id
title
description
video_url
thumbnail_url
duration
status
published_at
created_at
updated_at
deleted_at
Relationships
Trainer 1 ──────── N Feed Content

Each Feed item must belong to exactly one authorized creator.

trainer_id references the trainer responsible for publishing the content.

Content Status

Feed content should support a lifecycle.

Initial conceptual states:

DRAFT
PROCESSING
PUBLISHED
HIDDEN
DELETED

Only content in the appropriate published state may appear in the public Feed.

The exact implementation of status values may be adjusted during backend implementation, but the database must prevent unpublished content from being treated as publicly available content.

Feed Likes

Represents a user's Like on a Feed item.

Conceptual entity:

feed_likes

Fields:

id
feed_content_id
user_id
created_at

Relationships:

User        1 ──────── N Feed Likes
Feed Content 1 ─────── N Feed Likes

A user may Like a specific Feed item at most once.

Required constraint:

UNIQUE(feed_content_id, user_id)

Removing a Like should remove/deactivate the relationship according to the project's established deletion strategy.

The Like count should be derived from the stored relationships rather than duplicated as an authoritative value unless a future performance optimization explicitly introduces a denormalized counter.

Trainer Follows

Represents a user following a trainer.

Conceptual entity:

trainer_follows

Fields:

id
user_id
trainer_id
created_at

Relationships:

User     1 ──────── N Trainer Follows
Trainer  1 ──────── N Trainer Follows

A user may follow a specific trainer at most once.

Required constraint:

UNIQUE(user_id, trainer_id)

Following a trainer does not grant access to private trainer resources, paid plans, or client-only functionality.

It is primarily a Feed personalization relationship unless another system explicitly defines additional permissions.

Feed Personalization Data

The initial Feed does not require a permanent feed_history or video_views table.

The initial product intentionally does not expose a user-facing viewing history.

Likes and trainer follows are valid persistent personalization signals.

Additional recommendation signals may be introduced later without requiring the initial database schema to store a complete viewing history.

Indexing

The database must provide indexes for the main Feed access patterns.

At minimum, consider indexes covering:

feed_content.status
feed_content.trainer_id
feed_content.published_at
feed_likes.feed_content_id
feed_likes.user_id
trainer_follows.trainer_id
trainer_follows.user_id

Composite indexes should be introduced where they improve the actual query patterns used by the Feed API.

Indexes must be evaluated against real query patterns rather than added indiscriminately.

Data Integrity

Foreign keys must maintain the integrity of relationships between:

Users
Trainers
Feed Content
Feed Likes
Trainer Follows

Deleting or deactivating a trainer or Feed item must not leave invalid relationships.

The implementation must follow the project's established soft-delete and referential-integrity strategy.

Future Extensions

The schema should remain extensible for future Feed functionality such as:

Content categories
Tags
Content moderation records
Content reports
Trainer analytics
Video analytics
Featured content
Recommendation metadata
Content visibility controls

These should be introduced as separate entities when appropriate rather than adding unrelated fields to feed_content.

---

# FUTURE EXPANSION

The database should always be designed to support future features without requiring major structural changes.

Possible future features include:

- Marketplace expansion
- Native mobile applications
- AI-generated training programs
- AI-generated nutrition
- Additional payment providers
- Public API
- Multi-language support
- International taxation
- Analytics improvements

Whenever possible,
new features should extend the existing architecture rather than replacing it.

---

The database architecture is expected to evolve throughout the project's lifetime.

AI assistants are encouraged to improve the design whenever a better, more secure or more scalable solution is identified.

Every architectural modification must be documented inside update_logs.MD before implementation.
