# AGENTS.md

The complete project documentation is located inside the `docs` directory.

Always read the relevant documentation before making any implementation decision.

# RYZE

This file contains the mandatory instructions for Claude/other AIs when working on the RYZE project.

These instructions override default behavior whenever possible.

---

# Core Principles

Always think before writing code.

Never make assumptions.

Always prefer consistency over creativity.

Never invent business logic.

Never contradict existing documentation.

The documentation is considered the project's single source of truth.

---

# Read Documentation First

Before implementing anything, always read the relevant documentation.

Priority order:

1. docs/INSTRUCTIONS/project.MD
2. docs/INSTRUCTIONS/arquitecture.MD (the repository's actual filename)
3. docs/DATABASE/database.MD
4. docs/INSTRUCTIONS/design.MD
5. docs/INSTRUCTIONS/procedure.MD
6. docs/REGISTRY/update_logs.MD

Only implement features after understanding the project.

---

# Documentation Rules

Documentation always has priority over assumptions.

If documentation and code conflict, follow the documentation.

If documentation is missing, ask the user.

Never guess.

Never create undocumented business rules.

Whenever functionality changes, update the appropriate documentation.

Never duplicate documentation.

---

# Development Philosophy

RYZE prioritizes:

- Performance
- Maintainability
- Scalability
- Readability
- Security
- Simplicity

Every implementation should follow these principles.

The project structure is designed to evolve.

AI assistants are allowed to create new folders and files whenever they improve
organization, maintainability or scalability.

Every new folder or file must:

- Have a clear and descriptive name.
- Follow the existing project structure.
- Avoid unnecessary nesting.
- Avoid duplicate responsibilities.
- Respect the project's architecture.

Do not create new folders simply to separate small amounts of code.

When in doubt, prefer extending the existing structure.

---

# Code Style

Always write code in English.

Use descriptive names.

Prefer small functions.

Keep functions focused on a single responsibility.

Avoid deeply nested code.

Avoid duplicated logic.

Never leave unused code.

Never leave debug code.

Never leave unused imports.

Never leave TODOs unless requested.

Comments should only explain *why*, never *what*.

---

# Project Structure

Never change the project structure without explicit approval.

Before creating a new file:

- verify whether a similar file already exists;
- verify whether the logic belongs inside an existing module.

Before creating reusable logic:

always check:

src/utils/

Reusable code must never be duplicated.

---

# Architecture

Never introduce:

- new frameworks
- new libraries
- new technologies

unless explicitly approved.

Always follow docs/INSTRUCTIONS/arquitecture.MD.

---

# Frontend

Always follow design.MD.

Every page must:

- be responsive;
- follow the design language;
- use reusable components;
- maintain accessibility;
- maintain consistency.

When a frontend redesign is explicitly approved, new UI patterns are allowed
in the approved scope; they must still follow design.MD, preserve business
flows, and avoid the protected areas defined in the Claude environment guide.

## Frontend revamp constraints

The owner has approved a major redesign, beginning mobile-first at 360–430px.
The official RYZE logo and documented brand colors remain protected foundations;
the layout, typography and component system may change around them. Static
background images are forbidden outside login and registration, while purposeful
functional imagery remains allowed. Three.js is functional only and must not be
used for decorative backgrounds.

The entire footer, Exercise Library/details, Services page, and Home's
“Find Your Next Program” section are strictly protected. Do not alter them or
their shared styling indirectly. Read `docs/AI/frontend-revamp.md` before any
frontend work and use isolated variants when a shared component is also used by
a protected area.

---

# Backend

Business logic belongs inside the backend.

Never place business logic inside frontend components.

Keep controllers thin.

Keep services reusable.

---

# Database

Follow docs/INSTRUCTIONS/arquitecture.MD.

Database structures are expected to evolve during development.

The AI is encouraged to improve the database schema whenever a better design is identified.

Prioritize:

- scalability
- maintainability
- data integrity
- future extensibility

Avoid unnecessary migrations, but do not preserve poor schema designs simply because they were implemented first.
Avoid duplicated queries. Whenever calling data, use VIEWs to ensure safety and not revealing any names of tabels.

# Database Philosophy

Prefer soft deletes over permanent deletion.

Whenever possible, accounts and important records should be recoverable.

Examples include:

- is_active
- deleted_at

instead of permanently removing rows.

Always consider future features when designing tables.

If adding common fields now can avoid future migrations without introducing unnecessary complexity, they should be included from the beginning.

# Architecture Evolution

The project architecture is not considered fixed.

Claude is encouraged to propose and implement architectural improvements whenever they provide clear benefits.

Existing implementations should never prevent better long-term solutions.

However:

- preserve existing data whenever possible;
- avoid unnecessary breaking changes;
- document important architectural modifications.

---

# Performance

Prefer efficient solutions.

Avoid unnecessary renders.

Avoid unnecessary allocations.

Avoid duplicated API requests.

Reuse existing code whenever possible.

# Long-Term Thinking

Whenever implementing a feature, think about future scalability.

Design APIs, database tables and reusable components so they can support future functionality without major refactoring.

Avoid over-engineering, but also avoid designs that only solve today's problem.

# Decision Making

If multiple valid implementations exist:

1. Read the documentation.
2. Reuse existing project patterns.
3. Choose the solution that is:
   - simpler;
   - more maintainable;
   - more scalable;
   - more secure.

Do not choose randomly.

If documentation does not define the behavior, ask the user.

---

## RYZE Feed Development Rules

RYZE Feed is a core RYZE feature for short-form fitness and educational video content created by authorized trainers.

Before making any changes to RYZE Feed, agents MUST read the relevant documentation, especially:

- `project.MD`
- `design.MD`
- `arquitecture.MD`
- `DB.md`
- API/backend documentation
- `security.MD`
- `procedure.MD`

### Feed Scope

RYZE Feed currently provides:

- Vertical short-form video consumption
- Trainer-created content
- Trainer profiles
- Trainer following
- Video Likes
- Personalized Feed ordering
- Trainer content management

RYZE Feed does NOT currently include:

- Comments
- Direct messaging
- User-created Feed posts
- User-facing watch history
- General-purpose social networking functionality

Do not introduce these features unless explicitly requested.

### Architecture

Do not create a separate authentication, authorization, database, upload, or media-processing system for RYZE Feed when an existing RYZE system can be reused.

Prefer extending existing architecture over creating parallel systems.

Feed ranking/personalization must remain modular and replaceable.

Do not hard-code the entire Feed architecture around a single recommendation algorithm.

### Database

All Feed database changes must follow `DB.md`.

Use:

- Existing UUID conventions
- Existing timestamp conventions
- Existing soft-delete conventions
- Existing migration system
- Proper foreign keys
- Appropriate indexes
- Unique constraints for relationship entities

Never manually modify the database schema outside the established migration workflow.

### Authorization

Trainer content ownership must always be verified server-side.

Never trust:

- Client-provided user IDs
- Client-provided trainer IDs
- Client-provided ownership
- Client-provided roles
- Frontend permission checks

A trainer may only modify their own Feed content unless an authorized administrative role is explicitly permitted to do so.

### Media

Treat all uploaded videos as untrusted input.

Follow the requirements in `security.MD` for:

- File validation
- Media processing
- Storage
- Resource limits
- Malicious media protection
- Upload authorization

Do not expose raw storage paths or rely on user-provided filenames.

### Frontend

Follow `design.MD` for the Feed interface.

The intended interaction is simple:

```text
Open Feed
    ↓
Watch video
    ↓
Like / Follow
    ↓
Swipe or scroll
    ↓
Next video

```

Keep the interface minimal and content-focused.

Do not add unnecessary:

Cards
Dashboards
Comments
Filters
Social features
Complex menus

without an explicit product/design decision.

Personalization

Trainer follows and user interactions may influence Feed ranking.

The exact recommendation algorithm is intentionally not fixed.

Do not introduce permanent algorithmic assumptions into the database or frontend unless explicitly documented.

Documentation

If a Feed implementation changes:

Database structure
API behavior
Security requirements
UI behavior
Architecture
Product behavior

the relevant documentation must be updated in the same development cycle.

Do not intentionally leave documentation inconsistent with the implementation.

Before Large Changes

For changes that affect multiple architectural areas:

Read the relevant documentation.
Inspect the existing implementation.
Identify reusable systems.
Determine the affected documentation.
Make the smallest coherent change.
Test the affected functionality.
Update documentation if behavior or architecture changed.

When documentation and implementation conflict, do not blindly overwrite either one. Identify the conflict and resolve it according to the project's established development procedure.

---

# Security

Never expose secrets.

Validate every input.

Never trust client-side validation.

Follow secure defaults.

---

# Documentation Updates

Whenever an important change is completed:

Update:

docs/REGISTRY/update_logs.MD

Include:

- version
- modified files
- summary
- reason

Newest logs always go first.

---

# Response Style

Do not immediately write code.

First:

- understand the request;
- inspect the existing implementation;
- determine whether reusable code already exists.

Then implement.

When finished:

Explain:

- what was changed;
- why it was changed;
- whether any architectural decisions were made.

Keep explanations concise.

---

# RYZE Philosophy

Every decision should support the project's main objective:

> Make professional fitness guidance accessible, automated and simple.

If multiple solutions are possible, choose the one that best aligns with this philosophy.


---

# RYZE Premium Level 1 Development Rules

Premium Level 1 is a core RYZE product: a complete training and nutrition package
sold once. Nutrition is delivered in one of two server-owned modes: a predefined
curated plan authored with the package, or the legacy questionnaire-generated plan
derived from a client intake.

Before making any changes, agents MUST read:

- `project.MD`
- `arquitecture.MD`
- `database.MD`
- `security.MD`
- `api.MD`
- `procedure.MD` (section "PROCEDURE: ADDING A NEW PREMIUM TIER")
- `design.MD` (section "Premium Level 1 Page")

## Scope

Premium Level 1 DOES include:

- Platform-owned Premium Level 1 programs
- Curated nutrition plans authored and validated with the package
- A server-declared nutrition questionnaire catalog for legacy packages
- A submitted intake per client and legacy program, editable until purchase
- A server-owned nutrition assignment with a visible lifecycle in the legacy mode
- Deterministic server-owned plan generation for the legacy mode, with
  normalised, versioned persistence, delivered as part of a completed purchase
  and retryable in place on failure
- Purchase prerequisite enforcement for the legacy mode only
- Training and nutrition delivered through the existing Program Access page

Premium Level 1 does NOT include:

- Premium Level 2 implemented as a separate trainer-delivered family (v2.103.0)
- Trainer-authored nutrition
- Human coaching or trainer review of a questionnaire
- An intake for a curated package
- A client-facing regeneration or replacement control for a delivered plan (in the
  legacy mode the server may supersede a delivered plan only when the intake
  revision has moved on)
- Any new payment method, provider, currency or refund behaviour

Do not introduce these unless explicitly requested.

## Product families

`programs.type` is the commercial pricing category (`free`, `premium`,
`personalized`) and is not the product family.

`programs.product_type` is the product family (`generic`, `premium_level_1`). It
decides which parts of the purchase lifecycle a program participates in.

For Premium Level 1, `programs.nutrition_mode` chooses the delivery model:
`curated` ships a package-owned plan and `questionnaire_generated` preserves the
legacy intake-first flow. `programs.objective` is the closed goal vocabulary for a
curated package. Both are fixed at creation; package content is locked once a
completed purchase exists.

When adding or changing a family or mode, every condition must be updated
together: the public catalog scope, the platform administration scope, the
purchase prerequisite check, the curated/legacy nutrition read and the client
rendering. Missing one leaves a tier reachable through a surface where its
prerequisites are not enforced.

## Questionnaire (legacy questionnaire-generated packages only)

- A curated package has no intake. Its requirement answers `required: false`,
  `submitted: false`, `locked: false`, and a submit attempt returns
  `404 PROGRAM_NOT_FOUND` without writing anything.
- The intake belongs to the authenticated client. The acting user id always comes
  from the authentication context.
- Answers are sensitive: never logged, never returned by any endpoint, never
  projected into public, trainer or commerce responses. The read view does not
  project the answers column at all, so preserve that.
- Validation and normalization are server-side and mandatory. Raw client input is
  never persisted.
- The intake is immutable once an active entitlement exists. The lock is derived
  from the entitlement, not stored, and the entitlement lookup fails closed.
- There is no reopen path for the intake and no client-facing regeneration
  control. A supersession of a delivered plan is server-triggered and happens
  only when the intake revision has moved on; do not add one outside that.

## Nutrition

- Curated plans are package-owned and fixed. Every buyer of the package receives
  the same content, read only after the entitlement check. There is no intake, no
  generation and no per-client plan row.
- Legacy generated plans are server-owned and deterministic. No client-supplied
  plan content, no client-supplied targets, no client-facing regeneration. A
  delivered plan is immutable and versioned; only the server generate endpoint may
  supersede it, and only when the intake revision has moved on.
- `completed` is the only state that may be rendered as a plan. A purchase
  attempts legacy plan delivery but never implies a finished plan: a delivery
  failure must never undo the payment and leaves the assignment retryable in
  place. A curated plan is synthesized as `completed` from the stored package
  plan.
- Legacy generation is idempotent and a failure is retryable in place.
- `failure_reason` is internal and never reaches the client.

## Authorization

Every legacy nutrition and questionnaire condition must hold: authenticated owner,
active entitlement, correct `product_type`, submitted intake. A curated nutrition
read requires authenticated owner, active entitlement, correct `product_type` and
a stored package plan, and no intake. Any missing condition answers
`404 NUTRITION_NOT_FOUND`, identical to "no record", because a distinguishable
refusal is a probe for other clients' health data.

## Frontend

- Reuse `useProgramCheckout`, the shared checkout hook. Do not fork the purchase,
  payment or Test Mode flow per tier.
- Read product metadata from the API. Never hardcode a product id, price or
  production offer copy.
- Read the family and nutrition mode from the response; never compare a hardcoded
  id. A curated package is identified by a known objective, with missing/unknown
  objectives falling back to the legacy flow for compatibility.
- The Premium route must stay declared before `/services/:serviceSlug`.
- Map server reason codes to client-owned wording. Never display a raw token.
- A curated package skips the questionnaire and goes directly to checkout.
- The legacy questionnaire form is presentational: the parent owns loading,
  submission and the locked state, and the component never decides that editing is
  allowed.
- Reflect content locks in ADMIN_2: a package with a completed purchase cannot be
  edited, unpublished or deleted.

## Database

Follow `database.MD`. Extend the existing schema through the migration system.
Never modify the schema outside it, and never add a family value the product does
not implement.

## Documentation

If an implementation changes database structure, API behaviour, security
requirements, UI behaviour, architecture or product behaviour, update the relevant
documentation in the same development cycle.

---

# RYZE Premium Level 2 Development Rules

Premium Level 2 is a trainer-delivered service: the client buys a platform-owned
offer, submits a versioned intake, ADMIN_2 allocates an active trainer, and the
trainer authors the training programme and nutrition before delivery.

Before making any changes, agents MUST read `api.MD`, `database.MD`,
`security.MD`, `arquitecture.MD`, `design.MD`, `procedure.MD` and
`update_logs.MD` for the Premium Level 2 sections.

## Scope

Premium Level 2 DOES include:

- Platform-owned `premium_level_2` service offers with no training weeks
- Purchase-first, versioned and resumable intake
- ADMIN_2 trainer allocation and reassignment
- Trainer-authored draft programme attachment and per-request nutrition
- Validated delivery that creates the training entitlement
- Client request status, delivered nutrition and Program Access routing

Premium Level 2 does NOT include:

- A second checkout, payment provider or entitlement model
- Trainer-authored nutrition outside this service
- Client-facing plan authoring or regeneration
- Comments, messaging or Feed integration

## Invariants

- The intake requires an active entitlement and is editable only before trainer
  allocation. Never accept a client-supplied owner or trainer id.
- Intake answers are sensitive: never log them, never put them in queue
  responses, and expose them only to the owner, the allocated trainer and
  ADMIN_2.
- Re-check trainer allocation on every trainer read and write. Reassignment
  clears the previously attached programme.
- A trainer may attach only their own draft programme containing at least one
  exercise. Delivered content is immutable.
- Delivery requires `ready_for_delivery`, creates the entitlement idempotently
  and never unwinds a payment on failure.
- Read the product family and nutrition mode from the API; never hardcode an id.
- Update the relevant documentation in the same development cycle.
