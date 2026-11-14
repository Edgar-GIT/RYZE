# RYZE

Version: 2.103.0

---

# Overview

RYZE is a fully online fitness platform designed to automate the creation, distribution and sale of training and nutrition programs.

The platform aims to remove the complexity of getting started in fitness by providing structured, accessible and personalized guidance through a modern digital experience.

Rather than replacing personal trainers, RYZE provides a platform where trainers can scale their work, reach more clients and gradually benefit from automation powered by Artificial Intelligence.

The project is built around one core principle:

> Make professional fitness guidance accessible, simple and automated.

---

# Mission

RYZE exists to make professional fitness guidance available to everyone, regardless of experience.

By combining technology, automation and fitness knowledge, the platform reduces the time and complexity normally required for someone to start training with confidence.

---

# Vision

The long-term vision is to become a modern fitness ecosystem where users can
easily discover, purchase and follow high-quality training and nutrition programs
while certified trainers can publish and manage their own content.

Artificial Intelligence will progressively automate repetitive tasks, allowing
trainers to focus on quality rather than manual work.

The public `/our-vision` page communicates this direction as a technology-first
vision surface focused on the platform and roadmap, not founder biographies.

---

# Current Status

The project is under active development. The public visual frontend, the
authentication, database migrations, purchase flows and the ADMIN_1 / ADMIN_2
administration consoles are implemented; ADMIN_2 manages platform-owned Generic
Programs end to end (create, list, search, open, edit and soft delete) against
the generic program API.

The Generic Programs Marketplace displays real published Generic Program data
and its current sale price. Search, filtering, sorting and pagination are
server-side. The public contact form delivers through a configured SMTP relay;
when delivery is not configured it reports that honestly instead of accepting
the message. Paid programmes (Premium Level 1 and 2, and trainers' premium
and personalized plans) are fully purchasable end to
end: the checkout UX lets an authenticated user create a purchase, pick one of
the payment methods the backend currently supports (advertised by
`GET /api/v1/payments/methods`, never hard-coded in the frontend), pay through
the provider checkout flow and gain access once the payment is captured and
verified server-side. The chosen method is recorded on the purchase at
initiation — only after the provider accepts the initiation, so a provider
rejection leaves the method unbound and the buyer can retry with a different
method — and the later capture always resolves the correct provider without
trusting the browser. Stripe serves the
card and MB WAY methods through hosted Checkout Sessions and PayPal serves the
PayPal method; each provider is enabled only when it is completely configured.
MB WAY is offered through the same Stripe provider as card and therefore appears
and disappears with it. Purchase
creation snapshots the current price and currency, applies zero commission for
platform-owned content, blocks duplicate completed purchases, and exposes the
user's purchases. A pending purchase is payable only within its payment window
(24 hours, derived from its creation timestamp): once expired it cannot be paid,
and the next intent for the same program supersedes it (marks it `failed`) so
the checkout can start fresh. Payment completion is always authoritative on the
backend — a browser redirect never equates to a payment, only a verified
server-side capture or a verified payment provider webhook can complete a
purchase, and such a completion is never discarded for arriving after the window.
A pending purchase for a program that is no longer published is never re-opened
for payment.

Late settlements are reconciled, never discarded. A provider-verified payment
that arrives after the purchase left `pending` (a Supersede marked it `failed`,
or the payment was captured on the return leg while a completion raced ahead) is
honoured: a `failed` purchase is atomically returned to `completed` with its
entitlement, and the stale pending intent for the same `(user, program)` pair is
superseded inside the same transaction — but only when no other completed
purchase or active entitlement already resolved the pair, which is enforced by
two database uniqueness guards. When access was already granted, the late
payment cannot define a second sale and the `failed` record is retained as the
persisted, diagnosable recovery state; a verified approval that never moved any
money is never captured. No event can revert a completed purchase and no refund
workflow exists.

Every verified webhook delivery is durably recorded in
`purchase_reconciliation_events` with an immutable outcome code (visibility for
the Management Administrator only, never a client surface), so provider retries
are both honest and idempotent and the settlement trail survives without relying
on logs. Permanent mismatches and Test Mode bounds are answered 200 and
recorded — the provider stops redelivering and the diagnostic is preserved.

