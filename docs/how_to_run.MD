# RYZE
# How to Run

Version: 2.83.0

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

# Frontend Tests

Run:

```bash
npm test
```

The frontend unit tests run with Vitest (`vitest run`) in the jsdom
environment. Configuration lives in `vite.config.ts` (`test` block) and the
same path aliases used by the application are available inside tests. Test
files follow the `*.test.ts` / `*.test.tsx` suffix. Current coverage targets
the shared HTTP client consumers and the trainer session logic; the command is
part of the normal validation loop together with `npm run typecheck`.

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
- `JWT_ACCESS_TOKEN_TTL` (optional, defaults to `15m`; capped at `24h` — a
  larger value fails at startup)
- `COOKIE_SECURE` (optional, defaults to `true`; the auth cookie is served
  with the Secure flag, which must be `false` only for local HTTP development.
  Setting it to `false` prints a startup warning)

If `JWT_SECRET` is missing or too short, the backend fails at startup instead
of falling back to an insecure default secret.

Every issued token carries the issuer `ryze` and audience `ryze-api`, and
validation requires both plus an expiration; access tokens issued before an
upgrade to v2.93.0 lack these claims and are rejected, so users simply log in
again after deploying.

The access token is delivered as an HttpOnly cookie named `ryze_access_token`
(SameSite=Lax, Path=/), not in the JSON response.

State-changing browser requests are origin-checked: the request origin must be
the API's own host or a `CORS_ALLOWED_ORIGINS` entry, otherwise the request is
rejected with `403 CROSS_ORIGIN_FORBIDDEN`. Non-browser clients (curl, webhook
providers) send no origin and are unaffected.

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

## HTTP Server Safety Configuration

The backend runs an explicit `http.Server` with connection timeouts and a
request body limit, and drains in-flight requests on shutdown. Every value is
optional and has a safe default; a present-but-invalid value fails at startup so
a typo can never silently disable a timeout or the body limit.

- `SERVER_READ_HEADER_TIMEOUT` (optional, default `10s`) — time allowed to read
  the request headers.
- `SERVER_READ_TIMEOUT` (optional, default `30s`) — time allowed to read the
  full request, including the body.
- `SERVER_WRITE_TIMEOUT` (optional, default `60s`) — time allowed to write the
  response.
- `SERVER_IDLE_TIMEOUT` (optional, default `120s`) — time an idle keep-alive
  connection is kept open.
- `SERVER_SHUTDOWN_TIMEOUT` (optional, default `20s`) — time in-flight requests
  are allowed to finish after a termination signal.
- `SERVER_MAX_BODY_BYTES` (optional, default `1048576`, i.e. 1 MiB) — maximum
  request body size. A larger body is rejected with
  `413 REQUEST_ENTITY_TOO_LARGE` before any handler reads it.

A deploy or restart that delivers `SIGINT` or `SIGTERM` stops accepting new
connections, lets in-flight requests finish within the shutdown timeout, and
then closes the database handle, so a restart never drops an in-flight request.

## Authentication Rate Limiting Configuration

The credential and account endpoints (register, login, admin login/verify,
change-password, delete-account) are rate limited in memory, per client IP and
per account identifier. Every value is optional.

- `RATE_LIMIT_ENABLED` (optional, default `true`) — set to `false` to disable
  the limits entirely. The middleware is then not mounted at all.
- `RATE_LIMIT_MAX_KEYS` (optional, default `10000`) — maximum counters tracked
  per limiter, evicted least-recently-used first.
- `RATE_LIMIT_TRUSTED_PROXIES` (optional, default empty) — comma-separated IP
  addresses or CIDR ranges whose forwarded client-address headers are trusted.
  Leave empty unless the API runs behind a known reverse proxy; with the
  default the socket address is used, so a client cannot spoof its rate-limit
  key through `X-Forwarded-For`. Any entry that is neither an IP nor a CIDR
  range fails at startup.

An exhausted limit returns `429 RATE_LIMITED` with a `Retry-After` header.
Counting is per instance, so multi-instance deployments should complement it
with infrastructure-level limiting. Webhooks are never rate limited.

## Admin Authentication Configuration

