# RYZE FITNESS
# ARCHITECTURE.MD

Version: 2.1.0

---

# PURPOSE

This document defines the complete technical architecture of the RYZE platform.

It serves as the single source of truth regarding the project's technologies,
software architecture, folder organization, coding standards, infrastructure
and development guidelines.

The architecture is expected to evolve throughout the project's lifetime.

Whenever a better solution is identified, this document should be updated
before implementation.

---

# DESIGN PRINCIPLES

Every technical decision should prioritize:

- Simplicity
- Maintainability
- Scalability
- Performance
- Security
- Readability
- Reusability
- Future expansion

Avoid unnecessary complexity.

Whenever multiple solutions exist, prefer the one that is easier to maintain.

---

# PROJECT STACK

## Frontend

- React
- TypeScript
- Vite
- React Router
- Zustand
- CSS Modules
- Lucide React
- Framer Motion
- React Hook Form
- Zod

The frontend is developed as a Progressive Web Application (PWA).

Users can install the website as a desktop or mobile application.

No native mobile application is planned for the initial release.

---

## Backend

- Go
- Gin
- GORM
- Gorilla WebSocket
- golang-migrate

The backend exposes a REST API.

API Version:

/api/v1

Communication format:

JSON

---

## Database

Primary database:

MySQL

SQLite may be used during local development when appropriate.

Database documentation is maintained separately inside:

docs/DATABASE/database.MD

---

## Authentication

Supported methods:

- Email + Password
- Google OAuth

Passwords are never stored in plain text.

JWT is used for authentication.

Password recovery is performed via email.

The architecture should remain compatible with future two-factor authentication.

Administrator authentication uses the same JWT infrastructure with two
config-only factors (username/password, then an access code). Administrators
are defined exclusively in configuration (`.env`), never in the database. The
second factor is completed through a short-lived temporary token kind before
the admin session cookie is issued. See `security.MD` (Administrator
Authentication) and `api.MD` (Admin Authentication).

---

## Email

Provider:

Resend

---

## Realtime

Real-time communication uses:

Gorilla WebSocket

Current use cases include:

- Notifications
- Chat

Additional realtime features may be added in future versions.

---

## State Management

Frontend:

Zustand

Backend:

Stateless REST architecture.

---

# PROJECT STRUCTURE

```
main
└── src
    ├── backend
    ├── frontend
    ├── resources
    └── utils
```

The project structure is intentionally simple.

Folders may evolve as the project grows.

AI assistants are allowed to create new folders whenever they improve
organization or scalability.

Every new folder must:

- Have a descriptive name.
- Follow the existing architecture.
- Avoid unnecessary nesting.
- Represent a single responsibility.
- Never duplicate responsibilities already present elsewhere.

---

# BACKEND STRUCTURE

Example structure:

```
backend
├── ai
├── api
├── auth
├── config
├── constants
├── database
├── email
├── logs
├── middleware
├── models
├── routes
├── scheduler
├── services
├── uploads
└── websocket
```

Each folder has one responsibility.

Folders may be expanded whenever necessary.

---

# FRONTEND STRUCTURE

The frontend should remain modular.

Typical folders include:

- Components
- Pages
- Layouts
- Hooks
- Services
- Contexts
- Assets
- Styles

The exact structure may evolve as the project grows.

Current frontend implementation:

```
frontend
└── src
    ├── components
    ├── constants
    ├── hooks
    ├── pages
    ├── routes
    └── styles
```

Important public UI building blocks currently include:

- layout / navbar / footer / loading screen
- hero / showcase sections / product demos
- service cards / CTA band / contact experience
- glass panels and shared primitives

Frontend tooling files live at the repository root:

```
index.html
package.json
package-lock.json
tsconfig.json
vite.config.ts
```

The frontend uses React, TypeScript, Vite, React Router, CSS Modules,
Lucide React and Framer Motion as defined in the project stack.

The public visual system is dark-first, with electric cyan (`#2ED8FF`) as the
primary accent and the Hero as the visual source of truth.
Homepage sections prioritize product demos over text-heavy feature grids.
Detailed UI rules live in `docs/INSTRUCTIONS/design.MD` and
`docs/PAGES/design.MD`.