---

# Team

RYZE is currently developed by two founders.

## Edgar

Responsible for:

- Software Architecture
- Backend Development
- Frontend Development
- Infrastructure
- Database
- Security
- DevOps

Technical decisions are primarily led by Edgar.

---

## Sandro

Responsible for:

- Fitness
- Nutrition
- Marketing
- Advertising
- Business Strategy

Business decisions are discussed and approved jointly.

---

# Target Audience

RYZE is designed for anyone interested in improving their physical condition through structured training.

Although the platform is suitable for every fitness level, its primary audience consists of beginners and intermediate users who need clear guidance and motivation.

The initial marketing strategy focuses mainly on women between the ages of 20 and 40.

Since the majority of users are expected to access the platform using smartphones, the entire experience is designed following a Mobile First approach.

---

# Business Model

RYZE generates revenue through the sale of digital training and nutrition packs.

RYZE currently offers three different program categories, each designed for a different level of personalization.

## RYZE Free

The Free Program is designed to help new users get started with structured training at no cost.

Users can browse a catalog of generic training packs and activate one program at a time.

Each program is pre-built and intended for common goals such as muscle gain, fat loss or general fitness.

Features include:

- One active training program
- Generic workout program
- Exercise demonstrations
- PDF export
- No nutrition program
- No personalization
- No coach interaction
- Limited access duration

---

## RYZE Premium

Premium Level 1 combines a structured training program with a nutrition program. It supports two delivery models:

- **Curated** — the package ships a predefined nutrition plan authored with the package. Packages are grouped by objective (Gain muscle, Lose fat, Strength & conditioning), and every buyer of a package receives the same expert-defined plan.
- **Questionnaire-generated** — the legacy model. Before purchasing, the user completes a short nutrition questionnaire describing their goals and basic personal information, and the RYZE Recommendation Engine derives a nutrition programme from those answers.

In both models the workout and nutrition sides are delivered immediately after purchase, without waiting for manual preparation.

Features include:

- Structured training pack
- A curated package nutrition plan or an automatically generated nutrition programme
- Immediate delivery
- Exercise demonstrations

---

## RYZE Elite

Premium Level 2 provides the highest level of personalization available on the platform.

After purchasing the program, the client completes a comprehensive questionnaire covering every relevant aspect of their physical condition, lifestyle and objectives.

Information may include:

- Personal goals
- Daily routine
- Previous training experience
- Nutrition habits
- Allergies
- Medical conditions
- Injuries
- Sleep quality
- Stress levels
- Available training time
- Equipment availability
- Additional lifestyle information

Initially, every Premium Level 2 program is manually created by a certified trainer, ensuring that both the training and nutrition programs are specifically tailored to that individual.

As the platform evolves, this process may gradually incorporate Artificial Intelligence to assist trainers in generating personalized recommendations while maintaining professional oversight.

Since every program is created individually, delivery is not immediate.

Clients are informed of the estimated preparation time and receive access to their complete training and nutrition pack once it has been finalized.

The implemented flow (v2.103.0) is purchase first, then a comprehensive
server-catalogued intake, then trainer allocation by ADMIN_2, then
trainer-authored training and nutrition, then delivery. The offer itself carries
no training structure: the delivered programme and nutrition plan belong to the
client and are authored per request. The intake is editable and versioned until
a trainer is allocated; after allocation it becomes read-only.

Features include:

- Fully personalized workout program
- Fully personalized nutrition program
- Manual trainer review and authoring
- Individual program creation
- Versioned intake with resumable editing before allocation
- Status tracking from intake to delivery
- Exercise demonstrations
- Highest level of customization---

# Marketplace

RYZE includes a marketplace where certified trainers can publish their own training and nutrition packs.

Before joining the platform, every trainer must complete a validation process that may include:

- Curriculum Vitae
- Professional Certifications
- Identity Verification
- Manual Approval

Published packs are subject to a platform commission.

The commission structure may evolve as the platform grows.

Prices are fixed by category and nobody sets them (owner decision,
2026-11-14): Generic plans are free, Premium Level 1 costs 14.99 EUR and
Premium Level 2 costs 19.49 EUR. A trainer's plan sells at the price of its
type — free, premium (the Level 1 price) or personalized (the Level 2 price).
What administrators decide is the commission each trainer pays when one of
their plans is sold, typed as a plain percentage (30 = 30%).