Platform administrators are configuration-only accounts, never stored in the
database. Each administrator requires a username, a password and an access
code (the second authentication factor). The password must be at least 12
characters (the same minimum applied to user passwords) and the access code at
least 8 characters. If any value is missing or too short, the backend fails at
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

## Local Test Accounts

Local development credentials for test accounts live in the repository `.env`
file (git ignored).

### Regular user

A normal user account for exercising the product flows. The credentials are
defined in `.env` under `TEST_USER_EMAIL` / `TEST_USER_PASSWORD`. Current local
development values:

- Email: `teste1@gmail.com`
- Password: `RyzeTest1234` (12 characters, the minimum the password policy
  accepts)

The account is not seeded by migrations — it is created through the public
registration endpoint. After a database reset, recreate it with the same
endpoint (or simply register it again in the frontend):

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"teste1@gmail.com","password":"RyzeTest1234","first_name":"Teste","last_name":"Um"}'
```

Passwords must be 12 to 128 characters on every endpoint that sets one
(registration, change-password, administrative creation and administrative
reset); a shorter or longer value returns `400 VALIDATION_ERROR`.

### Administrator accounts

Platform administrators are configuration-only accounts defined entirely in
`.env` (`ADMIN_1_*` / `ADMIN_2_*`): username, password and access code. There
is no registration endpoint; the values only take effect after a backend
restart. See "Admin Authentication Configuration" above.

## Admin Test Mode (Optional)

Test Mode lets the Technical Administrator (ADMIN_1) impersonate a predefined
persona (Test Client / Test Trainer) to exercise the commerce flows with no
real payments. The feature is disabled by default and is switched on only
when the backend is configured with it:

```text
TEST_MODE_ENABLED=true
```

- When `TEST_MODE_ENABLED` is absent or `false`, every Test Mode action answers
  `403 TEST_MODE_DISABLED`, no Test Mode session can be created, and the
  feature is effectively absent.
- When enabled, sign in to the Admin Console as ADMIN_1 and open
  **Test Mode** in the Technical navigation to enter as one of the personas.
  Inside Test Mode all purchasable programs render as Free and buying completes a
  real zero-cost purchase + entitlement server-side — no payment provider is
  involved.
- Exiting Test Mode restores the original ADMIN_1 session and returns to the
  configured return path. Audit entries (`test_mode.entered` /
  `test_mode.exited` / `test_mode.purchase_completed`) are written through the
  standard logging pipeline.
- Test Mode requires a database migration: run `go run . migrate up` once in
  development. Migration `000023` creates the `test_sessions` table and the
  `purchases.test` marker.

Real money is never involved and no provider configuration is required for
Test Mode purchases.

## Stripe Checkout Provider (Optional)

Stripe serves the `card` payment method through Stripe-hosted Checkout Sessions.
Card details are entered only on the Stripe page: RYZE never receives, logs or
stores raw card data.

Stripe is enabled **only when both `STRIPE_SECRET_KEY` and
`STRIPE_WEBHOOK_SECRET` are set**. Webhooks are the reliable way to confirm a
payment, so a secret key without a signing secret leaves the `card` method
unadvertised and unprocessable rather than half-enabled.

```text
STRIPE_SECRET_KEY=sk_test_...
STRIPE_SUCCESS_URL=http://localhost:5173/services/generic-program/{program_id}/purchase/success?session_id={CHECKOUT_SESSION_ID}&purchase_id={purchase_id}
STRIPE_CANCEL_URL=http://localhost:5173/services/generic-program/{program_id}/purchase/cancelled?status=cancelled&purchase_id={purchase_id}
```

- `STRIPE_SECRET_KEY` — Stripe API secret key. Required to enable card
  processing. Use `sk_test_...` for development, `sk_live_...` for production.
  The key is loaded from the `.env` file and set on the global Stripe SDK key
  at startup. It is never logged or exposed to the client.
- `STRIPE_WEBHOOK_SECRET` — Stripe webhook signing secret. Also required; see
  [Webhook Configuration](#webhook-configuration).
- `STRIPE_SUCCESS_URL` — URL where the user is redirected after a successful
  Stripe Checkout payment. Optional; when empty, Stripe uses its default
  redirect behaviour. The `{program_id}` and `{purchase_id}` placeholders are
  replaced server-side for each session, and `{CHECKOUT_SESSION_ID}` is reserved
  for Stripe and must be kept on the success URL so the return leg can be
  verified. The frontend registers the documented
  `/services/generic-program/{program_id}/purchase/success` and
  `/services/generic-program/{program_id}/purchase/cancelled` paths (declared
  before the `/services/:serviceSlug` route) for both Generic and Premium
  purchases, since the return URL is shared by both product families.
- `STRIPE_CANCEL_URL` — URL where the user is redirected when they cancel the
  Stripe Checkout flow. Optional; when empty, Stripe uses its default redirect
  behaviour. The same `{program_id}` / `{purchase_id}` placeholders are
  substituted server-side; the frontend reads `purchase_id` and
  `status=cancelled` to recover the purchase.

One Checkout Session is created per purchase. Repeated initiations reuse the
same session, so a buyer can never end up with two payable sessions for the
same purchase.

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
PAYPAL_RETURN_URL=http://localhost:5173/services/generic-program/{program_id}?purchase_id={purchase_id}
PAYPAL_CANCEL_URL=http://localhost:5173/services/generic-program/{program_id}?purchase_id={purchase_id}&status=cancelled
```

