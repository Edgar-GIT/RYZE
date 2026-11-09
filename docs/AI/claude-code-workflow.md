# Claude Code workflow

This repository uses a short root `CLAUDE.md`, path-scoped `.claude/rules/`,
project-local `.claude/skills/`, three read-only reviewer agents, and minimal
project settings. Skills and references are selected per task; do not load the
whole library or all product documentation at startup.

## Start a task

1. State the requested scope and whether it changes application behavior.
2. Read only the relevant priority documentation and inspect existing routes,
   modules, tests and reusable utilities.
3. Check `.claude/rules/` and invoke the smallest relevant RYZE/technology
   skills. External examples must match installed library versions.
4. For independent high-volume research, use one read-only reviewer and ask for
   concise findings with exact paths. Do not delegate a small edit.
5. Implement the smallest coherent change, then verify proportionally.
6. Update authoritative documentation only for confirmed behavior changes;
   review `git diff`/`git status` and confirm application source was untouched
   for AI-environment work.

## Verified command surface

Verified with Claude Code `2.1.319` on 2026-11-09:

- `/context` and `/context all` — inspect context usage.
- `/compact [focus]` — compact context while optionally preserving a focus.
- `/memory` — inspect loaded memory files.
- `/doctor` — diagnose Claude Code installation/configuration.
- `/doctor prompt-audit` — audit instruction files (supported by this version).
- `/plugin` — inspect/manage plugins from a Claude session.
- `/usage` — inspect plan usage; `/cost` is the related cost view where enabled.

Outside a session, useful checks are `claude --version`, `claude doctor`,
`claude plugin list`, `claude plugin details frontend-design@claude-plugins-official`,
`npx skills list`, and `npx skills list --json`. The Skills CLI is project-local;
do not use `-g` and do not add it to application manifests.

## Verification discipline

Use `npm run typecheck`, focused `npm run test`, `npm run build` and
`npm run test:e2e` from the repository root only when the task warrants them.
Discover backend Go commands from `main/src/backend/` and documentation. Use
isolated databases for integration tests; never reset shared or production data.
For UI work, inspect real 360–430px browser screenshots plus larger viewports.
For migration work, begin read-only and separate schema, data, application,
performance, backup/restore and cutover checks.

No automatic hooks run tests or approve commands. Permissions remain explicit;
`.claude/settings.local.json` is ignored for personal settings.
