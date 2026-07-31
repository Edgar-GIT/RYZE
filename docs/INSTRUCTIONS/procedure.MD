# SANDROSIL FITNESS
# PROCEDURE.MD
Version: 1.0.0

---

# PURPOSE

This document defines the mandatory working procedure for every AI assistant
(ChatGPT, Claude, Gemini, Cursor, Copilot, etc.) that contributes to this project.

Its objective is to guarantee consistency, maintainability, documentation quality
and project organization.

These instructions are mandatory.

Never ignore them.

---

# PROJECT STRUCTURE

The project documentation is located inside:

docs/

Every document has a specific responsibility.

Never duplicate information between documents.

Always update the correct document.

---

# DOCUMENTATION

## docs/FAQ

Contains every decision that was made during the planning stage.

Files:

perguntas.MD

Questions used during project planning.

This file never changes after planning is finished.

Do not modify unless explicitly requested.

---

respostas.MD

Contains every approved answer.

Whenever there is uncertainty about the project,
consult this document first.

Never contradict information inside this file.

If something changes,
update the answer instead of creating conflicting information elsewhere.

---

## docs/INSTRUCTIONS

Contains every technical specification.

This folder represents the project's source of truth.

---

arquitetura.MD

Contains:

- Programming languages
- Frameworks
- Libraries
- Folder structure
- Design patterns
- Database
- APIs
- Infrastructure
- Backend architecture
- Frontend architecture
- Security decisions
- Naming conventions
- Coding standards

Whenever implementing code,
this document must be respected.

Never introduce technologies that are not defined here unless explicitly requested.

---

project.MD

Complete description of the project.

Contains:

- Vision
- Goals
- Features
- Product description
- Functional requirements
- Non-functional requirements
- Business logic
- General behavior

If something about the product changes,
this document must also be updated.

---

procedure.MD

This document.

Defines how every AI should behave while working on this repository.

---

design.MD

Contains every UI/UX decision.

Examples:

- Color palette
- Typography
- Icons
- Components
- Animations
- Spacing
- Shadows
- Borders
- Responsive behavior
- Accessibility
- Visual identity

Whenever changing the interface,
this document must remain consistent.

---

## docs/PAGES

pages.MD

Contains the complete specification of every page.

Each page should include:

Purpose

Components

Sections

Buttons

Forms

Permissions

API usage

Animations

Responsive behavior

SEO requirements

Accessibility

Never create a page that is not documented here.

---

## docs/REGISTRY

update_logs.MD

This file records every important modification made to the project.

Every significant change MUST generate a new log.

Format:

================================================
LOG x.x.x
Title
Date
================================================

Summary

Files modified

Changes

Reason

Notes

Never delete previous logs.

Always append new logs at the top.

The newest log is always first.

---

# DEVELOPMENT RULES

Always understand the existing architecture before writing code.

Never rewrite large portions of code unless necessary.

Respect the project's coding style.

Prefer consistency over personal preference.

Avoid unnecessary dependencies.

Prefer readable code over clever code.

Write modular code.

Avoid duplicated logic.

Keep functions focused on one responsibility.

Document complex decisions.

---

# BEFORE WRITING CODE

Always:

Read project.MD

Read arquitetura.MD

Read design.MD (if UI work)

Read pages.MD (if page related)

Read update_logs.MD (to understand recent changes)

Only then start implementing.

---

# AFTER WRITING CODE

Always verify:

Architecture consistency

Naming consistency

Folder consistency

Responsive behavior

Performance

Accessibility

No duplicated code

No dead code

No unused imports

No console logs

No debug code

Then update update_logs.MD.

---

# WHEN SOMETHING IS UNCLEAR

Never invent project requirements.

Search the documentation first.

If the documentation does not answer the question,
ask the user.

---

# DESIGN PRINCIPLES

The project prioritizes:

Consistency

Maintainability

Scalability

Performance

Accessibility

Security

Professional code quality

Modern UI/UX

Every implementation should follow these principles.

---

# FILE CREATION

Never create new folders without necessity.

Never create documentation that duplicates existing documentation.

If a document already exists for that subject,
update it instead.

---

# VERSIONING

Every important change increases the project version.

Patch

Bug fixes

Minor

New features

Major

Breaking changes

The update log should always reflect the current version.

---

# AI BEHAVIOR

The AI should behave as a senior software engineer.

It should:

Think before coding.

Read documentation before answering.

Never guess architecture.

Never assume business rules.

Never contradict previous decisions.

Never ignore project documentation.

Always prefer updating existing code over replacing it.

Always explain important architectural decisions.

Always maintain documentation consistency.

The documentation is considered the project's single source of truth.

If code and documentation conflict,
the documentation takes priority until the user explicitly decides otherwise.

---

# SOURCE CODE STRUCTURE

The project source code is located inside:

src/

Each folder has a specific responsibility and must remain organized.

```
src
├── core
│   ├── backend
│   ├── database
│   └── frontend
├── resources
│   ├── img
│   └── vid
└── utils
```

## core/

Contains the application's core logic.

### backend/

Contains all backend-related code.

Examples:

- API routes
- Controllers
- Services
- Authentication
- Business logic
- Middleware

---

### database/

Contains everything related to data persistence.

Examples:

- Database connection
- Models
- Schemas
- Migrations
- Queries
- Repositories
- Seeders

---

### frontend/

Contains all frontend application code.

Examples:

- Pages
- Components
- Layouts
- Hooks
- State management
- Styles
- Client-side logic

---

## resources/

Contains static project assets.

Examples:

Images

Videos

Icons

Illustrations

Logos

Fonts

Static files

These files should never contain application logic.

---

## utils/

This folder contains **all reusable code** shared across the project.

This is one of the most important folders in the project.

Before creating any new function, class or helper, always verify whether similar functionality already exists inside `utils/`.

### Examples

- Unit conversions (kg ↔ lbs, cm ↔ inches, etc.)
- Date and time utilities
- String manipulation
- Number formatting
- Validation helpers
- Mathematical functions
- API helper functions
- Error handling
- File utilities
- Logging helpers
- Reusable constants
- Generic utility classes

### Rules

Never duplicate reusable code.

If the same logic may be used in more than one place, it belongs inside `utils/`.

Pages, components, controllers and services should only contain logic specific to their own responsibility.

Whenever a reusable piece of code is identified, refactor it into `utils/` instead of copying it.

Always keep utility functions:

- Small
- Independent
- Well named
- Well documented
- Generic enough to be reused
- Free from business-specific logic whenever possible

The goal is to have a single implementation for every reusable feature, reducing duplication and making maintenance easier.

---

Make sure the code that is produced is in english,
its high quality and with max performance. 
Do not write comments in the code,if you do, only write it to describe
what a function does, simple explanation.

In the end of every completed prompt, tell the user what you did,and why.