- `PAYPAL_CLIENT_ID` — PayPal OAuth2 client ID. Required to enable PayPal
  payments. Leave empty to disable.
- `PAYPAL_SECRET` — PayPal OAuth2 client secret. Required to enable PayPal
  payments. Leave empty to disable.
- `PAYPAL_MODE` — PayPal environment mode. Defaults to `sandbox`. Use `live`
  for production.
- `PAYPAL_RETURN_URL` / `PAYPAL_CANCEL_URL` — URL templates where PayPal
  redirects the buyer after approving/cancelling a payment. Optional; the
  `{program_id}` and `{purchase_id}` placeholders are replaced server-side for
  each order. When both are empty, orders are created without a redirect
  context. PayPal appends the order token (`?token=<order_id>`) to both URLs,
  where `<order_id>` is the PayPal Order ID.

The PayPal credentials follow the same secrets management as Stripe: loaded from
the `.env` file at startup, never logged, never exposed to the client.

When the buyer returns from PayPal, the frontend calls the capture endpoint
(`POST /api/v1/me/purchases/:purchaseID/capture`) with the order token as
`provider_payment_id`. The backend loads the real order from PayPal, verifies it belongs to the purchase
(reference id, amount, currency) and captures it before completing the
purchase. A browser redirect alone never completes a purchase.

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
  `STRIPE_WEBHOOK_SECRET`; the same pair enables the `card` method, so the
  endpoint is registered only when card payments are actually possible
- `POST /api/v1/webhooks/paypal` — requires `PAYPAL_CLIENT_ID` AND
  `PAYPAL_WEBHOOK_ID`

The Stripe handler completes a purchase only for a fully paid one-time payment
Session (`payment_status == "paid"`, `mode == "payment"`) whose
`client_reference_id` matches the `purchase_id` metadata and whose amount and
currency match the purchase snapshot. Unpaid sessions (delayed payment
methods), refunds and disputes are acknowledged and ignored: RYZE purchases are
final and there is no refund workflow.

## Exercise Media (Optional)

The bundled exercise dataset media is third-party content (© Gym visual). Media
exposure is disabled by default and **must not be enabled in production until
commercial-use permission has been verified** (see `security.MD`).

```env
# Serve exercise media references (default: false)
EXERCISE_MEDIA_ENABLED=false

# Same-origin path the media references resolve against
EXERCISE_MEDIA_BASE_PATH=/media/exercises
```

While disabled, every API response reports `media_available: false` and empty
media fields, and the client renders its neutral placeholder. When enabled, the
base path must be a same-origin absolute path (no scheme, host, `..` or empty
value); validation rejects anything else at startup. The files themselves are
served by the deployment (for example nginx), not by the API.

## Trainer Evidence Storage (Optional)

Applicant evidence documents are private; only validated metadata is stored in
the database and the file bytes live in the configured directory.