The default platform commission on trainer-owned sales is 30% (3000 basis
points; owner decision, 2026-11-11). A trainer-specific rule set by the
Management Administrator replaces it; platform-owned content carries no
commission, and every completed purchase keeps the commission it was sold
with.

The first marketplace surface is the Generic Training Plans Marketplace: a
public catalog of published, platform-owned Generic Programs
(`scope=generic`) with live search, server-side filtering/sorting/pagination and
a per-program detail page. Generic programs are free (v2.114.0); paid programmes are purchasable end to end from their detail
page (see Current Status); pricing display, purchases, payments and
entitlements are implemented and backend-authoritative.

Trainer-owned programs (v2.111.0): a trainer's published generic programs are
discoverable through the "Featured coach programs" zone at the top of the
training plans marketplace — trainer programs ranked by views, each card
showing its coach — and through a public coach profile (`/coaches/:id`) that
lists every program the coach published. A view is one open of the program
page, counted at most once per viewer every 30 minutes and never for the
coach's own visits; programs with equal views follow newest first, so a new
program is visible from day one. A coach is public only once they publish;
only the registered name is shown. The platform-owned catalogue below the
zone is unchanged (`scope=generic`).

Purchased programs appear in the authenticated client "My Programs" area
(`/services/my-programs`): a chrome-free listing of the user's entitlements
served by `GET /api/v1/me/entitlements`, with each program opened through the
entitlement-protected structural read
(`GET /api/v1/me/programs/:programID`, "My Programs" detail). Access to that
read requires an active entitlement. A completed entitlement remains valid after
the program is unpublished; a missing entitlement and a soft-deleted or
nonexistent program are indistinguishable `404`s.

---

# User Roles

The platform currently defines four user roles.

## Client

Clients can:

- Purchase packs
- Follow training programs
- Track progress
- View statistics
- Participate in the community
- Manage their personal profile

The member area (`/profile`, `/services/my-programs`, `/services/progress`,
`/services/workout-history`, `/account/purchases`, `/account/settings`,
`/support`) is one app for desktop and phone: `/profile` is the member's
dashboard (next workout, week, totals, activity, current plan, recent workouts
and personal bests from real data), one shared app shell provides the
navigation, and every client's purchases are listed with amount, date, status
and a server-verified "Open program" action (see `docs/PAGES/design.MD`,
"Member Area").

---

## Trainer

A trainer is an extension of an existing user account: approving a trainer
application creates a trainer profile linked to the applicant's **same** user
identity (same UUID, email and password) — it never creates a second account.

Trainers can:

- Create packs
- Edit packs
- Publish packs
- View statistics
- Manage clients
- Access trainer dashboards

Trainer applications are created by authenticated users through
`POST /api/v1/trainer/apply` and reviewed by both administrators. The applicant
resolves their own state through `GET /api/v1/trainer/access`, which is the
single source of truth for "is this user a trainer" and for the latest
application outcome. The trainer web area (`/trainer/*`) is built on these
endpoints and on the existing trainer, client and program routes; it never
infers trainer status client-side.

Client management is a first-class trainer capability (under `trainer.clients`):
an active trainer can list their clients, resolve an exact email to confirm the
account, add or remove the relationship with any active user, assign one of
their own programmes, and read the client's recorded training overview, session
history and exercise performance. The relationship has its own lifecycle and is
independent of the user account and the trainer profile. Training insight is
read-only and derived from the client's real access grants, statistics and
immutable session snapshots; it never exposes nutrition, questionnaire answers,
medical data or payments (see the Trainer Clients sections in `database.MD` and
`api.MD`).

---

## Management Administrator

Management administrators are responsible for the operational side of the platform.

Responsibilities include:

- Managing trainers
- Managing clients
- Approving trainer applications
- Managing marketplace content
- Viewing business statistics
- Creating and editing platform packs
- Viewing platform sales (commerce)

The Management Administrator holds a read-only view of user accounts (list and
view, under `users.read`). The full user lifecycle (create, update,
soft-disable, reactivate and administrative password reset, under
`users.manage`) belongs to the Technical Administrator.

