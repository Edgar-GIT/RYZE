# RYZE FITNESS
# CONVENTIONS.MD

Version: 2.81.0

---

# PURPOSE

This document defines the coding standards, naming conventions and development guidelines used throughout the RYZE project.

Every contributor, whether human or AI, must follow these conventions to ensure the project remains consistent, readable and maintainable.

Whenever a convention needs to evolve, this document should be updated before the codebase.

---

# GENERAL PRINCIPLES

The project prioritizes:

- Readability
- Maintainability
- Simplicity
- Scalability
- Performance
- Security
- Consistency

Readable code is always preferred over clever code.

The objective is to create software that can be easily understood months or years later.

---

# LANGUAGE

Everything inside the project must be written in English.

This includes:

- Variables
- Functions
- Classes
- Components
- Database
- API
- Documentation
- Commit messages
- Logs
- Error messages

The only exceptions are user-facing translations managed through the internationalization system.

---

# NAMING CONVENTIONS

## Variables

Use camelCase.

Good examples:

```ts
currentUser
trainingProgram
profilePicture
totalCalories
```

Avoid abbreviations whenever possible.

Bad examples:

```ts
usr
tp
img
```

---

## Functions

Use camelCase.

Function names should begin with a verb.

Examples:

```ts
createUser()
updateProfile()
deleteNotification()
generatePdf()
sendEmail()
calculateCalories()
```

Never use generic names.

Avoid:

```ts
doStuff()
process()
handle()
run()
```

---

## React Components

Use PascalCase.

Examples:

```text
Navbar
ProfileCard
WorkoutChart
NotificationPanel
ExerciseCard
```

Each component should represent one responsibility.

---

## Interfaces

Use PascalCase.

Examples:

```ts
User
Workout
Notification
TrainerProfile
```

Avoid prefixes like:

```text
IUser
IWorkout
```

---

## Types

Use PascalCase.

Examples:

```ts
UserRole
WorkoutStatus
NotificationType
```

---

## Enums

Use PascalCase.

Enum values use UPPER_SNAKE_CASE.

Example:

```ts
enum UserRole {
    CLIENT,
    TRAINER,
    TECH_ADMIN,
    MANAGEMENT_ADMIN
}
```

---

## Constants

Use UPPER_SNAKE_CASE.

Examples:

```ts
MAX_UPLOAD_SIZE

DEFAULT_LANGUAGE

TOKEN_EXPIRATION

MAX_PROFILE_IMAGES
```

---

## Files

Use snake_case.

Examples:

```text
profile_card.tsx

user_service.go

notification_service.go

create_pack_modal.tsx
```

Avoid spaces.

Avoid capital letters.

---

## Folders

Use snake_case.

Examples:

```text
user_management

trainer_dashboard

email_templates

api_routes
```

Folders should describe their responsibility.

---

# REACT CONVENTIONS

One component per file.

Avoid components larger than approximately 300 lines.

Split large components into smaller reusable components.

Prefer composition over deeply nested components.

Business logic should remain outside UI components whenever possible.

Pages should orchestrate components, not contain large amounts of logic.

---

# CSS CONVENTIONS

RYZE uses CSS Modules.

SCSS is not used unless explicitly introduced in the future.

Global CSS should remain minimal.

Reusable styling belongs inside reusable components.

Use CSS variables whenever possible.

Examples:

- Colors
- Border radius
- Shadows
- Spacing
- Animation duration
- Font sizes

Avoid hardcoded values repeated throughout the project.

---

# TYPESCRIPT

Prefer explicit typing.

Avoid using `any`.

Use interfaces and types where appropriate.

Prefer immutable data whenever possible.

Use optional chaining when appropriate.

Keep functions pure whenever possible.

---

# GO CONVENTIONS

Packages use lowercase.

Files use snake_case.

Handlers should remain small.

Business logic belongs inside services.

Routes should never communicate directly with the database.

Database logic belongs inside repositories or dedicated database services.

Prefer dependency injection over global state.