```env
# Private directory for applicant documents (default: storage/trainer-evidence)
TRAINER_EVIDENCE_DIR=storage/trainer-evidence

# Maximum size of one document in bytes (default: 10485760 = 10 MiB)
TRAINER_EVIDENCE_MAX_BYTES=10485760
```

The directory is created with `0700` and files are written with `0600`. It must
not be inside a publicly served web root, and it must survive deployments and be
included in backups: the database only keeps opaque storage keys. Values above
`26214400` (25 MiB) are rejected at startup; keep the reverse proxy body limit
at least as large as `TRAINER_EVIDENCE_MAX_BYTES`.

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
is read from the `PORT` environment variable and defaults to `8080`. The server
applies the connection timeouts and the request body limit described in "HTTP
Server Safety Configuration", and shuts down gracefully on `SIGINT`/`SIGTERM`.

```bash
cd main/src/backend
go run .
```

## Available Endpoints

Current API surface (versioned under `/api/v1`):

- `GET /api/v1/health` — reports API status (`{"status":"ok"}`). Probes the
  database and answers `503 SERVICE_UNAVAILABLE` when it is unreachable, so a
  load balancer never routes traffic to an instance that cannot serve requests
  (public).
- `GET /api/v1/version` — reports the running API version the binary was built
  with (public).
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
- `POST/GET /api/v1/trainer/application/documents` — uploads or lists the
  applicant's own evidence documents; upload is allowed only while the latest
  application is PENDING. `GET
  /api/v1/trainer/application/documents/{documentID}/download` and `DELETE
  /api/v1/trainer/application/documents/{documentID}` operate on an owned,
  active document.
- `GET /api/v1/exercises` — lists published exercises (pagination + filters);
  `GET /api/v1/exercises/search?q=` searches names; `GET
  /api/v1/exercises/{exerciseID}` returns one published exercise with its
  alternatives (public).
- `GET /api/v1/admin/exercises` — lists catalogue entries for administration
  (pagination, search, status and body-part filters); `POST
  /api/v1/admin/exercises` creates a draft; `GET`/`PATCH
  /api/v1/admin/exercises/{exerciseID}` reads or updates one entry; requires
  `exercises.read`/`exercises.manage` (Management Administrator only).
- `GET /api/v1/admin/trainer-applications/{id}/documents` and `GET
  /api/v1/admin/trainer-applications/{id}/decisions` — reviewer evidence list
  and append-only decision history (chronological); `GET
  /api/v1/admin/trainer-applications/{id}/documents/{documentID}/download`
  streams one document; requires `trainer_applications.read` (both roles).
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
  accepts an optional `{"reason": "..."}` reviewer note; requires
  `trainer_applications.manage` (both roles).
- `POST /api/v1/admin/trainer-applications/{id}/reject` — rejects a PENDING
  application (kept in history; the user can re-apply); accepts an optional
  `{"reason": "..."}` reviewer note; requires
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

## Importing the Exercise Dataset

The canonical catalogue ships with a seed, but the full exercise dataset is
imported explicitly. Apply migrations first, then run the idempotent importer
from the backend directory:

```bash
cd main/src/backend
go run . import-exercises \
  --file ../resources/exercises/exercises-dataset-main/data/exercises.json
```

Use `--dry-run` to validate the file and report what would change without
writing. The importer matches records by `source` + `source_id`, then by
normalized name, and never overwrites an existing value; re-running it is safe.
It does not serve media — see "Exercise Media (Optional)" for the licensing
gate.

## Deployment

A production-like stack is defined in `docker-compose.yml` at the repository
root. It runs three services: `db` (MariaDB, never published to the host),
`backend` (the API image) and `frontend` (nginx serving the built client and
proxying `/api/` to the backend on the same origin, so the session cookie stays
first-party).

1. Create the environment file from the template and fill in every required
   value. The containerised database additionally needs `DB_ROOT_PASSWORD`:

   ```bash
   cp .env.example .env
   ```

   Set `COOKIE_SECURE=false` for a local HTTP stack; production behind HTTPS must
   keep it `true`.

2. Build and start:

   ```bash
   docker compose up -d --build
   ```

