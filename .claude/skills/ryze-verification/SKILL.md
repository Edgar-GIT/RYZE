---
name: ryze-verification
description: Use when planning or performing focused verification of RYZE frontend, backend, browser, documentation, or future database-migration work.
---

# RYZE verification

Discover commands from the relevant root `package.json`, `go.mod`, Makefiles,
Playwright/Vitest config and documentation before running them. Current frontend
scripts include typecheck, Vitest, Vite build and Playwright E2E; backend
verification is Go-package and focused integration testing.

Choose the smallest sufficient check: typecheck for types, focused unit tests
for logic, build for bundling, browser screenshots/interactions for UI, and
isolated database tests for persistence. Use `webapp-testing` only with the
existing Playwright setup; inspect the rendered page at 360–430px and larger
viewports for UI work.

Never reset shared or production databases. DB integration tests run against
PostgreSQL inside transactions; point them at the isolated `<DB_NAME>_test`
database (`DB_NAME=ryze_test go test ./... -p 1`) rather than the development
database. For migration work, validate schema/data parity, constraints, timestamps,
UUIDs, JSON, indexes, transactions, performance and backup/restore separately.

Report exact commands, environment assumptions, pass/fail results and unrelated
pre-existing failures. Do not claim a visual or migration check passed from
static code inspection alone.