---

# RESOURCES

Static assets belong inside:

resources/

Examples:

- Images
- Videos
- Fonts
- Logos
- Icons

Business logic should never exist inside this directory.

---

# UTILS

The utils folder contains every reusable piece of code.

Before writing any helper, utility or converter,
always verify whether a similar implementation already exists.

Examples:

- Validation
- Formatting
- Unit conversions
- Math
- Helpers
- Constants
- Generic utilities

Never duplicate reusable code.

---

# DATABASE DESIGN

The database follows these principles:

- UUID identifiers
- Soft Delete
- Historical preservation
- Versioning when appropriate
- Auditability
- Normalization
- Explicit relationships

Additional details are defined inside:

docs/DATABASE/database.MD

---

# API DESIGN

The REST API follows:

- JSON only
- REST principles
- JWT authentication
- Versioned endpoints

/api/v1

Endpoint documentation belongs inside:

docs/API/api.MD

---

## RYZE Feed Architecture

RYZE Feed is a dedicated content delivery system within the RYZE platform.

The system is responsible for storing, publishing, retrieving, ranking, and delivering short-form fitness content created by authorized trainers.

The Feed is designed as a modular system so that content delivery and recommendation logic can evolve independently from the rest of the platform.

### Core Components

RYZE Feed consists of the following logical components:

- Content Management
- Video Storage
- Video Processing
- Feed Delivery
- Feed Personalization
- Trainer Following
- Video Likes
- Content Moderation

### Content Flow

Trainer content follows this general lifecycle:

```text
Trainer
   │
   ▼
Create Content
   │
   ▼
Upload Video
   │
   ▼
Validate File
   │
   ▼
Process Video
   │
   ▼
Generate/Store Media
   │
   ▼
Content Review / Validation
   │
   ▼
Publish
   │
   ▼
RYZE Feed

nly published and valid content may be returned to normal Feed users.

Feed Delivery

The Feed should use incremental content loading rather than loading the entire content library at once.

The client requests a limited number of Feed items and requests additional items as the user approaches the end of the currently loaded content.

Conceptually:

Client
   │
   ▼
Feed API
   │
   ▼
Candidate Content
   │
   ▼
Personalization / Ranking
   │
   ▼
Paginated Results
   │
   ▼
Client

The exact pagination mechanism may evolve as the platform develops.

Feed Personalization

RYZE Feed supports personalized content discovery.

The system may consider signals such as:

Trainers followed by the user
Videos liked by the user
Content categories
Trainer relevance
Content popularity
Content freshness
Other future engagement signals

Following a trainer should increase the probability that content from that trainer appears in the user's Feed.

Following must not restrict the Feed exclusively to followed trainers.

The Feed should maintain a balance between:

Followed content
Relevant content
Discovery content

The recommendation algorithm must remain modular and replaceable.

No specific ranking formula should be considered part of the permanent architecture.

Trainer Following

Following is represented as a relationship between a user and a trainer.

User
 │
 └── follows ──► Trainer

This relationship may be consumed by the Feed personalization system.

Following a trainer does not automatically grant access to private trainer resources or paid plans.

It only affects Feed content personalization unless another system explicitly defines additional permissions.

Video Likes

Users may like published Feed videos.

The Like system must:

Allow one active Like per user per video
Allow users to remove their Like
Prevent duplicate Likes
Provide the current Like state to the client
Provide aggregate Like counts where required

Likes may be used as a personalization signal.

Content Ownership

Every Feed video must have a clearly defined creator.

Trainer
   │
   └── owns/publishes ──► Feed Content

Content ownership is required for:

Attribution
Content management
Editing
Removal
Moderation
Future analytics
Content States

Feed content should support a lifecycle rather than being immediately public after upload.

At minimum, the architecture should support states conceptually equivalent to:

Draft
   │
   ▼
Processing
   │
   ▼
Published
   │
   ├──► Hidden
   │
   └──► Deleted

The exact state names may be adjusted during database/API design.

Only appropriate published content should be visible in the public Feed.

Media Processing

Uploaded videos must not be assumed to be ready for immediate playback.

The system should support a processing stage where uploaded media can be:

Validated
Security checked
Transcoded if necessary
Optimized for delivery
Associated with a thumbnail/poster
Prepared for supported playback formats

The processing system should be designed independently from the Feed UI.

Content Moderation

RYZE Feed content must pass the platform's security and moderation requirements before becoming publicly available.

Content moderation and security validation are separate concerns:

Security validation protects the platform and infrastructure.
Content moderation determines whether content is appropriate for publication.

The Feed architecture must allow content to be hidden or removed without requiring physical deletion of all associated records.

Scalability

RYZE Feed should be designed so that increasing content volume does not require loading all videos or all users into memory.

The architecture should support future growth in:

Number of trainers
Number of videos
Number of Feed users
Number of Likes
Number of Follows
Video storage requirements
Feed requests

Feed retrieval, media storage, and recommendation logic should remain independently scalable where practical.

Future Extensibility

The initial RYZE Feed intentionally remains simple.

The architecture should, however, allow future functionality such as:

More advanced recommendations
Content categories
Hashtags/tags
Trainer analytics
Content analytics
Improved video processing
Content moderation tooling
Featured content
Personalized discovery
Additional creator roles

Future functionality must not require redesigning the fundamental relationship between users, trainers, and Feed content.

---

# FILE STORAGE

Uploads are stored locally during the initial phase.

```
backend/uploads/
```

Examples:

- Avatars
- Certificates
- PDFs
- Temporary files

Future cloud storage providers may replace local storage.

The architecture should make this migration straightforward.

---

# ERROR HANDLING

Every error should:

- Be logged
- Return meaningful messages
- Never expose sensitive information

The frontend displays errors using a modal overlay with a blurred background.

Users may dismiss non-critical errors by clicking outside the modal.

---

# LOADING EXPERIENCE

Whenever a page or major resource is loading,
the application displays the official RYZE logo.

Two rotating arrows surround the logo,
inspired by the Windows loading animation.

This loading screen should remain consistent across the entire platform.

---

# SEARCH

Searching is primarily performed through SQL queries.

Indexes should be created whenever necessary.

Future full-text search solutions may be adopted if justified.

---

# PDF GENERATION

PDF files are generated by the backend.

Temporary files may be created inside:

backend/uploads/temp

Temporary files should be automatically removed by scheduled jobs.

---

# SCHEDULER

Background tasks are handled by the scheduler.

Examples:

- Cleaning temporary uploads
- Removing expired notifications
- Updating statistics
- Calculating rankings
- Processing email queues
- Expiring programs
- Scheduled backups

---

# SECURITY

The architecture prioritizes:

- Password hashing
- JWT
- Input validation
- SQL injection protection
- XSS protection
- CSRF protection
- Rate limiting
- Secure file uploads

Security should always take priority over convenience.

---

# PERFORMANCE

The project prioritizes:

- Fast loading
- Lazy loading
- Efficient database queries
- Component reuse
- Small bundle sizes
- Optimized assets

Premature optimization should be avoided.

Optimize only where justified.

---

# CODE STANDARDS

The project follows these conventions:

- English only
- Descriptive naming
- Small functions
- Single responsibility
- Modular architecture
- No duplicated logic

Comments should only explain *why* something exists,
never what obvious code does.

---

# AI DEVELOPMENT

AI assistants are expected to behave as senior software engineers.

They are encouraged to improve:

- Performance
- Security
- Architecture
- Maintainability
- Folder organization
- Database structure

...whenever those improvements do not conflict with documented project decisions.

Whenever a better solution is identified:

1. Update the documentation.
2. Document the change inside `update_logs.MD`.
3. Apply the implementation.

Never introduce breaking architectural changes without documenting them first.

---

# FUTURE EVOLUTION

This architecture is intentionally flexible.

As the project grows, AI assistants may:

- Create new folders.
- Introduce new modules.
- Improve folder organization.
- Refactor code.
- Improve database structures.
- Improve APIs.

Every modification must respect the project's principles and be properly documented.

The goal is to continuously improve the platform while maintaining consistency,
clarity and long-term maintainability.