Return meaningful errors.

---

# DATABASE CONVENTIONS

Every entity uses UUID identifiers.

Never use sequential IDs.

Use snake_case.

Every table should contain timestamps whenever appropriate.

Prefer Soft Delete instead of permanent deletion.

Never duplicate data that can be referenced.

Create indexes whenever performance justifies them.

Never use SELECT * in production queries.

---

# API CONVENTIONS

REST architecture.

JSON only.

Versioned endpoints.

Example:

```
/api/v1/
```

HTTP methods should follow REST conventions.

GET

POST

PUT

PATCH

DELETE

Responses should remain consistent across the API.

---

# ERROR HANDLING

Errors should always be descriptive.

Never expose internal implementation details.

Never expose stack traces.

Sensitive information must never reach the client.

Errors should always be logged.

---

# LOGGING

Logs should contain meaningful information.

Avoid excessive logging.

Sensitive data must never appear inside logs.

Examples of forbidden information:

- Passwords
- Tokens
- Secrets
- API Keys

---

# COMMENTS

Avoid unnecessary comments.

Good comments explain:

- Business rules
- Security decisions
- Complex algorithms
- Architectural reasoning

Never explain obvious code.

Bad example:

```go
// Increment i
i++
```

Good example:

```go
// Soft delete is used to preserve historical purchases.
```

---

# REUSABLE CODE

Before creating new code, verify whether similar functionality already exists.

Shared logic belongs inside:

```
src/utils
```

Never duplicate reusable functionality.

Examples:

- Validators
- Formatters
- Unit conversions
- Date helpers
- String helpers
- Math helpers
- API helpers

---

# IMPORTS

Remove unused imports.

Keep imports organized.

Prefer absolute imports whenever configured.

Avoid circular dependencies.

---

# FILE ORGANIZATION

Each file should have one clear responsibility.

Avoid mixing unrelated logic.

Large features should be divided into multiple files.

---

# COMPONENT DESIGN

Components should be:

- Small
- Reusable
- Predictable
- Independent

Avoid creating components tightly coupled to a single page unless necessary.

---

# PERFORMANCE

Optimize only when justified.

Avoid premature optimization.

Prefer:

- Lazy loading
- Memoization
- Efficient rendering
- Proper indexing
- Pagination
- Code splitting

---

# ACCESSIBILITY

Accessibility is mandatory.

Always provide:

- Labels
- Keyboard navigation
- Focus states
- ARIA attributes where necessary
- Sufficient color contrast

---

# SECURITY

Never trust client input.

Always validate data on the backend.

Sanitize user input whenever required.

Use parameterized SQL queries.

Never expose secrets.

Passwords must always be securely hashed.

---

# GIT CONVENTIONS

Commit messages should be descriptive.

Preferred prefixes:

```
feat:
fix:
docs:
refactor:
style:
perf:
test:
build:
chore:
```

Examples:

```
feat: implement trainer dashboard

fix: correct JWT expiration

docs: update architecture documentation

refactor: simplify notification service
```

---

# VERSIONING

The project follows Semantic Versioning.

PATCH

Bug fixes.

MINOR

New features without breaking compatibility.

MAJOR

Breaking changes.

Documentation versions should remain synchronized whenever appropriate.

---

# DEVELOPMENT PHILOSOPHY

The simplest correct solution is usually the best solution.

Avoid unnecessary abstraction.

Prefer explicit code.

Prefer modularity.

Prefer maintainability over writing fewer lines.

Every implementation should make the next implementation easier.

Consistency is more valuable than personal coding preferences.

Whenever in doubt, choose the solution that another developer can understand immediately.

The codebase should always feel as if it was written by a single experienced engineer.

---

# FRONTEND CONVENTIONS (v2.81.0)

## Pages own their directory

Every page lives in its own snake_case directory named after the page, and imports
its own styles as a CSS module:

```
pages/<page_name>/<page_name>.tsx
pages/<page_name>/<page_name>.module.css
```