3. Apply migrations against the compose database (migrations never run on boot):

   ```bash
   docker compose run --rm backend migrate up
   ```

The API listens on `http://localhost:8080` and the client on `http://localhost`.
`VITE_API_BASE_URL` is a build-time value (default `/api/v1`); changing it
requires rebuilding the `frontend` image.

For a non-containerised host, `deploy/ryze-backend.service` is a hardened
systemd unit template. Install the binary at `/opt/ryze/ryze-api`, the
environment file at `/etc/ryze/ryze.env` and create the `ryze` user. The unit
sends `SIGTERM` and allows 30s for in-flight requests to drain
(`SERVER_SHUTDOWN_TIMEOUT` defaults to 20s).

### Health and version

Monitor `GET /api/v1/health`: `200` with `{"status":"ok"}` when the database is
reachable, `503 SERVICE_UNAVAILABLE` otherwise. `GET /api/v1/version` reports the
version the image was built with; inject it at build time so it matches the
release notes:

```bash
docker build --build-arg VERSION=2.89.0 -t ryze-api ./main/src/backend
```

### Backups

`scripts/backup_db.sh` writes a consistent, compressed snapshot to `backups/`
(override with `BACKUP_DIR`) using the connection settings from `.env`. It
supports optional GPG encryption via `BACKUP_GPG_RECIPIENT` and local retention
via `BACKUP_RETENTION_DAYS`.

Because the compose database is not published to the host, dump it through the
container instead:

```bash
docker compose exec -T db mariadb-dump \
  --user="${DB_USER}" --password="${DB_PASSWORD}" \
  --single-transaction --routines --triggers --events "${DB_NAME}" \
  | gzip -9 > "backups/${DB_NAME}-$(date -u +%Y%m%dT%H%M%SZ).sql.gz"
```

Restore (destructive, refuses to run without `--yes`):

```bash
scripts/restore_db.sh backups/ryze-20260101T000000Z.sql.gz --yes
```

`security.MD` requires backups to be encrypted before storage, kept off the
production host, and validated by periodic restoration tests. Validate a backup
by restoring it into a throwaway database first.

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
   purchase. `checkout.session.async_payment_succeeded` is handled the same way
   for delayed payment methods.

## MB WAY Testing with Stripe

MB WAY is served by the same Stripe provider and the same hosted Checkout
Session as card, so it uses the same configuration and the same CLI forwarding
command. There is no additional RYZE environment variable: `mbway` is advertised
whenever Stripe is completely configured.

One account-level Stripe setting is required, and it is not a RYZE setting:

1. Enable the MB WAY payment method on the Stripe account
   (Dashboard → Settings → Payment methods → MB WAY). The account must be
   eligible; the capability is listed as `mb_way_payments`.
2. RYZE cannot read that setting, so if it is missing Stripe rejects the Checkout
   Session and the purchase fails closed without ever completing. Card payments
   keep working.

To test, pick a program priced between 0.50 EUR and 5,000 EUR in EUR (the range
Stripe documents for MB WAY; RYZE enforces it server-side from the immutable
purchase snapshot) and choose MB WAY in the checkout selector. The buyer is sent
to the Stripe-hosted page and then approves the payment in the MB WAY app, which
is where the phone number is collected. The purchase completes from the same
`checkout.session.completed` /
`checkout.session.async_payment_succeeded` events as card.

## Payment Testing with PayPal Sandbox

PayPal sandbox webhooks can be configured in the PayPal Developer Dashboard:

1. Create a sandbox application at https://developer.paypal.com
2. Set `PAYPAL_CLIENT_ID`, `PAYPAL_SECRET`, `PAYPAL_MODE=sandbox`, plus
   `PAYPAL_RETURN_URL` and `PAYPAL_CANCEL_URL` pointing at the checkout page in
   `.env`
3. Create a webhook in the PayPal dashboard pointing to your local server
   (use ngrok or similar for local tunneling):

```bash
ngrok http 8080
```

4. Set the webhook URL in the PayPal dashboard to
   `https://<ngrok-id>.ngrok.io/api/v1/webhooks/paypal`
5. Copy the webhook ID and set `PAYPAL_WEBHOOK_ID` in `.env`

