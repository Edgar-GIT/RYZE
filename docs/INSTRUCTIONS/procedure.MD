# RYZE FITNESS
# PROCEDURE.MD

Version: 2.0.0

---

# PURPOSE

This document defines the mandatory workflow for every AI assistant contributing to the RYZE project, including (but not limited to):

- ChatGPT
- Claude
- Gemini
- Codex
- Cursor
- Copilot
- OpenCode
- Aider
- Any future AI development assistant

Its purpose is to ensure that every contribution follows the same standards, architecture, documentation practices and development philosophy.

These rules are mandatory.

They are not suggestions.

Every implementation must respect this document before, during and after development.

---

# DOCUMENTATION FIRST

RYZE follows a **Documentation First** development workflow.

Documentation defines the project.

Code implements the documentation.

The documentation is considered the project's primary source of truth.

Whenever documentation and code disagree, the documentation takes priority until the project owner explicitly approves a change.

Every important architectural, functional or business decision must exist in the documentation before becoming part of the implementation.

No feature should exist only inside the source code.

---

# DOCUMENTATION PRIORITY

Every task must consider both the implementation and its documentation.

A task is **not finished** when only the code works.

A task is only considered complete when:

- The implementation is complete.
- The architecture remains consistent.
- The documentation reflects the implementation.
- The update log has been updated whenever required.

Whenever new functionality is introduced, the AI must determine whether any documentation requires updating.

Examples include:

- New features
- New APIs
- Database modifications
- UI changes
- New business rules
- New AI behavior
- Folder restructuring
- New technologies
- New conventions

If documentation requires updating, it must be updated before considering the task complete.

---

# DOCUMENT READING ORDER

Before starting any task, the AI should read the documentation in the following order.

Only the documents relevant to the requested task are required, but this order should always be respected.

1. procedure.MD
2. project.MD
3. architecture.MD
4. conventions.MD
5. database.MD *(backend/database related tasks)*
6. api.MD *(backend/API related tasks)*
7. design.MD *(frontend/UI related tasks — prefer*
   `docs/INSTRUCTIONS/design.MD` for palette/tokens, then
   `docs/PAGES/design.MD` for page composition)*
8. ai.MD *(AI-related functionality)*
9. update_logs.MD *(to understand recent changes)*

Never begin implementing code without first understanding the relevant documentation.

---

# PROJECT STRUCTURE

The project's documentation is located inside the `docs/` directory.

Each document has a single responsibility.

Information should never be duplicated across multiple documents.

Whenever something changes, update the document responsible for that subject instead of repeating information elsewhere.

Every document contributes to a single, consistent source of truth.

---

# DOCUMENTATION

## docs/INSTRUCTIONS

This directory contains the project's core technical documentation.

It defines how the platform is built and how it should evolve.

---

### architecture.MD

Defines the complete technical architecture of the project.

Examples include:

- Programming languages
- Frameworks
- Libraries
- Folder structure
- Design patterns
- Backend architecture
- Frontend architecture
- Infrastructure
- Security decisions
- Performance guidelines
- Naming conventions
- Project organization

The architecture is expected to evolve throughout development.

AI assistants may propose improvements whenever they increase:

- Maintainability
- Performance
- Security
- Scalability
- Simplicity

Architectural modifications should always be documented before implementation.

---

### project.MD

Defines the product itself.

Contains information such as:

- Vision
- Mission
- Goals
- Features
- Product tiers
- Business rules
- Functional requirements
- Non-functional requirements
- User experience
- General platform behavior

Whenever the product changes, this document must also be updated.

---

### design.MD

Defines the complete visual identity of the platform.

Includes topics such as:

- Color palette (Hero section is the source of truth)
- Typography
- Layout
- Components
- UI patterns
- UX decisions
- Animations
- Loading screens
- Responsive behavior
- Accessibility

Whenever UI or visual identity changes, update:

- `docs/INSTRUCTIONS/design.MD` (palette + token system)
- `docs/PAGES/design.MD` (page composition + component behaviour)
- `main/src/frontend/src/styles/global.css` (runtime tokens)

Visual consistency is mandatory: the Hero palette is the only official palette.

Every frontend implementation must remain consistent with these documents.

---

### procedure.MD

This document.

Defines the mandatory workflow that every AI assistant must follow while contributing to the repository.

It explains how decisions are made, how documentation should be maintained and how development should be performed.

---

## docs/DATABASE

### database.MD

Defines the complete database architecture.

Includes:

- Entities
- Relationships
- Naming conventions
- Persistence rules
- Versioning strategy
- Soft deletion policy
- Auditability
- Historical data
- Future expansion

The database is expected to evolve during development.

