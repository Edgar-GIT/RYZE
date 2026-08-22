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

## Stripe Checkout Provider (Optional)

The payment system supports Stripe Checkout Sessions as the first real payment
provider. The provider is optional: when `STRIPE_SECRET_KEY` is not set, the
backend uses a placeholder that rejects all payment requests.

```text
STRIPE_SECRET_KEY=sk_test_...
STRIPE_SUCCESS_URL=http://localhost:5173/purchase/success
STRIPE_CANCEL_URL=http://localhost:5173/purchase/cancel
```

- `STRIPE_SECRET_KEY` — Stripe API secret key. Required to enable payment
  processing. Use `sk_test_...` for development, `sk_live_...` for production.
  The key is loaded from the `.env` file and set on the global Stripe SDK key
  at startup. It is never logged or exposed to the client.
- `STRIPE_SUCCESS_URL` — URL where the user is redirected after a successful
  Stripe Checkout payment. Optional; when empty, Stripe uses its default
  redirect behaviour.
- `STRIPE_CANCEL_URL` — URL where the user is redirected when they cancel the
  Stripe Checkout flow. Optional; when empty, Stripe uses its default redirect
  behaviour.

Real money is never processed during development. Use Stripe test keys and
test card numbers from the Stripe Dashboard.

## PayPal Provider (Optional)

PayPal is an alternative payment provider. When `PAYPAL_CLIENT_ID` and
`PAYPAL_SECRET` are left empty, PayPal payments are disabled and the payment
service rejects any request that selects the `paypal` method.

```text
PAYPAL_CLIENT_ID=...
PAYPAL_SECRET=...
PAYPAL_MODE=sandbox
```

- `PAYPAL_CLIENT_ID` — PayPal OAuth2 client ID. Required to enable PayPal
  payments. Leave empty to disable.
- `PAYPAL_SECRET` — PayPal OAuth2 client secret. Required to enable PayPal
  payments. Leave empty to disable.
- `PAYPAL_MODE` — PayPal environment mode. Defaults to `sandbox`. Use `live`
  for production.

The PayPal credentials follow the same secrets management as Stripe: loaded from
the `.env` file at startup, never logged, never exposed to the client.

## Webhook Configuration (Optional)

Webhooks enable server-to-server payment confirmation. Leave these empty to
disable webhook endpoints.

```env
# Stripe webhook signing secret (from Stripe dashboard → Webhooks → signing secret)
STRIPE_WEBHOOK_SECRET=

# PayPal webhook ID (from PayPal dashboard → Webhooks → webhook details)
PAYPAL_WEBHOOK_ID=
```

When both a provider secret AND a webhook secret are configured, the
corresponding webhook endpoint is registered:

- `POST /api/v1/webhooks/stripe` — requires `STRIPE_SECRET_KEY` AND
  `STRIPE_WEBHOOK_SECRET`
- `POST /api/v1/webhooks/paypal` — requires `PAYPAL_CLIENT_ID` AND
  `PAYPAL_WEBHOOK_ID`

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
  requires an authenticated admin session with `users.read` (both roles).
- `GET /api/v1/admin/users/{id}` — returns the safe representation of one
  active user; requires an authenticated admin session with `users.read` (both
  roles).
- `GET /api/v1/admin/users/deleted` — lists soft-deleted users; requires
  `users.manage` (Technical Administrator only).
- `POST /api/v1/admin/users` — creates a user (or reactivates a soft-deleted
  email); requires `users.manage` (Technical Administrator only).
- `PATCH /api/v1/admin/users/{id}` — updates the safe fields of a user;
  requires `users.manage` (Technical Administrator only).
- `PATCH /api/v1/admin/users/{id}/disable` — soft-disables a regular user
  account (revokes all sessions, blocks login); requires `users.manage`
  (Technical Administrator only).
- `POST /api/v1/admin/users/{id}/reactivate` — restores a soft-deleted user;
  requires `users.manage` (Technical Administrator only).
- `POST /api/v1/admin/users/{id}/password` — administrative password reset
  (invalidates all sessions); requires `users.manage` (Technical Administrator
  only).