After approving the order in the sandbox, the buyer is redirected to
`PAYPAL_RETURN_URL` with the order token; the checkout page calls the capture
endpoint, which verifies and captures the order server-side. The configured
webhook also receives `CHECKOUT.ORDER.APPROVED` and captures the order too —
whichever path completes first wins, the other is idempotent.

## Testing Without Real Providers

When Stripe is not completely configured (missing `STRIPE_SECRET_KEY` or
`STRIPE_WEBHOOK_SECRET`) and `PAYPAL_CLIENT_ID` is empty, no payment method is
advertised, and any payment initiation request fails closed with a generic
`502 PAYMENT_PROVIDER_ERROR`. This is the default state and is safe for
development without real provider accounts.

`mbway` is a reserved enum value with no provider: it is never advertised, never
routed, and always fails closed.

---

## Premium Level 1 (Local Walkthrough)

Premium Level 1 needs migrations `000025` and `000026` and a published Premium
Level 1 program before the page has anything to show.

### Apply the migration

```text
cd main/src/backend
go run . migrate up
```

Verify the schema version afterwards. The expected result is version `26` with
`dirty = false`:

```text
cd main/src/backend
go run . migrate status
```

### Create a Premium Level 1 program

The `/services/premium-level-1` page reads real product data, so it renders an
unavailable state until a program in the `premium_level_1` family is published.
Create and publish one as the Management Administrator (ADMIN_2):

1. Log into the administration console and open **Plans** (Programs → Plans).
2. Switch to the **Premium · Level 1** family tab.
3. Select **New plan**. The plan workspace opens in the Premium Level 1 family.
   Choose the **Premium plan** pricing category, set the price, build the weeks,
   training days and exercises, and publish it.

The Premium Level 1 family is administered through `/api/v1/programs/premium-level-1`:
the family is derived from the route alone and the request payloads carry no
product family field, so a client can never choose which family a plan lives in.
Generic management stays scoped to `trainer_id IS NULL` **and**
`product_type = 'generic'`, so it can neither create nor edit a Premium Level 1
program, and the family is locked once the plan exists. After publishing, the
program surfaces on `/services/premium-level-1` through the existing
`premium_level_1` catalog scope.

### Walk the flow

1. Open `/services/premium-level-1`. The product name, description, price, currency
   and duration come from the backend.
2. Select **Start my questionnaire**. The form renders from the server catalog; the
   client holds no hardcoded field list.
3. Submit a complete intake, then continue to checkout.
4. Complete the purchase. In a Test Mode session this needs no provider
   configuration and no real money.
5. The questionnaire becomes read-only immediately after the purchase. Confirm that
   resubmitting answers returns `409 QUESTIONNAIRE_LOCKED`.
6. The purchase itself attempts plan generation, so a completed purchase normally
   shows a `completed` assignment with a full plan. The generate endpoint is still
   the on-demand path: it is idempotent (repeat runs return the same plan), retries
   a `failed` assignment in place, and is the way to confirm a revision-moved intake
   supersedes the delivered plan.
7. Open **My Programs**. The entitlement carries the Premium Level 1 chip, and
   Program Access renders the training structure plus the nutrition panel.

### Verify without a client

Every step is observable through the API alone. With a valid session:

```text
GET  /api/v1/me/questionnaire/questions
GET  /api/v1/me/programs/{programID}/questionnaire
POST /api/v1/me/programs/{programID}/questionnaire
GET  /api/v1/me/programs/{programID}/nutrition
POST /api/v1/me/programs/{programID}/nutrition/generate
```

The requirement response reports `locked`. Before a purchase it is `false`; after
any completed purchase, regular or Test Mode, it is `true` and every further
mutation is refused.

Checkout against a Premium Level 1 program without a submitted intake answers
`409 PURCHASE_PREREQUISITE_NOT_MET`.

### Provider-dependent steps

Steps that require a real payment provider cannot be exercised locally without
credentials for that provider. Everything in the Premium Level 1 journey up to and
including a Test Mode purchase is provider-free; a real card, MB WAY or PayPal
purchase additionally requires the corresponding provider configuration described
above.
