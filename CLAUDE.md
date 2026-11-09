# RYZE Claude Code instructions

This is the concise Claude-specific entry point. `AGENTS.md` remains the full
Codex-compatible project instruction source; it is intentionally not imported
here because loading the 685-line file on every Claude session wastes context.
Use the path-scoped rules and skills below for detail.

## Precedence

1. Explicit owner instructions for the current task.
2. `docs/INSTRUCTIONS/procedure.MD` and relevant project documentation.
3. Security and architecture rules.
4. `.claude/rules/` path-scoped rules.
5. Relevant RYZE skills and installed technology skills.
6. Generic model defaults.

Never invent business logic. When documentation and code conflict, identify the
conflict and follow the documented procedure; ask before making an ambiguous
product decision. Write code and documentation in English.

## Read before acting

Read only documentation relevant to the task, beginning with:

- `docs/INSTRUCTIONS/project.MD`
- `docs/INSTRUCTIONS/arquitecture.MD` (actual repository filename)
- `docs/DATABASE/database.MD`
- `docs/INSTRUCTIONS/design.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/REGISTRY/update_logs.MD`

For Feed, Premium Level 1, or Premium Level 2, also read the matching API,
database, security and product sections. Sensitive answers, ownership, access
checks and entitlement rules are server-owned and must not be weakened.

## Verified repository map

RYZE is a React 19 + TypeScript + Vite 8 frontend with React Router, CSS
Modules, Framer Motion, Lucide and Three.js, and a Go 1.26 backend using Gin,
GORM and REST APIs. The verified root package does not currently list Zustand;
do not assume it is available. The backend is configured for MariaDB through
the MySQL driver; the future PostgreSQL migration is preparation only. Main
application roots are:

- `main/src/frontend/src/` — pages, components, stores, API clients and styles.
- `main/src/backend/` — Go API, services, repositories, models and database.
- `main/src/backend/database/migrations/` — embedded SQL migrations (000035 is
  the verified current migration).
- `docs/` — product, architecture, database, design, procedure and registry
  documentation.

Do not add frameworks, libraries, application dependencies, migrations or
database changes unless the owner explicitly requests that implementation.
Use existing modules and API contracts. Keep controllers thin, authorization
server-side, identity from authentication context, and sensitive data out of
logs and public responses.

## Frontend redesign guardrails

The owner has approved a major redesign, but it is 100% mobile-first: begin at
360–430px, support narrower screens, then adapt to tablet and desktop. The
experience should be distinctive, premium, straightforward and fast to
navigate. The official RYZE logo and documented brand palette are protected
(`docs/INSTRUCTIONS/design.MD` is authoritative; current accents are
`#18D8FF` and `#2A8CFF`); the visual system may be substantially redesigned
around those colors with
compatible neutrals and accessible semantic tokens.

Static background images are forbidden everywhere except login and registration.
Functional imagery (exercise media, avatars and useful product imagery) is
allowed. Three.js is functional technology only: preserve useful demonstrations,
load it lazily when appropriate, support WebGL fallback and reduced motion, and
consider mobile GPU/battery cost. Do not add R3F or other dependencies merely
because an installed skill contains examples.

Strictly protected frontend areas must remain unchanged, including indirect
shared-component effects: the entire footer; the Exercise Library and details;
the Services page; and Home's “Find Your Next Program” section. Before any
frontend work, read `docs/AI/frontend-revamp.md` and inspect the protected-area
inventory. Isolate redesigned variants instead of globally restyling shared
dependencies used by protected areas.

Preserve product behavior, API contracts, authentication, permissions,
purchase/entitlement rules and workout execution. Validate actual mobile
viewports, accessibility, reduced motion, keyboard behavior and responsive
states; CSS media queries alone are not proof of responsive behavior.

## Commands and verification

Frontend commands are run from the repository root: `npm run typecheck`,
`npm run test`, `npm run build`, and `npm run test:e2e`. Verify scripts before
using them. Backend commands are run from `main/src/backend/` with Go tooling;
integration tests require an isolated, explicitly approved database. Do not
reset shared or production databases. For this environment-preparation task,
validate configuration and documentation only; do not run application suites.

## Selective Claude environment use

Project skills live in `.claude/skills/` and are loaded only when relevant.
Use `frontend-design`, `ui-ux-pro-max`, `responsive-design` and
`design-system-patterns` for visual direction; the Vercel React skills,
`react-state-management` and `typescript-advanced-types` for implementation;
the relevant `threejs-*` skill(s) for actual 3D work; and `webapp-testing` for
browser verification. Follow installed library versions, not examples that
assume Next.js, R3F, Drei or another stack.

Use `.claude/agents/` reviewers only for independent, high-volume read-only
research or review. They have no edit or shell access. Keep one continuous
context for small changes. Read the matching custom RYZE skill before invoking
its workflow.

## Documentation and handoff

When confirmed implementation behavior changes, update the relevant source
documentation and put the newest concise entry first in
`docs/REGISTRY/update_logs.MD`. Do not claim unverified behavior. Keep the
AI-environment references in `docs/AI/` factual and separate from product
documentation. Review `git diff` and `git status` before handoff; never perform
Git commits, pushes, database resets or production changes as part of routine
work.
