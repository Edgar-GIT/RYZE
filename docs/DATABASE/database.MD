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

`session_version` is incremented whenever the account password changes or the
account is soft-deleted (by the account holder or by an administrator). Access
tokens embed the session version they were issued for, so such operations
revoke every previously issued token (see security.MD, Session Revocation).

The repository provides a paginated active-user query (`ListActive`) used by
the admin user-management foundation. It applies the soft-delete scope, orders
newest first and returns the active total; it reuses the same atomic soft-delete
primitive (`deleted_at` + `session_version` in one statement) used by account
deletion.

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

Stores trainer-specific information.

Examples:

- Biography
- Certificates
- Approval status
- Revenue
- Statistics

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