Trainer application review (list, view, approve and reject, under
`trainer_applications.read` / `trainer_applications.manage`) is implemented as a
backend foundation shared by **both** administrators, matching the operational
responsibility of approving trainer applications.

The admin sales view (list purchases, purchase details, per-program sales and
reconciliation events under `GET /api/v1/admin/purchases*`) is implemented as a
backend foundation exclusive to the Management Administrator under the
`commerce` permission. It is read-only and reports revenue from completed
non-test purchases; Test Mode purchases are accounted separately. The Technical
Administrator holds no commerce permission. The reconciliation events surface
(`GET /admin/purchases/reconciliation-events`) is visibility-only: it shows the
durable, immutable outcome each verified webhook delivery recorded, with an
"open" scope that surfaces only the settlements that still need attention. It
can never modify a purchase, entitlement or event — recovery stays server-side.

The Management Administrator answers the support chat (owner decision,
2026-11-11; `support.read` / `support.manage`, not held by the Technical
Administrator). Trainers, and clients with an active programme (paid or Free),
open conversations with RYZE support from `/support`; replies are signed
"RYZE support". Support is not coaching and never connects a client with a
trainer.

The generic program management workflow (plans list, plan detail) reads its
per-program sales exclusively from the same commerce surface: the plans list
batches `GET /admin/purchases/programs` with `program_ids` to show Price, Sales,
Test and Revenue per plan, and the plan detail page shows a compact Sales
section with a "View purchases" deep link to the Sales dashboard. Plans with
existing sales surface a warning before unpublishing, because unpublishing
revokes client access to the programme (their paid entitlement is preserved,
and access is restored when the plan is published again).

---

## Technical Administrator

Technical administrators have unrestricted access to the platform's technical side.

They are responsible for:

- Platform maintenance
- Infrastructure
- Security
- User management
- Software development
- Database administration

The Technical Administrator does not act as a super administrator: management
business responsibilities (plans, finance, marketing) belong exclusively to
the Management Administrator and are never granted automatically to the
Technical Administrator.

User management (full lifecycle: create, update, soft-disable, reactivate and
administrative password reset) is implemented as a backend foundation under the
`users.manage` permission and is exclusive to the Technical Administrator. The
Management Administrator keeps a read-only view under `users.read`.

Trainer application review is a shared responsibility: both administrators hold
`trainer_applications.read` and `trainer_applications.manage` and can list,
view, approve and reject applications.

---

# Platform Features

The platform currently includes the following core features:

- Training programs
- Nutrition programs
- Exercise library
- Progress tracking
- Marketplace
- Purchase history (client order history with status, Test Mode marker,
  server-verified access states inside the member app shell)
- User authentication
- PDF export
- Notifications
- Community system
- Statistics dashboard
- Multi-language support
- Gamification
- Admin Test Mode (see below)

Additional features will continue to be introduced as the platform evolves.

## Admin Test Mode

Test Mode lets a Technical Administrator impersonate one of two predefined
personas (Test Client, Test Trainer) to exercise the customer commerce flows
without real payments. It is a development/test companion, not a marketing or
sales tool.

- **Entry is exclusive to the Technical Administrator** (ADMIN_1), enforced
  server-side.
- Inside Test Mode every purchasable program (Generic or Premium Level 1)
  renders as Free, and buying completes a real zero-cost purchase + entitlement
  server-side — no payment provider, redirect or capture is involved. The same
  checkout prerequisites still apply, including the legacy Premium Level 1
  questionnaire. A curated package has no questionnaire prerequisite.
- Identity is entirely server-side: the backend mints the persona session
  cookie and restores the administrator identity on exit.
- The feature is globally controlled by `TEST_MODE_ENABLED` (default false) — a
  single kill switch that disables the whole experience.
- Frontend surfaces: a persistent banner with an Exit control and a
  Technical-navigation admin page to enter/exit with a configurable return
  path.

Test Mode is intentionally scoped: it does NOT create real money movement,
does not affect live data outside the persona's own purchases, and cannot be
used by Marketing/Front-desk roles.

---

## RYZE Feed

