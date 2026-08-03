# RYZE FITNESS
# CONVENTIONS.MD

Version: 1.0.0

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
trainingPlan
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
