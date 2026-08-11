# RYZE
# How to Run

Version: 1.0.0

---

# Purpose

This document explains how to install, run and build the RYZE frontend and
the Go backend foundation.

The current implementation contains the initial visual interface plus a
backend database connection layer. Other backend features are not implemented
yet.

---

# Prerequisites

Install the following tools before running the project:

- Node.js 22 or newer
- npm 10 or newer
- Git

This implementation was validated with:

- Node.js v26.5.0
- npm 12.0.2

---

# Installation

From the repository root, install dependencies:

```bash
npm install
```

The project uses the dependencies declared in `package.json` and locked in
`package-lock.json`.

---

# Dependencies

Runtime dependencies:

- React
- React DOM
- React Router DOM
- Lucide React
- Framer Motion

Development dependencies:

- Vite
- TypeScript
- Vite React plugin
- React type definitions
- Node type definitions

---

# Start Development Server

Run:

```bash
npm run dev
```

The development server starts on:

```text
http://localhost:5173
```

If the terminal shows a different port because `5173` is already in use,
open the URL printed by Vite.

---

# Production Build

Run:

```bash
npm run build
```

The build command performs:

1. TypeScript validation with `tsc --noEmit`.
2. Vite production bundling.

Production output is generated inside:

```text
dist/
```

---

# Preview Production Build

After building, run:

```bash
npm run preview
```

The preview server starts on:

```text
http://localhost:4173
```

---

# Folder Structure

Current frontend structure:

```text
main/src/frontend/src
├── components
├── constants
├── hooks
├── pages
├── routes
└── styles
```

Static resources remain outside the frontend source tree:

```text
main/src/resources
└── img
    └── logo
        ├── ryze_icon.png
        ├── ryze_set.png
        └── ryze_slogan.png
```

Shared reusable project utilities live in:

```text
main/src/utils
```

---

# Local Website Launch

To launch the website locally:

1. Open a terminal in the repository root.
2. Run `npm install`.
3. Run `npm run dev`.
4. Open `http://localhost:5173`.

Available frontend routes:

- `/`
- `/services`
- `/contact`
- `/our-vision`
- `/feedback`
- `/feed`
- `/profile`
- `/login`
- `/register`

`/about-us` redirects to `/our-vision`.

The `/login` and `/register` pages are connected to the backend: registration
calls `POST /api/v1/auth/register` and login calls `POST /api/v1/auth/login`.
Both redirect to `/profile` on success. Start the backend (see below) before
using the forms.

The `/profile` route is the simple authenticated profile page. It loads the
authenticated user from `GET /api/v1/me` (redirecting to `/login` on 401) and
offers a "Log out" action that calls `POST /api/v1/auth/logout`. This page
requires the HttpOnly `ryze_access_token` cookie set by login.

Routes that are not fully implemented yet display a professional under
development screen instead of a blank page or browser error.

---

# Backend

The backend lives in:

```text
main/src/backend
```

It uses Go with the documented GORM stack.

## Prerequisites

- Go 1.26 or newer
- A running MariaDB/MySQL server

## Database Configuration

The backend reads its database configuration from environment variables:

- `DB_HOST` (default `127.0.0.1`)
- `DB_PORT` (default `3306`)
- `DB_NAME` (required)
- `DB_USER` (required)
- `DB_PASSWORD`

Local development uses the system MariaDB instance on port `3306`. The
project uses a dedicated development administrator user with full privileges
only on the `ryze` database.

Create the database and user with root privileges:

```bash
sudo mariadb
```

Then run:

```sql
CREATE DATABASE IF NOT EXISTS ryze CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'ryze_admin'@'127.0.0.1' IDENTIFIED BY 'your_password';
CREATE USER IF NOT EXISTS 'ryze_admin'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON ryze.* TO 'ryze_admin'@'127.0.0.1';
GRANT ALL PRIVILEGES ON ryze.* TO 'ryze_admin'@'localhost';
FLUSH PRIVILEGES;
```

The `ryze_admin` user receives full privileges only on `ryze.*` and no
privileges over unrelated databases.

Add the variables to the repository root `.env` file (already ignored by Git):

```text
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=ryze
DB_USER=ryze_admin
DB_PASSWORD=your_password
```

A template without secrets is available at `.env.example`.

The backend loads the repository root `.env` automatically, so the database
credentials never need to be hardcoded or committed.

## JWT Configuration

