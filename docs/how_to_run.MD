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

`/about-us` redirects to `/our-vision`.

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