AI assistants are encouraged to improve the schema whenever a better, safer or more scalable solution is identified.

Database changes must preserve existing data whenever possible and be documented before implementation.

---

## docs/INSTRUCTIONS

### api.MD

Defines the complete REST API specification.

Includes:

- Endpoints
- Authentication
- Authorization
- Request formats
- Response formats
- Validation
- Error handling
- WebSocket events
- Versioning

The API should evolve together with the platform.

Whenever endpoints are added, modified or deprecated, this document must be updated accordingly.

---

## docs/INSTRUCTIONS

### security.MD

Defines every security standard that must be followed throughout the project.

This document covers:

- Authentication
- Authorization
- API security
- Database security
- File upload security
- Image validation
- Malware protection
- SQL Injection prevention
- XSS prevention
- CSRF protection
- SSRF protection
- DDoS mitigation
- Rate limiting
- Logging
- Audit trails
- Encryption
- Backups
- Privacy
- GDPR considerations
- Secure development practices

Security is a fundamental requirement of the project.

Whenever implementing any feature, AI assistants must verify that it complies with the rules defined in `security.MD`.

If a better or more secure solution is identified during development, the AI may improve the architecture, provided the change is documented in both `security.MD` and `update_logs.MD`.

No implementation should intentionally reduce the security level of the platform unless explicitly requested by the project owner.

---

## docs/INSTRUCTIONS

### ai.MD

Defines every AI-related feature implemented by the platform.

Examples include:

- AI-generated nutrition programs
- AI-generated workout programs
- Prompt engineering
- Model integrations
- AI workflows
- Context management
- Safety rules
- Future AI improvements

Whenever AI functionality changes, this document must remain synchronized.

---

## docs/INSTRUCTIONS

### conventions.MD

Defines the project's coding standards.

Examples include:

- Naming conventions
- File organization
- Folder organization
- Code style
- Formatting
- Error handling
- Logging
- Validation
- Git conventions
- General best practices

Every implementation should follow these conventions.

Consistency always takes priority over personal preference.

---

## docs/REGISTRY

### update_logs.MD

Records every important modification made to the project.

Every significant change must generate a new log.

Each log should contain:

- Version
- Title
- Date
- Summary
- Files modified
- Changes made
- Reason
- Additional notes

Never delete previous logs.

Always append new logs to the top of the document.

The newest log should always appear first.

# DEVELOPMENT WORKFLOW

Every implementation should follow the same development workflow.

Skipping steps is strongly discouraged.

The objective is to maintain consistency, quality and long-term maintainability.

The recommended workflow is:

1. Understand the task.
2. Read the relevant documentation.
3. Analyze the existing implementation.
4. Design the solution.
5. Implement the solution.
6. Validate the implementation.
7. Update the documentation.
8. Create an update log if necessary.

Only then should the task be considered complete.

---

# BEFORE WRITING CODE

Before writing or modifying any code, the AI must:

Always:

Read project.MD

Read architecture.MD

Read database.MD (if database related)

Read api.MD (if backend/API related)

Read security.MD

Read design.MD (if UI related)

Read update_logs.MD (to understand recent changes)

Only then start implementing.

- Understand the requested feature.
- Read every relevant documentation file.
- Understand the existing architecture.
- Check whether similar functionality already exists.
- Verify recent changes inside `update_logs.MD`.
- Consider the impact on:
  - Backend
  - Frontend
  - Database
  - API
  - Documentation
  - User experience

The AI should never begin coding immediately after reading the prompt.

Planning is always preferred over rewriting.

---

# DURING DEVELOPMENT

While implementing a feature, the AI should:

- Follow the existing architecture.
- Respect the project's coding conventions.
- Keep implementations modular.
- Avoid duplicated logic.
- Prefer reusable solutions.
- Prefer composition over duplication.
- Keep files organized.
- Use descriptive names.
- Respect folder responsibilities.

Whenever reusable logic is identified, it should be moved into `utils/`.

Business logic should never be duplicated.

---

# AFTER WRITING CODE

Before considering a task complete, always verify:

- Architecture consistency
- Folder organization
- Naming consistency
- Responsive behavior (frontend)
- Accessibility (frontend)
- Performance
- Security
- No duplicated code
- No dead code
- No unused imports
- No unnecessary dependencies
- No debug code
- No temporary files
- No commented-out code

The implementation should feel complete and production-ready.

---

# DOCUMENTATION EVOLUTION

Documentation evolves together with the project.

Whenever implementation changes any important aspect of the platform, the corresponding documentation must also be updated.

Examples include:

- New features
- Removed features
- New APIs
- Database changes
- Design changes
- New folders
- New conventions
- Architecture improvements
- Business logic changes