RYZE Feed is a vertical, short-form video feed designed to provide users with continuous educational and fitness-related content from RYZE trainers and creators.
Its not just another page like instagram, its for educational videos.
The experience is inspired by short-form video platforms such as Instagram Reels, but is intentionally focused on fitness, training, nutrition, recovery, and related educational content.

### Core Experience

- Users browse videos through a vertically scrollable feed.
- Videos are presented one at a time and optimized for visual consumption.
- The feed is primarily discovery-oriented rather than a traditional social network.
- Trainers can publish educational and informative fitness content.
- Content may cover topics such as:
  - Training techniques
  - Hypertrophy
  - Exercise execution
  - Nutrition
  - Recovery
  - Fitness education
  - General training advice
- Users can like videos.
- Comments are not supported.
- RYZE Feed does not provide a user-facing viewing history.
- The experience does not depend on traditional post feeds or long-form social interactions.

### Trainer Following

Users can follow trainers.

Following a trainer influences feed personalization, increasing the frequency at which content from followed trainers is presented to the user.

Following does not restrict the user to only that trainer's content. The feed may continue to include content from other trainers and creators.

### Feed Personalization

RYZE Feed should progressively personalize content based on user interactions.

Relevant signals may include:

- Trainers followed by the user
- Videos liked by the user
- General interaction with content
- Content categories and topics
- Trainer/content relevance

The exact recommendation algorithm is an implementation detail and may evolve independently from the product specification.

### Content Ownership

RYZE Feed content is published by authorized trainers or other roles explicitly permitted by the platform.

Each video belongs to its publishing creator and must maintain a relationship with that creator for attribution, management, and future personalization.

### Product Constraints

RYZE Feed intentionally excludes:

- Comments
- Public discussion threads
- User-created posts
- User-facing viewing history
- Traditional social-media timelines
- Direct messaging between users and trainers through the feed

The initial goal is to provide a focused educational content experience rather than reproduce a complete social network.

---

# Community

RYZE includes a community area available to users who have previously acquired at least one training pack.

The community encourages motivation and engagement through:

- Posts
- Images
- Comments
- Progress sharing

Certified trainers may also participate in community discussions.

The community is not intended to replace professional coaching.

---

# Gamification

Consistency is one of the main objectives of the platform.

To encourage long-term engagement, RYZE incorporates several gamification systems, including:

- Achievement badges
- Streaks
- Missions
- Events
- Progress notifications
- Milestone rewards

These systems are designed to improve user motivation without interfering with the training experience.

---

# Artificial Intelligence

Artificial Intelligence represents one of the most important long-term components of the project.

Initially, AI will support internal development.

Future releases will integrate AI directly into the platform, allowing automatic generation of personalized training and nutrition programs based on user questionnaires.

The long-term objective is to significantly reduce manual work while maintaining high-quality recommendations.

---

# Internationalization

RYZE is designed for an international audience.

The initial release supports:

- Portuguese
- English

Additional languages may be introduced in future versions.

Language rule (owner decision, 2026-11-11): the first visit chooses
Portuguese in Portugal and English elsewhere, detected in the browser (a
Portuguese time zone — Europe/Lisbon, Atlantic/Madeira, Atlantic/Azores — or a
`pt-PT` browser language; no external location service). A manual choice is
stored and always wins, so the language never changes on its own between
visits; signed-in accounts also save it, and it sets the language of their
emails. Dates and numbers follow the language (`en-GB` / `pt-PT`).

Translation status (v2.109.0): account recovery, the account area (profile,
My Programs, programme access, workout sessions and history, progress,
purchases, trainer application), notifications, support and the PWA prompts
are translated. The public website, the protected login and registration
copy, the Premium Level 1/2 service pages, the trainer area and the
administration consoles are still English-only pages (consistent, not mixed)
and are pending a translation pass with owner-reviewed Portuguese copy.
Content written by trainers or generated by the server (programme names,
exercise names, nutrition plans) is shown as stored.

---

# Design Philosophy

RYZE follows a modern, clean and futuristic design language.

The public website uses one official visual system. The Hero section is the
source of truth. Every other section must share the same near-black / electric
cyan palette and token set so the product feels coherent end to end.

The platform prioritizes:

- Simplicity
- Accessibility
- Performance
- Mobile First
- Consistency
- Fast navigation