The backend signs access tokens with HMAC-SHA256. The configuration is read
from environment variables:

- `JWT_SECRET` (required, at least 32 characters)
- `JWT_ACCESS_TOKEN_TTL` (optional, defaults to `15m`)
- `COOKIE_SECURE` (optional, defaults to `true`; the auth cookie is served
  with the Secure flag, which must be `false` only for local HTTP development)

If `JWT_SECRET` is missing or too short, the backend fails at startup instead
of falling back to an insecure default secret.

The access token is delivered as an HttpOnly cookie named `ryze_access_token`
(SameSite=Lax, Path=/), not in the JSON response.

The `.env.example` template uses placeholders. Generate a local secret (never
commit a real value) with:

```bash
openssl rand -hex 32
```

## CORS Configuration

The frontend runs on a different origin during development (`localhost:5173`)
and sends its requests with credentials (the auth cookie). The backend only
serves CORS headers to an explicit allow-list, never to `*`.

- `CORS_ALLOWED_ORIGINS` (optional, comma-separated). Defaults to
  `http://localhost:5173,http://127.0.0.1:5173` (the Vite dev server).

Add other origins (for example the Vite preview on `4173` or a deployed
frontend) by listing them:

```text
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173
```

## Admin Authentication Configuration

Platform administrators are configuration-only accounts, never stored in the
database. Each administrator requires a username, a password and an access
code (the second authentication factor). The access code must be at least 8
characters long. If any value is missing or too short, the backend fails at
startup instead of silently disabling an administrator.

```text
ADMIN_1_USERNAME=...
ADMIN_1_PASSWORD=...
ADMIN_1_ACCESS_CODE=...
ADMIN_2_USERNAME=...
ADMIN_2_PASSWORD=...
ADMIN_2_ACCESS_CODE=...
```

The access code is a secret: generate it locally and never commit a real
value. Changing the values in `.env` takes effect on the next server restart.

## Verify the Database Connection

From the backend directory, run:

```bash
cd main/src/backend
go run .
```

A successful connection prints:

```text
database connection established successfully (host=127.0.0.1 port=3306 db=ryze)
```

If the database is unreachable, misconfigured, or the credentials are
rejected, the backend prints a clear error and exits with a non-zero status.

After the connection succeeds the backend starts the HTTP API. The listen port
is read from the `PORT` environment variable and defaults to `8080`.

```bash
cd main/src/backend
go run .
```

## Available Endpoints

Current API surface (versioned under `/api/v1`):

- `POST /api/v1/auth/register` — creates a new user account (public).
- `POST /api/v1/auth/login` — authenticates with email/password and sets the
  `ryze_access_token` HttpOnly cookie (public).
- `POST /api/v1/auth/logout` — clears the `ryze_access_token` cookie (public;
  works even with a missing, expired or invalid cookie).
- `GET /api/v1/me` — returns the authenticated user's safe account
  information (requires the `ryze_access_token` HttpOnly cookie).
- `POST /api/v1/admin/auth/login` — step 1 of admin authentication
  (username/password); on success sets the short-lived
  `ryze_admin_stage_token` cookie (public).
- `POST /api/v1/admin/auth/verify` — step 2 of admin authentication (access
  code); on success sets the `ryze_admin_access_token` HttpOnly cookie and
  clears the stage cookie (public).
- `GET /api/v1/admin/users` — lists active users (pagination, newest first);
  requires an authenticated admin session.
- `GET /api/v1/admin/users/{id}` — returns the safe representation of one
  active user; requires an authenticated admin session.
- `PATCH /api/v1/admin/users/{id}/disable` — soft-disables a regular user
  account (revokes all sessions, blocks login); requires an authenticated
  admin session.

The three user-management endpoints are authorized for both administrator roles
(`RequireAdminRole(TECHNICAL_ADMINISTRATOR, MANAGEMENT_ADMINISTRATOR)`).

## Database Migrations

Database schema changes are managed with `golang-migrate`. The schema must
be reproducible entirely through migrations — never modify the database
manually (for example through DBeaver).

Migrations are versioned files inside:

```text
main/src/backend/database/migrations/
```

Each migration has an `.up.sql` and a `.down.sql` file.

Run all pending migrations:

```bash
cd main/src/backend
go run . migrate up
```

Roll back all applied migrations:

```bash
go run . migrate down
```

Check the current migration status:

```bash
go run . migrate version
```

The migration commands must be executed from the `main/src/backend`
directory.