Never leave documentation outdated.

---

# DATABASE EVOLUTION

The database architecture is expected to evolve during development.

AI assistants are encouraged to improve the database whenever a better solution is identified.

Possible improvements include:

- Better relationships
- Better normalization
- Better indexes
- Better naming
- Better constraints
- Better scalability
- Better security
- Better performance
- Better auditing

The database is not considered immutable.

Improving the schema is encouraged whenever it benefits the project.

However, AI assistants should always:

- Preserve historical information.
- Prefer soft deletion.
- Avoid destructive migrations.
- Avoid data loss.
- Preserve backward compatibility whenever possible.

Whenever database changes are introduced:

- Update `database.MD`.
- Update `update_logs.MD`.
- Ensure migrations remain consistent.

---

# API EVOLUTION

The REST API is also expected to evolve.

New endpoints, parameters and response fields may be introduced whenever they improve the platform.

The AI may:

- Add endpoints.
- Extend existing endpoints.
- Improve request validation.
- Improve response structures.
- Improve naming.
- Improve authentication.
- Improve permissions.
- Improve endpoint organization.

Breaking changes should be avoided whenever possible.

Whenever the API changes:

- Update `api.MD`.
- Update `update_logs.MD`.

The API should remain:

- Predictable
- RESTful
- Consistent
- Versioned
- Easy to maintain

---

## RYZE Feed Development Procedure

RYZE Feed is a core RYZE feature and must be developed according to the existing project architecture, database conventions, security requirements, and design system.

Before modifying the RYZE Feed, AI agents must read:

- `architecture.MD`
- `DB.md`
- `design.MD`
- `security.MD`
- API/backend documentation
- `AGENTS.md`
- `CLAUDE.md`

### Scope

RYZE Feed includes:

- Short-form fitness videos
- Trainer-created content
- Vertical video consumption
- Trainer profiles
- Trainer following
- Video Likes
- Feed personalization
- Content publishing
- Content management
- Video processing
- Content moderation

RYZE Feed must not be expanded into a general-purpose social network without an explicit product decision.

### Before Implementation

Before creating or modifying Feed functionality, the agent must:

1. Inspect the existing implementation.
2. Read the relevant documentation.
3. Identify existing reusable components and services.
4. Check existing authentication and authorization mechanisms.
5. Check existing database conventions.
6. Check existing media/upload infrastructure.
7. Check existing API conventions.
8. Determine whether the required functionality already exists elsewhere.

Agents should extend existing systems where appropriate rather than creating duplicate implementations.

### Database Changes

Database changes must follow the conventions defined in `DB.md`.

Before changing the schema:

- Verify existing relationships.
- Verify existing user and trainer entities.
- Verify existing UUID conventions.
- Verify timestamp conventions.
- Verify soft-delete conventions.
- Verify indexing conventions.
- Check whether an equivalent table or relationship already exists.

Schema changes must be implemented through the project's established migration system.

Do not modify production database structures manually.

### API Changes

New Feed endpoints must follow the existing RYZE API structure.

Before creating an endpoint:

- Check whether an equivalent endpoint already exists.
- Follow the existing authentication middleware.
- Follow the existing authorization system.
- Follow existing request validation.
- Follow existing response structures.
- Follow existing error handling.
- Follow existing API versioning.

Do not create parallel authentication, authorization, validation, or error-handling systems specifically for the Feed.

### Trainer Content

Trainer content must always maintain a clear ownership relationship.

Any operation that modifies Feed content must verify ownership or explicit administrative authorization on the backend.

Frontend ownership checks are not sufficient.

### Media

Video uploads must use the existing media infrastructure where possible.

If the existing infrastructure is insufficient, the agent must first document the required architectural change before introducing a completely separate storage system.

Media processing must follow the security requirements defined in `security.MD`.

### Feed Personalization

Feed ranking and personalization must remain modular.

Agents must not hard-code the Feed around a single permanent recommendation formula unless explicitly requested.

The initial implementation may use a simple deterministic ranking strategy.

The architecture must allow the ranking strategy to be replaced or extended later without rewriting the Feed UI or database model.

### Likes and Follows

Likes and trainer follows are relationships, not duplicated user state.

Agents must rely on the database constraints and backend validation to prevent duplicate relationships.

The frontend must not be treated as the source of truth.

### Frontend

The Feed interface must follow `design.MD`.

The intended experience is:

```text
Open Feed
    ↓
Active vertical video
    ↓
Minimal controls
    ↓
Scroll
    ↓
Next video
    ↓
Repeat

The Feed should remain visually simple and content-focused.

Do not introduce unnecessary cards, dashboards, filters, comment sections, or social-network features without an explicit design/product decision.

Performance

Feed development must prioritize:

Fast initial loading
Lazy loading
Efficient video loading
Automatic playback only for the active item
Pausing inactive videos
Incremental Feed loading
Efficient database queries
Minimal unnecessary API requests

The entire Feed must never be loaded into memory at once.

Documentation Synchronization

Whenever Feed architecture, database structure, API behavior, security requirements, or UI behavior changes, the relevant documentation must be updated in the same development cycle.

Documentation must reflect the current implementation.

Agents must not leave implementation and documentation intentionally inconsistent.

Safe Changes

If an implementation decision is unclear or conflicts with existing documentation:

Stop before making a broad architectural change.
Identify the conflict.
Check the relevant documentation.
Prefer the existing project convention.
If the decision requires a product or architectural choice, document the proposed change before implementing it.

Agents may make small implementation decisions independently when they do not affect the documented architecture or product behavior.

Future Changes

RYZE Feed is expected to evolve.

The initial implementation should not prevent future functionality such as:

Advanced recommendations
Video analytics
View/retention metrics
Categories
Tags
Trainer analytics
Content reports
Moderation tools
Featured content
Additional creator roles

Future functionality should be added incrementally rather than prematurely implementing speculative systems.

---

# FOLDER STRUCTURE

The current folder structure is a foundation, not a limitation.

AI assistants may create new folders and files whenever they improve:

- Organization
- Readability
- Scalability
- Maintainability
- Separation of responsibilities

Folder names should always be descriptive.

Avoid names such as:

- misc
- stuff
- temp2
- helpers2
- random

Instead prefer names that clearly describe their responsibility.

Examples:

- authentication
- middleware
- repositories
- notifications
- validators
- websocket
- scheduler
- uploads

Whenever a new folder becomes part of the project's architecture, it should also be documented.

---

# FILE CREATION

Before creating a new file, verify whether an existing file already fulfills that responsibility.

Never duplicate documentation.

Never duplicate code.

Whenever possible:

- Extend existing modules.
- Reuse existing abstractions.
- Improve existing implementations.

Create new files only when they improve organization or readability.

---

# CODE QUALITY

Every implementation should prioritize:

- Readability
- Simplicity
- Maintainability
- Performance
- Security
- Scalability

Readable code is preferred over clever code.

Small, well-structured functions are preferred over large monolithic implementations.

Every file should have a clear responsibility.

Business logic should remain independent from presentation logic.

Whenever complexity increases, split the implementation into smaller modules.

---

# DESIGN PRINCIPLES

Every user interface should remain consistent with the project's visual identity.

The interface should prioritize:

- Simplicity
- Clarity
- Accessibility
- Consistency
- Responsiveness
- Smooth animations
- Fast interactions

Animations should improve the experience without becoming distracting.

Loading states, transitions and feedback should always feel responsive and polished.

Whenever UI components are reused across multiple pages, reusable components should be created instead of duplicating layouts.

# WHEN SOMETHING IS UNCLEAR

AI assistants should never invent project requirements.

Whenever documentation does not answer a question:

1. Search the relevant documentation again.
2. Analyze the existing implementation.
3. Infer only when the decision is technically obvious.
4. Otherwise, ask the project owner.

Never fabricate business rules.

Never contradict documented decisions.

Whenever assumptions are made, they should be explicitly stated.

# DECISION MAKING

Whenever multiple valid implementations exist, the AI should choose the solution that best satisfies the following priorities:

1. Simplicity
2. Readability
3. Maintainability
4. Performance
5. Security
6. Scalability

The AI should not simply implement the first working solution.

Instead, it should evaluate the available alternatives and select the most appropriate long-term solution.

Whenever a significantly better architectural approach is identified, the AI may suggest it before implementation.

# DOCUMENTATION CONSISTENCY

Documentation should remain internally consistent.

Whenever information is moved to another document:

- Remove duplicated information.
- Update references.
- Avoid conflicting descriptions.

Every topic should have a single authoritative document.

Documentation should never contradict itself.

# VERSIONING

RYZE follows Semantic Versioning.

Major

Breaking architectural changes.

Major feature releases.

Large redesigns.

---

Minor

New functionality.

New modules.

New APIs.

New pages.

New integrations.

---

Patch

Bug fixes.

Performance improvements.

Documentation updates.

Minor refactoring.

---

Every significant modification should be reflected inside update_logs.MD.

# OWNERSHIP

The AI is encouraged to improve the project whenever it identifies opportunities for better architecture, security, performance or maintainability.

However:

- Business decisions always belong to the project owner.
- Visual identity decisions belong to the project owner.
- Product decisions belong to the project owner.

Technical improvements may be proposed proactively, but should never override documented project decisions.