All visual guidelines are documented in:

- `docs/INSTRUCTIONS/design.MD` — palette, tokens and consistency rules
- `docs/PAGES/design.MD` — page composition and component behaviour

---

# Technical Documentation

This document intentionally focuses on the business and functional aspects of the project.

All technical decisions are documented separately.

- `architecture.MD` — Technologies, frameworks, project structure and infrastructure.
- `docs/INSTRUCTIONS/design.MD` — Official palette, CSS tokens and visual rules.
- `docs/PAGES/design.MD` — User interface, branding, user experience and page specifications.
- `procedure.MD` — Development workflow, coding rules and AI instructions.
- `update_logs.MD` — Complete history of project modifications.

---

# Future Vision

RYZE is designed to evolve continuously.

Future releases may introduce:

- AI-generated programs
- Advanced marketplace features
- Additional payment providers
- Two-Factor Authentication
- Improved trainer tools
- New gamification systems
- Expanded analytics

Every new feature should remain aligned with the project's original philosophy:

> **Make professional fitness guidance accessible, automated and simple for everyone.**

---

# PREMIUM LEVEL 1

Premium Level 1 is a complete training and nutrition package sold once. It is
defined by two delivery models rather than by a single nutrition mechanism:

> A client chooses a training and nutrition package, buys once, and receives both
> halves immediately. A curated package ships a fixed nutrition plan authored for
> its objective; a legacy package derives a personalised plan from a questionnaire
> answered before checkout.

## What it includes

- A structured training plan, already part of the existing program hierarchy.
- Curated mode: a predefined package nutrition plan with a daily meal structure,
  portions, macronutrients, hydration guidance and cautions. Every buyer of the
  same package receives the same content.
- Legacy questionnaire mode: a nutrition programme generated on the server from
  the stored questionnaire, delivered as part of a completed purchase and
  persisted as a normalised, versioned plan (every delivered version is immutable
  and retained).
- Daily energy and macronutrient targets and a hydration target in both modes.
- Legacy mode: dietary rules built around the restrictions the client reported.
- Curated mode: the package's own predefined dietary pattern and cautions; there
  is no intake and therefore no per-client restrictions audit.

## What it is not

- It is not human coaching. No trainer reviews an intake, and nothing in the
  product promises one-to-one guidance.
- It is not a medical or clinical assessment. The delivered plan carries the
  server's or package's own cautions and the interface repeats that it is not
  medical advice.
- It is not a subscription. Entitlements do not expire.
- It is not an AI chat product. In v2.101.0 the legacy generator is a deterministic
  server-side function, which is what makes the plan reproducible and the
  questionnaire safely immutable; curated content is authored package content.

## The questionnaire is part of the legacy product

The intake exists only for a questionnaire-generated package. It is collected
before checkout because it is an input to that product, not a form for its own
sake. It is editable until the purchase and immutable after it, because the
delivered nutrition programme is derived from it: changing the answers would
silently change a plan the client already received. This is a product decision,
not a technical limitation, and the interface states it plainly instead of hiding
it behind a disabled control. A curated package has no intake and goes directly
from product to checkout.

## Delivery

Premium Level 1 uses the existing purchase, payment and entitlement flow. It adds
no payment method, no provider, no currency handling and no refund behaviour, and
it inherits every invariant established for v2.80.0. A curated plan is synthesized
from the package after the entitlement check; a legacy plan is generated after the
purchase. An entitlement keeps access after a package is unpublished.

## Scope discipline

Premium Level 2 is implemented as a separate product family
(`programs.product_type = 'premium_level_2'`). It reuses the existing catalogue,
purchase, payment, entitlement and access systems; it does not introduce a second
checkout or a parallel program hierarchy. A Premium Level 2 offer is a published
platform-owned program with no weeks or nutrition content. Purchase fulfilment
creates one service request. The client completes a detailed intake, ADMIN_2
allocates an active trainer, and that trainer attaches one of their own draft
programmes and authors the nutrition plan before delivery. Delivery creates the
training entitlement and releases the nutrition plan to the client.

Premium Level 3 and trainer-authored nutrition sold outside this service remain
out of scope. The intake answers are sensitive health data and are exposed only
to the owning client, the allocated trainer and authorised administrators.