- `POST /api/v1/trainer/apply` — submits a trainer application for the
  authenticated user (requires the `ryze_access_token` cookie); the applicant
  is always resolved from the session.
- `GET /api/v1/trainer/profile` — returns the authenticated trainer's own
  profile; requires an active trainer session with `trainer.profile`.
- `GET /api/v1/trainer/clients` — lists the authenticated trainer's active
  clients (pagination); requires `trainer.clients`.
- `POST /api/v1/trainer/clients` — adds an active user as a client of the
  authenticated trainer; requires `trainer.clients`.
- `DELETE /api/v1/trainer/clients/{userID}` — removes (soft-deletes) only the
  relationship between the authenticated trainer and the user; requires
  `trainer.clients`.
- `POST /api/v1/trainer/clients/{userID}/reactivate` — restores the exact same
  previously removed relationship; requires `trainer.clients`.
- `GET /api/v1/admin/trainer-applications` — lists trainer applications
  (pagination + optional `status` filter); requires `trainer_applications.read`
  (both roles).
- `GET /api/v1/admin/trainer-applications/{id}` — returns one trainer
  application; requires `trainer_applications.read` (both roles).
- `POST /api/v1/admin/trainer-applications/{id}/approve` — approves a PENDING
  application and creates the trainer profile for the same user atomically;
  requires `trainer_applications.manage` (both roles).
- `POST /api/v1/admin/trainer-applications/{id}/reject` — rejects a PENDING
  application (kept in history; the user can re-apply); requires
  `trainer_applications.manage` (both roles).

Authorization model: read endpoints run under `users.read` (both roles); every
user lifecycle endpoint runs under `users.manage` and is denied with HTTP 403
`FORBIDDEN` for the Management Administrator. Trainer-application review runs
under `trainer_applications.read`/`trainer_applications.manage` and is shared
by both roles. There is no permanent-deletion endpoint.

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

---

# Testing

The Go backend uses the standard `testing` package. No external test framework
is required.

## Run All Unit Tests

From the backend directory:

```bash
cd main/src/backend
go test ./services/... ./repositories/... -count=1
```

## Run All Tests Including Integration Tests

Integration tests require a running MariaDB instance and a valid `.env` file:

```bash
cd main/src/backend
go test ./... -count=1
```

Integration tests use database transactions that roll back automatically — no
manual cleanup is needed.

## Run a Specific Test

```bash
cd main/src/backend
go test ./services/purchases/ -run TestGetPurchaseByID -count=1
go test ./api/webhooks/ -run TestStripeWebhook -count=1
```

## Payment Testing with Stripe CLI

For local development with Stripe webhooks, use the Stripe CLI to forward
webhook events to the local server:

1. Install the Stripe CLI: https://docs.stripe.com/stripe-cli
2. Login: `stripe login`
3. Start forwarding:

```bash
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe
```

4. Copy the webhook signing secret printed by the CLI and set it in `.env`:

```text
STRIPE_WEBHOOK_SECRET=whsec_...
```

5. Use Stripe test card numbers from the Stripe Dashboard to complete a
   Checkout Session. The CLI forwards the `checkout.session.completed` event
   to the local server, which verifies the signature and completes the
   purchase.

## Payment Testing with PayPal Sandbox

PayPal sandbox webhooks can be configured in the PayPal Developer Dashboard:

1. Create a sandbox application at https://developer.paypal.com
2. Set `PAYPAL_CLIENT_ID`, `PAYPAL_SECRET`, and `PAYPAL_MODE=sandbox` in
   `.env`
3. Create a webhook in the PayPal dashboard pointing to your local server
   (use ngrok or similar for local tunneling):

```bash
ngrok http 8080
```

4. Set the webhook URL in the PayPal dashboard to
   `https://<ngrok-id>.ngrok.io/api/v1/webhooks/paypal`
5. Copy the webhook ID and set `PAYPAL_WEBHOOK_ID` in `.env`

## Testing Without Real Providers

When `STRIPE_SECRET_KEY` and `PAYPAL_CLIENT_ID` are both empty, the payment
service uses a placeholder provider that rejects all payment initiation
requests. This is the default state and is safe for development without real
provider accounts.