Components specific to one page live in that same directory and import their own
module. There are no index barrels: each import is the explicit path, so a file
move is a compile error rather than a silent resolution change.

## Shared components stay in components/

A component used by more than one page belongs in `components/<component_name>/`.
A component used by exactly one page stays in that page's directory. Moving a
component into `components/` before it has a second consumer creates a shared API
nobody needs yet.

## Page containers own orchestration

The page container owns state, loading, error handling and step sequencing. Its
presentational components receive plain props and return markup; they never fetch,
never decide whether the user may act, and never derive authorization state. This
is what lets the questionnaire form stay a pure component while the server stays
the only authority on whether the intake may be edited.

## Shared flows become hooks

When two pages implement the same state machine, it is extracted into a hook under
`hooks/`. The Premium Level 1 page and the Generic program detail page share
`useProgramCheckout`, which owns ownership resolution, purchase creation, payment
selection, provider redirect and return, capture, cancellation, duplicate-purchase
recovery, webhook race recovery and the Test Mode shortcut.

Extraction must not change behaviour for the page it came from. When a hook is
extracted, the original page keeps its semantics exactly; only the location of the
logic changes.

## Options are typed states

A multi-state flow is modelled as a discriminated union, not as a set of booleans.
`PurchaseState` is `{ status: "checking" | "guest" | "owned" | "ready" | ... }`, so
an impossible combination such as "loading and error at the same time" cannot be
represented, and each state carries only the fields that state actually has.

Where a single value must be narrowed for a render, it is derived once into a
named local rather than re-narrowed inline inside JSX.

## API modules are per-domain

Each domain owns a service module under `services/`, exporting typed contracts and
fetch functions. Components never build a URL or read `response.data` directly.
Types mirror the snake_case wire format exactly and are not renamed at the
boundary, so a contract change is visible in one file.

## Server codes are mapped to client wording

The backend returns short, stable reason codes; the client owns presentation
copy. A `Record<string, string>` map turns `"out of range"` into "This value is
outside the accepted range.", and an unknown code still renders a readable
sentence. Health-related fields must never display a raw server token.

## Attributes are forwarded only where they are valid

A server-declared constraint such as `max_length` is applied to the control it
belongs to. Spreading a shared attribute bag onto a `<select>` or a checkbox group
produces invalid markup, so each control declares its own attributes and only
adds the shared `aria-describedby` reference.

## Product metadata comes from the backend

Price, duration, level, frequency, family and description are always read from an
API response. A page never hardcodes a product id, price or product description,
and never hardcodes production copy that describes a specific offer. The product
family is a `product_type` value, not a hardcoded identifier compared in the
client.

---

# BACKEND CONVENTIONS (v2.81.0)

## Conditions that define a domain go in one helper

When several queries must agree on what a domain object is, the condition is
expressed once as a helper rather than repeated as a string literal. Generic
program management uses `genericScope` and `genericListScope`, and the public
catalog uses the `ScopeGeneric` and `ScopePremiumLevel1` filters.

A shared predicate string with a `?` placeholder is avoided on purpose. A
placeholder-count mismatch does not fail loudly at the call site: GORM's `First`
adds its own `LIMIT` placeholder, so a mismatched fragment binds an argument to
the wrong column and still runs.

## Product families are discriminators, not parallel systems

A new product family extends an existing constrained column. It does not get its
own table, its own purchase flow or its own entitlement model. When two families
must be separated, they are separated by two independent conditions rather than
one overloaded value.

## Failure modes are explicit and named

Every non-success outcome is a named error in the service layer and a documented
status and code at the handler. Services never return a partially applied result,
and a refusal that must be indistinguishable from an absence is a single named
error rather than a check repeated at each call site.

## Security refusals fail closed

When a decision depends on a lookup that can fail, an error is treated as the
restrictive outcome. The questionnaire lock treats an unreadable entitlement as
"locked". The opposite default would let a transient database failure reopen
editing of data a client already purchased against.
