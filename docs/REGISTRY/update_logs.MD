# Version 2.134.0

Title: Plan page redesign, trainer profile trimmed, hero in black, silver and chrome

Date: 2026-11-28

Summary:

- A plan in the store (`generic_program_detail_page`) was redesigned (owner:
  "refaz tudo"): the login page's split panel, with the cover (store card
  format, a price tag pinned to it) on the light-blue wave half and the name,
  coach, description, a spec sheet (length, training days, level, where) and
  the way to get it on the navy half; below, "The plan, week by week" as a
  timeline of weeks with the training days as tiles (short plans open in
  full). The free plan panel uses the public CTA and no longer repeats the
  back link. Purchase, Level 1 and archived states are unchanged in
  behaviour.
- Trainer profile: the views block and the "public coach profile" card were
  removed; the two profile forms are centred in one column.
- Home hero: the shaker is black with a grey lid and stands in front of the
  dumbbell's far end; the phone moved further right and forward; the chart is
  smoked black glass with a silver line whose arrow rises in an 8 s loop. The
  phone and laptop are now black, silver and chrome (`story_scene.ts`).

Verification: typecheck, unit tests and build; browser screenshots of the
plan page at 1440 and 390 px (no horizontal scroll), of the hero (four
frames of the chart's rise) and of the later chapters with the laptop.

---

# Version 2.133.3

Title: Home hero without the food plate

Date: 2026-11-28

Summary:

- Owner request: the food plate (and its steam) left the desktop hero, and
  the dumbbell, shaker, phone and progress chart moved further right of the
  copy (`heroX` 0.32 → 0.42; the chart re-centred on the remaining objects).
  The unused plate model and its textures were removed from `story_intro.ts`.

Verification: typecheck, unit tests and build; a browser screenshot at
1376×768.

---

# Version 2.133.2

Title: Desktop Home hero rebuilt on the owner's reference picture

Date: 2026-11-28

Summary:

- The hero picture test is over: the desktop hero is 3D again, after the
  owner's reference image — a round-plate dumbbell in front on the left, the
  RYZE shaker (navy body, teal lid) in the middle, the RYZE phone on its
  right, a plate on the far right and a frosted glass pane with a glowing
  cyan progress chart behind; no floor, a soft shade under each object, a
  near-frontal view. Built procedurally in `story_intro.ts`, reusing the
  plate model and the story's phone (which still flies on into the Train
  chapter); the drone's projection no longer plays (the scene starts at
  `INTRO_END`). Removed: the picture, `?hero=3d`, the napkin, the cutlery and
  the hex dumbbell models.
- Settings: Preferences, Sign-in methods and Privacy moved to the right
  column, under Delete account.

Verification: typecheck, unit tests and build; browser screenshots at
1376×768 compared with the reference (hero, and scrolling into the later
chapters) and at 390 px (no canvas on phones).

---

# Version 2.133.1

Title: Owner review fixes: calmer private areas, plan page layout, coach counters, trainer profile in place

Date: 2026-11-28

Summary:

- Private areas: softer contrasts; the sidebar, the top bar and the plan
  builder's sticky footer use one opaque bar colour (the footer was a black
  bar).
- Member sidebar: the training plan store and the Exercise Library removed
  (they are site pages).
- Free/paid plan page: picture and how to get the plan on the left, the plan
  (weeks and days, or what unlocks) on the right; a Premium Level 1 package
  points to the Level 1 questionnaire instead of a purchase panel.
- Coach public profile: the list of plans replaced by counters (plans
  created, subscriptions, views); the coach profile API returns
  `plans_count` and `subscriptions` (real-DB test).
- RYZE Personalised: the pay button is in the "What you get" block at the
  top (CheckoutPanel `actionOnly`); the separate payment section is gone.
- Trainer profile: photo, nickname, bio and social links are edited in place
  with the account settings' cards.

Verification: Go build/vet and the full suite on `ryze_test` (69 packages);
frontend typecheck, unit tests and build; browser checks of the Personalised
page, a free plan page and a coach profile at 390 px and 1280 px (no
horizontal scroll, no errors). Not verified in the browser: signed-in screens.

---

# Version 2.133.0

Title: Package versions, Level 1 ranges, meal photos and alternatives, livelier private areas

Date: 2026-11-28

Summary:

- Fixed a serious error: the featured programmes of the free marketplace
  listed Premium Level 1 packages; they now list free plans only (regression
  test with a real database).
- Buyers keep what they bought (owner decision): a sold package cannot change
  in place (route guard + checks inside the structure and meal-plan
  transactions, `409 PROGRAM_SOLD`). "Edit" on a sold package creates a new
  version; publishing it replaces the old one on the market, which stays for
  its buyers (migration 000073). A sold package cannot be deleted.
- Level 1 packages (trainer and RYZE) can say who they are for: height,
  weight and age ranges, each "Any" by default, applied by the Level 1
  questionnaire (migration 000074).
- Trainers' older Premium plans converted into Level 1 drafts (migration
  000075).
- Meal plans: a photo per meal and a line of alternatives with an optional
  photo (migration 000076), shown to buyers. Fixed: the trainer's meal photo
  upload needed the clients permission.
- Plan covers use the store card format (4:5) everywhere, previews included;
  the free plan panels show the plan's picture; "Created by" shows the
  coach's photo (initials without one).
- Private areas (member, trainer, admin) use the public site's field of blues
  and light glass (the Contact page model); the programme screen was redone
  (light-blue panel with the cover, glass panel with the actions).
- RYZE Personalised page: the payment section no longer repeats what the
  service includes.

Verification: Go build/vet and the full suite on `ryze_test` three times
(69 packages), new real-DB tests (versions end to end, ranges and versions in
matching, featured programmes) and unit tests (ranges, sold guard);
frontend typecheck, unit tests and build; migrations 000073–000076 applied to
`ryze_test` and the dev database. Not verified in the browser: the signed-in
trainer and admin screens.

---

# Version 2.132.0

Title: Programme archive, trainer Premium = Level 1 package, clearer trainer/admin UI

Date: 2026-11-27

Summary:

- Archive (migration 000072, `programs.archived_at`): a published programme
  can be taken off the market and put back later, by trainers (own
  programmes) and by ADMIN_2 (RYZE Free and Premium Level 1 plans, audited).
  Archived programmes leave the catalogue, store, coach profile, Level 1
  matching, new purchases (Test Mode too) and new free activations; people
  who already have them keep full access. Limit: 15 archived per trainer and
  15 for the platform. New endpoints in `api.MD`.
- Trainer Premium plans are real Premium Level 1 packages (owner decision):
  objective plus a complete curated meal plan (new builder step, saved via
  `PUT /trainer/programs/:id/nutrition`), required before publishing (`409
  NUTRITION_REQUIRED`); they appear on the coach profile, in the store and in
  the Level 1 questionnaire matching (the platform-only filter was removed).
  A published plan cannot switch between Free and Premium. Validation errors
  from the trainer API now carry the reason.
- Plan builder (trainer and ADMIN_2): each step lists what is missing and
  blocks "Next step" until the required fields are filled (name, description,
  package objective, at least one exercise, complete meal plan).
- After publishing, trainer and admin land on their programme list with a
  green confirmation ("Programme published", also for archive, restore and
  delete). Opening a programme shows a simple first screen: cover, name,
  state, and Edit (green), Archive / Put back on the market, Delete, each
  explained. The trainer builder moved to `/trainer/programs/:id/edit`.
  Lists gained an "Archived · n/15" filter.
- Coach public page = member profile (`/u/<username>`) with the "RYZE coach"
  badge, the plans they created and their views; `/coaches/:id` redirects
  there (the coach profile API now returns the username).
- Trainer sidebar regrouped (My work, Programmes, Profile and account) with
  explicit names, plus Create a programme, Archived programmes, See my public
  profile and a new Statistics and earnings page (`/trainer/stats`: sales,
  gross, RYZE commission, net, views per plan, programmes by state/type,
  clients, Level 2 requests). Member and ADMIN_2 sidebars use explicit names
  (phone tabs keep the short ones).
- Fixes found by the new tests: reading a curated meal plan always failed
  (GORM added a `deleted_at` filter on a view without that column), which
  also affected the meal-plan delivery of Premium Level 1 buyers; and a
  flaky admin purchase-totals test raced with two tests that commit
  purchases (now serialised with a test-only advisory lock).

Verification: Go build/vet and the full suite on `ryze_test` (69 packages,
four consecutive clean runs), new real-DB tests for the archive (limit,
state, catalogue exclusion) and the trainer Premium package lifecycle;
frontend typecheck, unit tests (incl. new ProgramOverview tests) and build;
migration 000072 applied to `ryze_test` and the dev database. Not verified in
the browser: the signed-in trainer and admin screens (no session available
to the assistant without the local credentials).

---

# Version 2.131.2

Title: Production readiness: production.MD, deploy templates, CI with PostgreSQL, Docker uploads fix

Date: 2026-11-27

Summary:

- `production.MD` (root, Portuguese): readiness assessment per area,
  infrastructure plan (Cloudflare Registrar/DNS/Pages/R2, OVH VPS, PostgreSQL
  in Docker), server hardening, deploy, backups (RPO/RTO proposals),
  monitoring, every environment variable, costs (three scenarios),
  emergencies, commercial/legal prerequisites and launch checklists.
  Nothing was bought, contracted, published or changed outside the repo.
- New `deploy/compose.app.yml` (one environment: API + PostgreSQL, no public
  ports), `deploy/compose.edge.yml` + `deploy/Caddyfile` (only public
  service; Cloudflare Origin Certificate; client IP only from Cloudflare),
  `deploy/deploy.sh` (pull, backup, migrate, swap, health check, automatic
  API rollback), `deploy/backup.sh` (age-encrypted DB + uploads to R2 with
  daily/weekly/monthly retention) and `deploy/restore.sh` (throwaway restore
  test, database and uploads restore).
- `.github/workflows/ci.yml` now applies migrations and runs the whole Go
  suite on a throwaway PostgreSQL 17 (validated locally on a fresh cluster
  without .env: 69 packages); reusable by the new
  `.github/workflows/deploy.yml` (staging on push to `staging`, production on
  `main` after Environment approval; image to GHCR; Pages after the API).
- Fixes: the Docker image wrote uploads to `/uploads` (not writable, not
  persistent) — now `UPLOADS_DIR=/var/lib/ryze/uploads` in the Dockerfile and
  compose; the local nginx did not proxy `/uploads/`; a name-sort test
  depended on the database collation and would fail on a fresh database.
- `public/_headers` for Cloudflare Pages (security headers with the API
  origins in the CSP, asset caching). `.gitignore` also ignores `.env.*`
  (except `.env.example`). `.env.example` lists every variable the code
  reads; the local `.env` got the missing names, empty (no values touched).

Verification: Go build/vet and the full suite on a fresh PostgreSQL without
`.env`; the backend boots with the updated `.env`; `docker compose config`
for both deploy compose files; workflow YAML parses; frontend build includes
`_headers`; shell scripts pass `bash -n`. Not verified: building the Docker
image (no Docker permission locally), the GitHub workflows (first real run on
GitHub), and anything on a real server.

# Version 2.131.1

Title: Support chat tables dropped

Date: 2026-11-24

Summary:

- Owner decision: migration 000071 drops `support_messages` and
  `support_conversations` (the chat was removed in v2.131.0); account
  anonymisation no longer touches them.

Verification: Go build passes; the whole Go suite passes on `ryze_test`.
Migration 000071 applied to `ryze_test` and the dev database.

# Version 2.131.0

Title: Support removed; Premium Level 2 rework (offers to all, RYZE official, plan workspace, meal photos); health consent; account anonymisation; refunds; discount codes; training reminders

Date: 2026-11-24

Summary:

- Support chat removed (owner decision: RYZE is a self-service experience;
  the Contacts page is the way to reach the company). Removed: `/support`,
  `/admin2/support`, the `support` service/handler/repository/model, the
  `/me/support/*` and `/admin/support/*` routes, its rate limits, admin
  permissions (`support.read`, `support.manage`), audit actions, notification
  kinds, the `support_reply` email and the dashboard counts. The
  `support_conversations`/`support_messages` tables were left in place
  (dropping data needs an explicit decision); anonymisation clears a deleted
  account's rows in them.
- Premium Level 2 (migrations 000064, 000065):
  - The service is one platform product (`RYZE Personalizado`, 19.49 EUR)
    created by migration 000064; administrators no longer create Level 2
    offers (category removed from the plan builder and the plans list; the
    create/update/publish/delete routes were removed).
  - After the questionnaire the request is offered at once to every trainer
    who accepts Level 2 requests; the first to accept takes it and the other
    offers are withdrawn (`uq_premium_level_2_offers_request_trainer`
    replaces one-open-offer-per-request). Declines no longer pass it on;
    turning availability off withdraws the trainer's open offers. Offers
    close after 24 hours.
  - RYZE itself (ADMIN_2) can take a waiting request
    (`POST /admin/premium-level-2/requests/:id/take`) and prepare it like a
    trainer (training programme, nutrition, delivery under
    `/admin/premium-level-2/requests/:id/...`); the client sees "RYZE".
  - New shared request workspace for trainers (`/trainer/premium-level-2/:id`)
    and RYZE (`/admin2/premium-level-2/:id`): the answers, the personal plan
    created right there (blank or from one of their plans, opened in the plan
    builder with `?request=`/`?base=` and linked on save, never published),
    the nutrition plan with a photo per meal (`programs/meals`, uploads
    `/trainer|admin/uploads/meal-image`), and delivery (client notified, plan
    in their area).
  - Delivery by a trainer records their share on the sale (their commission
    rule, 30% by default, on what was actually paid); their earnings now
    include delivered Level 2 services. RYZE deliveries keep it all.
- Health data consent (migration 000066): the Level 2 questionnaire and body
  measurements need explicit consent (checkbox, `health_consent`; otherwise
  `400 HEALTH_CONSENT_REQUIRED`), recorded on the account and the intake;
  `GET/POST/DELETE /me/health-consent`; withdrawing erases measurements and
  questionnaire answers (Account → Privacy and notifications).
- Account anonymisation (migration 000067): an hourly job anonymises accounts
  deleted more than 30 days ago (identity, profile, follows, tokens,
  notifications, health data, feedback, photos); sales documents stay.
  Re-registering within 30 days still restores the account.
- Refunds (migration 000068): ADMIN_2 refunds a completed purchase from the
  sales detail (`POST /admin/purchases/:id/refund`, reasons non-conformity,
  withdrawal, duplicate, goodwill). Card/MB WAY go back through Stripe (the
  payment is found by the purchase in its metadata, set from now on);
  PayPal or older payments are refunded in the dashboard and recorded with
  "already refunded". The purchase becomes `refunded`, its access ends at
  once (Level 2: the request is cancelled and the personal plan removed) and
  it leaves the trainer's earnings.
- Discount codes (migration 000069): ADMIN_1 and ADMIN_2 create codes
  (`/admin/discount-codes`, page in both consoles): 1–90% off, for every
  paid plan, Level 1, Level 2 or trainers' plans, valid 1 hour to 30 days
  (days with presets, or hours when 24 h or less), optional maximum uses,
  once per person. Buyers check a code on the pay screen
  (`GET /me/discount-codes/preview`); it is applied server-side when the
  payment opens (`discount_code`), recomputing the price and the split; the
  receipt shows the list price and the discount. Stripe idempotency keys now
  include the amount.
- Training reminders (migration 000070): members with a plan who have not
  trained for 4 days get an in-app notification and an email, at most once a
  week, between 10:00 and 20:00 in Portugal; toggle in Account → Privacy and
  notifications (`GET/PUT /me/training-reminders`). The home screen shows a
  "N days since your last workout" nudge.
- Legal documents updated accordingly (Level 2 by RYZE, discount codes,
  refunds and access, 30-day restore and anonymisation, health consent,
  reminders, no client–trainer contact).

Verification: Go vet passes; the whole Go suite passes on `ryze_test` (69
packages) with new tests for offers to all trainers and RYZE taking a request,
delivery commission on the price paid, meal photos, health consent, account
anonymisation, refunds (repository and money rules), discount codes and
training reminder candidates. Frontend typecheck passes; Vitest 29 files, 189
tests pass; production build passes. Migrations 000064–000070 applied to
`ryze_test` and the dev database. Note: while testing a down migration,
`go run . migrate down` (which reverts every migration) emptied the
`ryze_test` tables of migrations 45–64; the test database was restored to 70.

# Version 2.130.16

Title: Fix publishing an official RYZE plan (migration 000063)

Date: 2026-11-23

Summary:

- Publishing an official RYZE plan answered 500: the database check
  `chk_programs_cover_image` only allowed uploaded cover keys. Migration
  000063 also allows the fixed key `programs/official/ryze_base_plan.png`
  (the down migration clears it from official plans first).
- New repository test against the real schema
  (`TestProgramCoverConstraintAcceptsOfficialCover`); the earlier service test
  used a stub repository and could not see the constraint.

Verification: the new test fails before and passes after the migration; the
whole Go suite passes on `ryze_test` (68 packages). Migration 000063 applied to
`ryze_test` and the dev database.

# Version 2.130.15

Title: Official RYZE plan cover on wide views

Date: 2026-11-23

Summary:

- Owner request: on horizontal views an official RYZE plan's cover is a
  colour background (the logo's blues) with only the word RYZE, in the
  brand's wide Archivo; portrait plan cards (`ProgramCard`, 4:5) keep the
  RYZE "Base plan" picture.
- `/uploads/programs/official/ryze_base_plan.png` now serves the wide picture
  (1280×720); `/uploads/programs/official/ryze_base_plan_portrait.png` the
  portrait one. Both are embedded in the API. `constants/official_plan.ts`
  (`portraitCover`) swaps the URL on portrait cards; the plan builder previews
  the wide picture.

Verification: typecheck passes; Go build and the generic plan tests pass;
both pictures answer 200 image/png.

# Version 2.130.14

Title: Official RYZE plans (ADMIN_2, free plans)

Date: 2026-11-23

Summary:

- Owner request: when ADMIN_2 creates or edits a free (generic) plan, the
  cover panel offers "Plano oficial RYZE" — a plan of the platform itself
  instead of a custom one. Its only difference: the cover is always the RYZE
  "Base plan" picture and cannot be chosen, replaced or removed.
- API: the ADMIN_2 generic plan create/update body accepts `official`;
  responses carry `official`. The server forces the cover to
  `programs/official/ryze_base_plan.png` for an official plan, clears that key
  from a custom plan, and refuses `official` on Premium plans. Trainers'
  plans can never use it (their cover folder check rejects the key).
- The picture ships embedded in the API (no upload or migration needed) and
  is served at `/uploads/programs/official/ryze_base_plan.png`, so every plan
  card and page shows it without further changes. "Official" is derived from
  the cover key; no schema change.

Verification: typecheck passes; Vitest 29 files, 189 tests pass; Go tests for
`api/auth`, `generic_programs` (new official-cover test) and
`program_details` pass on `ryze_test`.

# Version 2.130.13

Title: Permanent @username, nickname every 7 days, cookie settings in the account, taller plan pictures

Date: 2026-11-23

Summary:

- Owner decision: the @username chosen at registration is permanent and
  unique to the account, like the email. `PUT /me/profile` and
  `PUT /me/username` answer `409 USERNAME_LOCKED` when a chosen @username
  would change; only a generated placeholder (`needs_username`) is chosen,
  once. The settings show it read-only with a lock.
- The nickname changes at most once every 7 days (migration 000062,
  `users.display_name_changed_at`): `409 NICKNAME_COOLDOWN`; account
  responses carry `nickname_change_at` while it is locked and the settings
  show the date.
- Account settings: "Privacidade e cookies" card with the current cookie
  choice and date, "Gerir cookies" (reopens the notice) and the legal links.
- My Programs: plan cover pictures 6.2rem → 11rem tall.

- Fix: re-registering a deleted account failed with 500 after the
  permanent-@username rule (the restored row still counted its @username as
  chosen). Reactivation now resets `username_chosen`, so the @username given
  at re-registration is set (or chosen once later when taken).
- Tests no longer depend on data already in the test database: the public
  catalogue tests hide existing plans inside their rolled-back transaction,
  and the reserved persona e-mail tests accept the row Test Mode itself
  creates. The 16 `seed-demo` plans found in `ryze_test` were archived (soft
  delete; `seed-demo` belongs to the dev database only).

Verification: typecheck passes; Vitest 29 files, 189 tests pass; the whole Go
suite passes on `ryze_test` (68 packages, no failures). Migration 000062
applied to `ryze_test` and the dev database.

# Version 2.130.12

Title: Wide payment screen with payment marks; legal pages, cookie notice and checkout consents (GDPR, EU consumer law)

Date: 2026-11-23

Summary:

- Payment screen redesigned to use the whole screen: the order card with the
  plan picture, what it includes, the total and trust points; the payment card
  in numbered steps — methods with drawn payment marks (Visa, Mastercard,
  Amex, Maestro, MB WAY, PayPal, Stripe), payment details (a card illustration
  for card), receipt (NIF), the consents and Pay. Phones get a compact order.
- Checkout consents (EU consumer rules; migration 000061): before paying, the
  buyer accepts the Terms and Privacy Policy and asks for immediate access to
  the digital content, acknowledging the loss of the 14-day right of
  withdrawal (Directive 2011/83/EU art. 16(m); DL 24/2014 art. 17). A paid
  payment without both answers `400 CONSENT_REQUIRED`; the server records
  `terms_accepted_at` and `immediate_access_requested_at` on the purchase
  (Test Mode records them when given).
- Legal documents at `/legal/termos`, `/legal/privacidade`, `/legal/cookies`
  and `/legal/aviso-legal` (PT/EN), describing what the code actually does
  (data, legal bases, processors, retention, rights, the exact cookies and
  storage). Company identity fields are placeholders in
  `constants/legal.ts` to fill in; a legal review is recommended.
- Cookie notice: "Rejeitar opcionais" and "Aceitar todos" with the same
  weight, "Personalizar", reopened from the Cookie Policy; the choice lasts 12
  months. RYZE uses no analytics or advertising cookies today.
- Footer: the existing Privacy, Terms and Cookies links now open the documents
  (they pointed to Our vision); its layout is unchanged.

Verification: typecheck passes; Vitest 29 files, 189 tests pass; Go tests for
the consent requirement and the purchase flows pass on `ryze_test` (the four
known seeded-data failures remain). Playwright: payment screen at 1838 px and
390 px (mocked API), legal pages and cookie notice on the dev site at 1440 px
and 390 px. Migration 000061 applied to `ryze_test` and the dev database.

# Version 2.130.11

Title: Checkout split: a simple summary on the page, the payment on its own screen

Date: 2026-11-23

Summary:

- The paid journeys (Premium Level 1 and Level 2) show, in the site's glass,
  only what the member gets, the price and one "Pagar" button (owner request).
- "Pagar" opens the payment screen (`?pagamento=1`, so the browser's back
  closes it): a bar with "Voltar", the order on the left (picture, item, what
  it includes, total) and "Finalizar compra" on the right — "Pagamento seguro
  via stripe", the methods (Cartão with the card brands, MB WAY, PayPal; rows
  on phones), what happens next, the NIF, Pay, and the provider's own form
  (Stripe embedded Checkout) when configured. Success shows in place; Test
  Mode keeps its own purchase.

Verification: typecheck passes; checkout hook tests pass; Playwright with a
mocked API at 1280 px and 390 px (summary, payment screen, not-set-up message).

# Version 2.130.10

Title: Stripe's own payment form inside the RYZE checkout (embedded Checkout)

Date: 2026-11-23

Summary:

- Owner rule: payments always use the provider's own screens. With the new
  optional `STRIPE_PUBLISHABLE_KEY`, card and MB WAY create an embedded Stripe
  Checkout Session (`ui_mode=embedded_page`, the success URL as return URL) and
  the checkout mounts Stripe's form in place — card number, expiry and CVC with
  the brand recognised from the number — with "Trocar método de pagamento" to
  go back. Without the key the redirect to Stripe's page stays as before.
  Stripe.js is loaded from `js.stripe.com` (no new npm dependency). Card data
  goes only to Stripe; completion is verified server-side as before.
- The initiation response carries `client_secret` and `publishable_key` for
  the embedded page; the production CSP allows the Stripe hosts it needs.

Verification: typecheck passes; Go tests for the embedded session (ui_mode,
return_url, no success/cancel URL, client secret) and the purchase and payment
packages pass; checkout hook tests pass. Not exercised against Stripe: no keys
in this environment.

# Version 2.130.9

Title: Checkout always shows the payment methods, as tiles

Date: 2026-11-23

Summary:

- The payment step no longer stops at "Payments are not available yet" when no
  provider is configured (owner request): it always shows the whole screen —
  the order, the three methods RYZE supports (Cartão, MB WAY, PayPal) as
  tiles, what happens next for the chosen one (accepted card brands for card),
  the NIF and the Pay button. Which methods can actually be paid with still
  comes from `/payments/methods`; a method not set up carries a "Por
  configurar" badge, and paying with it explains that it is not set up —
  nothing is created or charged. Card details are still entered on the
  provider's secure page (RYZE never handles card numbers).

Verification: typecheck passes; checkout hook tests pass; Playwright with a
mocked API (no provider configured): tiles, card and MB WAY details, the
not-set-up message.

# Version 2.130.8

Title: Feedback gallery redesigned: clean glass bento grid

Date: 2026-11-23

Summary:

- Owner feedback: the tilted cards looked blurred and felt laggy. The gallery
  is now a clean grid of glass cards (the public surface style), the first card
  with a photo featured (two columns by two rows on desktop), a gentle hover
  lift and photo zoom (no filters, owner decision), a staggered fade-in, and an enlarged view with previous / next (buttons
  and arrow keys). No 3D transforms. Still six cards first, "Mostrar mais" /
  "Mostrar menos".

Verification: typecheck passes; Playwright against the dev site at 1440 px and
390 px (grid, enlarged view).

# Version 2.130.7

Title: Feedback with a photo and a gallery; demo data seed; Test Mode opens the home

Date: 2026-11-23

Summary:

- Feedback (migration 000060): one feedback per member — pending or published;
  a rejected one can be sent again — with one optional JPEG/PNG picture
  (re-encoded without metadata, longest side 1280 px, stored in the uploads
  `feedback` folder). The picture is served by `GET /feedback/:id/image` only
  once approved, and to reviewers by `GET /admin/feedback/:id/image`; the
  ADMIN_2 review queue shows it. `POST /me/feedback` accepts multipart
  (`rating`, `message`, `image`) or JSON as before.
- Feedback gallery: the ask card above, then a mosaic of slightly tilted cards
  (three columns on desktop, two on tablets, one on phones) that opens with six
  and grows with "Mostrar mais" / "Mostrar menos" — never an endless scroll;
  each card opens enlarged with the full picture and message.
- `go run . seed-demo`: one published free plan per category (6), ten
  published Premium Level 1 packages across the four objectives and the three
  audiences (to try the Level 1 assignment), ten demo members with an approved
  feedback and picture, and the demo member account from
  `RYZE_DEMO_CLIENT_EMAIL` / `RYZE_DEMO_CLIENT_PASSWORD` (added to `.env`).
  Images live in `main/src/backend/devdata/seed/` (`make_images.sh`
  regenerates them). Idempotent; run on the dev database.
- Entering Test Mode always opens the site's home, where the Test Mode bar
  shows; the return path is only used when exiting.

Verification: typecheck passes; Go tests for member feedback (service and
handler) pass; the seed ran on `ryze_test` twice (second run created nothing)
and on the dev database; the feedback API returns the ten pictures. Playwright
against the dev site at 1440 px and 390 px (gallery, enlarged card).
Migration 000060 applied to `ryze_test` and the dev database.

# Version 2.130.6

Title: Workout photo; compare your lifts with RYZE

Date: 2026-11-22

Summary:

- Workout photo (owner request, as in Lyfta; migration 000059): before saving a
  workout — or later, opening it from the history — the member can add a photo
  of themselves (JPEG or PNG, re-encoded without metadata, longest side 1280
  px). It is stored in the private `session_photos` folder, never served
  publicly: `GET/POST/DELETE /me/workout-sessions/:id/photo`, owner only,
  `private` cache. The history marks workouts with a photo. Abandoned workouts
  take none.
- "Compara-te com a RYZE" in Achievements (like Lyfta's Lyftoff): pick an
  exercise and see where your best lift (estimated 1RM, Epley, reps capped at
  12) stands among every member who trained it — "Top 1%", rank, the share of
  other lifters you beat. `GET /me/achievements/rankings` returns only the
  member's own numbers and aggregates.
- The request body limit accepts a path pattern (for the photo upload).

Verification: typecheck passes; Go tests for the image store, ranking score
and the ranking query (on `ryze_test`) pass. Playwright with mocked APIs at
1440 px (rankings) and 390 px (workout photo box). Migration 000059 applied to
`ryze_test` and the dev database.

# Version 2.130.5

Title: ADMIN_2 request list spacing; commission quick values 0, 5 and 10 %

Date: 2026-11-22

Summary:

- Premium Level 2 requests (ADMIN_2): the search, status filter and empty
  message no longer touch the card edges.
- Trainers & commissions: quick values "No commission" (0 %: the trainer
  receives the whole sale), 5 % and 10 % join 20-35 %. The server already
  accepted 0 %.

Verification: typecheck passes.

# Version 2.130.4

Title: The plan in focus on the home, "Trocar plano", other plans, "Começar treino" opens the next workout

Date: 2026-11-22

Summary:

- The home shows the plan of the member's latest action — adding a plan,
  training one or choosing one — and a workout in progress always wins (was:
  the last trained plan, so a newly added plan never appeared until trained).
- The main card says "O teu primeiro treino" / "Começar primeiro treino" until
  the plan has a completed workout; "Ainda não tens um plano" offers free plans
  and Premium; a complete plan offers "Continuar com <next plan>".
- "Trocar plano" (when there is more than one active plan) lists the plans
  with their next workout; the choice is kept on the account (migration
  000058, `GET/PUT /me/plan-focus`, access checked by the server) so phone and
  computer agree.
- "Os teus outros planos" lists up to four other active plans with their next
  workout, a start button and "Pôr na home".
- "Começar treino" in the side bar opens `/services/next-workout`: the workout
  in progress, else the next workout of the plan in focus (started), else the
  complete plan, else the library.
- Phones keep four bottom tabs; Recovery is in "Mais".

Verification: typecheck passes; Vitest (27 files, 172 tests) passes, with the
focus rule tests; Go tests for the plan focus service and repository pass on
`ryze_test`. Playwright with mocked APIs at 1440 px and 390 px (home with three
free plans). Migration 000058 applied to `ryze_test` and the dev database.

# Version 2.130.3

Title: Several free plans at once, library split, Premium levels exclusive, plan day tools

Date: 2026-11-22

Summary:

- Several Free plans can be active at once (owner decision, as in Lyfta;
  migration 000057): the one-active rule is now per user and program.
  Activating another Free plan adds it; activating an active one is
  idempotent. Statistics take as current the active Free plan trained most
  recently.
- The library ("Os meus planos") is split: "O teu Premium" (Level 1 or Level 2,
  with links when there is none) and "Os teus planos grátis" (any number, with
  "Adicionar plano grátis").
- Premium Level 1 and Level 2 are never active together: buying one while the
  other is active answers `409 PREMIUM_LEVEL_CONFLICT` (also in Test Mode),
  before any purchase row exists; the checkout explains it.
- Trainer Programmes: the lead no longer says trainers sell Personalized plans
  (Free or Premium €14.99; Level 2 plans are made inside each request).
- Plan builder: every training day has tools — copy, cut, paste over the day,
  duplicate, move earlier or later, remove all exercises and remove the day —
  and a copied day can be pasted as a new day in any week.

Verification: typecheck passes; Vitest for the day tools passes; Go tests for
the Premium exclusivity, the Free activation e2e flow, statistics and
repositories pass on `ryze_test`. Migration 000057 applied to `ryze_test` and
the dev database.

# Version 2.130.2

Title: Workout player shows the exercise; set inputs validated; new summary and history

Date: 2026-11-21

Summary:

- Home hero picture test: the picture moved well to the right (objects past
  the middle of the screen) and is centred on the screen height; only the
  hero's 3D (the opening and the table) stays hidden, the rest of the story
  keeps its 3D (phone, computer, planet). `/?hero=3d` shows the 3D hero.
  The scene plays exactly as with the 3D hero (only the table and drone are
  hidden) and the canvas fades in as the hero scrolls away; the shadow map is
  now drawn on the first frame (never drawn, it left the phone's frame dark
  and see-through in this mode).
- Workout player: the current exercise plays its demonstration (GIF or video)
  in a loop with the description beside it — name, prescription, muscle,
  equipment, rest, the trainer's note and the first steps of how to do it —
  and the sets below. On phones the demonstration is shorter and the steps
  stay behind "Instruções completas".
- Set inputs: letters or other text in the weight or reps no longer log the
  target or last weight; the field turns red, an error explains the format
  (weight 0-1000 kg with up to two decimals, reps a whole number) and the set
  cannot be ticked until it is fixed.
- Workout summary: a headline card, four stat tiles, records as gold cards and
  every exercise as a card with each set as a chip (the best set starred,
  skipped sets dimmed, a PR badge on records); two columns on desktop.
- History: the last 30 days at a glance (workouts, time, volume, records), the
  month with its count, and each workout as a card with the first exercises,
  time, volume, sets with a bar and records; two columns on desktop.

Verification: typecheck passes; Vitest for the set input parsing passes;
Playwright with mocked APIs at 1440 px and 390 px (workout player, summary,
history) and 1660 px (Home hero).

# Version 2.130.1

Title: Level-up seal of light, no preview button, Home hero picture test

Date: 2026-11-21

Summary:

- Level-up animation (owner feedback: the chains and padlock looked like two
  emojis): the badge is now held by two crossed, segmented bands of light in
  the level's colour and a hexagonal seal with a line lock and a turning ring;
  the seal cracks, the bands snap (a ring of light at the break) and the seal
  bursts into slivers of light. The rest of the timeline is unchanged.
- The "Testar subida de nível" button is gone everywhere (Test Mode banner and
  local development): what development shows is what production shows. EXP for
  tests is set with `go run . set-xp`.
- Home hero test (owner request): the picture `main_hero_replace.png` stands
  where the 3D composition stood, right of the copy; the 3D opening and table
  are hidden and the phone grows in as the hero scrolls away. `/?hero=3d`
  shows the 3D for comparison. If the picture is kept, the drone opening can
  be redone around it and the 3D hero removed.

Verification: typecheck passes; Playwright at 1440 px (a temporary harness,
removed) showed the seal and bands and the hero picture.

# Version 2.130.0

Title: Trainer earnings and views, Recovery page, level-up animation, Home hero composition

Date: 2026-11-21

Summary:

- Trainer profile: the cards no longer touch (one grid with spacing; on
  desktop the identity beside a "Visualizações" card and the coach card).
- Trainer earnings (`GET /trainer/earnings`): the dashboard shows "Ganho
  total" (gross sales), "Lucro" (the trainer's share) and "Comissões" (what
  RYZE kept), from completed, non-Test-Mode purchases of the trainer's plans.
- Views, the stat only coaches have: on the trainer profile page and as
  `stats.views` on the public profile (`/u/:username`), from the views of the
  coach's published plans.
- Recovery has its own page and menu entry (`/services/recovery`, "Recuperação"),
  out of Progress (`?tab=muscles` redirects). The side of the body map is a
  detail card for the selected (or least recovered) muscle and compact chips
  grouped by status, so it stays about as tall as the map; the body drawing
  has more detail (muscle heads, tendons, knees, Achilles, a soft volume
  light).
- Level-up animation: the screen darkens, the new badge appears blurred under
  two crossed chains and a padlock, everything shakes harder, the chains snap,
  the padlock explodes (flash, shock wave, shards, sparks) and the new level
  is revealed glowing in its own colour. It plays once after the workout that
  levels up; "Saltar" jumps to the reveal; reduced motion shows the reveal.
  Test Mode (banner) and local development (Achievements) have "Testar subida
  de nível" with the eight levels (preview only).
- EXP adjustments (migration 000056, `go run . set-xp`); @edgar ("teste1") is
  at 39,999 EXP, one short of Lifter Lendário.
- Home hero: the 3D composition follows the owner's reference (shaker behind
  on the left, the plate in front of it, the phone large at the centre, the
  dumbbells on its right); napkin and cutlery sit under the plate.

Verification: typecheck passes; focused Vitest (navigation history, Test
Mode) passes; Go tests for EXP adjustments and the earnings query pass on
`ryze_test`. One Playwright pass with mocked APIs at 1440 px: Home hero,
Recovery and the level-up animation. Docs: `api.MD`, `database.MD`,
`how_to_run.MD`, `PAGES/design.MD`, `docs/AI/frontend-revamp.md`,
`CLAUDE.md`. Migration 000056 applied to `ryze_test` and the dev database.

# Version 2.129.0

Title: Trainer profile shows the one identity; profile "about" with social networks; nickname at registration

Date: 2026-11-21

Summary:

- The trainer profile page no longer shows the registered name and the email
  or says the profile cannot be edited: it shows the photo, nickname (else the
  @username), the @username, biography and social links, with "Editar perfil"
  and "Ver perfil público". Both shells show the @username, never the email.
- Profile "about" (migration 000055): biography (500 characters), date of
  birth (only the age is public), training start year (years of training),
  weight and social networks (Instagram, TikTok, YouTube, X, Facebook,
  Strava) — `GET/PUT /me/profile/details`, edited in Account, shown on the
  public profile. Only handles are stored; links are built by the server.
- Registration: the @username stays required and unique (the database
  refuses two active accounts with the same one); the nickname is an optional
  field (`display_name`).

Verification: typecheck passes; `npm run test` 26 files, 179 tests pass; Go
tests for handle normalisation, details validation and the details
repository pass on `ryze_test`; `api/auth` and `registration` keep their known
failures from seeded data. Playwright with mocked APIs at 390 px and 1440 px:
trainer profile, public profile, Account, registration. Docs: `api.MD`,
`security.MD`, `project.MD`, `database.MD`, `PAGES/design.MD`,
`docs/AI/frontend-revamp.md`, `CLAUDE.md`. Migration 000055 applied with
`go run . migrate up`.

# Version 2.128.0

Title: Back to where you came from, locked completed days, nickname and chosen @username, public profiles with following, Premium Level 2 allocation by offer

Date: 2026-11-21

Summary:

Owner requests of 2026-11-21.

- Back buttons: coming from another area (personal area → training plans) or
  from the page above, "back" returns to exactly that page; within an area it
  goes up to the parent. Navbar, member shell and in-page back links share
  `resolveBack` / `useBackTarget`.
- My plan: a completed day is done once — the card turns green with
  "Concluído a <date> · não se repete" and the server refuses a restart
  (`409 WORKOUT_ALREADY_COMPLETED`). The plan cover shows under the title.
- Fixes: short public pages no longer leave a dark band before the footer;
  the coach page clears the navbar and, for the coach themselves, explains why
  the profile is not public (no published plan, or a Test Mode account).
- Name: the platform shows the nickname ("alcunha"), else the @username —
  never the email or the registered name (coach names included). The
  @username is required at registration; generated ones (Google, older and
  administrator-made accounts) are replaced through a first-visit dialog
  (`PUT /me/username`, migration 000052).
- Public profiles at `/u/:username`, open to everyone: photo, nickname,
  member since, workouts, followers, following, level, record medals; follow
  and unfollow (migration 000053). Signed in, the public navbar shows the
  bell and the account photo and name.
- Trainers create free and Premium Level 1 plans only; older personalised
  plans keep their type.
- Premium Level 2 allocation by offer (migration 000054): after the intake
  the request is offered to the available trainer with the fewest open
  requests; accept allocates it, decline or 24 hours without an answer passes
  it to the next trainer; with nobody left ADMIN_2 allocates by hand. New
  trainer panel with the availability switch, notification
  `pl2_request_offered` with email.
- Answers: the password-reset email depends on the SMTP credentials (recorded
  as skipped without them); profile photos exist since v2.127.0.

Verification: typecheck passes; `npm run test` 26 files, 179 tests pass; Go
build and vet pass; new tests for the completed-day rule (route test),
required @username, following, Premium Level 2 offers (pick, one open offer,
decline → next, expiry, accept) and trainer plan types pass on `ryze_test`;
`api/auth` and `registration` keep their known failures from seeded data.
Playwright with mocked APIs at 390 px and 1440 px: plan page, public profile,
coach page, @username dialog, no horizontal overflow. Docs: `api.MD`,
`security.MD`, `project.MD`, `database.MD`, `PAGES/design.MD`,
`how_to_run.MD`, `docs/AI/frontend-revamp.md`, `CLAUDE.md`. Apply migrations
000047-000054 with `go run . migrate up`.

# Version 2.127.0

Title: Explicit back buttons, free plan overview, public profile, complete trainer application with document screening, muscle recovery and body measurements

Date: 2026-11-21

Summary:

Owner requests of 2026-11-20, in the agreed order (quick fixes first).

- Navigation: the navbar and member-shell back buttons open the named parent
  page ("Voltar à Área pessoal", "Voltar aos Planos de treino", ...) in one
  click instead of stepping back through history. The free and premium
  detail pages no longer sit under the navbar.
- Password: the current-password fields (change password, delete account) no
  longer autofill and have no reveal button. Re-entering the password stays,
  because it is what protects the account if a session is stolen.
- Free plans: the detail page shows the plan's weeks, day titles, focus and
  exercise counts (`GET /programs/:id/outline`, free plans only, no
  exercises), and "Obter plano grátis" opens a checkout-like get step without
  payment or purchase document.
- Public profile (migration 000049): unique @username (generated for every
  account, optional at registration, editable), display name and photo
  (re-encoded to a 512 px JPEG). Shown in the account settings, the dashboard
  and the user menu; coach names use the display name.
- Trainer application (migration 000050): one complete multipart submission:
  personal details, work history, education, a free text and the documents
  (certification and identification required). Every file is screened
  locally: declared AI generation, AI generator metadata, unsafe PDF content
  or a file another account submitted refuse the submission with the reason;
  softer signals are flagged for the reviewer. The admin review shows the
  form and the screening, and admins can make a user a trainer by @username.
  `start_test_mode.sh` promotes its trainer that way.
- Statistics (migration 000051): the progress page has Training / Muscles /
  Measurements tabs. Muscles: an estimate of each of 16 groups' recovery from
  the last 7 days of sets (`GET /me/statistics/muscles`), shown on a front and
  back body map beside the list with percentages. Measurements: weight, body
  fat and 13 circumferences, logged per day, with the latest values, an
  evolution chart and the history (`/me/measurements`). Dashboard shortcuts
  open both.
- Email verification (question answered): the flow is implemented end to
  end (sent after registration, `/verify-email`, resend); without SMTP
  credentials the emails are recorded as skipped and not sent.

Verification: typecheck passes; `npm run test` 26 files, 175 tests pass; new
Go tests for the outline, profile, avatars, screening, submission, promotion,
recovery and measurements pass (`ryze_test` for SQL); the `api/auth` and
`registration` packages keep their known failures from seeded `ryze_test`
data (public program counts, reserved Test Mode addresses). Progress tabs and
the application form checked with Playwright against mocked APIs at 390 px
and 1366 px (no horizontal overflow). Docs: `docs/INSTRUCTIONS/api.MD`,
`docs/INSTRUCTIONS/security.MD`, `docs/DATABASE/database.MD`,
`docs/PAGES/design.MD`, `docs/how_to_run.MD`, `docs/AI/frontend-revamp.md`,
`CLAUDE.md`. Apply migrations 000047-000051 with `go run . migrate up`.

# Version 2.126.2

Title: Simple mobile site without 3D stand-ins; registration fixed; sign-in in the start script

Date: 2026-11-20

Summary:

- Mobile/tablet: the Home story no longer renders the drone opening or planet
  RYZE below 1024 px (only as the desktop fallback); phones get copy plus the
  real app screens (no lifted cards, from 768 px only). The stacked story is one
  column up to 1023 px (fixes a heading/paragraph overflow in two columns at
  768-1023 px). Our Vision shows no tunnel stand-in on phones. `Reveal` is
  static below 1024 px. Phones get tighter section spacing (`--r-section-y`
  3.25rem). `useMediaQuery` now reads the query on the first render.
- Registration: the backend requires 12-128 character passwords, but the form
  neither said so nor checked it and showed "Validation failed."; it also
  opened /profile without a session. The form now shows the rule, checks it,
  explains validation/rate-limit errors and signs the new account in after
  registering (owner request; login/registration area touched only for this).
- `start_test_mode.sh`: sets PUBLIC_APP_URL and GOOGLE_REDIRECT_URL for
  localhost when GOOGLE_CLIENT_ID/SECRET are in .env, refuses a partial Google
  config, and prints whether account creation, Google sign-in and emails are
  active. `.env.example` documents this.

Verification: typecheck passes; `npm run test` 26 files, 175 tests pass; Home
and Our Vision checked at 390 px and 820 px (screens and lifted cards load);
the start script's sign-in report run against the local API. Docs:
`docs/PAGES/design.MD`, `docs/how_to_run.MD`, `docs/AI/frontend-revamp.md`,
`CLAUDE.md`.

# Version 2.126.1

Title: The Feed page drops its 3D scene

Date: 2026-11-20

Summary:

Owner request: remove the Feed's 3D animation and keep a simple "in
development" scene. `/feed` now uses the shared under-development placeholder
with the "no entry" sign hologram, like the other unfinished pages. Removed
`feed_construction_scene.ts`, `feed_construction_hologram.*` and the larger
Feed stage. `home_story/story_hologram.ts` and `story_athlete.ts` are again
unused (kept, as before). Docs: `docs/PAGES/design.MD`.

Verification: typecheck passes.

# Version 2.126.0

Title: Automatic purchase documents for every completed purchase; Test Mode recipient review

Date: 2026-11-20

Summary:

Owner requirement: every completed purchase gets a document automatically,
shown to the client and emailed, mandatory and without opt-in or opt-out
(this supersedes FAQ answer 47, "only if the person wants"). Real and Test
Mode purchases are covered; free activations are not purchases and get none.
Owner decisions of 2026-11-20: until certified invoicing software is
connected the document is a purchase receipt with no tax value (an
app-generated document is not a valid Portuguese tax invoice); VAT follows the
small-business exemption (article 53 of the VAT Code); checkout offers an
optional, server-validated NIF; the format is an HTML page with a print layout
(Save as PDF) plus an email carrying the document, with no new dependency.

Backend:

- Migration `000048_create_invoices`: `purchases.buyer_tax_id`,
  `invoice_counters` (gapless numbering per series and year in the issuing
  transaction) and `invoices` (one per purchase, full snapshot of seller,
  buyer, item, amounts, VAT treatment, currency and payment method; a trigger
  rejects every update and delete). Series `RYZE` for sales and `TEST` for
  Test Mode.
- `services/invoices`: a purchase fulfilment hook (idempotent, never fails or
  undoes the purchase), on-demand issue when the owner opens a missing
  document, and a background sweep every five minutes (`jobs.go`) that issues
  missing documents and records sales-document emails never recorded. Email
  template `invoice_issued` (EN/PT) carries the document and links to its page;
  a Test Mode document goes to the session's validated recipient.
- Endpoints: `GET /api/v1/me/invoices`, `GET /api/v1/me/purchases/:purchaseID/invoice`
  (owner only, 404 for anything else) and `GET /api/v1/admin/purchases/:id/invoice`
  (commerce permission). Optional `tax_id` on
  `POST /api/v1/me/purchases/:purchaseID/payment` and on the Test Mode purchase
  (`services/taxid`, check digit). Seller identity from `INVOICE_SELLER_NAME`
  (default "RYZE"), `INVOICE_SELLER_TAX_ID`, `INVOICE_SELLER_ADDRESS`.

Test Mode review (fixes):

- The persona email recipient ignored the session lifetime and fell back to an
  older session's address: a newer session without an address, or an expired
  but not exited one, could still send a persona's emails to someone else's
  inbox. `test_mode.RecipientResolver` now uses only the persona's most recent
  active session, within its lifetime and while Test Mode is enabled; the
  repository method became `FindLatestActiveByPersona`.
- The handler validated the email before trimming it; it now validates the
  normalised address like the service.
- SMTP failure reasons are stored and logged with addresses redacted.
- The persona notification test no longer depends on the shared test
  database's seeded persona. Migration `000047` reviewed; `000046` unchanged.

Frontend:

- `components/checkout_tax_id/`: the optional NIF field (label, hint, inline
  error announced politely) on the shared checkout panel, the generic plan
  page and Premium Level 1, real and Test Mode; the pay/retry buttons stay
  disabled while a typed NIF is invalid.
- `pages/account_receipt_page/`: the receipt page
  (`/account/purchases/:purchaseId/receipt`) with a print layout; a "Receipt"
  button on completed purchases in Purchases; receipt links on every checkout
  success state. The Test Mode email field now announces its error and says the
  receipt goes to that address.

Docs: `api.MD`, `database.MD`, `arquitecture.MD`, `security.MD`,
`project.MD`, `PAGES/design.MD`, `how_to_run.MD`, `FAQ/respostas.MD` (dated
note on answer 47), `CLAUDE.md` (latest migration). Package version 2.2.61.

Verification:

- `go build ./...` and `go vet ./...` pass; `gofmt` clean.
- New tests: invoice repository (sequential numbering, one document per
  purchase, separate test series, owner-scoped reads, immutability; concurrent
  issues giving one document per purchase and gapless numbers) on `ryze_test`;
  invoice service (idempotent issue and email, Test Mode recipient,
  suppression without one, owner-only on-demand issue, retries); invoice
  handler (owner only, no account id); NIF validation (backend and frontend);
  payment and Test Mode handlers with `tax_id`; recipient resolver; invoice
  email rendering; address redaction.
- Passing on `ryze_test`: `./email`, `./services/...` except
  `services/registration`, `./repositories`, `./api/webhooks`, `./routes`.
  `./api/auth` keeps only its four known failures from data seeded in
  `ryze_test` (public programme counts, reserved Test Mode emails);
  `services/registration` `TestRegisterRejectsReservedTestModePersonas` fails
  for the same seeded-persona reason (untouched code).
- `npm run typecheck` passes (the `story_scene.ts` syntax error noted in
  2.125.0 was fixed in 2.125.1); `npm run test`: 26 files, 175 tests pass.
- Migration `000048` applied to `ryze_test`; the development database still
  needs `go run . migrate up` (`000047` and `000048`). Real email delivery was
  not verified (no SMTP credentials configured).

# Version 2.125.1

Title: Hero scene: nothing touches the phone, dumbbells sideways and more detailed; the phone spins again

Date: 2026-11-20

Summary:

Owner feedback on the hero: the phone looked as if it stood inside the plate;
the dumbbells should lie sideways, the shaker should stand behind the plate and
the dumbbells need more detail. The phone's spins in the story chapters, removed
in 2.123.1, are back, and its frame is white metal again rather than a lit edge.

Main changes:

- `home_story/story_intro.ts`: a new layout, checked in screen space at the
  hero camera so that nothing overlaps the phone: the plate further left and
  forward (about 30 px clear of the phone at 1440 px), the RYZE shaker behind
  it, a smaller napkin with the fork and knife in front of the phone (under the
  plate's rim), and the two dumbbells lying sideways on the right, one in front
  of the other. Each dumbbell now has a cyan band round both heads, the RYZE
  mark with "10 KG" on the side face in view, chrome flanges, and a knurled
  grip between smooth chrome sleeves with grooves. The beam is a little wider so
  it covers the whole composition.
- `home_story/story_scene.ts`: the phone spins in the Train, Nutrition and
  Evolve chapters again, with the screen swapped while its back is in view
  (the cross-fade overlay is gone); the frame is white polished metal
  (`#f2f5fa`, clearcoat); the hero sits slightly higher (`heroY` -1.32). A
  leftover fragment of the removed cross-fade function, which broke the full
  typecheck, was removed.

Verification (short, as asked): typecheck passes; one hero screenshot at
1440 px with the 3D running.

Also in this version — the Feed's construction hologram (owner request,
2026-11-20: the Home's drone doing the hologram scene, "really good 3D", a
hologram person building the platform):

- `pages/under_development_page/feed_construction_scene.ts` (rewritten): the
  Home's scanning drone flies in, opens its beam over a platform and projects
  a hologram person who takes post cards from a floating stack and pushes them
  into a four-slot Feed column; the full column completes with a sweep and
  clears, and the loop starts again. The former boxy worker is gone.
- `home_story/story_hologram.ts`: options for an opening (drone fly-in, beam
  opening from the sensor down, the hologram building up) and a building mode;
  exposes its rig, floor, scale and the stack's pick point. Unused by the Home.
- `home_story/story_athlete.ts`: a building routine (reach, lift, carry, push,
  rest) and the fists' position; no dumbbells or muscle glow while building.
  Unused by the Home.
- `feed_construction_hologram.*`: the light version is now an SVG of the same
  drone projecting the Feed column while its cards build in (still, complete,
  with reduced motion); `under_development_page.*`: a larger stage for the
  Feed only.
- Docs: `docs/PAGES/design.MD` (Feed), `docs/AI/frontend-revamp.md`.

Verification: typecheck passes; the Feed checked at 1440 px through a full
card cycle and at 390 px (light version); no console errors.

# Version 2.125.0

Title: Test Mode purchases send lifecycle emails to a validated checkout recipient

Date: 2026-11-18

Summary:

Owner request of 2026-11-18: a Test Mode purchase must run the real commerce
lifecycle and send its emails to an inbox the administrator can read. Test Mode
personas use non-deliverable `@ryze.local` addresses, so the checkout now asks
for one bare email address and the backend validates it before completing the
zero-cost purchase.

The recipient is stored only on the active `test_sessions` row (migration
`000047`) and is never logged or returned by an API. While the session is
active, the notifications service uses it for the persona's normal lifecycle
emails, including `access_granted`; in-app notifications are unchanged. Without
a typed recipient, emails stay suppressed. Exiting Test Mode soft-deletes the
session and stops further use of the address.

Test Mode now documents and supports Generic, Premium Level 1 and Premium Level
2 purchases, and a disabled purchase endpoint answers the documented
`403 TEST_MODE_DISABLED` instead of the inactive-session code. Invoice
generation remains out of scope: no numbering, tax fields, PDF or attachment
was added.

Modified files:

- `main/src/backend/email/address.go`
- `main/src/backend/email/email_test.go`
- `main/src/backend/database/migrations/000047_add_test_session_recipient_email.up.sql`
- `main/src/backend/database/migrations/000047_add_test_session_recipient_email.down.sql`
- `main/src/backend/models/test_session.go`
- `main/src/backend/repositories/test_session_repository.go`
- `main/src/backend/services/test_mode/test_mode.go`
- `main/src/backend/services/test_mode/test_mode_test.go`
- `main/src/backend/services/notifications/notifications.go`
- `main/src/backend/services/notifications/notifications_test.go`
- `main/src/backend/api/auth/test_mode_purchase_handler.go`
- `main/src/backend/api/auth/test_mode_handler_test.go`
- `main/src/backend/api/auth/premium_e2e_test.go`
- `main/src/backend/routes/routes.go`
- `main/src/frontend/src/components/test_mode/test_mode_email.ts`
- `main/src/frontend/src/components/test_mode/test_mode_email.test.ts`
- `main/src/frontend/src/components/test_mode/test_mode_email_field.tsx`
- `main/src/frontend/src/components/test_mode/test_mode_email_field.module.css`
- `main/src/frontend/src/services/test_mode_api.ts`
- `main/src/frontend/src/hooks/use_program_checkout.ts`
- `main/src/frontend/src/hooks/use_program_checkout.test.tsx`
- `main/src/frontend/src/components/checkout_panel/checkout_panel.tsx`
- `main/src/frontend/src/pages/generic_program_detail_page/generic_program_detail_page.tsx`
- `main/src/frontend/src/pages/premium_level1_page/premium_level1_page.tsx`
- `main/src/frontend/src/pages/premium_level1_page/premium_level1_page.pt.ts`
- `docs/INSTRUCTIONS/api.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/INSTRUCTIONS/security.MD`
- `docs/DATABASE/database.MD`
- `docs/PAGES/design.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Test Mode purchases previously exercised the lifecycle without leaving the
persona inbox, so no email could be inspected. A checkout-scoped recipient keeps
the address out of the account and purchase records while letting the existing
notification/outbox pipeline deliver the real test emails.

---

# Version 2.124.0

Title: Premium Level 2 pays first, then opens a complete post-purchase experience and academy

Date: 2026-11-16

Summary:

Owner request of 2026-11-15: Premium Level 2 now follows payment first. The
service page sells the offer and the full questionnaire opens immediately after
the purchase, instead of being saved as a pre-purchase draft. The pre-purchase
draft table, model, repository methods, handler and routes were removed.

The post-purchase experience is new: an entry resolver selects the newest
non-cancelled request; a short preparing moment leads into the expanded
bilingual intake (catalog version 2, eight sections); after submission the page
shows an honest plan-on-the-way timeline; and the summary presents a
deterministic analysis, the completed purchase receipt, every answer and a
print-only answers/receipt sheet for the browser's Save as PDF. No PDF
dependency was added and answers are never written to browser storage.

A new exclusive academy is included with a delivered Premium Level 2 service.
It contains a complete bilingual 12-chapter practical guide with references,
a desktop table of contents, mobile topic chips, reading progress and a scroll
spy. The member navigation item and the page itself both require a delivered
request; the page never reads intake answers.

The intake catalog grew to version 2 with personal details, goals, training
history, body measurements, lifestyle, nutrition, health and coaching
preferences. Boolean questions now render as explicit Yes/No choices.

Modified files:

- `main/src/backend/services/premium_level_2/catalog.go`
- `main/src/backend/services/premium_level_2/service.go`
- `main/src/backend/services/premium_level_2/validation.go`
- `main/src/backend/services/premium_level_2/validation_test.go`
- `main/src/backend/services/premium_level_2/drafts.go` (removed)
- `main/src/backend/services/premium_level_2/drafts_test.go` (removed)
- `main/src/backend/api/auth/premium_journeys_handler.go`
- `main/src/backend/models/premium_journeys.go`
- `main/src/backend/repositories/premium_journeys_repository.go`
- `main/src/backend/repositories/premium_journeys_repository_test.go`
- `main/src/backend/routes/routes.go`
- `main/src/backend/database/migrations/000046_drop_premium_level_2_intake_drafts.up.sql`
- `main/src/backend/database/migrations/000046_drop_premium_level_2_intake_drafts.down.sql`
- `main/src/frontend/src/pages/premium_level2_experience/premium_level2_experience_page.tsx`
- `main/src/frontend/src/pages/premium_level2_experience/premium_level2_experience_page.module.css`
- `main/src/frontend/src/pages/premium_level2_academy/premium_level2_academy_page.tsx`
- `main/src/frontend/src/pages/premium_level2_academy/premium_level2_academy_page.module.css`
- `main/src/frontend/src/pages/premium_level2_academy/premium_level2_academy_content.ts`
- `main/src/frontend/src/pages/premium_level2_journey/premium_level2_journey_page.tsx`
- `main/src/frontend/src/pages/premium_level2_page/premium_level2_page.tsx`
- `main/src/frontend/src/components/checkout_panel/checkout_panel.tsx`
- `main/src/frontend/src/components/premium_level2_intake/question_field.tsx`
- `main/src/frontend/src/components/app_shell/member_shell.tsx`
- `main/src/frontend/src/routes/app_routes.tsx`
- `main/src/frontend/src/routes/member_routes.tsx`
- `main/src/frontend/src/services/premium_journeys_api.ts`
- `main/src/frontend/src/i18n/catalogs/premium_level2.pt.ts`
- `main/src/frontend/src/i18n/catalogs/questionnaire.pt.ts`
- `docs/INSTRUCTIONS/api.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/DATABASE/database.MD`
- `docs/PAGES/design.MD`
- `docs/AI/frontend-revamp.md`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Verification:

- `go build ./...` passes.
- Full backend suite `DB_NAME=ryze_test go test -count=1 -p 1 ./...` passes on
  the isolated PostgreSQL test database; migration `000046` applied there.
- The new pages, member shell and member routes pass TypeScript compilation in
  isolation.
- Full frontend `npm run typecheck` remains blocked by the pre-existing syntax
  error in the user-modified `home_story/story_scene.ts`; that file was not
  changed in this task.

# Version 2.123.3

Title: Feed construction hologram model quality pass

Date: 2026-11-16

Summary:

The Feed-only construction hologram was refined to match the procedural 3D
quality of the Home experience. The worker now has recognisable construction
equipment and layered clothing, the frame uses I-beam geometry with gussets
and fasteners, and the hammer has a clearer two-material silhouette. The
scanner drone now uses the site's metallic pod, sensor lens, ducted rotors and
lighting treatment. The Feed route, timing and fallbacks remain isolated; no
Home scene or other page was changed.

Verification:

- The Feed scene passes isolated TypeScript compilation.
- Existing full-project typecheck remains blocked by the pre-existing syntax
  error in the user-modified `home_story/story_scene.ts`.

# Version 2.123.2

Title: Generic public login failure and Feed construction hologram

Date: 2026-11-16

Summary:

The public login no longer reveals the administrator login route when a
non-email identifier is entered. Administrator authentication remains a
separate backend flow; the member login endpoint has no administrator-route
disclosure and no admin route or authentication contract was changed.

The `/feed` under-development page now has an isolated procedural construction
hologram. A worker hammers a metal frame while a drone performs the existing
Home-style spawn-in/reveal timing. The scene is lazy-loaded, desktop-only and
has mobile, reduced-motion and WebGL fallback behaviour. Other
under-development pages and the Home scene are unchanged.

Modified files:

- `main/src/frontend/src/pages/login_page/login_page.tsx`
- `main/src/frontend/src/pages/login_page/login_page.pt.ts`
- `main/src/frontend/src/pages/under_development_page/under_development_page.tsx`
- `main/src/frontend/src/pages/under_development_page/feed_construction_hologram.tsx`
- `main/src/frontend/src/pages/under_development_page/feed_construction_hologram.module.css`
- `main/src/frontend/src/pages/under_development_page/feed_construction_scene.ts`
- `main/src/frontend/src/routes/app_routes.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Verification:

- The isolated Feed scene passes TypeScript compilation.
- `git diff --check` passes for the changed tracked source files.
- Full frontend typecheck remains blocked by the pre-existing syntax error in
  the user-modified `home_story/story_scene.ts`; it was not changed in this
  task.

# Version 2.123.1

Title: Hero scene without the table, closer and deeper; the phone always faces the visitor

Date: 2026-11-15

Summary:

Owner feedback: no table needed; a perspective with more depth, the scene a
little to the right and turned slightly to the left, the elements closer
together (like the reference food-and-fitness photo); and the same phone look
as on the hero in the other sections, front only, with its brighter edge.

Main changes:

- `home_story/story_intro.ts`: the stone table is gone; an invisible floor
  only receives the shadows, with a soft contact shade under each object. The
  composition is held close: the plate in front on the left, on a linen napkin
  with the fork and knife, overlapping the phone's corner; the phone at the
  centre; the two dumbbells on the right with their axes towards the visitor
  (the RYZE end faces in view); the shaker behind them.
- `story_scene.ts`: the hero group is tilted more (0.36) for depth, turned
  slightly left (towards the copy) and moved right (`heroX` 0.38). The phone no
  longer spins in the Train, Nutrition and Evolve chapters: it always faces the
  visitor and its screens cross-fade (training → nutrition at 0.44–0.49,
  nutrition → synced at 0.66–0.70). Its titanium edge has a little emissive
  light, so it reads near-white at every angle, as on the hero.

Verification (short, as asked): typecheck passes; the 3D starts with no
console errors; the hero checked at 1440 px.

# Version 2.123.0

Title: Hero becomes one scene on a stone table

Date: 2026-11-15

Summary:

Owner feedback: the hero still read as three separate objects by three
different artists (a quality phone, a blurry plate, a toy dumbbell) over a
black reflective patch where nothing seemed to rest; it lacked life. Wanted: a
composed scene like a food and fitness photograph, one continuous surface with
depth, contact shadows and very subtle reflections, and small signs of life
(steam, light on the phone, a slight camera drift, discreet particles, light
that breathes), without turning it into a circus of effects.

Main changes (`home_story/story_intro.ts`, `story_scene.ts`):

- One light stone table (chamfered slab with thickness, cool marble-like
  texture, a mid-light tone so shadows read), receiving the key light's
  shadows, with contact shade under each object and a soft shade below it.
  The dark reflective disc and its reflection pass are gone (also the main
  cause of the lag).
- Props redone to one standard: a pair of matte black rubber hex dumbbells
  (chamfered heads, knurled chrome handle, chrome collars, the RYZE mark
  printed on the end faces); a plate like a food photograph — grilled chicken
  breast with grill bars, broccoli florets of many buds, quinoa grain by grain
  and diced peppers (instanced, one draw call each), herb flakes; a linen
  napkin under the fork and knife; a RYZE shaker at the back (smoky body,
  black lid, cyan level line, the mark on a curved label).
- Life: steam rising from the plate, a streak of light crossing the phone's
  glass every few seconds, a slow camera drift (the table no longer turns with
  the pointer, so its shadows stay valid), motes drifting in the light, the
  cyan rim light breathing. Shadows are redrawn only while the opening plays
  or the story scrolls.
- The opening (drone, beam, holograms, change to solid) is unchanged and now
  materialises the whole scene; hologram twins support instanced parts.

Verification (short, as asked): typecheck passes; the 3D starts with no
console errors; the hero checked at 1440 px.

# Version 2.122.2

Title: Lifter badges showcase in a modal

Date: 2026-11-15

Summary:

Owner request: the "see all badges" control should match the card's design
and open a pop-up over a dimmed page showing every badge in colour, animated,
as a showcase (not greyed out as locked).

Main changes:

- `components/lifter/`: "See all badges ›" is now a link-style button under
  the level line of the lifter card. It opens a native `<dialog>` (dimmed and
  blurred backdrop; Esc, the close button or a click outside close it; focus
  returns to the button) with the 8 levels in a grid — 4 across, 2 on phones —
  each tile with the badge floating and glowing, its level number, name and
  starting EXP; the member's own level is tagged. No locked or grey state.
  Reduced motion stops the animations.

Verification: typecheck passes.

# Version 2.122.1

Title: Hero composition refinements and a lighter 3D frame

Date: 2026-11-15

Summary:

Owner feedback: the opening was laggy, the scene sat too close to the copy,
the dumbbell should be larger and show its RYZE mark, the plate larger and
further forward, and the near-white metal of the hero phone should be on the
phone everywhere on the page.

Main changes:

- Performance: the surface reflection renders at a third of the canvas width
  (384–640 px) and refreshes every other frame (every third once the opening
  is over); shadows use a 1024 map and are redrawn every other frame during
  the opening, every fourth after; small parts (seeds, leaves, lenses) cast no
  shadow; the dumbbell's and plate's bounds are measured once; after the
  opening only the steam is updated.
- Layout: the composition moved right (`heroX` 0.34); the dumbbell is 30%
  larger, turned so its near end cap — polished steel, a dark face with the
  RYZE mark and a cyan ring — faces the visitor, with matte rubber hex heads;
  the plate is 20% larger and further forward, the fork and knife angled
  beside it.
- The 3D phone's frame is near-white titanium in every chapter (Train,
  Nutrition, Evolve), not only on the hero.

Verification (short, as asked): typecheck passes; the 3D starts with no
console errors; the hero composition checked at 1440 px.

# Version 2.122.0

Title: Hero final composition on one surface; all lifter badges; "no entry" hologram

Date: 2026-11-15

Summary:

Owner requests of 2026-11-15: (1) keep the hero opening but reorganise its
final composition — the phone as the protagonist standing on a surface, the
dumbbell resting behind it on the left, a served plate with a fork, a knife
and a little steam behind it on the right, nothing floating; (2) in the member
area, a small button that shows all 8 lifter badges in one row, animated, with
each level's name and EXP; (3) on the under-development page, a hologram of a
"no entry" sign on a post instead of the planet.

Main changes:

- `home_story/story_intro.ts`: one deep-blue presentation surface (a
  `Reflector` with a discreet reflection that fades out, a shadow catcher for
  real shadows from the key light, contact shade); the phone stands on its
  bottom edge on a clear acrylic stand, leaning back, about 55% of the stage
  height, its pose locked to the surface at the top of the page and easing into
  the story pose as it scrolls (`seat`); the dumbbell rests on a flat face of
  its hex heads, partly behind the phone; the plate (grilled salmon on quinoa,
  avocado, asparagus, halved cherry tomatoes, lemon) with a fork and a knife
  beside it and a soft steam once solid. Every object is set on the surface by
  its lowest point. The opening (drone, beam, holograms, change to solid) is
  unchanged and ends in this composition; shadows grow with the change line.
- `story_scene.ts`: shadow map on (the key light frames the composition; drawn
  only while the hero is on screen); the surface tipped towards the viewer for
  a three-quarter view.
- `home_story/hero_showcase.*` (phones): the same arrangement — the phone in
  front, the dumbbell lying behind on the left, the plate with cutlery and
  steam on the right, on one surface.
- `components/lifter/`: "See all badges" on the member home's lifter card
  opens the 8 level badges in one row (floating, the current one glowing,
  levels not reached dimmed) with name and starting EXP.
- `pages/under_development_page/blocked_hologram.*`: a cyan hologram of a
  "no entry" sign on a post, projected from a floor disc (scan lines, a
  climbing band, particles); replaces `BrandGlobe` there.

Verification (kept short at the owner's request): typecheck and build pass;
the desktop 3D starts with no console errors and the final composition was
checked at 1440 px; the phone layout's final composition at 390 px.

# Version 2.121.0

Title: Home hero opening (drone projection and materialisation), planet RYZE back, lighter devices

Date: 2026-11-15

Summary:

Owner requests of 2026-11-15: (1) restore the planet in the "Get started"
chapter with "Your next level starts here, join planet RYZE now!"; (2) the
phone and laptop looked odd against the more vivid background; (3) a new
cinematic opening for the Home hero: a drone flies in from the right, projects
the phone (real RYZE interface), a dumbbell and a meal as holograms, closes its
beam with a flash and leaves as the holograms turn into solid objects that stay
as the hero's composition — reusing the drone, projection and phone, never
blocking the CTAs, with reduced motion going straight to the final
composition and a light version on mobile without WebGL.

Main changes:

- `components/home_story/story_intro.ts` (new): the opening, procedural.
  Drone in (overshoot and settle) → beam opens (lens flash, scan pulses,
  particles, light pool) → phone, dumbbell and meal build as holograms from
  the bottom up (hologram twins with the same geometries: fresnel, scan lines,
  glowing edges; the phone's screen shown as a cyan hologram of the real
  training screen) → suspension → beam closes with a flash → drone leaves
  while each object turns solid behind a bright climbing line (its real
  materials clipped by a moving plane, the twin above it), with a flash, a
  shock ring and a contact shadow. About 7 s; plays once when the story is
  first on screen; any scroll jumps to the final composition; full frame rate
  while it plays. The dumbbell (knurled steel, chrome collars, gunmetal hex
  heads with a cyan inlay and the RYZE mark) and the meal (white ceramic bowl
  with grains, salmon and sesame, avocado, cherry tomatoes, broccoli, edamame,
  chickpeas) float and turn slowly afterwards.
- `story_scene.ts`: the hero composition is the story's own phone at the
  centre, with the dumbbell and meal; as the hero scrolls on they leave up and
  away and the phone rises out of the frame, coming back for the Train chapter
  (the rest of the timeline is unchanged). Local clipping on; the device metals
  are light (natural titanium phone with a frosted blue back and a soft light
  rim and shadow; silver-blue aluminium laptop; light glows instead of the
  saturated blue halos). The hologram athlete, its platform and the readout
  are no longer in the scene.
- `components/home_story/story_planet.ts` (new): planet RYZE restored from the
  committed original unchanged (land mask, clouds, routes from Lisbon, cities,
  the badge), shown in the last chapter; `story_content.ts`: "Your next level
  starts here, join planet RYZE now!" / "O teu próximo nível começa aqui,
  junta-te já ao planeta RYZE!".
- `components/home_story/hero_showcase.tsx` (new, HTML/SVG/CSS): the same
  opening for phones, reduced motion and no-WebGL (the phone with the real
  screen in `PhoneMockup`, SVG dumbbell and meal, one run of about 7 s; with
  reduced motion the final composition). The stacked last chapter shows
  `BrandGlobe`. `components/showcase/` mockups: natural titanium and silver
  aluminium frames, light rim and a deeper shadow.
- The desktop loading placeholder is gone (the opening starts from an empty
  stage). `story_athlete.ts`, `story_hologram.ts`, `story_readout.tsx` and
  `components/brand_hologram/` are no longer used; kept in the working tree
  pending the owner's decision (they were never committed).

Verification: frontend typecheck, 168 tests and the production build pass
(`story_scene` chunk 60 kB, 20.7 kB gzip). Browser checks on the local stack:
1440 px software-WebGL frame sequences of the whole opening (drone in, beam,
holograms, beam closing, the change to solid, the final composition), 1280
and 1920 (EN) final compositions beside the copy, the hero CTAs on top and
clickable during the opening, the scroll chapters (phone out and back, lighter
devices, planet RYZE with the new headline) and no console errors beyond the
unauthenticated 401s; at 390 px the light opening frozen at several moments,
the final composition, reduced motion, the lighter mockups, the planet in the
last chapter (also at 820 px), no Three.js download and no horizontal
overflow. Real-GPU frame rate was not measured here.

# Version 2.120.0

Title: Home hologram becomes a training analysis that links the scene to the app

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: the hologram should become an analysis — the
drone takes information while the athlete trains, so the scene connects to the
app — reusing what exists; with a brief to evolve (not replace) the scene into
a premium "Human Performance" visual: a translucent holographic body with the
wireframe as a subtle overlay, a refined scanning device instead of a toy-like
craft, a platform with depth, reflections and energy pulses, controlled light,
accurate RYZE interface elements, and no invented biometric or AI body
analysis (the scan is a visual metaphor). Checked at 1440 px and 390 px.

Main changes:

- What the scan "collects" is only what RYZE records, in the workout player's
  terms: the set (kg × reps, "Logged"), the exercise's muscles, the rest before
  the next exercise and, when the routine ends, the saved workout's summary
  with the EXP the backend awards (workout +100, 5 per set, +50 per record),
  the Average Lifter level bar and the record medal. Sample data, labelled.
- `components/home_story/story_readout.tsx` (new) with
  `story_readout.module.css`: the hero readout in the navy-glass style of the
  story's callouts — the session card ("Set 2 of 3 · Logging", exercise,
  kg × reps, a "Logged" pill when the set's data reaches the scanner, set
  progress, muscles; a rest view with "Up next") and the workout card ("Today's
  workout", x/12 sets; then "Workout saved", +210 EXP with its breakdown, the
  level bar and "New medal"). A line joins it to the worked muscle (the side
  nearer the card) and a dashed data line to the sensor. Placed every frame
  from anchors the scene projects (no React re-render per frame); it measures
  the hero copy's real text lines and the story steps and takes the first
  place that covers neither, sliding down if needed, or steps aside when there
  is no room (narrow desktop stages around 1024 px). Everything follows from
  the scene's time, so it cannot drift. Translated (PT/EN copy in
  `story_content.ts`); `aria-hidden` and pointer-transparent. Leaves as soon
  as the story scrolls on.
- `story_hologram.ts`: the drone is now a scanner (satin graphite lens-shaped
  pod with a dome, a light ring, a sensor array of rings and a core, slim arms
  to ducted rotors; pitched slightly towards the viewer; its lights brighten
  when data arrives). The beam is softer; the descending rings gave way to a
  scan plane that follows the scanner's line up and down the body. Every rep
  sends a burst of data from the worked muscle to the sensor. The platform is
  a dark navy disc (unlit top, so the grazing view never mirrors the studio
  light) with a metal side and bevel, a light rim, a drifting light strip,
  rings, a pulse when a set is logged, and a faint reflection of the athlete
  masked by the stencil to the platform.
- `story_athlete.ts`: translucent material (fresnel rim, inner volume, key
  light, faint slices) with a crisp scan contour and afterglow; the light
  network is now a quiet overlay, lit where the scan passes and over the
  worked muscles; a slow breath and head turn; a short rest after each
  exercise's last set; the reflection (the same skinned body mirrored in the
  shader); the state the readout needs (exercise, time into it, round).
- `story_scene.ts`: stencil buffer on; projects the scanner, worked muscle and
  platform to stage pixels for the readout every drawn frame; the hologram
  sits slightly further right (`holoX` 0.41) to leave room for the readout.
- `components/brand_hologram/` (phones, reduced motion, no-WebGL, loading
  placeholder): the scanner redrawn, a dark platform that pulses at the end of
  every rep, a scan line running up and down, and on the phone layout a small
  readout card (curl, then squat) that lights "Logged" with each rep.

Verification: frontend typecheck, 168 tests and the production build pass
(`story_scene` chunk 61.5 kB, 21.8 kB gzip). Browser checks on the local stack
with software WebGL: 1440×900 frame sequences across the routine (logging,
rest view, the workout summary after the last set, the next round), 1024, 1280
and 1920 (EN) placements, the scroll chapters (the readout leaves with the
hero), and no console errors beyond the unauthenticated 401s. At 390 px the
phone layout shows the SVG scanner and its card, loads no Three.js and has no
horizontal overflow, also with reduced motion. Real-GPU frame rate was not
measured here.

# Version 2.119.0

Title: Home hologram: a high-level training mannequin that cycles exercises

Date: 2026-11-15

Summary:

Owner feedback on v2.118.0: the drone, beam and spotlight are right, but the
athlete looked like "a set of balls stuck together". Requested: a hologram
that really shows a human, ideally a genderless figure with no defined face,
like the owner's reference (a glowing triangulated network body); the figure
should alternate between exercises to fill the space and show progress
everywhere, and the area being worked should pulse gold or red.

Main changes:

- `components/home_story/story_athlete.ts` (new, procedural, no model files,
  no new dependency): one continuous faceless mannequin built from signed
  distance fields (ellipsoids and round cones for pelvis, glutes, ribs, pecs,
  deltoids, biceps, triceps, forearms, quads, hamstrings, calves, an egg head
  with a soft jaw) joined with smooth unions, meshed once with surface nets
  (vertices projected onto the exact surface, normals from the field) into
  ~17k vertices, and skinned to a 17-bone skeleton with weights from the same
  fields. The build (~0.6 s of script on a desktop CPU) runs in slices of at
  most ~12 ms while the SVG placeholder shows. Over the skin, a triangulated
  light network (~900 nodes, ~2,100 links along the skin only) with twinkling
  nodes, skinned on the CPU. Exercises with dumbbells, three reps each: biceps
  curl, shoulder press (racks the dumbbells on the way in and out), squat and
  Romanian deadlift; the feet stay planted (exact leg solve), and between
  exercises the figure turns to the angle that shows the next one best. The
  worked muscles (biceps; deltoids and triceps; quads and glutes; hamstrings
  and glutes) pulse gold, turning red at the hardest point, with a soft glow
  over each.
- `story_hologram.ts`: the sphere athlete and the bench are removed; the
  drone, beam, scan rings, particles, floor disc and halo stay. The beam now
  leaves a wide lens (a frustum) so the press and the hip hinge stay inside
  the light; `createHologram` is async (the scene awaits it). A WebAssembly
  mesh simplifier was rejected because the production CSP (`script-src
  'self'`) would block it; the CSP is unchanged.
- `story_scene.ts`: the hologram's base scale is slightly larger (0.95).
- `components/brand_hologram/`: the SVG/CSS fallback (phones, reduced motion,
  no-WebGL, loading placeholder) now shows the same faceless mannequin in
  profile, covered in a triangle-and-node network pattern, alternating three
  curls and three squats in a 12 s cycle (the joints fold ankle → knee → hip),
  with the biceps, then the quads and glutes, pulsing gold to red. With
  reduced motion it stands still with a steady biceps glow.

Verification: frontend typecheck, 168 tests and the production build pass
(`story_scene` chunk 53.8 kB, 19.4 kB gzip). A Node check of the athlete
builder: consistent winding (1 of 33,824 triangles off), no invalid values,
feet at the same height in every pose. Browser checks on the local stack
(software WebGL at 1440×900): frame sequences of all four exercises and the
transitions, the press staying inside the beam, the story chapters (hero,
corner, out of frame, back for the call to action) and no console errors
beyond the unauthenticated 401s; the SVG frozen at several points of its
cycle on a 390 px phone. Real-GPU frame rate was not measured here.

# Version 2.118.0

Title: Home hero: a drone-projected hologram athlete replaces the Earth

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: the planet in the Home story "does not say
anything"; replace it with a 3D glass hologram of an athlete (suggested: a
drone projecting, from above, an athlete sitting on a bench doing a dumbbell
curl), conveying the hero's message — train anywhere, progress your body in
every way — with the effects proposed (muscles lighting up, light, glass).

Main changes:

- `components/home_story/story_hologram.ts` (new, procedural, no model files):
  a drone (metal hull, LED strip, spinning rotors, a glowing lens) hovers and
  projects a cone of light (streaks, three descending scan rings, rising
  particles) onto a floor disc (rings, a turning dashed ring, a pulse). Inside,
  the athlete — a V-taper lathe torso with pecs and abs, deltoids, biceps,
  triceps, forearms, quads and calves on a joint hierarchy — sits on a bench
  and does alternating dumbbell curls (2.6 s per rep). Hologram shader: bright
  rim, horizontal scan slices, a light band climbing the body on every rep, a
  faint flicker and a rare glitch, over a depth pre-pass so only the nearest
  surface shows; the working biceps light up with each contraction; a soft
  inner glow; the hologram materialises from the floor up when the scene
  starts.
- `story_scene.ts`: the Earth (sphere, continents, clouds, routes, cities,
  Lisbon badge, their shaders and the land-mask download) is removed; the
  hologram takes its timeline (`holo*` state keys): hero → corner while the
  programs pass → flies up out of the frame while the devices are on stage →
  back for the last chapter. Calmer while in the corner.
- `components/brand_hologram/` (new): the SVG/CSS equivalent for phones,
  reduced motion and no-WebGL (drone, beam with descending rings, the athlete
  in profile curling with alternating arms, a climbing light band); also the
  placeholder while the 3D scene loads. The SVG globe stays on the
  under-development pages.

Verification: frontend typecheck, 168 tests and the build pass (the 3D scene
stays a lazy desktop-only chunk, 37.6 kB). Local site in headless Chromium
with software WebGL: no shader or page errors; frame sequences show the
alternating curls, the reveal, the climbing band and the biceps glow; the
whole story at 1440 px (hero, corner, Train without the drone, return in the
last chapter) and the hero at 1920 and 1100 px; phones at 390 px show the SVG
version (no canvas, no Three.js, no overflow) and stay still with reduced
motion.

---

# Version 2.117.5

Title: Story steps show only the current chapter

Date: 2026-11-15

Summary: owner feedback of 2026-11-15 — the side steps should not show all
five, only the current one, and without the bubble (the glassy look stays).
The current step reads "02 — TRAIN" in glass type (white to ice gradient, soft
glow) with a light-bar dash, fading and sliding in on each chapter; the others
appear on hover or keyboard focus to keep the navigation. Checked on the local
Home at 1920 and 1440 px (one step visible per chapter, all five on hover, no
page errors); typecheck, tests and build pass.

---

# Version 2.117.4

Title: Story steps without a panel

Date: 2026-11-15

Summary: owner correction of 2026-11-15 — no block around the five story
steps, only the glass on each step. The panel was removed; every row now has
its own frosted glass (stronger once passed, bright with a cyan glow when
current), which keeps the rows readable over the 3D phone and laptop.
Verified on the local Home at 1920 and 1440 px through the chapters;
typecheck, tests and build pass.

---

# Version 2.117.3

Title: Story steps as glass rows

Date: 2026-11-15

Summary:

Owner request of 2026-11-15, with a reference image: show the Home story's
side steps as "01 — ASSESS / 02 — TRAIN …" rows, the current one in bright
glass.

Main changes:

- `components/home_story/story_immersive.module.css`: the steps are a small
  frosted panel on the right edge; each row reads "01 — NAME" (names always
  written from 1600px), the current chapter is bright glass with a cyan edge
  and glow, the passed ones faint glass, the next ones plain text, with a
  cross-fade between states. Below 1600px the rows show numbers only, because
  the phone and laptop would otherwise sit under the names. The step buttons
  carry `data-state` (current, done, next) for this.

Verification: frontend typecheck, 168 tests and the build pass; the local
Home at 1920, 1536, 1440, 1280 and 1100 px through the chapters (the steps'
states follow the chapter on screen; no page errors). At 1920 px the full
panel is clear of the phone in Train and touches the edge of the small phone in
Evolve under its glass.

---

# Version 2.117.2

Title: No empty space between "Find your next program" and Train

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: there should not be so much free space between
the "Find your next program" section and the Train section of the Home.

Main changes:

- Desktop story (`components/home_story/story_immersive.tsx`): the Train copy
  follows the leaving program cards 48px below them (as if it scrolled with the
  page), fades in over its first 220px on screen and settles in its place once
  the cards have gone; it still fades out with its window. The programs keep
  only 1.5rem of bottom padding inside the story. Before, two thirds of the
  screen stayed empty for about 500px of scrolling.
- Phones: the Train chapter after the programs has a short top padding (96px
  from the cards to the Train label), and the swipeable cards' shadow now fits
  inside the row (it was cut in a straight line on the brighter field).

Verification: frontend typecheck, 168 tests and the build pass; on the local
Home at 1440 px the Train copy stays 72px under the cards while they leave
(measured at six scroll positions) and rests in place after; at 390 px the
gap is 96px with no edge under the cards; no page errors.

---

# Version 2.117.1

Title: Language switch written as "PT / EN"

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: replace the PT | EN pill in the navigation bar —
write "PT / EN" like the rest of the navigation; the language in use gets a
small light and the other is darker; animate the change.

Main changes:

- `components/language_switcher/`: "PT / EN" in the navigation's type, no
  pill or bubble; the active language is white with a glow and a cyan light
  under it, the other dimmed (hover brightens it). On change the light
  stretches to the other side (leading edge first, trailing edge 110 ms later)
  and the new language lights up; no motion with reduced motion. Behaviour,
  keyboard, accessible name and saving the choice are unchanged. Used by the
  public navbar, the app shell's top bar, the account page and the account
  recovery pages.

Verification: frontend typecheck, the language tests (9) and the build pass;
the local site at 1440 and 390 px shows PT active, the transition frames and EN
active after one click, with no page errors.

---

# Version 2.117.0

Title: Back buttons everywhere, Premium pages on the public layer, 60-30-10 colour

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: back buttons across the site (sign-in, the
administration consoles, the plans catalogue — "what if I just want to go back
one page?"); the Premium Level 1/2 pages showed the bare page background; and
the site's background was "opaque", without contrast — apply the 60-30-10 rule
so the last colour brings it to life, attract attention instantly (reference:
a vivid blue/white 3D site), optionally a lighter glassy theme from the brand
icon, navbar included.

Main changes:

- Back buttons: `components/back_button/`, `hooks/use_go_back.ts` and
  `services/navigation_history.ts` (the tab's RYZE history, kept across
  reloads). In the public navbar (not on Home), on login/registration,
  password recovery and admin login, and in the app shell's top bar (member,
  trainer, ADMIN_1, ADMIN_2; hidden on each area's home). With nothing behind,
  the button goes to the page's parent, never to another site.
- Premium pages: the Level 1/2 journeys, the legacy Level 1/2 pages, the Level 2
  request page, the plan detail and the coach profile now use the public layer
  (`ui.page`), so they no longer show the bare background or a seam before the
  footer.
- Colour (60-30-10, `public_ui`): 60% a continuous field of the logo's blues
  (brightest on the first screen, a rhythm of light, cyan haze and deep pools,
  a fine CSS sparkle; no images); 30% light glass (translucent white with blur)
  for cards, panels and the navbar; 10% cyan and white accents (primary buttons
  glow). Lighter secondary text for contrast. Applied to the public layer, the
  navbar (light frosted glass), the 3D Home story backdrop palette, Services,
  the plans catalogue (glass tokens), the L1/L2 journey cards, the footer (deep
  end of the field; layout, columns and copy unchanged) and the Exercise
  Library and detail (page tokens only; layout, components and viewer
  unchanged).
- Fixed while verifying: the contact form kept a blur from the shared glass
  inside the split panel (a lighter box); a deep-pool glow crossed its 125rem
  tile and showed a line every 2 000 px; the back button had no accessible
  name when its label is hidden on phones.

Verification:

- Frontend: typecheck, 168 tests (new: navigation history) and the build pass.
- Local site (public pages) at 1440 and 390 px before and after: Home, Services,
  L1, L2, plans, login, contact, Our Vision, feedback, Exercise Library, the
  footer; no page errors.
- Isolated stack (API :8081 on `ryze_test`, Vite :5175): 10/10 back-button
  checks (Home without a back button; navbar back to the previous page; a
  directly opened page back to its parent; login back to the previous page;
  member area back and phone fallback to the member home; ADMIN_2 fallback to
  its dashboard; no page errors). Demo data removed afterwards.

Open:

- The private areas (member, trainer, administration) keep the app's navy
  theme; the field and glass can be extended to them if the owner wants.
- The Services tiers keep their own dark glass panels (the owner's design),
  which now stand out against the field.

---

# Version 2.116.2

Title: Level badges from the owner's artwork

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: use the eight level icons the owner placed in
`main/src/resources/img/badges/badges.png` (a 4 × 2 sheet), cut one by one.
Each badge was cut from its cell in rank order (Rookie, Beginner, Average,
Dedicated, Advanced, Elite, Master, Legendary), keeping its body and nearby
glow and fading out the stray glow specks, then centred on a transparent
256 × 256 WebP (`rookie.webp` … `legend.webp`, 30–36 kB each, next to the
original sheet, which stays unchanged and is not bundled).

Main changes:

- `LevelBadge` (`components/lifter/`) shows the level's artwork with the level
  number on a small chip (no chip in the level ladder) instead of the drawn
  hexagon; the picture is decorative because the level name is always written
  next to it.
- Used in the member home's lifter card, the achievements hero and level
  ladder (levels not reached are shown in grey) and the level-up message of a
  saved workout.
- The lifter card's first column is wider so the level name stays on one line
  at 1100–1440 px.

Verification: frontend typecheck, 164 tests and the build pass. Isolated stack
with the demo member at 1440 and 390 px: the badges load on the home (1), the
achievements page (9) and the level-up card, the level name fits one line at
1440 and 1100 px, no page errors. Demo data removed afterwards.

---

# Version 2.116.1

Title: Home "Pick how much guidance you want" section removed

Date: 2026-11-15

Summary:

Owner request of 2026-11-15: remove the "Pick how much guidance you want."
section from the Home page. The Home now goes from "Find your next program"
(and the story chapters) straight to "From discovery to your first session".
`components/category_overview/` and `hooks/use_service_catalog.ts`, used only by
that section, were deleted; the Home no longer loads the catalogue. The
Services tiers still present the three levels.

Verification: frontend typecheck, 164 tests and the build pass; the local Home
at 1440 px (EN) and 390 px (PT) no longer shows the section, with no horizontal
overflow and no page errors.

---

# Version 2.116.0

Title: EXP, lifter levels and record medals; plans for women or men; sex in Premium Level 1

Date: 2026-11-14

Summary:

Owner request of 2026-11-14: a medal every time a weight record is beaten
("like Lyfta"), EXP for every workout towards lifter levels such as "average
lifter" and "elite lifter" (levels left to the implementation), shown among
the first things in the member area; the member's sex in the Premium Level 1
journey; a "plan for women / for men" option when creating a plan; and, the
first time someone opens the training plans, a question about their sex and
whether they prefer to see the plans for it. One additive migration (000045).

Main changes:

- Achievements (`services/achievements`, `GET /me/achievements`,
  `GET /me/achievements/sessions/:id`): computed on read from the completed
  workout sessions, nothing stored. 100 EXP per finished workout + 5 per
  completed set (max 40) + 50 per record medal; at most two workouts a day
  earn EXP. A medal when a workout's heaviest set of an exercise beats the
  previous best (warm-ups ignored; the first time sets the record; one per
  exercise per workout). Eight levels: Rookie 0, Beginner 500, Average 1500,
  Dedicated 4000, Advanced 8000, Elite 15000, Master 25000, Legendary 40000.
- Member area: the lifter card is the first card of the home (level badge,
  EXP bar, medal count and latest medals); new "Achievements" page
  (`/achievements`, secondary navigation) with all medals, the level ladder and
  the rules; the saved-workout summary shows the EXP earned, a level up and
  the medals.
- Plan audience (`programs.audience`): "Designed for" (Everyone / Mostly women
  / Mostly men) in the shared plan builder (ADMIN_2 and trainers), the plan
  facts and the review; never on a Level 2 offer. The public catalogue accepts
  `audience=female|male` (plans for everyone plus that audience) and returns
  `audience`.
- Plan preferences (`users.sex`, `users.plan_audience_filter`,
  `PATCH /me/plan-preferences`, both on `GET /me`): the training plans ask once
  (account for members, browser for visitors) and apply the "Designed for"
  filter; "Plans for me" reopens the question.
- Premium Level 1: the journey asks Woman / Man (pre-filled when known);
  `POST /me/premium-level-1/match` requires `sex`, only picks packages for
  everyone or that sex, and stores it on the match.
- Fixed while verifying: `/achievements` was missing from the app's private
  paths, so the public navbar appeared over the member area.

Verification:

- Backend: `gofmt`, `go vet ./...` and `go test -p 1 ./...` on `ryze_test`
  pass (60 packages; new tests: achievement rules, levels, audience
  validation, plan preferences, and a repository test for the completed-set
  query, session ownership, preferences and the catalogue audience filter).
- Frontend: typecheck, 164 tests and the build pass.
- Isolated stack (API :8081 on `ryze_test`, Vite :5175): 51/51 checks. The API
  EXP (2 815) and medals (22) of a demo member with eleven workouts match an
  independent copy of the rules; a live workout with heavier loads earned 3
  medals and 300 EXP; another member gets 404 for that award; preferences,
  catalogue audience, Level 2 refusal and the sex-aware match behave as
  documented. Browser at 1440 and 390 px: lifter card first on the home,
  achievements page, workout rewards, Level 1 sex choice, the plans question
  for a visitor (filtered for men, not asked again) and for a member (saved on
  the account), and the builder's "Designed for"; no horizontal overflow and
  no page errors. Demo data removed afterwards.

Open:

- The EXP rules and level names are the implementation's proposal (the owner
  asked to create the levels); change them in `services/achievements` and the
  `LEVEL_NAMES` copy.
- Workouts recorded only through the legacy `POST /me/workouts/:id/complete`
  (without sets) earn no EXP; the player always creates sessions.
- Exercise names on medals are the names saved with each workout (some
  imported catalogue names are lower-case English).
- The "plan for women / men" applies to new and edited plans; existing plans
  are for everyone until edited.

---

# Version 2.115.1

Title: Home goal cards restored

Date: 2026-11-14

Summary:

Owner report of 2026-11-14: Home's "Find your next program" still showed the
published plan "Plano 1" (its name and description) in place of the original
"Train for free" card. The section now always shows the three original goal
cards — Build muscle → Premium Level 1, Lose fat → Premium Level 2, Train for
free → training plans — with their images (`advert1–3`, protected) and no
dependency on the catalogue. `buildProgramHighlights` and its tests were
removed (no other consumer). Services and the category overview still list
what is published.

Verification: frontend typecheck, 164 tests and the build pass; the local
Home (one generic plan published) shows the three goal cards with the right
links and images at 1440 px (EN) and 390 px (PT), with no page errors.

---

# Version 2.115.0

Title: Services tiers, Premium Level 1 and Level 2 purchase journeys, Lyfta-style player

Date: 2026-11-14

Summary:

Owner request of 2026-11-14: Services and Home's "Find your next program"
showed a single card after one plan was published; a complete rework of the
Services cards copying the owner's image (three rising levels in the RYZE
colours, so the card background images can go; everything below the cards
unchanged); the Premium Level 1 flow as planned (service → "Learn more" → goal
cards → weight, age and height → RYZE picks a fitting plan at random, never
the same plan twice → payment → plan assigned); the Premium Level 2 flow
(service → goal cards → long questionnaire → payment last → service
assigned); a visible payment screen; and a workout mode closer to Lyfta. One
additive migration (000044).

Main changes:

- Cause of the single card: Services and Home were catalogue-driven and showed
  only published families (the fallback appeared only when nothing at all was
  published). Services now always shows the three tiers; Home fills the
  missing categories with the original goal cards (always three).
- Services: `components/service_tiers/` replaces `components/service_card/`
  and `components/service_showroom/` (deleted, with `SERVICE_SHOWCASE`):
  Nível 1 Grátis, Nível 2 Premium 14,99 €, Nível 3 Personalizado 19,49 € with
  the crown and the main CTA, rising glass panels with slanted lit edges, the
  arrow and the benefits strip; stacked on phones. `rcc1–3.png` are no longer
  referenced.
- Premium Level 1 journey (`pages/premium_level1_journey/`) and
  `POST /api/v1/me/premium-level-1/match` (rate scope `premium-match`): a
  random published curated package for the objective, excluding packages the
  client owns or bought (`409 ALL_PLANS_OWNED`, `404 NO_MATCHING_PLAN`). New
  objective `sport_specific`. Each pick is stored in `premium_level_1_matches`.
- Premium Level 2 journey (`pages/premium_level2_journey/`):
  `GET|PUT /api/v1/me/premium-level-2/offers/:programID/intake-draft` saves the
  questionnaire section by section; checkout requires a complete draft
  (`409 PURCHASE_PREREQUISITE_NOT_MET`, regular and Test Mode); fulfilment
  turns it into intake revision 1 and deletes the draft. The legacy `?program=`
  page no longer sells: it links to the journey for that offer, and the Home
  Level 2 card opens the journey (`?offer=`).
- Payment screen: `components/checkout_panel/` (order summary, total,
  payment-method cards, Test Mode notice, a plain "payments are not available
  yet" message when no provider is configured, cancelled/error/confirmed
  states), shared by both journeys. Guests return to the journey after signing
  in (`services/after_login.ts`).
- Workout player: exercise pictures, the training day's name, a stats strip
  (duration, volume, sets), current/done states with a "Current" tag, a rest
  chip per exercise and a ring rest timer with −15 s / +15 s / Skip (screen
  only; the countdown still follows `rest_ends_at`).
- CORS: `PUT` added to the allowed methods. It was missing, so every
  cross-origin browser `PUT` failed preflight (the Level 2 questionnaire, the
  trainer plan structure and the Level 2 trainer steps in the development
  setup).

Verification:

- Backend: `gofmt`, `go vet ./...` and `go test -p 1 ./...` on `ryze_test`
  pass (57 packages; new tests: match service, partial answers, journey
  repository, CORS methods). Two failures during the run came from a Test Mode
  persona left in `ryze_test` by the browser checks; it was removed and the
  cleanup script now removes personas.
- Frontend: typecheck, 167 tests and the build pass.
- Isolated stack (API :8081 on `ryze_test`, Vite :5175), at 1440 and 390 px:
  Services tiers; Level 1 pick and Test Mode payment; the same goal again
  never returns the owned plan; Level 2 purchase refused before the
  questionnaire; partial and complete drafts; Level 2 payment step and Test
  Mode payment (request `awaiting_trainer_assignment` with intake v1); legacy
  Level 2 link and Home card lead to the journey with the right offer; no
  horizontal overflow and no page errors. Demo data removed afterwards.
- Payments were tested in Test Mode only; no real payment provider was
  exercised.

Open:

- AI analysis of the Level 2 questionnaire is not implemented; the allocated
  trainer prepares the plan and nutrition as before.
- The Level 1 match is a recommendation: checkout does not require it, and a
  package can still be bought from its detail page.
- What the Level 1 body data changes in the plan is still to be decided.
- No retention period for questionnaire drafts of offers never bought.
- Provider return URLs are deployment configuration (`STRIPE_*_URL`,
  `PAYPAL_*_URL`); they were not changed.

---

# Version 2.114.0

Title: Fixed catalogue prices, plan builder rework, plan covers, admin consoles rebuilt

Date: 2026-11-14

Summary:

Owner request of 2026-11-14: an account menu with sign-out; trainers land in
their trainer area after signing in; nobody sets prices (Generic is free,
Premium Level 1 costs 14.99 EUR, Premium Level 2 costs 19.49 EUR) while
administrators set each trainer's commission as a plain percentage; a far
richer first screen for ADMIN_2 and a complete rework of both administration
consoles in the look of the other signed-in areas; a Lyfta-style plan creation
workflow (classification and cover first, weeks created automatically, named
training days with a muscle-group focus, an exercise picker with pictures,
inline prescriptions, adaptive rest times, no price or category choice at
publishing); easier trainer and commission management; and an `uploads`
folder for plan images split into generic, l1 and l2 (with room for profile
and feedback photos). One additive migration (000043).

Main changes:

- Prices (owner decision): `models.CatalogPriceMinorUnits` fixes the price of
  every plan. Platform plans take their family (generic free, Level 1 1499,
  Level 2 1949, EUR); trainer plans take their type (free 0, premium 1499,
  personalized 1949). Create/update stamp it and ignore client prices; publish
  re-checks it; `PATCH /admin/programs/:id/pricing` can only re-apply it
  (`409 PRICE_FIXED`). `/admin/settings/commerce` returns `catalog_prices`.
  Existing purchases keep the price they were sold at.
- Commissions: typed and shown as percentages (30 = 30%) on "Trainers &
  commissions"; the API keeps basis points. Fixed a PostgreSQL bug that made
  every commission override fail (the model wrote the generated
  `active_commission_rule` column, SQLSTATE 428C9).
- Plan covers and training days (migration 000043): `programs.cover_image`
  and `program_workouts.name/description/focus`. Covers are uploaded through
  `POST /admin/uploads/program-cover` and `POST /trainer/uploads/program-cover`
  into `UPLOADS_DIR/programs/{generic,l1,l2}` (JPEG/PNG/WebP by magic bytes,
  5 MiB, UUID names, atomic writes) and served read-only at
  `/uploads/programs/...`. Covers show on the marketplace cards and on Train;
  day names show on the member home and the plan schedule.
- Trainer plans: catalogue metadata (level, goal, weeks, days per week, cover)
  and `GET|PUT /trainer/programs/:id/structure`, which saves the whole
  structure with the platform builder's rules.
- Plan builder (`components/plan_builder`), shared by ADMIN_2 and trainers:
  Plan (category with the fixed price, cover, name, description, goal,
  difficulty, weeks and days) → Structure (weeks created automatically, week
  tabs, copy a week forward, named days with a focus, app-style exercise picker
  with stills and an animated preview, inline sets/reps/kg/rest/RIR and notes,
  a full sheet per exercise) → Nutrition (curated Level 1) → Review (summary,
  checklist, Save draft / Publish). Rest reads 45 s, 1 min 1 s, 1 min 30 s;
  the player uses the same format.
- Account menu and landing: the avatar opens a menu with the area's shortcuts
  and "Sign out"; phones get a "More" sheet. Login and Google sign-in go to
  `/app`, which sends approved trainers to `/trainer/dashboard` (the only
  change to the protected login page).
- Administration consoles: both now use the shared app shell. ADMIN_2:
  `GET /admin/overview/business` (revenue and trend, shares, categories,
  members, activity, catalogue, best sellers, most viewed) and
  `GET /admin/overview/queues` (navigation badges); ADMIN_1:
  `GET /admin/overview/platform` with system health, configuration and the
  latest admin actions. Navigation reorganised; the Marketplace, Analytics and
  Development pages were removed (replaced by the plans list and overviews);
  ADMIN_2 reviews trainer applications in its own console. Plans list rebuilt
  as cards per category.

Verification:

- Backend: `go vet ./...` and `go test -p 1 ./...` on `ryze_test` pass
  (56 packages; new tests: catalogue pricing, fixed-price endpoint,
  commission override repository, uploads store).
- Frontend: typecheck, 167 tests (new: adaptive durations) and the build pass.
- Isolated stack (API :8081 on `ryze_test`, Vite :5175): 27/27 API checks
  (prices, uploads incl. non-image and traversal rejection, cover category,
  day details, trainer structure and type price, permissions, commission) and
  browser checks at 1440 and 390 px (trainer sign-in lands on the trainer
  dashboard, member menu sign-out, both consoles, the whole builder flow
  including save and delete) with no horizontal overflow and no page errors.
  Demo data removed afterwards.

Open:

- Legacy rows priced before this rule keep their price until saved again
  (the local database had no programmes).
- The administration consoles stay in English; the plan builder and the
  member/trainer areas are PT/EN.
- Avatars and feedback photos: folders reserved, no endpoint yet.

---

# Version 2.113.0

Title: Member and trainer areas rebuilt as a desktop and phone app

Date: 2026-11-14

Summary:

Owner request of 2026-11-14: a complete rework of the reserved user areas —
the best possible experience with Lyfta-style information about everything,
built for both PC and phone (the old area looked like a phone screen on a
desktop), in the colours of the login and registration page — and the same for
the trainer area. ADMIN_1 and ADMIN_2 stay as they are. Frontend only: no API,
database, price, access, purchase or workout rule changed; every value shown
comes from the existing endpoints.

Main changes:

- App shell: `components/app_shell/` adds `AppShell`, one layout for the member
  and trainer areas in the login palette (logo navy, royal panels, cyan light,
  the light-blue wave gradient for the main action). Desktop: a 17rem sidebar
  with primary and secondary navigation, a highlighted action card and an
  area switch; at 1180px and below an icon rail; at 767px and below a top bar
  and a bottom tab bar. The top bar carries the PT | EN toggle, the
  notification bell and the account avatar. Tokens are scoped to the shell, so
  public and admin pages are unaffected. Loading states use a shimmer skeleton
  (`AppLoading`) instead of the full-screen loader.
- Member area: `MemberShell` loads `GET /api/v1/me` once for all member pages
  (401 → `/login`) and exposes it through `useMember()`. Navigation: Home,
  Train, Progress, History; secondary: Training plans, Exercises, Purchases,
  Support, Account; a "Start workout" action; a switch to the trainer area for
  approved trainers. Member routes load lazily and keep one shell mounted while
  moving between member pages.
- Member home (`/profile`): replaces the profile card with a dashboard — the
  next workout in the light wave panel (start, resume the live session, plan
  complete or no plan), the current week with a day strip and streak, totals
  (workouts, training time, volume, completed sets),
  an 8-week activity chart, the current plan with a completion ring and
  per-week bars, recent workouts, personal bests from completed sets, and
  shortcuts. Twelve-column grid on desktop, stacked below 1024px.
- Account settings (`/account/settings`): profile, language preference, sign-in
  methods, email verification, password change, trainer status, sign-out and
  account deletion moved here unchanged in behaviour, in two columns on
  desktop.
- Member pages (Train, plan detail, Progress, History, Purchases, Support,
  trainer application) render inside the shell without the old account bar and
  background, use the full width, and gain desktop layouts (plan detail with a
  sticky side column, two-column session lists). `components/account_nav` and
  the old profile page were removed.
- Workout player: still full screen; from 1100px a sticky overview column shows
  the progress ring, the clock, the exercise list with jump links and Finish.
- Trainer area: `TrainerLayout` now uses `AppShell` (Dashboard, Clients,
  Premium L2, Programmes; secondary: Profile, Support; a "New programme" action
  that opens the creation form; a switch to the member area). The trainer
  dashboard is rebuilt in PT and EN: a light hero with "Add client" and "New
  programme", public profile views, tiles for clients, published, drafts and
  Premium L2, and lists of recent clients, programmes and the Premium L2 queue.
  Shared trainer styles follow the shell; the client detail rounds completion
  and fits long programme names.

Not changed / open:

- The admin console (ADMIN_1 and ADMIN_2) is unchanged, as requested.
- Trainer pages other than the dashboard and the shell (clients, client
  detail, programmes, programme detail, Premium L2, profile) keep their English
  copy; translating them is a separate step.
- The Premium Level 2 request page (`/services/premium-level-2/requests/:id`)
  still renders outside the member shell.

Verification:

- `npm run typecheck`, `npm run test` (164 tests) and `npm run build` pass; the
  member area is a separate lazy chunk.
- Browser checks on an isolated stack (`ryze_test`, demo data removed
  afterwards): member home, Train, plan detail, Progress, History, Purchases,
  Support and Account, and the trainer dashboard, clients, client detail and
  programmes at 1440, 1024 and 390 px with no horizontal overflow; "Start
  workout" from the home opens the full-screen player without the shell and
  the home then offers "Continue workout".

---

# Version 2.112.0

Title: Services showroom, PT | EN toggle, glass Exercise Library, Home globe and laptop

Date: 2026-11-14

Summary:

Owner request of 2026-11-14: a glassier Exercise Library with the same layout,
a better Home globe, a fixed and better laptop, a 3D showroom for the Services
cards (arrival from the left, the depth and the right; hover brings a card
forward with floating fact bubbles while the background blurs), a "Free" tag on
the generic plan card and a reworked PT | EN control. Frontend only: no API,
database, price or access change.

Main changes:

- Language: the PT | EN block is one toggle button — a click anywhere switches
  the language (before, only a click exactly on the other option did), a cyan
  light slides to the active language, arrow keys pick a side, and the
  accessible name states the current and the next language.
- Services: `ServiceShowroom` presents the unchanged `ServiceCard`s as products
  on display. After owner feedback the first version (grid floor, spotlight
  cones, page-wide blur, rotated cards) was replaced the same day: from 1024px
  with a precise pointer the cards stand flat and crisp on real 3D pedestals
  (a lazily loaded WebGL stage drawn on demand: navy two-tier pedestals, cyan
  light rings, light pools, ground glow, a soft light behind each card), with a
  placard under each pedestal stating the route's goal; hovering or focusing a
  card lifts it (2D only), lights its pedestal, dims the others and floats out
  four fact bubbles. Phones and touch screens keep stacked cards with the goal
  and the facts as chips. The generic plan card carries a mint "Free" tag.
- Exercise Library: frosted glass toolbar and inset fields, translucent glass
  cards with a top highlight, thumbnail reflection and hover sheen, glass
  tags, fixed page light; translucent tokens on the detail page; filter labels
  translated in Portuguese. Layout and components unchanged.
- Home story: the closed laptop lid no longer cuts through the keys; keys use
  one rounded geometry per size, a cyan backlight, speaker grilles and sharper
  legends. The globe gains a dot-matrix land, night-side city lights, an ocean
  glint, procedural clouds and a brighter atmosphere (an orbit ring with
  satellites was tried and removed at the owner's request).

Verification:

- `npm run typecheck`, `npm run test` (164 tests; the language test now covers
  one-click toggling and arrow keys) and `npm run build` pass.
- Browser checks: Services showroom at rest (pedestals under each card,
  placards under each pedestal), on hover of each card and on keyboard focus
  (1440 px), stacked with goal and chips at 390 px, no horizontal overflow, no
  console errors; the toggle switches when clicked on the active side; Exercise
  Library and detail at 390 and 1440 px; Home story globe and laptop (closed,
  opening, open) rendered with WebGL.

---

# Version 2.111.0

Title: Admin audit log, coach showcase, test stack fixes, Feedback and Our Vision motion

Date: 2026-11-13

Summary:

Owner request of 2026-11-13: verify the local test accounts and the one-command
test stack (with exercise media for local testing only), add life to the
Feedback page, implement the administrative audit log and a Lyfta-style public
showcase of trainer programs ("Featured coach programs" ranked by views, with
clickable coaches), and give the Our Vision hero a 3D "WOW". Two additive
migrations (000041, 000042). No price, commission, duration, access or
purchase rule changed.

Main changes:

- Admin audit log: `admin_audit_logs` (migration 000041) is append-only (a
  trigger rejects UPDATE and DELETE). `middleware.AdminAudit` records every
  state-changing admin request, every evidence download and denied attempts;
  `middleware.AdminSignInAudit` records second-factor sign-ins. Records hold
  the admin identity, action key, method, route template, target id, status,
  outcome and client address — never request content.
  `GET /api/v1/admin/logs` (ADMIN_1, `security` permission, filters admin and
  outcome) and the ADMIN_1 "Audit log" page replace the placeholder.
- Coach showcase: `program_view_stats` (migration 000042) counts program page
  views (`POST /api/v1/programs/:id/views`, once per viewer and program every
  30 minutes, owner visits excluded, `program-view` rate limit 120/10 min).
  `GET /api/v1/coaches/featured-programs` ranks published trainer generic
  programs by views (ties newest first); `GET /api/v1/coaches/:trainerID` is
  the public coach profile; `GET /api/v1/programs/:id` adds `coach {id, name}`
  for trainer-owned programs. Coaches are public only after publishing;
  disabled trainers and Test Mode personas never appear; no contact data is
  exposed. UI: "Featured coach programs" zone at the top of Training plans
  with the coach under each card, "Created by" on the program detail, the
  `/coaches/:trainerId` page and "Open public profile" on the trainer profile.
- Local test stack (`start_test_mode.sh`): account lookups walk every page
  (the first run stopped because the client was beyond the first 100 users);
  accounts that already sign in are left untouched; the trainer can be
  approved through the normal application flow; the API runs as a built
  binary so Ctrl-C stops it; an existing Vite server is reused; new
  `--accounts-only` mode. All four test accounts were verified (client,
  approved trainer, ADMIN_1, ADMIN_2) and the exercise media preview answers
  locally.
- Login fixes (owner report "I can't log in"): the member login form invites
  "Email or username" but member accounts only have emails — a value without
  "@" now shows "Use the email address of your account. Administrators sign
  in at /admin/login." instead of "Validation failed."; the admin login trims
  stray spaces from the username and access code (never the password); in
  local development the API base follows the page host, so opening the site
  as `127.0.0.1:5173` keeps the SameSite session cookie; the local `.env`
  admin usernames were unquoted so a copied value signs in.
- Feedback page: drifting brand lights, staggered entrance and a light sweep
  in the header; a word-only highlights ticker; a breathing light band, a
  travelling edge light, hover lift and Before / After labels around the
  story cards. CSS only; reduced motion stops everything.
- Our Vision hero: from 768px with WebGL, a lazily loaded Three.js tunnel of
  identical app cards streams past the still RYZE card (rings, sparks, halo,
  pointer tilt and parallax; pauses off screen). Phones keep the flat wall
  (now animated in CSS) and never download Three.js.

Verification:

- `go test -p 1 ./...` on `ryze_test`: 55 packages ok; `gofmt` and `go vet`
  clean. New tests: admin_audit and coach_showcase services, AdminAudit /
  AdminSignInAudit middleware, public coaches handler, coach showcase and
  audit repositories (including immutability).
- `npm run typecheck`, `npm run test` (164 tests) and `npm run build` pass.
- Browser E2E on an isolated stack (`ryze_test`, API :8081, Vite :5175):
  27/27 — coach creation, publish, featured listing, owner view not counted,
  client view counted once, "Created by", coach profile, audit page; no
  horizontal overflow or page errors at 390 and 1440 px. Fixtures removed
  afterwards.
- Feedback and Our Vision checked at 320, 390, 820 and 1440 px.

---

# Version 2.110.0

Title: Training app, language toggle, private-area polish, security and architecture fixes

Date: 2026-11-13

Summary:

Owner request of 2026-11-13: a Lyfta-style training app, a PT | EN block in
the navigation with a first-visit language prompt, the public site in
Portuguese, one consistent design for every private area, mobile checks, and a
complete platform audit with security and architecture fixes. No migration,
dependency, price, commission or access rule changed.

Main changes:

- Training app: plan screen with week tabs and day cards (exercises, sets,
  estimated minutes, done/next/in progress), full-screen workout player (clock
  with pause, SET / PREVIOUS / KG / REPS / ✓ rows prefilled from the
  prescription or the last session, rest bar, exercise info sheet, screen wake
  lock), end-of-workout report with personal records, save / keep going /
  abandon; history grouped by month; progress page with tiles and SVG charts
  (per week, volume per workout, per exercise best weight / estimated 1RM /
  volume). Insights are computed client-side from `GET /me/workout-sessions`
  (`services/training_insights.ts`, unit tests). Phones get a bottom tab bar.
- Language: compact PT | EN block in the public navigation (desktop and phone,
  current language highlighted); first-visit prompt in the detected language
  that confirms or switches it once; `useGettext` catalogues and `Copy<T>`
  objects. Translated: Home, Services (except the protected cards), training
  plan catalogue, program detail and checkout, Premium Level 1 and 2 (including
  the server questionnaires' labels and options), Our Vision, Contact,
  Feedback, Feed/404, Exercise Library chrome, footer labels and login /
  registration. Protected parts keep their English text unchanged; Portuguese
  is added beside it. The Premium Level 1 questionnaire now shows readable
  option labels instead of raw values.
- Private areas: every account and trainer surface uses the `body.ryze-app`
  navy tokens (near-black cards replaced, new `--app-inset`); the trainer area
  no longer renders the public navbar and footer and uses the member
  navigation (header links, phone tab bar); admin consoles get a phone top bar
  and admin tables become stacked cards on phones (`app-stack-table`); the My
  Programs empty state stacks its buttons on phones.
- Public navigation: signed-in visitors see "My account" instead of "Log in /
  Start free"; the phone menu and the narrow-desktop bar no longer overflow at
  320 px or between 821 and 960 px.
- Architecture: ADMIN_1, ADMIN_2 and trainer routes are lazy-loaded modules;
  the main bundle fell from 1,099 KB to 731 KB (303 → 216 KB gzip).
- Security: API security headers (`nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, `Cross-Origin-Resource-Policy: same-site`,
  `Cache-Control: no-store` on private paths); rate limits on purchase
  creation, payment start and capture (30/10 min), nutrition generation
  (20/hour), trainer client lookup (60/15 min) and application document
  upload (30/hour).
- Admin: `GET /admin/settings/commerce` (ADMIN_2) and `GET /admin/system/status`
  (ADMIN_1) back the Settings, System, Configuration and Dashboard pages with
  real values instead of static copy.

Verification:

- `go test -p 1 ./...` green on `ryze_test` (53 packages). In parallel mode
  `TestAdminCommerceListPurchasesAndSummary` can fail because packages share
  `ryze_test` (pre-existing isolation gap, documented in the audit).
- `npm run typecheck`, `npm run test` (23 files, 164 tests), `npm run build`.
- Browser checks on the isolated stack (API 8081 on `ryze_test`, Vite 5175):
  training flow (13 checks), language prompt and toggle (10 checks), public
  pages in PT at 320/360/1366/1440 px, navigation overlap at 821–1920 px,
  every client, trainer, ADMIN_1 and ADMIN_2 route at 390 and 1440 px with no
  horizontal overflow and no page errors.

Not done / open:

- Trainer pages' content and the admin consoles remain in English (the trainer
  navigation is translated).
- The Services cards (protected, English text inside their images) are not
  translated.
- Terms of Service and Privacy Policy links point nowhere and no legal pages
  exist; legal text needs the owner.

# Version 2.109.0

Title: Platform completion — account recovery, Google Sign-In, notifications, support chat, PWA and Portuguese

Date: 2026-11-12

Summary:

Completion round from the owner's "RYZE — Final Platform Completion" brief.
Adds self-service account recovery and transactional email (Resend over SMTP),
Google Sign-In, 24-hour notifications, a support chat answered by the
Management Administrator, a PWA, and an English/Portuguese system with the
account area translated. Owner decisions applied: default platform commission
30% (3000 bps), support answered by ADMIN_2, first language detected in the
browser (Portugal → Portuguese, elsewhere English; manual choice wins).

Main changes:

- Migrations `000037`–`000040` (additive; applied to `ryze` and `ryze_test`
  after a backup each, schema version 40): `users.email_verified_at`,
  `preferred_language`, `password_set`; `user_tokens`, `email_outbox`,
  `user_identities`, `notifications`, `support_conversations`,
  `support_messages`. Each `down` was validated inside a rolled-back
  transaction.
- Email: outbox with dedupe keys, retrying worker (`jobs.go`), mandatory
  STARTTLS, EN/PT templates; disabled delivery records emails as `skipped`.
- Account recovery: forgot/reset password (60-minute single-use fragment
  links, session revocation, change notice), email verification (48 hours),
  language preference; pages `/forgot-password`, `/reset-password`,
  `/verify-email`; the login page's "Forgot your password?" now opens them.
- Google Sign-In: OIDC code flow with PKCE, signed state cookie, ID-token
  verification against Google's keys, no automatic linking of existing
  accounts, linking/unlinking from Profile → Sign-in methods, Google-only
  accounts keep an unusable password until they set one. The protected login
  page only wires its existing Google button (English notices).
- Notifications: created on purchase completion, Premium Level 2 intake,
  trainer assignment and delivery, and support replies; visible 24 hours;
  dedupe tombstones kept 30 days; hourly purge; bell in the account and trainer
  headers (both headers now wrap to two rows on phones instead of overflowing).
- Support chat: `/support` for trainers and clients with an active programme,
  `/admin2/support` for ADMIN_2 (`support.read` / `support.manage`);
  eligibility on every call, idempotent messages, Server-Sent Events with
  resume after reconnection and client-side recovery when the browser gives
  up; staff identities never shown.
- PWA: manifest, icons generated from the official logo on brand navy,
  service worker caching only public assets and the page shell (never the
  API), update prompt.
- i18n: typed dictionaries without a dependency; translated auth recovery,
  profile, My Programs, programme access and structure, workout sessions and
  history, progress, purchases, Premium Level 1 nutrition labels, trainer
  application, notifications, support and PWA prompts. Dates and money follow
  the language.
- Unknown addresses show "Page not found" instead of an "under development"
  message (the Feed placeholder is unchanged).
- Commission default 3000 bps; `.env.example` documents `PUBLIC_APP_URL`,
  `MAIL_*` and `GOOGLE_*`.

Verification:

- `go test ./...` green against `ryze_test`; new unit/integration tests for
  email, recovery, Google (local fake provider with RSA keys), notifications,
  support (service + SSE handler).
- Frontend `tsc --noEmit`, vitest (148 tests) and production build green.
- Live checks: Google button and profile methods at 390/320 px; notification
  bell at 390 px and desktop; a two-browser support E2E on an isolated backend
  instance over `ryze_test` (18 checks, reply pushed in ~0.1 s, message
  recovered after an offline period, closed/ineligible/cross-account
  refusals); PWA on a production preview (worker control, cache contents,
  offline shell). The E2E fixtures were removed from `ryze_test` afterwards.

Independent review (read-only backend, database and frontend reviewers) and
fixes applied in this version:

- High: linking Google now requires a confirmed email; a password reset
  removes Google links from accounts that had a password; re-registration
  removes old links (prevents someone who registered or knew an account from
  keeping a way in after the owner recovers it).
- An administrator email change clears the verification and invalidates
  links sent to the old address; a password change retires reset links.
- Support: live streams capped (3 per account, 500 per instance), scoped
  wake-ups, session and eligibility re-checked every 15 s; retried "open" is
  idempotent; the open limit is counted under the insert lock; another
  author's client message id is refused.
- Outbox outcomes apply only to rows still in `sending`; Google identity locks
  use `FOR NO KEY UPDATE`; fixed advisory-lock key.
- Emails greet by name only after the address is confirmed; Premium Level 2
  reassignment notifies again.
- Frontend: a slow response for one conversation can no longer be shown while
  replies go to another (stale responses dropped, composer reset per
  conversation); screen-reader announcements only for new replies; streams
  stop retrying once the session or support access ends; the bell stops
  polling after sign-out; the service worker caches only hashed assets and
  icons; iOS status-bar style fixed; manifest no longer locks orientation;
  Support keeps trainers one tap from their dashboard; sign-in methods show an
  error with retry.
- The MariaDB copy tool refuses to empty PostgreSQL-only tables that hold
  data.
- Documented as accepted or pending: live reset links in pending outbox rows,
  residual enumeration (registration 409, small timing difference), Google
  sign-in restoring an admin-disabled account (owner decision, same as
  re-registration), one-way `down` for 000038/000040.

Not verified / pending:

- No live Resend send, no real Google OAuth client and no live payment provider
  in this environment.
- Owner input needed: Free "limited access duration" value, Free "PDF
  export", translating the public site and protected login copy, Premium
  service pages, trainer area and admin consoles; RYZE Feed.

Files: see `git status` for the full list (backend `email/`,
`services/{account_recovery,google_auth,notifications,support}/`, new
repositories/models/handlers, migrations `000037`–`000040`; frontend `i18n/`,
`components/{language_switcher,notification_bell,support_chat,pwa_update_prompt}/`,
`pages/{account_recovery_page,support_page,admin2/admin2_support_page}/`,
`public/`; documentation api, database, security, project, how_to_run,
product completeness, CLAUDE.md).

Reason: owner request (completion brief) and owner decisions of 2026-11-11.

# Version 2.108.0

Title: Backend database migrated from MariaDB to PostgreSQL

Date: 2026-11-11

Summary:

The backend now runs on PostgreSQL (owner request). The 36 migrations were
converted to PostgreSQL with the same numbers, the connection and migration
runner use pgx, the MySQL-specific SQL in the repositories was ported, and the
complete MariaDB data set was copied into PostgreSQL with verified row counts.
The backend was restarted on PostgreSQL and `go run . migrate up` applied the
schema (version 36). MariaDB was left intact for rollback.

Main changes:

- Migrations `000001`–`000036` rewritten for PostgreSQL: `TIMESTAMPTZ`,
  `JSONB`, `NUMERIC`, `VARCHAR(36)` ids, stored generated columns with
  `CASE`/`||`, `CHECK (>= 0)` for former unsigned columns, the
  `ryze_touch_updated_at()` trigger (28 tables) replacing
  `ON UPDATE CURRENT_TIMESTAMP`, `CITEXT` for `users.email` and
  `active_name = lower(name)` for exercises. Every `down` migration was
  verified by a full down/up round trip.
- `database/database.go` (pgx, UTC sessions, UTC `timestamptz` scanning,
  microsecond `NowFunc`), `database/migrate.go` (golang-migrate pgx driver),
  `config` (`DB_PORT` default 5432, new `DB_SSLMODE`).
- Repositories: `UTC_TIMESTAMP(6)`/`NOW(6)` → `CURRENT_TIMESTAMP`;
  `ON DUPLICATE KEY UPDATE` → `ON CONFLICT`; boolean `SUM`s → `COUNT(*)
  FILTER`; `LIKE` → `ILIKE` (case-insensitive searches as before); MySQL error
  1062 → SQLSTATE 23505 (`repositories/pg_errors.go`); statements whose
  duplicate error is mapped to a domain error run in a savepoint, because
  PostgreSQL aborts a transaction after a failed statement (the Premium Level
  2 delivery relied on continuing after a tolerated duplicate).
- Tests: statements expected to fail inside a test transaction run in a
  savepoint (`repositories/savepoint_helpers_test.go`).
- Scripts: `scripts/postgres_setup.sh` (role, `DB_NAME` and an isolated
  `<DB_NAME>_test`, ICU `und` collation), `scripts/migrate_mysql_to_postgres.sh`
  + module `scripts/mysql_to_postgres/` (one-off data copy), `backup_db.sh` /
  `restore_db.sh` on `pg_dump` / `psql`; `docker-compose.yml` uses
  `postgres:17`; `.env.example` (port 5432, `DB_SSLMODE`, no
  `DB_ROOT_PASSWORD`).

Reason:

The owner asked for a complete migration of the backend from MySQL/MariaDB to
PostgreSQL, including the data, a backend restart and the golang-migrate run.

Modified files:

- `main/src/backend/database/` (`database.go`, `migrate.go`, all 72 migration
  files), `main/src/backend/config/config.go`, `config/env.go`,
  `main/src/backend/go.mod`, `go.sum`
- `main/src/backend/repositories/` (`pg_errors.go` new; user, exercise,
  trainer, trainer_client, trainer_application, member_feedback, entitlement,
  purchase, reconciliation_event, nutrition_questionnaire,
  nutrition_assignment, premium_level_2, program, generic_program,
  free_program_activation, statistics, trainer_client_insight,
  commission_rule repositories; tests and `savepoint_helpers_test.go`)
- `scripts/postgres_setup.sh`, `scripts/migrate_mysql_to_postgres.sh`,
  `scripts/mysql_to_postgres/` (new); `scripts/backup_db.sh`,
  `scripts/restore_db.sh`; `docker-compose.yml`; `.env.example`
- `docs/DATABASE/database.MD`, `docs/INSTRUCTIONS/arquitecture.MD`,
  `docs/INSTRUCTIONS/api.MD`, `docs/INSTRUCTIONS/security.MD`,
  `docs/how_to_run.MD`, `docs/AI/database-migration-inventory.md`,
  `docs/AI/README.md`, `docs/REGISTRY/update_logs.MD`, `CLAUDE.md`,
  `.claude/rules/database.md`, `.claude/rules/backend.md`,
  `.claude/skills/ryze-postgresql-migration/SKILL.md`,
  `.claude/skills/ryze-verification/SKILL.md`,
  `.claude/agents/database-migration-reviewer.md`

Verification:

- `go build ./...`, `go vet ./...`; `DB_NAME=ryze_test go test ./... -p 1`
  against PostgreSQL 18.6: 48 packages pass (none fail).
- Migrations: up to 36, full down and up again on `ryze_test`; up on `ryze`.
- Data: dry run, then copies into `ryze_test` and `ryze`: 3791 rows in 40
  tables, all counts equal to MariaDB; value aggregates (users, exercises with
  JSONB and microsecond timestamps, programs, trainers) equal.
- API on PostgreSQL: public endpoints 200 with `Z` timestamps; on a second
  instance against `ryze_test`: register, case-different duplicate (409),
  login with a lower-cased email, `/me`, feedback eligibility, account
  deletion.
- Backup/restore round trip on `ryze_test` (counts, version 36, 28 triggers,
  `citext`).
- MariaDB safety dumps: `backups/ryze-mariadb-final-*.sql.gz` (local,
  unencrypted, gitignored).

# Version 2.107.0

Title: Home story steps restored, Earth to the corner, Contact in the login look

Date: 2026-11-11

Summary:

The owner kept the order hero → programs → chapters but rejected the opening
that left the Earth alone on screen and asked for the earlier story scheme. The
desktop Home is again one pinned 3D stage with the five story steps on its right
edge (current step follows the chapter; each step scrolls to its chapter). As
the hero scrolls away the Earth travels to the upper right corner and waits
there while the programs pass; the steps step aside over the programs; the phone
flies in as the programs leave, so there is no empty stretch. The stage keeps
its height in the flow and is released when the track ends, so the next section
never runs over a pinned "Get started" (the bug fixed in 2.106.0 stays fixed).
The Contact page now uses the login page's split card: the light-blue wave half
with the title and channels, the navy half with the form.

Main changes:

- `components/home_story/story_immersive.tsx` + CSS — one stage again, story
  steps (`nav` "Story steps"), hero copy fade, programs hide the steps, track
  starting at 55% of the viewport, instant jump for keyboard focus.
- `components/home_story/story_scene.ts` — keyframes: Earth to the corner
  during the hero, tucked during the programs, phone fly-in 0.14 → 0.25;
  diagram windows retimed.
- `components/contact_section/` — split card with `Waves` and light channel
  cards; the form drops its own card frame inside it.

Reason:

Owner feedback on the Home opening and the request to give the Contact page
more life.

Modified files:

- `main/src/frontend/src/components/home_story/story_immersive.tsx`,
  `story_immersive.module.css`, `story_scene.ts`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`,
  `contact_section.module.css`
- `docs/INSTRUCTIONS/design.MD`, `docs/PAGES/design.MD`,
  `docs/AI/frontend-revamp.md`, `docs/REGISTRY/update_logs.MD`

Verification:

- `npm run typecheck`, `npm run test` (148/148), `npm run build` and
  `npm run test:e2e` (8/8) passed.
- Browser checks (headless Chromium, SwiftShader WebGL): 29 story checks —
  five steps on the hero with step 1 current and clickable, steps hidden over
  the programs, program CTAs clickable, step 2/3 reaching Train/Fuel, Train
  copy on screen as soon as the programs have left, keyboard focus revealing a
  faded chapter, the stage released before the next section, rendering paused
  off screen, disposal on navigation, reduced-motion/no-WebGL/mobile fallbacks
  without Three.js; interaction checks; no horizontal overflow on 10 routes × 8
  widths; screenshots of Home (1440×900) and Contact (1440×900, 390×844).

# Version 2.106.0

Title: Home story fixes, linked diagrams and the login look on Services

Date: 2026-11-11

Summary:

The desktop Home story now uses two pinned stages that share one WebGL canvas:
the opening (the hero copy fades while the Earth turns to the centre, then the
stage scrolls away), the programs in normal flow with nothing behind them, and
the device chapters, whose stage scrolls in with the phone already in place
and scrolls away after "Get started". This fixes four reported bugs: the lost
hero scroll animation, the Earth behind the program cards, the empty stretch
after them, and the last chapter staying pinned under the next section. Each
Train/Fuel card is now joined to the screen element it shows (outline on the
screen, rounded leader, travelling light pulse); the "Synced" badge sits over
both devices. The program cards lose their Premium Level tags. The login
page's split panel becomes a public primitive and gives Services a route guide
with a "Start free today" panel; the free training-plan marketplace and Our
Vision get the public colours and more light without layout changes.

Main changes:

- `components/home_story/story_immersive.tsx` + CSS (two stages, canvas moves
  between them, opening fade, release after the track), `story_scene.ts`
  (`setProgress(progress, immediate)`, new opening/track keyframes, leader-line
  diagram with outlines and pulses, badge light, all materials precompiled),
  `story_content.ts` (callouts ordered like their screen elements),
  `story_stacked.tsx` (keeps the set/rest and protein/meal cards).
- `components/programs_overview_section/` — only free programs carry a tag;
  transparent section with a soft blue light.
- `components/public_ui/public_ui.module.css` — `split`, `splitPane`,
  `splitLight`.
- `pages/services_page/` — route guide split panel, gradient "RYZE", benefits
  rewritten to existing features (the former AI and analytics claims removed).
- `pages/generic_plan_marketplace_page/`, `components/program_card/`,
  `program_row/`, `filter_sidebar/`, `search_bar/`, `carousel_arrow/` (all used
  only by the marketplace) — public colours, brand-gradient covers, "Free"
  badge, Archivo titles.
- `pages/our_vision_page/` — light-blue RYZE tile in the hero wall, gradient
  step numbers, a light ring on "Today", a brighter RYZE comparison column.

Reason:

Owner feedback on the Home story bugs and diagrams, the wrong impression given
by the level tags, and the request to apply the login page's colours where
RYZE wants attention (especially Services) and to give the marketplace and Our
Vision more life without changing their layout.

Modified files:

- `main/src/frontend/src/components/home_story/` (`story_immersive.tsx`,
  `story_immersive.module.css`, `story_scene.ts`, `story_content.ts`,
  `story_stacked.tsx`)
- `main/src/frontend/src/components/programs_overview_section/`
- `main/src/frontend/src/components/public_ui/public_ui.module.css`
- `main/src/frontend/src/pages/services_page/`
- `main/src/frontend/src/pages/generic_plan_marketplace_page/`,
  `components/program_card/`, `components/program_row/program_row.module.css`,
  `components/filter_sidebar/filter_sidebar.module.css`,
  `components/search_bar/search_bar.module.css`,
  `components/carousel_arrow/carousel_arrow.module.css`
- `main/src/frontend/src/pages/our_vision_page/`
- `docs/INSTRUCTIONS/design.MD`, `docs/PAGES/design.MD`,
  `docs/AI/frontend-revamp.md`, `docs/REGISTRY/update_logs.MD`, `CLAUDE.md`,
  `AGENTS.md`, `.claude/rules/frontend.md`,
  `.claude/skills/ryze-frontend-revamp/SKILL.md` (the split panel as the look
  for attention moments)

Verification:

- `npm run typecheck`, `npm run test` (148/148), `npm run build` and
  `npm run test:e2e` (8/8) passed.
- Browser checks (headless Chromium, SwiftShader WebGL): 29 story checks —
  opening fade while pinned, no stage behind the program cards, canvas moving
  between stages, Train visible on arrival, the chapter stage released before
  the next section, rendering paused off screen and between the stages,
  disposal on navigation, reduced-motion/no-WebGL/mobile fallbacks without
  Three.js; interaction checks; no horizontal overflow on 10 routes × 8 widths;
  screenshots at 1024×768, 1280×720, 1440×900, 1920×1080 and 390×844.

# Version 2.105.0

Title: Member feedback with administrator review

Date: 2026-11-11

Summary:

Members can rate RYZE (1–5 stars) and leave a short message from the public
Feedback page; submissions are published under "Other feedback from our
athletes" only after the Management Administrator approves them in the admin
console. Eligibility is checked by the server: an active account that holds or
held any plan, free ones included (an entitlement or a Free activation; expired
access counts, revoked access does not). One pending submission per member, the
public list shows only a "First L." name snapshot, and submissions are rate
limited.

Main changes:

- Migration `000036_create_member_feedback` (table `member_feedback`, CHECK
  constraints on rating and status, a generated `pending_user_id` with a unique
  index for the one-pending rule).
- Backend: `models/member_feedback.go`, `repositories/member_feedback_repository.go`,
  `services/member_feedback/` (validation, eligibility, public name, review),
  `api/auth/member_feedback_handler.go`; routes `GET /api/v1/feedback`,
  `GET /api/v1/me/feedback/eligibility`, `POST /api/v1/me/feedback` (new
  `feedback` rate-limit scope: 10/hour/IP), `GET /api/v1/admin/feedback`,
  `POST /api/v1/admin/feedback/:id/approve|reject`; new admin permissions
  `feedback.read` and `feedback.manage` (Management Administrator only).
- Frontend: `services/member_feedback_api.ts`, the Feedback page's
  `community_feedback` section, and the real review queue on
  `/admin2/feedback` (`services/admin2_api.ts`).

Reason:

The owner asked for a way for athletes to leave feedback and for a public
comments area, restricted to accounts with a plan and moderated before
publication.

Modified files:

- `main/src/backend/database/migrations/000036_create_member_feedback.{up,down}.sql` (new)
- `main/src/backend/models/member_feedback.go`, `repositories/member_feedback_repository.go`,
  `services/member_feedback/` , `api/auth/member_feedback_handler.go` (+ tests) (new)
- `main/src/backend/routes/routes.go`, `routes/ratelimit.go`,
  `middleware/adminroles/adminroles.go`
- `main/src/frontend/src/services/member_feedback_api.ts` (new),
  `services/admin2_api.ts`, `pages/feedback_page/` , `pages/admin2/admin2_feedback_page/`
- `docs/INSTRUCTIONS/api.MD`, `docs/INSTRUCTIONS/security.MD`,
  `docs/DATABASE/database.MD`, `docs/PAGES/design.MD`, `docs/REGISTRY/update_logs.MD`
- Agent instructions aligned with the owner's 2026-11-10/11 decisions (with the
  owner's permission): `CLAUDE.md`, `AGENTS.md`, `.claude/rules/frontend.md`,
  `.claude/skills/ryze-frontend-revamp/SKILL.md`,
  `.claude/agents/frontend-reviewer.md` — the narrowed protected parts, the one
  public design principle, the approved desktop 3D story and migration 000036.

Verification:

- `go build ./...` and `go vet` passed; `go test ./services/member_feedback/`
  (9 tests), `go test ./api/auth/ -run Feedback` (5 tests) and
  `go test ./middleware/...` passed. Database-backed suites were not run: they
  connect to the configured (owner's) database.
- The migration has not been applied to any database and was not executed
  against MariaDB in this session; apply it with `go run . migrate up` from
  `main/src/backend/`.
- Browser checks with a mocked API: 13 public-flow checks (signed out, no plan,
  pending, form and keyboard rating, submission, empty list, pagination, no
  overflow at 390px) and 5 admin review checks.

# Version 2.104.0

Title: Public website redesign — one consistent RYZE world and desktop 3D story

Date: 2026-11-10

Summary:

Redesigned the public website around one principle: it is built from the same
materials as the app. Every page is one continuous logo-navy surface (navy
`#02041F`; the logo's royal `#072A77`, ring blue `#0481DC`, Z cyan `#00CBFF`
and chrome `#E4E9F1` were sampled from the official artwork) with a repeating
rhythm of soft blue light; sections are transparent, so they no longer jump
between unrelated colours; cards are the app's blue glass surfaces; cyan is the
light; and the only bright element is a light-blue wave panel. Archivo (OFL,
self-hosted) gives expanded display type that echoes the wordmark; headlines
keep the original brand-coloured continuations.

Pages: Home, Our Vision (`/our-vision`; `/about-us` redirects), Contact,
Feedback, the public under-development placeholder, and — after the owner
narrowed the protected areas on 2026-11-10 — Services (cards unchanged),
login/registration (complete rework), the Home "Find Your Next Program" section
(cards unchanged), the Exercise Library and detail (colours only) and the
footer (layout, columns and slogan image unchanged).

Home tells RYZE in five chapters. On desktop they are a scroll-driven Three.js
story: the Earth with real continents and RYZE routes leaving Lisbon (the brand
travelling the world) under the headline "Train anywhere. Progress
everywhere.", a modelled phone and laptop showing illustrative app screens
designed after the owner's phone reference, then the Earth again for the call
to action. Phones, tablets, reduced motion and no-WebGL get an SVG
globe with the same routes and CSS device frames, and never download Three.js.

Two earlier passes (a dark, minimal one, then a multi-colour one with paper,
cyan and royal sections and a dotted planet) were rejected by the owner and
replaced before release.

Main changes:

- `components/public_ui/`: the layer (`page` tokens and continuous light;
  `toneNavy`, `toneLight`; `surface`, `lightPanel`, `iconTile`; type scale;
  `PublicAction`; `BrandStop`; `Waves`) and the "RYZE Archivo" `@font-face`.
- `components/home_story/` (new): chapter content, shared copy parts, the
  immersive desktop stage, the dynamically imported scene (`story_scene.ts`:
  Earth shader with land mask, merged route tubes with travelling pulses, city
  points, Lisbon badge; extruded phone and laptop with physical materials,
  instanced keys, edge glow) and the stacked layout;
  `hooks/use_immersive_story.ts` and `hooks/use_media_query.ts` (new).
- `components/brand_globe/` (new): SVG globe and its generated data;
  `components/showcase/` (new): phone and laptop frames.
- `components/category_overview/` and `components/purchase_journey/` (new);
  redesigned `contact_preview/`, `contact_section/`, `contact_form/` (same
  fields, messages, honeypot, limits and API; persistent status live region;
  dark fields with 3:1 borders).
- `pages/login_page/`: one card with both forms and a light-blue wave panel that
  slides across when switching mode (the form swaps sides); phones get a wave
  header. Field names, validation messages, API calls and redirects are
  unchanged; the one-second loading overlay between modes was removed; the
  inactive form is inert and hidden from assistive technology and focus moves
  to the new heading. The login background image is no longer used here (the
  admin login still uses it).
- `pages/services_page/`: background image removed; new intro, states and
  benefits panel around the unchanged `ServiceCard`s. The heading now wraps on
  phones; previously it forced the cards to 445–528px inside a 360–430px screen,
  where they were cut off.
- `components/programs_overview_section/`: section surface and heading only.
- `pages/exercise_library_page/`, `pages/exercise_detail_page/`: page-scoped
  colour token overrides and the library title typeface; layout unchanged.
- `components/footer/`: navy surface, cyan top line, cyan column headings and
  Archivo; layout and content unchanged.
- `pages/feedback_page/`: hero restored to the original composition (slogan
  level with the headline), stories as glass cards on navy, wave-panel CTA.
- Owner follow-up (same day): the story backdrop now dissolves into the page,
  removing the edge before "Find Your Next Program"; display and heading line
  heights opened slightly; the laptop screen was redesigned at a larger scale
  (2560×1600) with maximum texture filtering and a square-on view, and the
  keyboard gained backlit legends, Touch ID and arrow keys, a chamfered glass
  trackpad, an opening scoop and a display notch; the Our Vision hero globe was
  replaced by a platform visual (the RYZE core wired to its modules and the
  roadmap's next ones); the Feedback slogan is centred on the headline block.
- Owner follow-up (second pass): the Services cards and the Home goal cards
  are back as a showcase whenever no offer is published (the original static
  content from before the catalogue became API-driven; live data still wins);
  Train and Fuel gained cards lifted out of the phone screen (3D on desktop,
  overlapping the phone on mobile); Evolve now shows laptop and phone side by
  side joined by a sync stream; the hero Earth moved slightly right; the
  Feedback hero is centred with the members' own results as links to their
  stories; the Our Vision hero shows an adaptive-versus-static training-load
  illustration.
- Owner follow-up (third pass, 2026-11-11): Home now runs hero → Find your
  next program → device chapters, with one 3D stage pinned behind all three;
  the programs section was redesigned as the main call to action (tall photo
  cards, level tags, CTA on every card, swipeable on phones); the chapter
  navigation was removed; Train and Fuel cards hang from a right-angled tree;
  Evolve lost its connecting line and uses large-type screens; device screens
  are sampled one mip level sharper; Services' Premium Level 2 card carries a
  purple "Personalized" tag; the Feedback result chips were removed; Our
  Vision opens with "Thousands of fitness apps. All the same. RYZE isn't."
  and a new "Why RYZE" comparison section.
- `pages/our_vision_page/`, `pages/contact_page/`,
  `pages/under_development_page/`, `pages/home_page/home_page.tsx`: moved to
  the single navy system.
- Navbar: glass tinted to the brand navy; earlier invisible accessibility fixes
  kept (44px menu hit area, hidden links out of the tab order, Escape closes).
- Removed `components/hero/`, `components/platform_section/platform_section.*`,
  `components/section_title/` and `pages/home_page/home_page.module.css`.
- Assets: `main/src/resources/img/showcase/` (three illustrative screens,
  208 KB), `main/src/resources/img/globe/earth_land_mask.webp` (47 KB),
  `main/src/resources/fonts/archivo/` (90 KB + OFL). `scripts/generate_globe_assets.py`
  regenerates the globe assets from Natural Earth (public domain).
- Bundle: main JS 884.0 → 915.6 kB (gzip 240.7 → 253.2 kB); CSS 264.8 →
  260.3 kB. Three.js is a shared lazy chunk (590 kB) used by the Exercise
  Library viewer (8 kB chunk; previously one 544 kB chunk) and the Home scene
  (25 kB chunk).

Reason:

The owner asked for one coherent site ("is this the same site?" between
sections and against the footer), a clean design that speaks for itself, a
realistic phone and laptop matching their reference, a globe that conveys RYZE
travelling the world, a fixed Feedback page and a complete login/registration
rework, while keeping the Services and program cards, the Library layout and
the footer design.

Modified files:

- `main/src/frontend/src/components/public_ui/`, `home_story/`,
  `brand_globe/`, `showcase/`, `category_overview/`, `purchase_journey/` (new
  or rewritten), `contact_preview/`, `contact_section/`, `contact_form/`,
  `navbar/navbar.module.css`, `navbar/navbar.tsx`, `footer/footer.module.css`,
  `programs_overview_section/programs_overview_section.module.css`
- `main/src/frontend/src/hooks/use_immersive_story.ts`,
  `hooks/use_media_query.ts` (new)
- `main/src/frontend/src/pages/home_page/`, `our_vision_page/`,
  `feedback_page/`, `contact_page/`, `under_development_page/`, `login_page/`,
  `services_page/`, `exercise_library_page/exercise_library_page.module.css`,
  `exercise_detail_page/exercise_detail_page.module.css`
- Removed `components/hero/`, `components/platform_section/platform_section.*`,
  `components/section_title/`, `pages/home_page/home_page.module.css`
- `main/src/resources/img/showcase/`, `main/src/resources/img/globe/`,
  `main/src/resources/fonts/archivo/` (new); `scripts/generate_globe_assets.py`
  (new)
- `docs/PAGES/design.MD`, `docs/INSTRUCTIONS/design.MD`,
  `docs/AI/frontend-revamp.md`, `docs/REGISTRY/update_logs.MD`

Verification:

- `npm run typecheck` passed; `npm run test` 21 files / 148 tests passed;
  `npm run build` passed (the >500 kB chunk warning predates this change);
  `npm run test:e2e` 8/8 passed.
- Browser QA with a mocked API (headless Chromium, SwiftShader WebGL): no
  horizontal overflow and no page errors on 10 public routes at 320, 360, 390,
  430, 768, 1024, 1280 and 1440px; 35 interaction checks (every CTA lands on its
  documented route, anchors, mobile menu, focus, contact validation, touch
  targets, stories carousel, catalogue error/empty states); 29 authentication
  checks at 1440 and 390px (hidden form unreachable by keyboard, focus after a
  switch, every validation and API error message, unchanged request bodies,
  redirect to `/profile`, reduced motion).
- 3D (22 checks): renders on desktop, draws stop off screen and after leaving
  Home (scene disposed, canvas removed), chapter navigation, chapters stay in
  the accessibility tree, keyboard focus reveals its chapter; reduced motion,
  no WebGL and 360–768px never request Three.js.
- Protected parts compared with a pristine `HEAD` build (computed styles of
  every descendant plus geometry): the Home program cards are identical at 360,
  390, 430, 768, 1024 and 1440px; the Services cards are identical at 768, 1024
  and 1440px and differ at 360–430px only in width/height-derived values (they
  now fit the screen). Exercise detail layout is identical; the library list and
  footer keep their structure, moved only by the new typeface's line heights
  (2–4px and 6–16px). The loading screen appears and dismisses on every route in
  both builds.

# Version 2.103.0

Title: Claude Code engineering environment preparation

Date: 2026-11-09

Summary:

Prepared a project-local Claude Code environment for the planned frontend
revamp, backend hardening and future PostgreSQL migration. Added concise root
instructions, scoped rules, read-only reviewer agents, RYZE-specific skills and
factual AI workflow/inventory documents. Installed and pinned reviewed project
skills for frontend design, React, responsive UI, Three.js and webapp testing;
enabled the official Anthropic frontend-design plugin at project scope.

Reason:

The repository needs a maintainable, token-efficient AI engineering workflow
without changing application functionality, dependencies, database schema or
production configuration. The current MariaDB/MySQL architecture and frontend
protected areas are recorded for future work.

Modified files:

- `CLAUDE.md`, `AGENTS.md`, `.gitignore`
- `.claude/settings.json`, `.claude/rules/`, `.claude/skills/`, `.claude/agents/`
- `skills-lock.json`
- `docs/AI/`

Verification:

- Claude Code `2.1.319`, project plugin list and plugin details inspected.
- Skills CLI `1.7.6`; project-local skill list, hashes and frontmatter inspected.
- JSON/frontmatter, protected-area inventory and MariaDB/MySQL inventory checked.
- No application source, package manifest, migration, database or production
  configuration was modified.

# Version 2.103.0

Title: Premium Level 2 trainer-delivered service

Date: 2026-11-09

Summary:

v2.103.0 implements Premium Level 2 end to end. A client buys a platform-owned
service offer, completes a versioned intake, ADMIN_2 allocates an active trainer,
and that trainer authors the training programme and nutrition before delivering
the pack. Delivery creates the existing training entitlement and releases the
nutrition plan to the client. The feature reuses the existing program
hierarchy, checkout, Test Mode, entitlement and program-access systems; it adds
no second checkout, no new payment provider and no separate media pipeline.

Main changes:

- Added migration `000035_add_premium_level_2`: widens `programs.product_type`
  and the product-metadata constraint, and creates the request, versioned intake,
  per-request nutrition plan/meals/foods and request-event tables plus their
  views and indexes. Verified up/down/up; schema version `35`.
- Added `models/premium_level_2.go`, `repositories/premium_level_2_repository.go`
  and `services/premium_level_2` for the request lifecycle, server-catalogued
  intake validation, trainer allocation, readiness and idempotent delivery.
- Added `api/auth/premium_level_2_handler.go` and all client, ADMIN_2 and trainer
  routes, including the shared staff intake catalogue.
- Added the purchase fulfilment multiplexer hook so a completed purchase creates
  exactly one request in `awaiting_intake` after the entitlement check.
- Added the platform-owned `premium_level_2` offer family to the ADMIN_2 program
  service and plan builder: empty offer validation, premium-only pricing, no
  structure or audience metadata, and no publish content lock regression.
- Added the public `premium_level_2` catalogue scope and the API-backed Services,
  Home and My Programs integrations.
- Added the customer offer/purchase page and the request/intake/status page with
  delivered nutrition and training access.
- Added the ADMIN_2 Premium L2 request queue with search, status, pagination,
  intake review, trainer allocation and delivery history.
- Added the trainer Premium L2 workspace for allocated requests, draft programme
  attachment, shared nutrition authoring and confirmed delivery.
- Added frontend L2 contracts, nutrition mapping, answer formatting, shared
  intake renderer and L2 unit/E2E coverage. Updated stale product-family tests to
  recognise `premium_level_2` and to reject genuinely unknown families.
- Updated `project.MD`, `arquitecture.MD`, `security.MD`, `api.MD`,
  `database.MD`, `design.MD`, `procedure.MD`, `conventions.MD`, `how_to_run.MD`,
  `product_roadmap.MD`, `product_completeness.MD` and this registry.

Reason:

Premium Level 2 was the next product tier and was explicitly excluded while
unimplemented. The implementation needed to preserve the platform's existing
purchase, authorization, entitlement and soft-delete guarantees while keeping
sensitive intake data private to the owner, allocated trainer and authorised
administrators. The new family is now documented as implemented instead of being
a placeholder.

Tests:

- `go -C main/src/backend build ./...` — passed.
- `go -C main/src/backend test -count=1 ./...` against isolated
  `ryze_test_premium_l2` — all packages passed.
- `npm run typecheck` — passed.
- `npm test` — 21 files / 148 tests passed.
- `npm run build` — passed.
- `npm run test:e2e` — 8 tests passed, including the L2 catalogue cards and
  highlighted service.
- Migration up/down/up and `migrate version` verified version `35
  (dirty=false)`.

Known verification gaps:

- A live Premium Level 2 purchase, real payment-provider delivery and a complete
  multi-identity browser walkthrough remain unverified; the E2E catalogue paths
  use mocked APIs and the lifecycle is covered by backend tests.
- Richer browser coverage for the request/intake, admin allocation and trainer
  authoring pages is not yet implemented.

---

# Version 2.102.0

Title: Secure trainer client detail with training insight and programme assignments

Date: 2026-11-07

Summary:

v2.102.0 completes the trainer client-management detail surface. A trainer can
open one of their active clients and see the client's real programme access,
statistics, live session, finished session history and exercise performance,
then assign or remove their own programme without exposing another trainer's
mutable assignment. Every insight read revalidates the active trainer-client
relationship and remains strictly read-only.

The release adds no database migration and no parallel statistics, access or
session model. It reuses `trainer_clients`, `statistics`, `workout_sessions`,
program entitlements and Free activations, and presents legacy completion-only
history honestly as a count and latest date.

Main changes:

- New repository `repositories/trainer_client_insight_repository.go` reads
  active assignments, live entitlements, live Free activations, finished and
  active sessions, set counters, legacy summaries and exercise snapshots. All
  queries receive the verified client user id explicitly and are read-only.
- New service `services/trainer_client_insights` verifies the active
  trainer-client relationship before every operation, reuses the statistics and
  workout-session services, maps access statuses and builds the overview,
  paginated session list, full session detail and exercise history DTOs. Recent
  sessions are limited to 10; history pages default to 20 and cap at 100.
- New handler `api/auth/trainer_client_insights_handler.go` and five routes under
  `/api/v1/trainer/clients/:userID`:
  `overview`, `statistics`, `workout-sessions`,
  `workout-sessions/:sessionID` and `exercises/:exerciseID/history`. Every route
  requires `trainer.clients`; malformed input answers `VALIDATION_ERROR`,
  unrelated clients `CLIENT_RELATION_NOT_FOUND`, foreign sessions
  `WORKOUT_SESSION_NOT_FOUND`, and internal details never reach the response.
- Access overview keeps trainer assignments, purchase entitlements and Free
  activations separate. Assignment and Free access become `unavailable` when the
  programme is missing, deleted or unpublished; live entitlements stay `active`
  after unpublishing. Another trainer's assignment is visible as read-only
  context but its assignment id is omitted.
- Frontend `services/trainer_api.ts` gains the insight types and fetch calls.
  The new `/trainer/clients/:userID` page reuses the shared trainer styling,
  `Button`, `LoadingScreen` and `workout_sessions_api` session shape. It presents
  real stats, a live-workout card, access sources, assign/remove controls,
  expandable session snapshots, paginated history, exercise history and the
  legacy note with honest empty/error states. The client list now links each row
  through "View client".
- No nutrition, questionnaire answers, medical data, payments, trainer-authored
  performance or client-generated training records are exposed. No new
  permission, role, upload, notification or messaging surface is introduced.
- Docs updated: API, security, architecture, database, design, project vision,
  local endpoint list, roadmap, completeness audit and this registry. No
  migration file was added; schema version remains `34`.

Reason:

Trainers could manage client relationships and assignments but could not see the
client's actual training record, so the client detail area stopped at identity
metadata. The product needs a secure, useful training overview without turning
the trainer area into a second statistics engine or granting access to sensitive
non-training data. This release composes the existing modules behind one
relationship-scoped read surface, preserves historical snapshots and legacy
history semantics, and keeps assignments scoped to their owning trainer.

Validation:

- Isolated MariaDB on `127.0.0.1:33107`, database `ryze_test_workout`;
  `migrate version` reports `34 (dirty=false)`.
- `DB_HOST=127.0.0.1 DB_PORT=33107 DB_NAME=ryze_test_workout DB_USER=root DB_PASSWORD= go test -count=1 ./...` passed every package uncached (`api/auth` 34.953s, `repositories` 2.825s, `routes` 5.248s).
- `go vet ./...`, `go build ./...` and a clean `gofmt -l` check passed.
- `npm run typecheck` passed.
- `npm run test` passed 19 files / 137 tests.
- `npm run build` passed.
- `npm run test:e2e` passed 8/8 Playwright tests in 19.5s.
- Focused coverage: `trainer_client_insights` service tests, isolated
  `TrainerClientInsightRepository` integration tests, insight handler tests,
  25 trainer API tests and 6 client-detail page tests.

---

# Version 2.101.0

Title: Objective-based Premium Level 1 catalogue and curated nutrition

Date: 2026-11-05

Summary:

v2.101.0 completes the Premium Level 1 catalogue remodel. Premium Level 1
packages are now grouped by a closed objective vocabulary and can deliver one of
two server-owned nutrition models: a predefined curated plan authored with the
package, or the legacy questionnaire-generated plan derived from a client intake.
The Management Administrator can author both training and curated nutrition in
one workspace, completed purchases freeze package content, and purchasers keep
access after a catalogue entry is unpublished.

The legacy intake flow remains compatible: existing rows are backfilled as
`questionnaire_generated`, old questionnaires/plans/entitlements are untouched,
and packages without a known objective continue to render through the legacy
client path.

Main changes:

- Migration `000034_add_curated_premium_level_1_nutrition` adds
  `programs.objective`, `programs.nutrition_mode` and
  `programs.content_version`, backfills existing Premium Level 1 rows as legacy,
  adds metadata/content CHECK constraints and indexes, and creates the
  package-owned `curated_nutrition_plans`,
  `curated_nutrition_plan_meals` and `curated_nutrition_plan_meal_items` tables
  plus their safe read views.
- `models/program.go` adds the objective, nutrition-mode and content-version
  fields and constants. `models/curated_nutrition.go` and
  `services/curated_nutrition` add the curated aggregate, normalization and
  validation.
- The generic/platform program service, repository and handler now share one
  family-bound administration surface for Generic and Premium Level 1 packages.
  It validates objective/mode, requires a complete curated plan for a curated
  package, revalidates at publish, increments `content_version` on successful
  edits, and refuses metadata changes after creation
  (`409 PROGRAM_METADATA_LOCKED`) and content edits/deletion after a completed
  purchase (`409 PROGRAM_CONTENT_LOCKED`).
- Checkout prerequisites now apply only to questionnaire-generated Premium
  Level 1 packages. Curated packages skip the intake entirely.
- `services/questionnaires` adds `requiresIntake`: a curated/generic package
  answers `required: false`, `submitted: false`, `locked: false`, and a submit
  attempt returns `404 PROGRAM_NOT_FOUND` without writing.
- Nutrition delivery gains a curated branch that reads the package plan after the
  entitlement check and projects it onto the existing client plan contract with
  `engine_version = 0`, `questionnaire_version = 0`, fingerprint
  `curated:<program-id>:<plan-version>`, empty exclusions and macros derived from
  the authored foods. Legacy generation, supersession and retry behavior are
  unchanged.
- Program access now treats an entitlement as status-agnostic: a purchased
  program stays readable and executable after unpublishing. Assignment and Free
  grants still require a published program.
- The public catalogue exposes `objective`; `nutrition_mode` and curated plan
  content remain private.
- Frontend: ADMIN_2 family tabs, objective/mode authoring, curated nutrition
  editor, read-only curated detail section and content-lock actions. The Premium
  page groups packages by objective with `?objective=` filtering, keeps legacy
  objective-null packages under "Other packages", sends curated packages
  directly to checkout, and maps curated/legacy nutrition copy and states.
- E2E fixtures and the Premium catalogue test assert objective grouping and
  package selection.
- Docs updated: API, architecture, database, security, design, project vision,
  local walkthrough, roadmap, completeness audit, procedures, agent rules and
  this registry.

Reason:

Premium Level 1 only supported a questionnaire-generated plan, while the product
needs objective-based packages that ship a predefined nutrition plan. The previous
catalogue also gave no product-level way to group packages by goal, and
unpublishing a sold program revoked paid access. This release adds the curated
model without a parallel product family or payment flow, preserves the legacy
intake path, and makes purchased access independent of catalogue publication.

Validation:

- Isolated MariaDB on `127.0.0.1:33107`, database `ryze_test_workout`;
  `migrate down 1` → `migrate up` succeeded and `migrate version` reported
  `34 (dirty=false)`.
- `DB_HOST=127.0.0.1 DB_PORT=33107 DB_NAME=ryze_test_workout DB_USER=root DB_PASSWORD= go test -count=1 ./...` passed every package uncached (`api/auth` 38.186s, `repositories` 3.730s, `routes` 5.499s).
- `go vet ./...`, `go build ./...` and a clean `gofmt -l .` check passed.
- `npm run typecheck` passed.
- `npm run test` passed 18 files / 123 tests.
- `npm run build` passed.
- `npm run test:e2e` passed 8/8 Playwright tests in 21.1s, including the
  objective-grouped Premium catalogue flow.

---

# Version 2.100.0

Title: Secure programme access, Free activation and interactive workout sessions

Date: 2026-11-02

Summary:

v2.100.0 completes the programme access and workout execution milestone. A
client can now enter a published programme through an active entitlement, an
active trainer assignment or an activated Free programme, and execute its
prescribed workouts as server-owned sessions. The public catalogue is now
metadata-only: programme structure is never exposed without an access grant.

Workout sessions snapshot the prescription, enforce one live session per user at
the database level, log actual performance server-side, provide absolute
server-derived rest timers, and feed real history/progress statistics. Free
programmes gain a dedicated activation grant instead of abusing checkout,
entitlements gain an optional expiry, and the legacy assignment read and workout
history remain available.

Main changes:

- Migration `000033_secure_program_access_and_workout_sessions` adds
  `free_program_activations`, `entitlements.expires_at`, `workout_sessions`,
  `workout_session_exercises` and `workout_session_sets`, including generated
  one-active constraints, foreign keys, indexes and CHECK constraints.
- Free activation repository/service/handlers and client Free programme list;
  one active activation per user, idempotent same-programme activation and
  transactional replacement.
- Unified `program_access` service resolving entitlement, assignment or Free
  activation, loading structure through the shared `program_content` mapper.
- `program_content` now carries product metadata (`product_type`, `level`,
  `duration_weeks`, `frequency_per_week`, `training_type`) without exposing
  trainer or parent identifiers.
- Interactive session repository/service/handlers: start/idempotent resume,
  active session, owned read, set updates, pause, resume, complete, abandon and
  history. Prescription snapshots are immutable; server timestamps drive elapsed
  time and `rest_ends_at`.
- Statistics now resolve the current programme from assignment, entitlement or
  Free activation, scope unique completions to active workouts, clamp completion
  percentage and add all-time session/set counters.
- Public programme detail is metadata-only; `trainer_id` and all structure were
  removed from public responses.
- Frontend: access-only `ProgramStructure`, Free activation on the generic
  detail, Premium access-step handoff, merged My Programs, active-session banner,
  workout player, workout history, progress page, routes, AccountNav and profile
  link.
- Tests: repository entitlement-expiry coverage, rewritten public/access service
  and handler tests, corrected statistics expectations, and a full backend
  Free-activation-to-session end-to-end test.
- Docs updated: API, database, architecture, security, page design and this
  registry.

Reason:

Clients could purchase or activate a programme but could not securely read its
structure, execute workouts, log sets or see meaningful progress. The previous
public detail also leaked the full programme structure without an access check.
This release closes that access gap, gives every grant a single authorization
path, and delivers the interactive training experience without changing the
legacy assignment/history endpoints.

Validation:

- Isolated MariaDB on `127.0.0.1:33107`, database `ryze_test_workout`;
  `migrate up` reported `migrations applied successfully`.
- `DB_HOST=127.0.0.1 DB_PORT=33107 DB_NAME=ryze_test_workout DB_USER=root DB_PASSWORD= go -C main/src/backend test -count=1 ./...` passed every package uncached.
- `DB_HOST=... go -C main/src/backend test ./routes -run TestFreeActivationWorkoutSessionE2E -v` passed.
- `DB_HOST=... go -C main/src/backend test ./repositories -run 'TestEntitlementRepositoryExpiry|TestStatisticsRepository' -v` passed.
- `go -C main/src/backend build ./...` and `go -C main/src/backend vet ./...` passed; `gofmt` reported no files.
- `npm run typecheck` and `npm run build` passed.
- `npm test` passed 18 files / 123 tests.
- `npm run test:e2e` passed 8/8 Playwright tests, including the metadata-only
  public detail and guest Free activation prompt.

---

# Version 2.99.0

Title: Mandatory SMTP TLS and production deployment hardening

Date: 2026-11-01

Summary:

v2.99.0 closes security and deployment gaps found during the production audit.
The contact notifier now requires STARTTLS for every delivery and refuses a
relay that does not advertise it; the previous opportunistic upgrade could send
envelope and message content in plaintext. The release binary embeds the
migration files, so the distroless image can migrate without a source tree. The
backup and restore scripts now preserve explicitly exported recovery settings,
fail closed on encryption errors and remove plaintext artefacts; the production
nginx, Compose and systemd configuration now sets security headers and a CSP,
trusts only the compose network for forwarded client addresses, allows the
shutdown drain to finish, and gives private trainer evidence a persistent
writable location.

No database schema, payment-provider, nutrition-generation, Premium Level 2,
product-scope or media-licensing behaviour changed in this release.

Main changes:

- `services/contact/smtp_notifier.go`: STARTTLS is mandatory before `MAIL FROM`;
  authentication still happens only after the TLS upgrade.
- `services/contact/smtp_notifier_test.go`: regression coverage proving a relay
  without STARTTLS receives no envelope and no message.
- `database/migrate.go`: migrations embedded with `//go:embed` and exposed
  through `source/iofs`; `migrate up` and `migrate version` work outside the
  source tree.
- `scripts/env_loader.sh`: loads the project `.env` without overwriting values
  already exported by a recovery command.
- `scripts/backup_db.sh`, `scripts/restore_db.sh`: GPG preflight, `0700`/`0600`
  permissions, plaintext and partial-ciphertext cleanup on encryption failure,
  and explicit restoration of caller-supplied connection settings.
- `main/src/frontend/nginx.conf`: HSTS, CSP, clickjacking/MIME/referrer/
  permissions headers, `server_tokens off`, and a 26 MiB body limit.
- `docker-compose.yml`: private trusted-proxy subnet, 30s stop grace period,
  `TRAINER_EVIDENCE_DIR`, and a persistent `backend_data` volume.
- `main/src/backend/Dockerfile`: non-root evidence directory and the matching
  runtime default; migrations now embedded in the shipped binary.
- `deploy/ryze-backend.service`: `StateDirectory`, evidence path, `UMask` and
  embedded-migration runtime notes.
- `.env.example`: v2.99.0 version, compose subnet and container evidence-path
  guidance.
- Docs updated: `how_to_run.MD`, `security.MD`, the product roadmap, the
  completeness audit and this registry.

Reason:

The audit found a security-relevant fallback in contact delivery and several
production-only failures that the development setup did not expose: the image
had no migration files, Docker could SIGKILL the API before its drain finished,
nginx did not emit the documented security headers, forwarded client addresses
were not trusted behind the proxy, and trainer evidence defaulted to a
non-writable path. Fixing them closes verified gaps without changing product
behaviour.

Validation:

- `gofmt -l .` produced no files; `go build ./...` and `go vet ./...` passed.
- The exact backend CI subset (`config`, `middleware`, `version`, `services/token`,
  `services/contact`, `services/public_programs`) passed with the race detector.
- Full database-backed `go test ./...` passed against a disposable MariaDB. A
  first run hit a transient MariaDB deadlock in
  `services/delete_account`; that package passed alone and the full suite passed
  on rerun.
- `npm run typecheck` passed; `npm test` passed 18 files/123 tests; `npm run
  build` passed.
- `npm run test:e2e` passed 7/7 tests.
- Contact delivery matrix passed: STARTTLS success, authenticated STARTTLS,
  no-STARTTLS refusal, recipient rejection, auth failure, connection refusal and
  untrusted TLS certificate. Validation, attachment type/size, honeypot, CRLF,
  body limit and 10/hour rate limiting behaved as documented.
- Production bundle with the documented CSP produced no
  `securitypolicyviolation` on Home, Services, Generic Marketplace, Premium
  Level 1, Exercise Library or Contact; a real contact submission succeeded
  through the same-origin proxy.
- Plain and GPG-encrypted backups restored into disposable databases with 33
  tables, migration version 32, matching row counts; an encryption failure left
  no artefact. Refusal without `--yes` exited 2.
- The embedded-migration binary ran from `/tmp` against a fresh database and
  reported version 32, `dirty=false`.
- `docker compose config` parsed successfully; `systemd-analyze verify` accepted
  the unit apart from the intentionally absent deployed executable.
- No Git operations were performed.

Residual and unverified:

- The Docker daemon is inaccessible and nginx is not installed locally, so the
  image was not built, the compose stack was not started and nginx did not load
  the configuration in this environment. Compose syntax and the CSP policy were
  verified separately.
- No real SMTP relay, Stripe/PayPal sandbox, hosted CI run or production
  infrastructure was available. Live delivery, provider settlement and hosted
  CI remain unverified.
- The Gym visual media licence remains unresolved; production media must stay
  disabled.
- Credentials that appeared in historical repository state still require
  operator rotation.
- The transient full-suite database deadlock is recorded as a local test
  reliability caveat, not a reproduced product failure.

# Version 2.98.0

Title: Server-side marketplace and real contact delivery

Date: 2026-10-30

Summary:

v2.98.0 completes the two remaining Phase B product gaps. The public catalogue
now owns marketplace filtering, sorting and pagination, including multi-value
filters, duration ranges and deterministic `updated_at` ordering. The generic
marketplace requests each row from that contract instead of downloading and
filtering the whole catalogue, and its discovery rows no longer imply invented
popularity. The contact form now submits multipart data to a real SMTP-backed
endpoint with validation, attachment limits, a honeypot, rate limiting and an
honest unavailable state. Playwright coverage now includes marketplace filtering
and both contact outcomes, and the E2E job is wired into CI.

No database schema, payment-provider, nutrition-generation, Premium Level 2 or
media-licensing behaviour changed in this release.

Main changes:

- `repositories/program_repository.go`: slice-based filter conditions, ORed
  values within a dimension, `duration_ranges`, `updated_at` sorting and an
  `id ASC` deterministic tie-breaker.
- `services/public_programs/public_programs.go`: multi-value validation with a
  20-value cap, frequency/range normalization and the expanded sort allowlist.
- `api/auth/public_programs_handler.go`: comma-separated parsing,
  `duration_ranges` parsing and `400 VALIDATION_ERROR` for an unknown scope.
- `services/contact`: validation/normalization service and SMTP STARTTLS
  notifier with server-owned sender/recipient and in-memory attachments.
- `api/auth/contact_handler.go`, `routes/routes.go`, `routes/ratelimit.go`:
  `POST /api/v1/contact`, honeypot handling, 1 MiB attachment cap, a 64 KiB
  multipart body-limit exception and a 10/hour per-IP scope.
- `config/config.go`, `config/env.go`: `ContactConfig` and fail-fast
  `CONTACT_*` loading.
- `services/marketplace_api.ts`, `generic_plan_marketplace_page.tsx`: server-side
  filter/sort/pagination per row, bounded 12-program sections, stale-request
  guard and honest result/empty copy.
- `hooks/use_service_catalog.ts`, `premium_level1_page.tsx`: page through the
  complete Premium Level 1 response set.
- `services/contact_api.ts`, `components/contact_form`: multipart submission,
  submit states, client-side size/type pre-checks, honeypot and `aria-live`
  feedback.
- `constants/marketplace.ts`, `components/service_card`, `admin2_marketplace_page`:
  replaced invented popularity labels with deterministic catalogue labels and
  "Featured".
- Tests: contact service/handler/config, public-program multi-value filters and
  sorting, marketplace/contact API clients, checkout cancellation/duplicate
  return/entitlement refresh, marketplace browser filtering and contact browser
  success/unavailable states.
- `playwright.config.ts` and `.github/workflows/ci.yml`: CI uses the
  Playwright-managed Chromium and runs the E2E suite with report artifacts.
- `.env.example`: documented `CONTACT_SMTP_*`, `CONTACT_FROM` and
  `CONTACT_RECIPIENT`.
- Docs updated: `api.MD`, `security.MD`, `arquitecture.MD`, `project.MD`,
  `design.MD`, `how_to_run.MD`, the product roadmap, the completeness audit and
  this registry.

Reason:

The catalogue contract still accepted only one value per filter and the generic
marketplace downloaded the whole catalogue before filtering it client-side. The
contact form was visual only, so the product advertised a channel that could not
deliver a message. Closing both gaps makes the API the single source of
filtering truth and makes the contact experience truthful about delivery.

Validation:

- `gofmt -l .` produced no files.
- `go vet ./...` passed.
- `go test ./...` passed across every backend package.
- `go test -race ./config/... ./middleware/... ./version/... ./services/token/... ./services/contact/... ./services/public_programs/...` passed; the CI job now runs the new database-independent contact and public-program tests.
- `npm run typecheck` passed.
- `npm test` passed: 18 files, 123 tests.
- `npm run build` passed; Vite emitted only the existing native-config and
  large-chunk warnings.
- `npm run test:e2e` passed: 7 tests (Services, Home, Premium catalogue,
  marketplace filtering, trainer prescription and both contact outcomes).
- Live API at `http://localhost:8080/api/v1`: health `200`; unknown scope `400`;
  invalid training type `400`; multi-value filter query `200`; contact
  submission `503 CONTACT_UNAVAILABLE` because SMTP is not configured.
- Live browser at `http://localhost:5173`: marketplace empty state rendered from
  the API, no "Most Popular"/"Trending This Week" copy, no horizontal overflow
  at 1440px or 390px, contact displayed the honest unavailable message and no
  page errors were observed.
- No Git operations were performed.

Residual and unverified:

- Contact positive delivery is unverified: no SMTP relay/credentials were
  available, so the endpoint correctly returns `503 CONTACT_UNAVAILABLE`.
- The live positive marketplace path is unverified: the database currently has
  zero published generic and Premium Level 1 programs. Server filters and
  sorting are covered by repository/service/handler tests and mocked browser
  tests.
- Stripe/PayPal/MB WAY sandbox purchase, return and entitlement flows remain
  unverified without provider credentials.
- The CI E2E job was added but has not run on a hosted runner in this cycle.
- Exercise-media licensing remains unresolved; production media stays disabled.
- No isolated disposable database, production deployment or real refund flow was
  exercised.

---

# Version 2.97.0

Title: API-driven catalogue, complete Premium package listing and trainer prescription editor

Date: 2026-10-29

Summary:

v2.97.0 closes the remaining frontend product-consistency gaps in Phase B. The
trainer program detail page now edits workout-exercise instructions, notes and
ordered sets through the Phase A prescription API. Home and Services read the
published generic and Premium Level 1 catalogue instead of hardcoded products,
and Premium Level 1 lists every returned package instead of only the first.
Deterministic Playwright browser tests now cover the public catalogue, Premium
package selection and trainer prescription editing against a mocked API.

No backend, database, payment-provider, nutrition-generation or media behaviour
changed in this release.

Main changes:

- `services/trainer_api.ts`: added the workout set vocabulary, prescription
  input/response types and `updateWorkoutExercisePrescription` for the
  trainer-scoped exercise PATCH.
- `components/trainer/prescription_editor`: inline editor for instructions,
  notes and ordered sets; supports add/remove/reorder, all five set types,
  numeric fields, tempo, max 50 sets and client-side validation mirroring the
  server contract.
- `pages/trainer/trainer_program_detail_page`: opens the editor from each
  exercise, saves through the API, refreshes the exercise row and renders a
  concise prescription summary.
- `hooks/use_service_catalog.ts` and `services/service_catalog.ts`: load both
  published families and derive category cards, price labels and Home
  highlights. Prices are derived from the catalogue; no product price or offer
  copy is hardcoded.
- `constants/services.ts`: replaced static `SERVICE_PROGRAMS` with
  `SERVICE_CATEGORY_PRESENTATION`; removed the Premium Level 2 placeholder and
  its `rcc3.png` asset reference.
- `components/programs_overview_section`, `pages/home_page` and
  `pages/services_page`: API-backed highlights, category preview and explicit
  loading/error/empty states. Home Premium highlights deep-link to the selected
  package.
- `pages/premium_level1_page`: lists every returned Premium Level 1 package,
  selects one through `?program=<id>`, offers "All packages" and resolves a
  provider return to the purchased package without changing the existing
  questionnaire, checkout, access or nutrition flow.
- Playwright: added `@playwright/test`, `playwright.config.ts`, a reusable API
  mock, public catalogue specs and a trainer prescription spec. The config uses
  system Chromium when available and runs a dedicated Vite server on port 5174.
- `.gitignore`: ignores Playwright `test-results/` and `playwright-report/`.
- Docs updated: `design.MD` (API-backed Home/Services, two category cards,
  Premium package catalogue, no Elite/Premium Level 2 page),
  `arquitecture.MD` (frontend catalogue and browser-test modules),
  `how_to_run.MD` (browser E2E command and mocked-API scope), the product
  roadmap, the completeness audit and this registry.

Reason:

The frontend still duplicated product data and pricing that the backend already
owns, exposed a Premium Level 2 placeholder that must not exist, rendered only
one Premium Level 1 package, and lacked browser-level coverage for the trainer
prescription editor. Closing these gaps makes the published catalogue the single
source of product truth and gives deterministic regression coverage for the
critical UI paths that unit tests cannot exercise.

Validation:

- `npm run typecheck` passed.
- `npm test` passed: 16 files, 111 tests.
- `npm run build` passed; Vite emitted only the existing native-config and
  large-chunk warnings.
- `npm run test:e2e` passed: 4 tests (Services catalogue, Home highlights,
  Premium package catalogue/selection, trainer prescription editing).
- The browser suite installs a request-level API mock and does not contact the
  backend, database or a payment provider; it is browser evidence, not a live
  integration run.
- No Go tests were rerun because no backend code changed in this release.

Residual and unverified:

- The backend at `http://localhost:8080/api/v1` was not responding at the last
  check, so no live browser/API/database validation was performed in this cycle.
- No isolated database, live payment credentials, production deployment or
  confirmed Gym media licence; provider purchase, restore drills and production
  media remain unverified.
- Marketplace server-side filtering/sorting/pagination and the contact
  submission backend remain open Phase B items.
- Playwright is not wired into CI, and a production-like live browser run,
  accessibility/responsive regression and checkout/access/evidence journeys
  remain open.
- Premium purchase-return handling and checkout-state switching were not
  exercised against a real provider return in this cycle.
- No Git operations were performed.

---

# Version 2.96.0

Title: Test Mode persona ownership, importer reporting and marketplace consistency audit

Date: 2026-10-24

Summary:

v2.96.0 closes the Test Mode persona trust boundary and corrects the exercise
importer's duplicate/validation reporting. Test Mode previously relied on two
fixed email addresses as the only link between a persona and its account, so a
public registration could squat a persona address and the enter flow could adopt
any pre-existing account. Persona ownership is now a durable database marker,
reserved addresses are refused by public registration, and every active-session
lookup re-verifies the marker instead of trusting the session row alone.

The exercise importer now distinguishes duplicate source IDs from duplicate
normalized names, rejects a source ID that maps to conflicting exercises, and
requires exactly one JSON array. The CLI report exposes the separated counters.
A full marketplace/publication/entitlement audit found no additional code defect;
the existing family scoping, publication gate and entitlement checks were
confirmed and remain covered by their regression tests.

Main changes:

- Migration `000032_add_test_mode_persona_ownership`: nullable
  `users.test_persona` (`client`/`trainer`, check constrained), generated
  `active_test_persona` (`IF(deleted_at IS NULL, test_persona, NULL)`) with a
  unique key. Existing rows are deliberately not backfilled; an unmarked
  account is never retroactively trusted.
- `models.User` carries the `test_persona` marker and never serialises it.
- `services/test_mode`: reserved-email helper, marker ownership checks,
  `ErrPersonaConflict`, marker writes during provisioning, duplicate-race
  revalidation, and `ActivePersona` re-loading the bound user on every status
  and purchase call. A legacy or forged session without the matching marker
  fails closed as `ErrNoActiveSession`.
- `services/registration`: the two fixed persona addresses, including
  case/whitespace variants, return `ErrDuplicateEmail` before lookup or
  creation, blocking public claims and soft-deleted-account reactivation.
- `api/auth/test_mode_handler.go`: maps persona conflicts to
  `409 TEST_MODE_PERSONA_CONFLICT` without exposing the conflicting account id.
- `services/exercise_import`: report fields split into
  `SkippedDuplicateSourceIDs` and `SkippedDuplicateNames`; same source ID with a
  different normalized name aborts the import; trailing JSON is rejected;
  source-linkage uniqueness failures are surfaced as integrity errors.
- `repositories/exercise_repository.go`: `ErrExerciseSourceTaken` distinguishes
  the active source-id unique key from the active-name unique key.
- `main.go`: importer report labels now print duplicate source ids, duplicate
  names and name conflicts separately.
- Tests: Test Mode ownership, conflict, forged-session, retry and reserved-email
  regressions; registration reserved-email coverage; importer duplicate
  separation, conflicting-source-id and trailing-data regressions; repository
  source-linkage coverage.
- Test reliability: the trainer-application document repository test now sets
  explicit distinct timestamps, removing a same-tick ordering flake.
- Docs updated: `api.MD` (reserved registration, Test Mode conflict/status,
  corrected purchase status and inactive-session errors, Premium Level 1 Test
  Mode scope), `security.MD` (durable persona ownership and reservation),
  `arquitecture.MD` (ownership model and importer classification),
  `database.MD` (users marker, Test Sessions ownership revalidation, migration
  032), `project.MD` (Test Mode Premium Level 1 prerequisites), `how_to_run.MD`
  (importer report/validation), and the product roadmap.

Reason:

A fixed email is not an ownership proof: without a durable marker, a public
registration or a legacy account could become a Test Mode identity. The
importer also conflated duplicate source IDs with duplicate names, making
integrity failures and intentional folds indistinguishable to operators.

Validation:

- `go run . migrate version`: version `32`, `dirty=false`.
- Real dataset dry run: `records=1324 unique=1318 created=0
  matched_by_source=1318 matched_by_name=0 updated=0
  duplicate_source_ids_skipped=0 duplicate_names_skipped=6 name_conflicts=0`;
  the transaction was rolled back and no catalogue row changed.
- `gofmt -l .` clean; `go vet ./...` clean; `go test ./...` all packages pass.
- Targeted race run passes: `go test -race ./services/test_mode
  ./services/registration ./services/exercise_import ./repositories
  ./api/auth`.
- Live backend rebuilt and restarted on the development database:
  `/api/v1/health` and `/api/v1/version` return 200, reserved-email registration
  returns `409 EMAIL_ALREADY_REGISTERED`, `/api/v1/auth/test-mode` reports
  `active:false`, and no persona account or persona marker exists.
- Marketplace audit: public catalogue scope, generic/Premium Level 1 admin
  family isolation, purchase prerequisite enforcement, published-program access
  gate and entitlement history were traced end to end; no new defect found.

Residual and unverified:

- No isolated/disposable database, provider sandbox credentials, browser E2E
  run, restore drill or trainer prescription editor was available.
- Gym visual media remains disabled pending commercial-use confirmation.
- Existing reserved accounts are not migrated to personas; an operator must
  review any conflict before that persona can be entered.

---

# Version 2.95.1

Title: Server-wide exercise-media gating for every embedded exercise summary

Date: 2026-10-23

Summary:

v2.95.1 closes a media-policy gap found during the v2.95.0 security audit. The
exercise catalogue already honoured `EXERCISE_MEDIA_ENABLED`, but public,
trainer, generic and client-program responses copied raw stored media fields.
With media disabled those endpoints still returned dataset-relative media paths,
contrary to the documented production gate; with media enabled they omitted the
configured base path and attribution.

All exercise summaries now use one shared media resolver. Disabled exposure
returns empty URLs and attribution with `media_available: false`; enabled
exposure prefixes the configured same-origin base path and returns attribution.
The catalogue's own previously ungated `video_url` is fixed by the same path.

Main changes:

- New `services/exercise_media`: shared `Resolver` for video, image, animation
  and attribution gating plus base-path normalization.
- `services/exercises`: catalogue media projection now routes every field,
  including `video_url`, through the resolver.
- `services/public_programs`, `services/client_programs`,
  `services/workout_exercises` and `services/generic_programs`: constructors
  accept `config.ExerciseMediaConfig`; every embedded exercise summary resolves
  media through the shared gate and exposes `media_available` and
  `media_attribution`.
- `api/auth/public_programs_handler.go`,
  `api/auth/client_program_handler.go` (through the shared trainer summary) and
  `api/auth/generic_programs_handler.go` expose the gated fields.
- `routes/routes.go`: passes `mediaCfg` to all affected services.
- Tests: resolver disabled/enabled/empty behaviour, catalogue video gating, and
  disabled/enabled media assertions for public, trainer, generic and client
  program services.
- Docs updated: `api.MD` (program exercise media fields and gate behaviour),
  `security.MD` (gate applies to every embedded exercise summary) and
  `product_roadmap.MD` (Phase A media coverage and Test Mode hardening item).

Reason:

The production media kill switch must not depend on which endpoint embeds an
exercise. Serving stored dataset paths while licensing is unresolved violates
the documented deployment gate and gives clients unusable URLs.

Validation:

- `gofmt` applied to changed Go files.
- `go vet ./...` clean.
- `go test ./...` all packages pass, including the new media-gating tests.

Residual and unverified:

- Production media remains disabled pending Gym visual commercial-use
  permission; no media is served by this change.
- Test Mode persona-email reservation was confirmed as a separate conditional
  hardening item and is tracked in Phase B; Test Mode remains disabled by
  default.

---

# Version 2.95.0

Title: Canonical trainer workout-exercise prescriptions and client prescription reads

Date: 2026-10-22

Summary:

v2.95.0 completes the backend gap between the trainer program structure and the
persisted exercise prescription. Trainers can now replace the instructions,
notes and ordered set list of an exercise in their own program workout, and both
trainer reads and the authenticated client's assigned-program read expose the
prescription. The same validation vocabulary and numeric bounds are now shared
by trainer-owned and platform-owned programs, and the generic program update
path was corrected to apply the submitted set values instead of reusing the
previous row values.

No schema migration was required. The `workout_exercise_sets` table and the
`instructions`/`notes` columns already existed; this release wires them through
the trainer API and the client assignment read, adds the missing set preloads,
and documents the contract. Removed tail sets are soft-deleted and historical
rows remain preserved. Client-visible responses continue to omit the owning
trainer and all parent identifiers.

Main changes:

- New `services/training_prescription` package: shared `SetInput`, controlled
  set-type vocabulary, 50-set cap, 20-character tempo cap and validation for
  reps, weight, RIR, RPE, rest and mandatory set type.
- `services/generic_programs` delegates set validation to the shared package;
  `repositories/generic_program_repository.go` now writes the desired set values
  when reusing an existing row (regression covered by
  `generic_program_repository_test.go`).
- `services/workout_exercises`, `repositories/workout_exercise_repository.go`
  and `api/auth/trainer_workout_exercise_handler.go`: prescription update with
  ownership-chain verification and row locking, ordered set reads, safe
  `instructions`/`notes`/`sets` responses, and `PATCH
  /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises/:workoutExerciseID`.
- `repositories/program_assignment_repository.go`,
  `services/client_programs/client_programs.go` and
  `api/auth/client_program_handler.go`: `/api/v1/me/program` now preloads and
  returns the assigned program's instructions, notes and ordered sets.
- `routes/routes.go`: trainer PATCH route registered under
  `trainer.programs`, alongside the existing add/list/order/read/delete routes.
- Tests: service validation/forwarding, handler success/IDOR/validation,
  client assignment response, production-router client E2E, and the generic
  set-update regression.
- Docs updated: `api.MD` (client and trainer prescription contracts),
  `database.MD` (repository methods, update lifecycle and tempo width), and the
  new canonical `docs/ROADMAP/product_roadmap.MD`.

Reason:

The persisted prescription schema and the client Program Access contract already
expected per-assignment sets, but the trainer API had no write/read path and the
assigned-program endpoint omitted them. Closing the gap makes trainer-authored
programs usable by clients without inventing a second prescription model or a
new schema.

Validation:

- `gofmt` applied to changed Go files.
- `go vet ./...` clean.
- `go test ./...` all packages pass, including real-database integration tests
  for trainer prescription success, replacement/history, IDOR, invalid sets and
  the client assigned-program response.

Residual and unverified:

- No trainer-facing prescription editor was added; the existing frontend is
  unchanged and browser-level editing remains a roadmap item.
- No migration was added and no destructive database operation was performed.

---

# Version 2.94.0

Title: Persisted Exercise Library with dataset import, 3D demonstrations, trainer-verification evidence review, and administrative catalogue management

Date: 2026-10-20

Summary:

v2.94.0 closes the product-completeness gaps identified in the audit at
`docs/AUDIT/product_completeness.MD`. The exercise catalogue is no longer a
static read-only seed: it is a persisted, administratively managed entity with
source provenance, body-part classification, ordered instruction steps, media
references and publication status, populated from the bundled dataset through
an idempotent import command (1,305 records imported and 13 existing rows
enriched; 1,353 active exercises, 6 duplicate dataset names skipped). The
public client gains a real Exercise Library (search, five server-side filters,
pagination, detail page, alternatives) and a genuine Three.js viewer for four
exercises. Trainer verification gains the review workflow it was missing:
applicants can attach optional evidence documents while an application is
pending, reviewers can download them and read an append-only decision history,
and approve/reject accept an optional reason. Administrators can manage the
catalogue at `/admin2/exercises` under new `exercises.read` /
`exercises.manage` permissions. Three migrations were added (000029–000031);
the development database is at version `31`, `dirty=false`.

Media is a deliberate exception: the dataset media is © Gym visual, not
MIT-licensed. It is not bundled or served; `EXERCISE_MEDIA_ENABLED` defaults to
false, the API reports `media_available=false` and empty media fields, and the
UI shows a neutral placeholder. This is a deployment blocker until
commercial-use permission is verified, and it is documented in `security.MD`
and `how_to_run.MD`.

The public UI was also corrected before release: the navbar now sizes itself so
the six links are distributed evenly between the brand and the auth actions
without overlap, the library renders only static images and fills each square
card, the detail page owns the animated media in a proportional split preview,
and the Home/Our Vision headings align to the shared page container instead of
drifting toward the centre or the viewport edge.

Main changes:

- Database, migrations `000029_extend_exercise_catalog_source_media_publication`,
  `000030_trainer_application_evidence_and_decisions` and
  `000031_add_exercise_three_d_demonstrations`: new exercise columns (`source`,
  `source_id`, `body_part`, `instruction_steps`, `animation_url`,
  `source_media_id`, `media_attribution`, `status`, `owner_trainer_id`,
  `three_d_key`, generated `active_source`), constraints/indexes, the evidence
  and decision tables, and the `three_d_key` constraint. Each migration has a
  record-preserving down file.
- Backend catalogue: `models/exercise.go`, `repositories/exercise_repository.go`
  (filters, search, publication scope), `services/exercises/exercises.go`
  (validation, status lifecycle, body-part vocabulary) and
  `services/exercises/admin_exercises.go` (draft-on-create, immutable name,
  status transitions, media reference validation); public handlers and the new
  `api/auth/admin_exercises_handler.go`; routes for `/exercises`,
  `/exercises/search` and `/admin/exercises`.
- Dataset import: `services/exercise_import/importer.go` and the
  `import-exercises --file <path> [--dry-run]` command in `main.go`; matching by
  `source`+`source_id` then normalized name, never overwriting existing values.
- Media gate: `config` reads `EXERCISE_MEDIA_ENABLED` and
  `EXERCISE_MEDIA_BASE_PATH` (same-origin validation); repository responses
  null the media fields and set `media_available=false` while disabled.
- Development-only media preview:
  `main/src/frontend/dev/exercise_media_preview.ts` (+ test) is a Vite
  serve-only middleware (`vite.config.ts`) that streams the downloaded dataset
  `images/`/`videos/` at `/dev-exercise-media` to loopback clients only, with
  fixed directories/extensions, realpath containment, no-store/nosniff/
  noindex headers and no build or preview counterpart. The preview endpoint is
  registered only when both local media variables are set; it also answers
  `404` for the dataset's project-relative and `/@fs/` URLs (in every dev
  session), so Vite's own file serving cannot bypass it. It requires the
  backend media flag and base path to be set locally; the library thumbnail and
  detail preview now also fall back to the placeholder when an available asset
  fails to load.
- Backend trainer evidence: `services/trainer_applications/documents.go`
  (magic-byte PDF/JPEG/PNG validation, size ceiling, atomic private storage,
  pending-only upload/delete),
  `repositories/trainer_application_document_repository.go`,
  `repositories/trainer_application_decision_repository.go` (append-only,
  chronological), and handler/routes for applicant documents and reviewer
  documents/decisions/download. Approve and reject record a decision row in the
  same transaction, with an optional normalized reason. Config reads
  `TRAINER_EVIDENCE_DIR` and `TRAINER_EVIDENCE_MAX_BYTES` (default 10 MiB,
  ceiling 25 MiB); `middleware/body_limit.go` exempts only the document upload
  path.
- Frontend public library: `services/exercise_library.ts`,
  `components/exercise_library/*` (preview body, media placeholder, library
  dialog) and `pages/exercise_library_page` /
  `pages/exercise_detail_page`; navbar route `/exercises` with debounced
  search, server filters, alternatives and honest loading/error/empty states.
  The list loads strict 10-item batches through an intersection sentinel,
  appends without duplicates, retries a failed batch in place, and renders only
  the static `image_url`, filling each square card with `object-fit: cover`
  and a wrapping attribution overlay; the grid uses two columns on small
  phones. Animated media is confined to the detail page, which uses the shared
  preview body in a split layout (content left, media right, `66rem` card cap,
  `18rem` media cap, media first when stacked below `860px`). Both public pages
  clear the fixed navbar with page-container top spacing.
- UI layout corrections: the navbar uses an `auto minmax(0,1fr) auto` grid and
  distributes the six links with `space-evenly` across the middle column, so
  they sit evenly between the brand and the auth actions instead of leaving a
  gap before Services; compact link metrics and an icon-only breakpoint at
  `1360px` keep the group clear of the auth actions and the Start free pill
  inside the bar (`navbar.module.css`). Home/Platform and Our Vision layouts
  keep their page content inside the shared container instead of full-bleed
  overrides, so the headings and copy align to the same left edge as the rest
  of the page.
- Frontend 3D: `components/exercise_library/exercise_3d_demo.tsx` (lazy chunk)
  and `exercise_3d_viewer.tsx` (procedural Three.js scene, orbit/zoom,
  play/pause, speed, WebGL fallback) for `squat`, `push_up`, `biceps_curl` and
  `lunge`.
- Frontend verification UIs: `trainer_application_documents.tsx` (+ CSS) in the
  pending state of `trainer_apply_page.tsx`; the review panel in
  `admin_trainer_applications_page.tsx` (+ CSS) with document list/download,
  decision history and optional decision note.
- Frontend catalogue administration: `services/admin2_api.ts` exercise section,
  `pages/admin2/admin2_exercises_page` (+ CSS), the `/admin2/exercises` route
  and the Programs navigation entry; create/edit modal with classification,
  content, instruction steps and relative media fields.
- Shared `main/src/utils/format_bytes.ts` (extracted from the trainer document
  panel and reused by the admin review panel).
- Docs updated in the same cycle: `api.MD` (exercise fields/filters, admin
  catalogue, applicant documents, reviewer evidence/decisions, optional
  decision reason), `arquitecture.MD` (canonical catalogue and evidence
  layering), `security.MD` (evidence uploads, exercise media exposure),
  `database.MD` (exercise schema plus evidence and decision tables),
  `design.MD` (Exercise Library pages/nav, Apply evidence panel, admin review
  panel, `/admin2/exercises`), `how_to_run.MD` (media/evidence configuration,
  dataset import, new endpoints) and this log.
- Tests: backend service/repository/handler/e2e coverage for import, public and
  admin catalogue, evidence upload/download/delete and decisions; frontend
  tests for the library, detail, media fallbacks, 3D viewer, document panel,
  admin review page, admin exercises page and API clients. The dev media
  middleware has its own resolver and HTTP test suite.

Validation:

- Backend: `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` all
  packages pass.
- Frontend: `npx tsc --noEmit` clean, `npx vitest run` 14 files / 99 tests
  pass, `npm run build` succeeds (index 746.77 kB / 210.47 kB gzip; CSS
  222.08 kB / 39.29 kB gzip; lazy 3D chunk 544.45 kB / 136.65 kB gzip;
  chunk-size warning accepted).
- Layout: headless Chromium checks at 1920×1080, 1366×768, 768×900 and 390×844
  confirmed the navbar links are evenly distributed with the Start free pill
  inside the bar (the first link starts 34px after the brand at 1366px), the
  Home and Our Vision headings share the container's 360px left edge at 1920
  and 83px at 1366, the library fills each card and shows two columns at
  390px, the detail split renders content left/media right at desktop and
  stacks media-first below 860px, both pages clear the fixed navbar, and the
  library shows `1–10` initially; batch append/dedup stays covered by the
  library tests.
- Local preview: the Vite dev server returned the shipped GIF/JPEG at
  `/dev-exercise-media/...` with `image/gif`/`image/jpeg`, `no-store` and
  `nosniff`; an encoded traversal request and the dataset's direct
  project-relative and `/@fs/` URLs returned `404`. The production build
  contains no GIF and no `dev-exercise-media` reference, and `vite preview`
  answered the same path with the SPA HTML instead of media. With
  `EXERCISE_MEDIA_ENABLED=false`, a separate dev server answered both the
  preview and direct URLs with `404`.
- Disabled media: a backend started with `EXERCISE_MEDIA_ENABLED=false` returned
  empty media fields, `media_available: false` and no attribution; the frontend
  fallback states are covered by tests.
- Database: development database at migration version `31`, `dirty=false`;
  1,353 active exercises; 4 entries with `three_d_key`.
- Import: 1,305 dataset records imported, 13 existing rows enriched, 6
  duplicate names skipped.

Residual and unverified:

- Exercise media is not bundled or delivered in production and may not be
  enabled there until Gym visual commercial-use permission is verified; the
  production delivery mechanism (e.g. nginx location) is an operator decision
  and was not implemented. Local inspection is limited to the loopback-only
  Vite dev middleware.
- The 3D viewer was validated by unit tests and a production build, not in a
  live browser/WebGL session; unsupported devices fall back to a message.
- No disposable database was available, so migrations were applied forward and
  verified on the development database only; the down files were not exercised
  (never run `migrate down` on dev, because migration `000018.down` has a
  pre-existing FK order flaw).
- Only four exercises have 3D demonstrations; the viewer is procedural and does
  not use dataset video or motion capture.
- Dataset media is 180×180. Filling wider library cards and the `18rem` detail
  column necessarily upscales it; sharper rendering requires higher-resolution
  source assets.
- The import command is explicit and is not run automatically on deploy.
- The administrative catalogue API does not support rename or permanent
  deletion; archived entries stay in the database.
- Evidence withdrawal is a soft delete and there is no restore endpoint; the
  evidence directory must be backed up with the database.
- Documentation header version strings were intentionally left unchanged; this
  log remains the version registry.

---

# Version 2.93.0

Title: Security audit and hardening — credential-leak remediation, CSRF origin guard, required JWT claims, rate-limiter and backup hardening

Date: 2026-10-18

Summary:

v2.93.0 is a full security audit with implemented remediation. Six audit domains
were reviewed: authentication/session/JWT, route authorization and ownership
(IDOR), SQL construction, uploads/media, frontend security, and
infrastructure/secrets/CI. The audit verified several strengths (parameterized
SQL everywhere, server-side ownership checks on every route, no XSS sinks or
token storage in the frontend, no upload surface exists) and one critical
finding: live administrator credentials were committed verbatim in backend test
fixtures, in a publicly reachable repository, alongside `.env` permissions of
644. The fixtures now use fake values, `.env` was rotated (new passwords and
access codes, mode 600), and the documentation states that a credential which
ever reached a repository must be treated as compromised and rotated by the
operator, since Git history cannot be un-published. High-severity gaps closed:
Test Mode `Exit` ignored the `TEST_MODE_ENABLED` kill switch, and state-changing
requests relied on `SameSite=Lax` + CORS alone (a same-site sibling host was
implicitly trusted). Both are fixed: `Exit` now refuses to restore any identity
from a stored session when the feature is disabled, and a new
`middleware.OriginGuard` requires browser-originated state-changing requests to
come from the API's own host or a `CORS_ALLOWED_ORIGINS` entry
(`403 CROSS_ORIGIN_FORBIDDEN`). Medium gaps closed: JWTs now carry and require
issuer `ryze` and audience `ryze-api`, the access-token lifetime is capped at
24h, the account rate limiter reads its identifier up to the configured
`SERVER_MAX_BODY_BYTES` (a raised body limit can no longer become a limiter
bypass), `POST /admin/auth/verify` gained a stage-cookie-keyed limiter, `/0`
trusted-proxy ranges are rejected at startup, GORM logs parameterized SQL only,
and the backup script runs under `umask 077` with an explicit warning when
unencrypted. Low gaps closed: admin credential checks are constant-time and
non-short-circuiting, `COOKIE_SECURE=false` prints a startup warning, frontend
service modules encode path segments, CI runs the race detector plus the
frontend test suite, and the two dev-only `npm audit` advisories were resolved.
No schema changed and no migration was added: the database stays at migration
version `28`, `dirty=false`.

Main changes:

- Backend, new `middleware/csrf.go` + `middleware/csrf_test.go`: `OriginGuard`
  rejects a state-changing request whose `Origin` (or fallback `Referer`) is
  neither an allow-listed origin nor the request's own host; malformed and
  opaque (`null`) origins are rejected; requests without any browser origin
  (curl, webhooks) pass; safe methods are exempt. Normalized to lowercase
  scheme/host; the scheme is not compared so a TLS-terminating proxy stays
  compatible.
- `routes/routes.go`: `middleware.OriginGuard(corsCfg.AllowedOrigins)` mounted
  after CORS and before the body limit; `newAuthRateLimiters` now receives
  `serverCfg.MaxBodyBytes`.
- `routes/ratelimit.go` + `routes/ratelimit_test.go`: `authRateLimitScope`
  gained `accountCookie`; the admin verify scope is keyed by
  `auth.AdminStageTokenCookieName`; `newAuthRateLimiters(cfg, maxBodyBytes)`
  carries the body bound into every account limiter.
- `middleware/ratelimit.go` + `middleware/ratelimit_test.go`:
  `AccountRateLimitKey(scope, field, maxBodyBytes)` reads the identifier up to
  the configured bound (fallback constant documented); new `CookieRateLimitKey`
  keys by an HMAC digest of a cookie value, so a credential cookie never enters
  a limiter table in clear. Tests for bound reading, oversized-body fallback,
  body restoration and cookie keying.
- `services/token/token.go` + `token_test.go`: all tokens are issued with
  issuer `ryze`, audience `ryze-api` and issued-at; parse requires
  issuer/audience/expiration and validates issued-at. Tests forge claims with
  the pinned values; new negative tests for missing issuer and wrong audience.
- `api/auth/admin_login_handler_test.go`: the forged expired stage token now
  carries issuer/audience so it exercises expiry instead of claim rejection.
- `config/env.go` + `config/env_test.go`: `JWT_ACCESS_TOKEN_TTL` is capped at
  24h (`maxTokenTTL`); `isTrustedProxyEntry` rejects `/0` ranges (`0.0.0.0/0`,
  `::/0`) with a clarified startup error. Tests for the ceiling and both
  wildcard forms.
- `services/test_mode/test_mode.go` + `test_mode_test.go`:
  `Exit` returns `ErrDisabled` when the feature is disabled; enforces the kill
  switch while still ignoring session age. `api/auth/test_mode_handler_test.go`
  asserts `403 TEST_MODE_DISABLED`.
- `services/admin_login/admin_login.go`: every configured credential is
  evaluated with digest-based constant-time comparisons and no short-circuit,
  so timing does not reveal whether a username exists or its list position.
- `database/database.go`: the GORM logger is configured with
  `ParameterizedQueries`, so error and slow-query logs emit SQL structure, not
  interpolated values.
- `main.go`: startup warning when `COOKIE_SECURE=false`.
- Credential fixtures scrubbed (fake values, usernames kept as identifiers):
  `api/auth/admin_login_handler_test.go`, `api/auth/admin_me_handler_test.go`,
  `config/env_test.go`, `services/admin_login/admin_login_test.go`. `.env`
  administrator passwords and access codes rotated and the file restricted to
  600 (not tracked).
- Frontend: new `main/src/utils/http_client.ts` helper `encodePathSegment`
  (+ `http_client.test.ts`); path parameters encoded in `admin2_api.ts`,
  `admin_api.ts`, `marketplace_api.ts`, `premium_level1_api.ts`,
  `purchases_api.ts`, `test_mode_api.ts`.
- Repository hygiene: `.gitignore` adds `backups/`, `uploads/`, `*.log`; both
  `.dockerignore` files exclude `uploads/`; `scripts/backup_db.sh` sets
  `umask 077` and warns when producing an unencrypted dump; CI runs
  `go test -race` on database-independent packages plus `npm test` on the
  frontend.
- `package-lock.json`: dev-only `nanoid` and `source-map-js` advisories
  resolved; `npm audit` reports zero vulnerabilities.
- Documentation: `security.MD` (implemented CSRF guard, required token claims
  and TTL ceiling, constant-time admin login and stage-cookie limiting, Test
  Mode kill switch on exit, body-bound and cookie-limiter rate limiting, `/0`
  proxy rejection, GORM parameterized logging, backup/umask practice, CI and
  dependency practice), `api.MD` (`403 CROSS_ORIGIN_FORBIDDEN`, upload
  endpoints marked not implemented), `arquitecture.MD` (middleware and
  validation list), `how_to_run.MD` (TTL cap, `COOKIE_SECURE` warning, origin
  guard, token-claim session invalidation on upgrade), `update_logs.MD`.

Reason:

A security audit found a committed credential leak (critical), a kill-switch
bypass on Test Mode exit and a CSRF gap beyond SameSite (high), missing JWT
claim validation, a body-limit-dependent rate-limiter bypass, an IP-only
access-code limiter, SQL values in ORM logs, insecure backup handling and CI
coverage gaps (medium/low). Test fixtures must never contain deployed
credentials, a disabled feature must be fully disabled, browser requests must
be origin-checked beyond SameSite, and identifiers read for rate limiting must
follow the configured body bound.

Validation:

- Backend: `gofmt -l .` empty; `go vet ./...` clean; `go test ./...` exit 0
  (40 tested packages, 0 failures, including database-backed suites);
  `go test -race ./config/... ./middleware/... ./version/... ./services/token/...`
  green.
- Frontend: `npx tsc --noEmit` clean; `npx vitest run` green (4 files, 23
  tests).
- Dependencies: `npm audit` reports 0 vulnerabilities.
- Secrets: the four previously committed credential literals and all rotated
  `.env` values have zero matches in tracked files at `HEAD`.
- `bash -n scripts/backup_db.sh` clean.

Residual and unverified (not claimed):

- The previous administrator credential values remain in the Git history of
  the public repository. Rotating the deployed environments overrides the
  leaked values operationally; history remediation (history rewrite) or
  provider-side secret scanning remains an operator action and was not
  performed here.
- `__Host-` cookie prefixes were not introduced (they conflict with the
  documented local HTTP development mode); the residual risk is mitigated by
  `SameSite=Lax`, `Secure` in production and the new origin guard.
- Browser-level CSRF/origin behaviour was verified by deterministic middleware
  tests, not by a live browser run.
- The race detector ran on the database-independent packages; the full
  database-backed suite ran without `-race`.
- Backup script hardening was verified by syntax check and inspection; no live
  backup/restore drill was executed.
- Live Stripe/PayPal provider flows were not executed (unchanged from v2.92.0).
- No schema change was made, so no migration up/down drill was required.

---

# Version 2.92.0

Title: Commerce reliability — PayPal readiness parity, duplicate-sale acknowledgement, capture race recovery and resolved-pair guards

Date: 2026-10-18

Summary:

v2.92.0 is a reliability pass over the purchase lifecycle, driven by an audit
of the commerce surfaces. Five verified defects were fixed. (1) The PayPal
provider and its webhook endpoint disagreed on readiness: provider creation
required a client id, secret and a supported mode, but the webhook was mounted
on a weaker condition and the verifier silently fell back to the sandbox API
for an unsupported mode, so a mistyped mode could accept deliveries for a
disabled provider. Both now share one readiness predicate and the verifier
fails closed. (2) A verified provider delivery for a purchase whose (user,
program) pair had already resolved another way (a sibling purchase completed
first) surfaced as an internal error and was retried forever with a
`transient_failure` record; it is now acknowledged with 200 and recorded as
`conflict_pair_resolved`. (3) The browser capture return leg could surface a
stale `404 PURCHASE_NOT_FOUND` when a concurrent verified webhook completed
the purchase mid-transaction; it now re-reads the durable state and returns
the idempotent success (or `409 PURCHASE_NOT_PENDING` when the purchase was
superseded) so the existing ownership-polling recovery applies. (4) The
completed-purchase unique index collision on the atomic completion paths was
unmapped, feeding the same retry loop; both repository paths now map it to
`ErrCompletedPurchaseExists` and the service reports it as
`ErrDuplicateEntitlement`. (5) Payment initiation could open a provider
session for a pending purchase whose pair was already owned, taking money
that could never grant access; initiation and capture now refuse resolved
pairs before the provider is contacted. The checkout hook correspondingly
recovers 409 `DUPLICATE_ENTITLEMENT` by resolving real ownership instead of
showing a dead-end error. No schema changed and no migration was added: the
database stays at migration version `28`, `dirty=false`.

Main changes:

- `routes/routes.go`: new `paypalEnabled` single readiness predicate (client
  id and secret present, mode `sandbox` or `live`). `resolvePaymentProviders`
  and the PayPal webhook mount share it (the webhook additionally requires
  `PAYPAL_WEBHOOK_ID`), so the provider and the endpoint can never disagree.
  `createPayPalWebhookVerifier` fails closed on an unsupported mode instead of
  silently verifying against the sandbox API.
  `routes/payment_providers_test.go`: predicate table test, provider-creation
  parity test, verifier fail-closed test.
- `services/purchases/purchases.go`: `InitiatePayment` refuses a purchase
  whose pair already resolved (`ErrDuplicateEntitlement`) before any provider
  is resolved or contacted. `captureAndComplete` refuses a pair that already
  owns an active entitlement before the capture call, and re-reads the durable
  state when its completion observes the purchase moving on: completed →
  idempotent success with the delivery hook, superseded →
  `ErrPurchaseNotPending`. `completeAndCreateEntitlement` maps the
  completed-purchase collision on both atomic paths to `ErrDuplicateEntitlement`.
  `ErrDuplicateEntitlement` documentation extended to mean "the pair is
  already resolved (active entitlement or completed sibling purchase)".
- `repositories/purchase_repository.go`: `CompleteWithEntitlement` and
  `CompleteWithRestoredEntitlement` map a duplicate-entry error on the
  purchase status update (the completed-purchase unique index) to
  `ErrCompletedPurchaseExists` instead of a raw wrapped driver error.
- `api/webhooks/stripe_handler.go` and `api/webhooks/paypal_handler.go`: both
  handlers map `ErrDuplicateEntitlement` to `200 "pair already resolved"` and
  record the durable `conflict_pair_resolved` outcome (PayPal covers both the
  capture-on-approval path and the capture-completed path). A redelivery can
  no longer loop with `transient_failure`.
- Frontend `hooks/use_program_checkout.ts`: shared `recoverOwnershipOrFail`
  helper; 409 `DUPLICATE_ENTITLEMENT` from intent creation, initiation and
  Test Mode completion now resolves the real access state (owned) instead of
  displaying a failure for a program the user owns.
- Tests: service regression tests for initiation refusal, capture refusal,
  completion collision mapping on both atomic paths, concurrent-completion
  recovery and superseded reporting; webhook acknowledgement tests for Stripe
  and both PayPal completion paths; new `use_program_checkout.test.tsx`
  (guest resolution, capture return leg, raced-capture recovery,
  duplicate-entitlement initiation, Test Mode conflict).
- Documentation: `api.MD` (initiation/capture `409 DUPLICATE_ENTITLEMENT`,
  capture race semantics, webhook resolved-pair acknowledgement, outcome
  vocabulary), `security.MD` (PayPal endpoint readiness parity, fail-closed
  verifier, mismatch verdicts corrected to the acknowledged/recorded
  behaviour, duplicate-sale acknowledgement), `arquitecture.MD` (PayPal
  availability rule, initiation guard, capture guard and moved-on recovery,
  repository collision mapping, webhook resolved-pair acknowledgement).

Reason:

The commerce audit found a retry loop on duplicate sales (money already
settled, pair already owned, delivery rejected forever with a 500), a
readiness asymmetry that could mount a webhook for a disabled PayPal, stale
failures on the capture return leg for buyers whose payment was already
settled, and provider sessions opened for purchases that could never
complete. Payment flows must be exactly-once per sale and fail closed, and a
buyer whose access settled by any path must never see a failure. Documentation
was corrected where it no longer matched the implementation (the webhook
mismatch verdicts were already acknowledged with 200 and recorded; some
documents still described 500 responses).

Validation:

- Backend: `gofmt -l`, `go vet ./...` clean; `go test ./...` green (full
  suite, including repository and route tests against the development
  database).
- Frontend: `npx tsc --noEmit` clean; `npx vitest run` green (3 files, 20
  tests).

Residual and unverified (not claimed):

- Live Stripe/PayPal sandbox flows were not executed.
- The concurrency races (sibling completion, capture race) are covered by
  deterministic stub-based tests, not by a live concurrent payment run.
- No schema change was made, so no migration up/down drill was required.

---

# Version 2.91.0

Title: Trainer platform — access status, exact-email client lookup, self-add guard, trainer web area and migration rollback fix

Date: 2026-10-16

Summary:

v2.91.0 ships the trainer platform end to end and fixes three verified
defects. The backend gains the two endpoints the trainer frontend needs
(`GET /api/v1/trainer/access` and `GET /api/v1/trainer/clients/lookup`) and the
existing `POST /api/v1/trainer/clients` self-relationship guard now actually
works: it previously compared the trainer profile UUID with the user UUID,
which can never be equal for the same person, so a trainer could add their own
account as a client. The guard now resolves the trainer's linked user id and
rejects the self-add with `400 VALIDATION_ERROR` (verified live). The trainer
web area is built entirely on existing RYZE systems: the shared HTTP client,
the shared exercise-library dialog and the incremental program-structure
endpoints; no parallel auth, upload, media or catalog system was introduced.
The frontend also gains its first automated test baseline (Vitest + Testing
Library, explicitly approved) and the trainer pages are validated live against
the development database. Three down migrations were also corrected: MariaDB
rejects `DROP CHECK`, so `000019`, `000024` and `000025` now use
`DROP CONSTRAINT`. No schema changed and no migration was added; the database
stays at migration version `28`, `dirty=false`.

Main changes:

- Trainer access status: new `FindLatestByUserID` on
  `TrainerApplicationRepository` (`created_at DESC, id ASC`), `AccessStatus`
  and `GetAccessStatus` on the `trainer_applications` service, and the
  `TrainerApplicationHandler.GetAccessStatus` handler for
  `GET /api/v1/trainer/access` (authenticated user, 401
  `AUTHENTICATION_REQUIRED`, 500 `INTERNAL_ERROR`). Response:
  `{is_trainer, application|null}` with the applicant's own view of the latest
  application (no nested user object). Registered in `routes/routes.go` right
  after `POST /trainer/apply`.
- Exact-email client lookup: new `FindByEmail` on `UserRepository`,
  `ClientCandidate` and `LookupClientByEmail` on the `trainer_clients` service
  (trimmed exact match, `400 VALIDATION_ERROR` on empty, `404 CLIENT_NOT_FOUND`
  when no active user matches) and the `TrainerClientsHandler.LookupClient`
  handler for `GET /api/v1/trainer/clients/lookup` behind `trainer.clients`.
  Registered before `/clients/:userID`.
- Self-add guard fix: `trainer_clients` service now depends on a
  `TrainerRepository`, resolves the authenticated trainer's linked user id and
  rejects adding that same user as a client. Dead `trainerID == userID`
  comparison removed. New tests `TestAddClientRejectsTrainersOwnAccount` and
  `TestAddClientTrainerCheckRepositoryFailure`; the invalid-id test loop keeps
  only genuinely malformed identifiers.
- Migration rollback: `DROP CHECK` replaced with `DROP CONSTRAINT` in
  `database/migrations/000019_add_generic_program_fields.down.sql`,
  `000024_add_payment_method_to_purchases.down.sql` and
  `000025_add_premium_level_1.down.sql`. Statement-level verified against
  MariaDB with disposable clone tables; a full up/down/up chain drill was not
  run because no disposable database is available in this environment.
- Trainer web area (`main/src/frontend/src`): new `services/trainer_api.ts`
  (typed client over the shared `http_client`), `components/trainer/`
  (`trainer_session_context.tsx` as the single backend-driven source of trainer
  status; `trainer_layout/` shell with loading-login-apply redirects and
  navigation; shared `trainer_page.module.css`), and `pages/trainer/` (apply,
  dashboard, clients, programs, program detail with the week/workout/exercise
  builder, profile). Routes wired in `routes/app_routes.tsx` with
  `/trainer/apply` declared before the `/trainer` shell; the Profile page hub
  links to "Trainer Dashboard" or "Become a Trainer" based on the access
  endpoint. All totals shown are real (pagination totals); no fake statistics.
- Trainer list responses: the frontend client unwraps the actual
  `{weeks: [...]}` and `{exercises: [...]}` envelopes of
  `GET /trainer/programs/:programID/weeks` and
  `GET .../workouts/:workoutID/exercises` (an earlier client draft assumed raw
  arrays; caught by the live smoke test).
- Repository test robustness: `TestTrainerApplicationRepository` now expresses
  list/count assertions as deltas against a baseline captured before seeding
  (the development database already contained committed applications),
  consistent with the existing trainer-repository test fixes. No assertion was
  weakened: the created application must still be listed and the status filter
  must still count it.
- Frontend test setup (new dev dependencies, explicitly approved: `vitest`,
  `jsdom`, `@testing-library/react`): Vitest configuration in `vite.config.ts`
  (`jsdom`, same path aliases), `npm test` script, and the first test files —
  `services/trainer_api.test.ts` (endpoint contracts, id encoding, the
  `{weeks}` / `{exercises}` envelope unwrapping) and
  `components/trainer/trainer_session_context.test.tsx` (loaded payload, 401 →
  unauthenticated, non-401 failures not treated as unauthenticated).
- Documentation: `api.MD` (access status, client lookup, self-add wording),
  `project.MD` (access endpoint and trainer web area), `arquitecture.MD`
  (trainer web area module map), `PAGES/design.MD` (Trainer Area pages),
  `how_to_run.MD` (frontend test command).

Reason:

The trainer platform is a core RYZE capability: certified trainers manage their
clients and their programs. The work order required a complete, verified
trainer surface built only from existing systems, an honest audit of what
exists, and fixes for verified defects. The self-add guard was ineffective
(proved by a live end-to-end run creating a trainer → self client
relationship), and the migration down files would have failed on MariaDB.

Validation:

- Backend: `gofmt -l`, `go vet ./...` clean; `go test ./... -count=1` green
  (40 packages).
- Frontend: `npm run typecheck` and `npm run build` green; `npm test` green
  (2 files, 15 tests).
- Live end-to-end smoke test against the development database covering:
  register/login, access before/after apply, duplicate apply conflict, trainer
  routes blocked before approval, admin login + verify, approval, access after
  approval, profile, lookup (found/unknown), add/re-add/self-add/remove/re-remove
  client, program create/update/list/detail, price validation, client-supplied
  `product_type` not honored, weeks/workouts/exercise add-reorder-delete,
  publish/unpublish, cross-user access denied, and public catalog visibility
  (draft hidden, published visible, unpublish and delete hide again). Test
  accounts created during the run remain in the development database.

Residual and unverified (not claimed):

- Full migration up/down/up chain could not be executed (no disposable
  database; Docker unavailable in this environment).
- Provider sandbox/Live payment flows and off-host backups remain unverified.
- Browser/responsive validation of the trainer area was not performed.
- Frontend automated coverage is a baseline (API client and trainer session);
  page-level component tests are not part of this milestone.

---

# Version 2.90.0

Title: Authentication hardening — rate limiting, uniform password policy, admin session revocation and token-kind enforcement

Date: 2026-10-15

Summary:

v2.90.0 hardens the authentication surface. The credential and account
endpoints had no abuse protection, every path that sets a password enforced a
different rule (client registration and self-service change required a
non-empty password while the administrator bootstrap required 6 characters),
admin sessions could not be revoked without rotating the JWT secret, and token
validation still accepted an access token that carried no `kind` claim. Now the
authentication endpoints are rate limited, a single reusable password policy of
12–128 characters is applied everywhere a password is set, an administrator
password or access-code rotation revokes outstanding admin sessions, and token
validation requires the expected kind. No migration was added and no schema
changed: the database stays at migration version `28`, `dirty=false`. No
existing endpoint changed shape; the only contract additions are the `429
RATE_LIMITED` response and the tightened password-validation outcomes, both
documented in `api.MD`.

Rate limiting is applied only to the six credential and account endpoints
(`/auth/register`, `/auth/login`, `/admin/auth/login`, `/admin/auth/verify`,
`/auth/change-password`, `/auth/delete-account`) so provider webhooks and every
other endpoint are never rejected by it. Each endpoint limits independently by
client IP, and the endpoints that carry an account identifier add a second
counter keyed by the normalised identifier, so credential stuffing against one
account is slowed even from many addresses. Identifiers are never stored in
clear: the counter key is an HMAC-SHA256 digest under a process-local random
key. Counters are in memory, per instance, and bounded with least-recently-used
eviction. The client address is only taken from forwarded headers when the
proxy is explicitly trusted, so a client cannot spoof its own key.

Login no longer leaks account existence through timing: an unknown email still
performs a dummy Argon2id verification, so it is indistinguishable from a wrong
password in both response and timing. Registration keeps its
`EMAIL_ALREADY_REGISTERED` conflict as an accepted, documented residual
enumeration risk because the frontend and the (as-yet unbuilt)
email-verification recovery flow depend on it; the registration rate limit
mitigates it.

Main changes:

- Rate limiting: new `middleware/ratelimit.go` (`Limiter`, `RateLimit`,
  `IPRateLimitKey`, `AccountRateLimitKey`; fixed window, injectable clock,
  LRU-bounded), new `routes/ratelimit.go` (per-scope policies and the
  `authRateLimiters` chain), and the six authentication routes wired through
  it in `routes/routes.go`. An exhausted limit returns `429 RATE_LIMITED` with a
  `Retry-After` header.
- Trusted proxies: `routes.Setup` now calls `SetTrustedProxies` with
  `RATE_LIMIT_TRUSTED_PROXIES` (default none, so the socket address is used);
  an invalid list fails closed by trusting nothing.
- Password policy: `services/password` gained `ValidatePolicy`,
  `ErrPasswordTooShort`, `ErrPasswordTooLong`, `MinPasswordLength` (12) and
  `MaxPasswordLength` (128). Registration, change-password, trainer creation
  and the administrative create/reset paths all apply it and report
  `400 VALIDATION_ERROR`. The admin bootstrap minimum in `config/env.go` now
  reuses `password.MinPasswordLength`.
- Admin session revocation: `services/token` registers, at startup, an
  HMAC-SHA256 credential version per configured administrator derived from the
  password and access code; admin tokens embed it and the validator requires a
  constant-time match against the registered version, so rotating an admin
  credential revokes every outstanding admin session.
- Token kinds: `ValidateAccessToken` now rejects any token whose `kind` is not
  `user`, so a token that omits the claim can no longer be treated as a user
  session (admin tokens already required `kind = admin`).
- Login hardening: unknown-account logins verify against a fixed dummy
  Argon2id hash before returning the shared invalid-credentials error.
- Config: `config.RateLimitConfig` and `config.LoadRateLimit` with
  `RATE_LIMIT_ENABLED` (default true), `RATE_LIMIT_MAX_KEYS` (default 10000)
  and `RATE_LIMIT_TRUSTED_PROXIES`; `main.go` loads it and passes it to
  `routes.Setup`.
- Tests: `middleware/ratelimit_test.go` (window, reset, isolation, disabled,
  LRU, 429 + `Retry-After`, identifier normalisation, body preservation,
  pass-through), `routes/ratelimit_test.go` (scope wiring), rate-limit config
  tests in `config/env_test.go`, token tests for admin credential revocation and
  missing/empty kind, and updated auth/admin-auth test helpers to register
  administrator credentials.
- Docs: `security.MD` (password policy, account-enumeration resistance,
  admin session revocation, token kind strictness, implemented rate limiting,
  brute-force mitigations), `api.MD` (429 `RATE_LIMITED` shape, per-endpoint
  rate limits, password policy, login/enumeration behaviour), `arquitecture.MD`
  (authentication middleware/configuration layer), `conventions.MD`
  (password policy), `how_to_run.MD` and `.env.example` (`RATE_LIMIT_*`,
  administrator password minimum, compliant test password).

---

# Version 2.89.0

Title: Launch infrastructure — health and version endpoints, build CI, deployment manifests and backup tooling

Date: 2026-10-13

Summary:

v2.89.0 adds the operational surface the documentation promised but the code
never exposed, and the tooling needed to build, deploy and back it up
reproducibly. `api.MD` documented `GET /api/v1/health` and
`GET /api/v1/version`, but neither route existed; the version the API reported
had no single source. Now both endpoints exist, the version comes from the binary
the process was built from, and a continuous-integration workflow, container
definitions, a systemd unit and backup scripts accompany them. No migration was
added and no schema changed: the database stays at migration version `28`,
`dirty=false`. No existing endpoint changed shape and no product behaviour
changed.

The health endpoint probes the database and answers `503 SERVICE_UNAVAILABLE`
when it is unreachable, so a load balancer never routes traffic to an instance
that cannot serve requests. When no database handle is configured it reports
healthy without probing, which keeps the endpoint usable during startup wiring.
The version endpoint reports a build-time value that defaults to `dev` for local
builds and is injected in production, so the reported version and the release
notes cannot drift.

The deployment surface is host-agnostic. `docker-compose.yml` runs the database,
the API and an nginx frontend; the frontend proxies `/api/` to the backend so the
session cookie stays first-party and no CORS exception is needed. `scripts/`
provide a consistent, optionally encrypted dump and a restore that refuses to run
without an explicit confirmation.

Main changes:

- Endpoints: public `GET /api/v1/health` (database probe; `200 {status:"ok"}` or
  `503 SERVICE_UNAVAILABLE`) and `GET /api/v1/version` (`{version}`), registered
  in `routes.Setup` and served by `auth.HealthHandler`.
- Version: new `version` package whose `Version` variable is overridden at build
  time with `-ldflags "-X ryze/backend/version.Version=<version>"` and defaults
  to `dev`.
- Tests: `api/auth` cover a healthy database, a missing database handle, an
  unreachable database (503 envelope) and the build version reported by the
  version endpoint; `gofmt`, `go build ./...` and `go vet ./...` are clean.
- CI: `.github/workflows/ci.yml` runs backend formatting, build, vet and the
  database-independent unit tests, plus frontend typecheck and build. The
  database-backed suite is intentionally excluded because it requires a
  provisioned MariaDB and still carries known pre-existing failures.
- Deployment: `docker-compose.yml`, `main/src/backend/Dockerfile` (distroless,
  non-root, version injected), `main/src/frontend/Dockerfile` (Vite build served
  by nginx), `main/src/frontend/nginx.conf` (SPA fallback + `/api/` proxy),
  `deploy/ryze-backend.service` (hardened systemd unit) and `.dockerignore`
  files.
- Backups: `scripts/backup_db.sh` (transactionally consistent, compressed,
  optional GPG encryption and local retention) and `scripts/restore_db.sh`
  (gzip/GPG aware, refuses without `--yes`).
- Docs: `how_to_run.MD` (endpoints list, deployment, health/version, backups),
  `api.MD` (health/version response shapes and 503) and `.env.example`
  (`DB_ROOT_PASSWORD`, `RYZE_VERSION`).

---

# Version 2.88.0

Title: HTTP server hardening — connection timeouts, request body limit and graceful shutdown

Date: 2026-10-13

Summary:

v2.88.0 hardens the HTTP transport the API is served over. The server was
previously started with Gin's `Run`, which builds an `http.Server` with no
read/write/idle timeouts and terminates abruptly on a signal, and no endpoint
bounded the size of a request body. Now `main.go` runs an explicit `http.Server`
with bounded connection timeouts, a global request body limit that rejects
oversized bodies with `413 REQUEST_ENTITY_TOO_LARGE`, and a graceful shutdown
that drains in-flight requests on `SIGINT`/`SIGTERM` before closing the database
handle. No migration was added and no schema changed: the database stays at
migration version `28`, `dirty=false`. The only API contract addition is the new
413 status; every existing behaviour, including webhook signature verification
against the raw request body, is unchanged.

The body limit never mutates a request. The middleware buffers each body up to
the configured limit, rejects anything larger with a deterministic 413 before
any handler runs, and re-exposes the exact bytes it read. Buffering in the
middleware, rather than relying on `http.MaxBytesReader` alone, is what makes the
rejection a 413: the JSON binding layer would otherwise report an oversized body
as a generic 400. The webhook handlers therefore still verify their provider
signatures against the precise raw body they receive.

Two failure modes are closed. A slow or stalled client can no longer hold a
connection open indefinitely (read-header, read, write and idle timeouts), and a
single client can no longer force unbounded memory allocation by streaming an
arbitrarily large body. A deploy or restart now stops accepting new connections,
lets in-flight requests finish within the shutdown timeout, and only then closes
the database handle, so a restart never drops an in-flight request.

Main changes:

- Config: `config.ServerConfig` plus `LoadServer`, reading
  `SERVER_READ_HEADER_TIMEOUT` (10s), `SERVER_READ_TIMEOUT` (30s),
  `SERVER_WRITE_TIMEOUT` (60s), `SERVER_IDLE_TIMEOUT` (120s),
  `SERVER_SHUTDOWN_TIMEOUT` (20s) and `SERVER_MAX_BODY_BYTES` (1 MiB). Every
  value has a safe default; a present-but-invalid value fails startup.
- Middleware: `middleware.BodyLimit` rejects a declared oversized body up front,
  bounds an undeclared (chunked) body through a limited reader, and returns the
  project error envelope with 413 `REQUEST_ENTITY_TOO_LARGE`.
- Routes: `routes.Setup` takes the server configuration and installs
  `BodyLimit` after `CORS`, so even a body-limit rejection carries the CORS
  headers a browser needs to read it.
- Main: an explicit `http.Server` (read-header/read/write/idle timeouts) run
  until it fails or receives `SIGINT`/`SIGTERM`, then `Shutdown` within the
  configured timeout and an explicit database close.
- Tests: `config` cover the server defaults, custom values and the invalid
  duration / body-limit rejections; `middleware` cover a body within the limit,
  a body exactly at the limit, a declared oversized body, an undeclared
  (chunked) oversized body, the disabled limit and the 413 error envelope.
  `gofmt`, `go build ./...` and `go vet ./...` are clean.
- Docs: `how_to_run.MD` (server safety section, env template), `security.MD`
  (transport hardening), `api.MD` (413) and `.env.example` document the new
  behaviour and configuration.

---

# Version 2.87.0

Title: Family-aware platform-owned program management — Premium Level 1 becomes administrable through its own route group

Date: 2026-10-12

Summary:

v2.87.0 makes the platform-owned program administration family-aware end to end,
so a Premium Level 1 program can be created, edited, listed, published and
soft-deleted by the Management Administrator instead of being seeded directly. No
migration was added and no schema changed: the database stays at migration
version `28`, `dirty=false`. No refund, regeneration, Premium Level 2 or new
payment provider surface was introduced, and the storefront keeps rendering the
premium family through the existing `premium_level_1` public catalog scope.

The family is a route property, never client data. The generic administration
service accepts an explicit product family on every operation
(`SupportedProductTypes` = `generic` and `premium_level_1`), and one handler
instance per family stamps the family of the route the request arrived on
(`/api/v1/programs/generic` and `/api/v1/programs/premium-level-1`, both under
the same `PermissionPlans` gate). Requests carry no `product_type` field, the
family can never be reclassified through an edit, and every repository query is
scoped to `trainer_id IS NULL` **and** one family, so a generic operation can
never touch a Premium Level 1 program and vice versa. Responses expose the
family as `product_type` so the client renders the right lifecycle.

The management console follows the same rule. The Plans page lists one family at
a time (Generic / Premium · Level 1 tabs, never a merged view), the plan
workspace selects the family at creation and locks it afterwards, and the plan
detail page reads, edits, publishes and deletes through the family route and
links the Premium Level 1 storefront for published premium programs. The labels
are also corrected: FREE/PREMIUM/PERSONALIZED are the commercial pricing
categories (Free Plan / Premium Plan / Personalized Plan) and are no longer
described as product tiers, which removes the "Premium · Level 2" suggestion for
a category whose family is absent from the constraint on purpose.

Main changes:

- Service: `generic_programs` takes the family as an explicit parameter on every
  method, validates it against the closed `SupportedProductTypes` list, stamps it
  onto the model and echoes it in summaries; coverage includes
  `TestCreateProgramStampsProductFamily`,
  `TestUnsupportedProductFamilyRejectedEverywhere` and family-forwarding checks.
- Handler: `NewGenericProgramsHandler(svc, productType)` binds the route's
  family; the response gains `product_type` and a body-supplied family is ignored
  (`TestGenericProgramsFamilyBinding`).
- Routes: `/programs/premium-level-1` mirrors the generic group under
  `AdminAuthenticate` + `RequireAdminPermission(adminroles.PermissionPlans)`.
- Repository: every query scoped to `trainer_id IS NULL` plus one family; the
  family is never part of an update, so reclassification is impossible
  (`TestGenericProgramRepositoryPremiumLevel1Family` proves isolation in both
  directions).
- Tests (all passing): services `generic_programs`, handler suite across both
  route families, repository suite, and transactional routes e2e; `gofmt`,
  `go build ./...` and `go vet ./...` clean. The api/auth suite still fails only
  the 4 known baseline tests and repositories only the known
  `TestTrainerRepository`.
- Frontend: `admin2_api.ts` adds the family constants/helpers and separates the
  commercial-category labels from the family labels, exposes `product_type` on
  program summaries, and routes the six generic program calls through
  `/programs/{family}` with the family as the first argument;
  `plan_builder_model.ts` carries the family on the draft; the Plans page uses
  family tabs plus a pricing-category filter; the create workspace adds a family
  selector (locked on edit); the detail page acts through the family route and
  links the premium storefront; dashboard and marketplace rows keep the family on
  their plan links. Frontend typecheck and production build pass.
- Docs: `how_to_run.MD` replaces the "seed a Premium Level 1 program directly"
  step with the real ADMIN_2 workflow through the Plans page.

---

# Version 2.86.0

Title: Commerce integrity, durable reconciliation diagnostics and read-only administrative visibility — every verified settlement leaves an immutable audit record

Date: 2026-10-12

Summary:

v2.86.0 hardens the settlement audit and gives the Management Administrator a
read-only reconciliation view. One direct migration (`000028`) recording every
verified provider webhook delivery, the webhook layers answering permanent
mismatches with 200 plus a durable outcome, and a read-only
`/admin/purchases/reconciliation-events` surface (ADMIN_2, `commerce`
permission only) were added. No refund, regeneration, Premium Level 2 or new
payment provider surface was introduced; reconciliation logic, the single-pending
guarantee, Test Mode, Premium Level 1 privacy and the Generic checkout are
unchanged. Schema version moves to `28`.

Every verified webhook delivery is now durably recorded. Each Stripe or PayPal
event that passes verification writes one row to
`purchase_reconciliation_events`, keyed by `(provider, event_id)`, stamped with a
closed outcome vocabulary (15 codes). Recording is best-effort observability: a
persistence failure never changes the HTTP verdict and never decides whether a
purchase completes. Outcomes are immutable once resolved — only the provisional
codes (`transient_failure`, `provider_uncertain`) may be overwritten by a later
redelivery of the same event — so a provider retry can never flip a settled fact
and duplicate deliveries stay idempotent.

Permanent contract mismatches and Test Mode bounds are answered 200 and recorded
instead of returned as 500. An amount, currency, payment method or reference
mismatch can never settle the purchase, so retrying is pointless: RYZE now
acknowledges with 200, records the outcome (`amount_mismatch`,
`currency_mismatch`, `method_mismatch`, `reference_mismatch`,
`permanent_invalid`, `test_purchase_event`), and lets the provider stop
redelivering while the administrative view preserves the diagnostic. 500 remains
only for transient failures and integrity violations so the provider genuinely
retries.

Administrative visibility without a write surface. `GET
/admin/purchases/reconciliation-events` (ADMIN_2, `commerce` permission, route
published before `/purchases/:id`) returns the audit trail newest-first with
scope (`open` default — resolved settlements hidden — vs `all`), optional exact
`outcome` and `provider` filters. It is read-only: no endpoint can change a
purchase, an entitlement or an event, and it exposes no decision a browser can
take — recovery stays server-side. The response never projects user ids,
`failure_reason`, secrets, raw payloads or any revenue/commission breakdown. The
frontend Sales page adds the Reconciliation section: scope + outcome selects,
an outcome-badged events table, pagination, loading / error+retry / empty states
and the shared purchase detail modal — visibility only, with client-owned
wording for every outcome code.

Main changes:

- Migration: `000028_create_purchase_reconciliation_events` (applied, DB at
  version `28`, `dirty=false`) adds the audit table, `(provider, event_id)`
  unique index and helper indexes; down drops it. The pre-existing `000025`
  `down` migration remains broken on MariaDB (error 1064), documented and
  unrelated.
- Model: `PurchaseReconciliationEvent` with the closed outcome vocabulary,
  provisional/resolved helpers and `IsValidReconciliationOutcome`.
- Repository: `reconciliation_event_repository.go` adds `Upsert`
  (final-outcomes-immutable upsert), `Record` (the webhook
  `ReconciliationOutcomeRecorder` adapter) and `ListAdmin` (outcome, provider
  and open/all scope filters, unscoped purchase preload, safe projection without
  `failure_reason`).
- Webhooks: shared outcome recorder thread through both handlers; Stripe and
  PayPal record every outcome, return 200 on permanent mismatches and Test Mode
  bounds, 500 only on transient/integrity failures, and stay best-effort on
  persistence errors.
- Admin commerce: `ListReconciliationEvents` service method + validated filter,
  handler, DTO and route before `/purchases/:id`; ADMIN_1 forbidden, ADMIN_2
  `commerce` only.
- Tests: webhook recorder + permanent-mismatch/200 suites for both providers,
  events repository suite (provisional overwrite, immutable finals, filters,
  validation), admin endpoint suite (scope, filters, 400s, authorization, lean
  projections), real-DB concurrency suites (reconcile and complete, single
  winner under `-count=3`).
- Frontend: `admin2_api.ts` adds reconciliation types, outcome metadata with
  client-owned wording and the events fetcher; the Sales page adds the
  read-only Reconciliation section (types, tests and typecheck/build pass).
- Docs: `api.MD` (endpoint contract + outcome vocabulary + webhook
  semantics), `security.MD` (recording best-effort, immutability, read-only
  surface, lean projections), `project.MD`, `database.MD` (table + migration
  `000028`) and `design.MD` (Reconciliation section) updated.

---

# Version 2.85.0

Title: Payment reconciliation, single pending intent and webhook settlement durability — late verified payments are never discarded

Date: 2026-10-11

Summary:

v2.85.0 closes the settlement window where a provider-verified payment could be
silently lost. One direct migration (`000027`) and one webhook-bound work
surface were touched; the database moves to migration version `27`. No refund,
regeneration, Premium Level 2 or new payment provider surface was introduced and
the proven PayPal/Stripe return-context handling is unchanged; Test Mode,
Premium Level 1 privacy and the Generic checkout are unaffected.

Late settlements are reconciled, never discarded. Previously both webhooks
answered "purchase not pending" with 200 whenever a verified payment arrived
after the purchase left `pending`, so a payment captured on the browser return
leg while a supersede was landing could grant nothing and be silently dropped.
Now a `failed` purchase is the recovery state: `ReconcilePurchase`
(service) / `ReconcileFailedPurchase` (repository) atomically returns it to
`completed` — superseding any stale pending sibling, running the guarded
failed→completed transition and restoring-or-creating the entitlement in one
transaction. The two database uniqueness guards
(`uq_purchases_completed_purchase`, `uq_entitlements_active_entitlement`) make a
second sale impossible: a conflict rolls everything back and the failed record
stays as the persisted, diagnosable recovery state. A webhook answers 200
(reconciled) on success, 200 with a diagnostic on a conflict or purchase-that-moved-on,
and 500 only on an integrity violation or transient failure so the provider
redelivers. `ReconcilePurchase` is internal-only: no route accepts a purchase id
without an authenticated owner, so a browser can never trigger it.

Single pending intent at the database level. Migration `000027` adds the
`pending_purchase` generated column with a unique index, so the engine itself
enforces at most one active pending intent per `(user, program)` pair. The
repository maps a violation to `ErrPurchaseAlreadyExists` and intent creation
re-evaluates the resolved ownership (`DUPLICATE_ENTITLEMENT` or
`DUPLICATE_PURCHASE`) instead of leaking a raw driver error.

PayPal capture settlements ordered by proof, not by notification. The handler
now processes `PAYMENT.CAPTURE.COMPLETED`: the order is loaded from the PayPal
API (never trusted from the event), its single purchase-unit reference recovers
the RYZE purchase, the captured amount and currency are verified against the
snapshot, and the purchase is completed (pending) or reconciled (failed). A late
`CHECKOUT.ORDER.APPROVED` for a failed purchase is consent, not money — it is
never captured and is acknowledged with a diagnostic, letting the order lapse at
PayPal; when funds actually moved, the capture event reconciles.

Client recovery. `useProgramCheckout` now polls ownership briefly (5 × 1 s)
after a 409 `PURCHASE_NOT_PENDING` instead of immediately declaring failure,
and the fallback message no longer claims "no money was taken" when the payment
may already have moved.

Main changes:

- Migration: `000027_enforce_single_pending_purchase` (applied, DB at version
  `27`, `dirty=false`) adds `uq_purchases_pending_purchase`; down migration drops
  it. Repository `Create` maps the violation to `ErrPurchaseAlreadyExists`.
- Repository: `repositories/purchase_repository.go` adds
  `ReconcileFailedPurchase` (one transaction: supersede stale pending siblings,
  guarded failed→completed with `ErrCompletedPurchaseExists` /
  `ErrPurchaseNotFound` verdicts, restore-or-create entitlement with
  `ErrEntitlementAlreadyExists`).
- Service: `services/purchases/purchases.go` adds the `ReconcilePurchase` service
  method and `ErrReconciliationConflict`; an unknown purchase is a settled state
  (`ErrPurchaseNotCompleted`), a completed purchase is idempotent, and intent
  creation handles the concurrent-writer duplicate.
- Webhooks: shared `reconcileLateSettlement` helper; Stripe routes a `failed`
  purchase into reconciliation; PayPal gains the combined verifier surface
  (`PayPalWebhookVerifier`), a `PAYMENT.CAPTURE.COMPLETED` handler backed by a
  server-side order read, and the late-approval diagnostic (never captured).
- Tests: repository reconcile suite (success, restore, conflicts, sibling
  supersede, single-pending rejection), service reconcile suite, and webhook
  late-settlement suites for both providers including the no-refund sweep.
- Frontend: `use_program_checkout.ts` polls ownership after 409
  `PURCHASE_NOT_PENDING` and uses an honest confirmation message.
- Docs: `api.MD`, `security.MD`, `project.MD` and `database.MD` document the
  reconciliation lifecycle, the single-pending guarantee and migration `000027`.

---

# Version 2.84.0

Title: Purchase integrity, payment recovery and transaction safety — expired-purchase supersede, published-program revalidation at initiation, and atomic entitlement restoration

Date: 2026-10-09

Summary:

v2.84.0 closes the purchase-integrity audit of the payment lifecycle with three
production fixes and one removal. No schema change was made: the database stays
at migration version `26` with `dirty=false`, and no migration file was added,
modified or removed. Expiry is derived and never stored, so superseding an
expired purchase needs no new column. No refund, regeneration, Premium Level 2
or new payment provider surface was introduced; the proven PayPal/Stripe
return-context handling was re-verified and left untouched.

Second charge protection for abandoned checkouts. A pending purchase could be
paid at the provider without any server-side time limit, so a buyer who created
an intent and abandoned it could be charged by a provider session opened long
after the checkout was meant to end. A pending purchase is now payable only
within a payment window measured from its creation timestamp (24 hours by
default, configurable server-side via `WithPurchaseExpiryWindow`): once the
window lapses, `InitiatePayment` refuses with `409 PURCHASE_EXPIRED` and never
opens a new provider session. The buyer recovers by starting a fresh intent,
which supersedes the expired purchase — it is marked `failed`, closing the
pending slot — and returns a new pending purchase. The supersede only ever acts
on the pending row and is guarded at the database level (`WHERE status =
'pending'`), so a concurrently completed purchase or a competing supersede is
left untouched and ownership is re-evaluated before a fresh intent is allowed.
A `failed` purchase is terminal: it cannot be initiated, completed or reach
access. Completion paths deliberately ignore expiry: a provider-verified
payment that settles after the window still completes instead of being
discarded, so a buyer who genuinely paid is never stranded.

No provider payments for retired programs. `InitiatePayment` revalidated
nothing about the program after the intent was created, so a pending purchase
for a program that was unpublished or soft-deleted in the meantime could still
open a new provider session. Initiation now reloads the program through the
published-only lookup and refuses new provider contact with
`404 PROGRAM_NOT_FOUND` when it is no longer published. Completion is
deliberately unaffected: money already captured at the provider still
completes, and access rendering already derives from publication.

Atomic entitlement restoration. When a buyer completed a repurchase over a
soft-deleted entitlement, the restore and the completion were separate
database operations: a crash or a racing completion between them could leave an
active entitlement with a pending (or missing) purchase. The purchase
repository now provides `CompleteWithRestoredEntitlement`, which restores the
soft-deleted entitlement and completes the purchase inside a single transaction
and rolls everything back if the purchase is not pending. The service drops its
two-step restore-then-complete path and the dead `payment payments.Provider`
constructor argument from `purchases.NewService`. Error mapping for the new
expiry refusal was added to the handler surface.

Frontend recovery. The shared checkout hook recovers from `PURCHASE_EXPIRED`
transparently: it creates a fresh intent once (the backend supersedes the
expired purchase) and continues the checkout instead of showing a dead end. The
guard is bounded — a checkout freshens its intent at most once.

Main changes:

- Expiry: `services/purchases/purchases.go` gains `ErrPurchaseExpired`,
  `defaultPurchaseExpiryWindow` (24h) and `WithPurchaseExpiryWindow`;
  `CreatePurchaseIntent` supersedes an expired pending purchase (`Fail`) and
  re-checks ownership; `InitiatePayment` refuses expired purchases.
- Published revalidation: `InitiatePayment` reloads the program via
  `FindPublishedByID` and refuses (`ProgramNotFound`) before contacting any
  provider; completion paths are untouched.
- Atomic restoration: `repositories/purchase_repository.go` adds
  `CompleteWithRestoredEntitlement` (single tx, restore rolled back on
  non-pending) and `Fail` (pending-guarded); the service completion path uses
  the atomic method and falls back to a fresh entitlement only when nothing is
  left to restore; the service interface drops `Complete` and
  `RestoreByUserAndProgram`.
- Removal: `payment payments.Provider` parameter removed from
  `purchases.NewService` (dead argument); `routes/routes.go` and all test call
  sites updated.
- Handler: `api/auth/purchase_handler.go` maps `ErrPurchaseExpired` to
  `409 PURCHASE_EXPIRED`.
- Tests: service tests cover expiry boundaries, supersede vs completion race,
  unpublished-program refusal, failed-purchase refusal and the not-expired
  happy path; repository tests cover `Fail` guards, restore+complete success,
  replay fail-closed and rollback on non-pending purchases; handler table adds
  the `PURCHASE_EXPIRED` code.
- Frontend: `use_program_checkout.ts` handles `PURCHASE_EXPIRED` with a
  one-shot fresh-intent recovery.
- Documentation: `api.MD` (initiation validity window + `409 PURCHASE_EXPIRED`
  + `404 PROGRAM_NOT_FOUND` at initiation, completion ignoring expiry, restore
  atomicity, failed status semantics), `security.MD` (expiry, published check,
  restore rollback), `DATABASE/database.MD` (derived expiry — no migration;
  `CompleteWithRestoredEntitlement`; failed status), `project.MD` (payment
  window and authoritative completion wording), and this update log. Versions
  bumped to 2.84.0 where the document changed.

# Version 2.83.0

Title: End-to-end Premium Level 1 integration audit and production hardening — envelope consistency, questionnaire contract agreement, bounded Test Mode sessions and safer payment method binding

Date: 2026-10-08

Summary:

v2.83.0 is the result of an end-to-end audit of the Premium Level 1 purchase
lifecycle (catalog → intake → purchase → payment → capture → entitlement → plan
delivery → access). The milestone found and fixed genuine integration defects
across the backend contract, the questionnaire payload agreement, the payment
flow and the frontend checkout recovery, and hardened two production
behaviours. No schema change was made in this build: the database stays at
migration version `26` with `dirty=false`, and no migration file was added,
modified or removed. No refund, regeneration or Premium Level 2 surface was
introduced; audit-only notes are recorded below and were deliberately left
out of scope.

Response envelope consistency. The three questionnaire endpoints
(`GET /me/questionnaire/questions`, `GET /me/programs/{id}/questionnaire`,
`POST /me/programs/{id}/questionnaire`) and the two nutrition assignment
endpoints returned their payloads flat while every other endpoint in the
platform returns `{ success, message, data }`. All five now wrap their payload
under the standard `data` envelope. The premium E2E test was updated to read
through the envelope, so the integration test asserts the real wire contract.

Questionnaire contract agreement. The intake submission endpoint binds on the
server side, so a catalog type that does not match the payload type silently
fails to save. `training_days_per_week` was declared as a `select` with string
options while the payload field is an integer; the catalog now declares it as
`number` with the server's bounds, matching the int field. The catalog also
declares several `list` questions (allergies, intolerances, excluded foods,
disliked foods, medical conditions, injuries, limitations) that the form could
not render — they fell through to the default text input and produced a string
that the `*[]string` server field rejects. The form now renders `list` fields as
one free-typed entry per line in a multi-line control (blank lines dropped,
entries trimmed), so every server-declared type has a faithful control.

Bound Test Mode sessions. The persona session had no server-side lifetime:
`GET /auth/test-mode` and persona purchases accepted any session whose cookie
had not expired, regardless of how old the entry was, because the server kept no
record of time alongside the session. Persona sessions are now bounded by the
same lifetime as the access token (`ACCESS_TOKEN_TTL`, configured server-side
via `WithSessionTTL`); an expired session has no active persona and an absent
persona answers `false`/`403` exactly as if no session existed. TTL zero means
unlimited (test-only defaults preserved). Exit is deliberately exempt from the
expiry check so the stored administrator identity can always be restored and an
admin is never stranded in an impersonated context. The existing disabled-path
behaviour is now explicitly tested (`TEST_MODE_DISABLED`), and the
socket/restore surfaces preserved the previous behaviour.

Safer payment method binding. `InitiatePayment` recorded the validated method
on the purchase as soon as the provider was resolved, so a provider rejection
left a method bound to a purchase that had no provider payment — the buyer could
not retry with another method without a mismatch refusal. The method is now
written only after the provider accepts the initiation; a rejection leaves the
method unbound and retryable with any other method. The repository
`SetPaymentMethod` gained a `WHERE payment_method IS NULL` single-writer guard
as defense in depth, and a same-method replay (the supported retry path) is no
longer re-persisted. Capture and webhook flows are unchanged: they still re-derive
the provider only from the recorded method, never from client input.

Frontend checkout recovery. Payment return URLs are configured for the Generic
Program page and are shared by both product families, but the frontend had no
route matching `/services/generic-program/{program_id}/purchase/success` or
`.../cancelled`, so a returning buyer landed on the 404/`NotFound` fallback
instead of the checkout detail page. Both exact routes now render
`GenericProgramDetailPage` and are declared before the `/services/:serviceSlug`
catch-all. The `useProgramCheckout` hook no longer resolves the retry method
from the currently selected control (which may not be the method the purchase is
bound to) — a retried payment now reads the purchase's recorded method from the
purchase history and resolves the provider from it. The mount-time capture
return leg runs before and independently of the product-detail load, so a buyer
who already paid at the provider is never left un-captured because the program
data failed to load.

The database `payment_method` column documentation is unchanged in structure
(it still exists, migration `000024`) but the documented write timing now
reflects "only after provider acceptance". Test session documentation was added
to `api.MD` and `security.MD`.

Main changes:

- Backend contract: `questionnaire_handler.go` and `nutrition_handler.go` wrap
  success payloads in `{ success, message, data }`; `premium_e2e_test.go`
  asserts through the envelope.
- Questionnaire: the catalog (`services/nutrition_questionnaire/questionnaire.go`)
  declares `training_days_per_week` as `number` (Min/Max from server bounds)
  instead of `select`; frontend `premium_questionnaire_form.tsx` added a `list`
  multi-line control.
- Test Mode: `services/test_mode/test_mode.go` gains `WithSessionTTL` +
  `Option` and an `enabled` gate that is now tested; `routes/routes.go` wires
  `WithSessionTTL(jwtCfg.AccessTokenTTL)`; new tests cover expiry, zero-TTL,
  disabled and exit-after-expiry.
- Payments: `services/purchases/purchases.go` records the method only after
  provider acceptance and only when unbound; `repositories/purchase_repository.go`
  guards with `WHERE payment_method IS NULL`; `purchases_test.go` covers the
  provider-rejection-releases-method path and the no-re-persist replay.
- Frontend: `app_routes.tsx` registers the two payment return routes before the
  slug catch-all; `use_program_checkout.ts` resolves the retry method from the
  purchase record and runs the return-leg capture independently of the detail
  load.
- Documentation: `api.MD` (envelope, catalog types incl. `list`, payment
  binding timing, Test Mode lifetime), `security.MD` (payment single-writer,
  persona session TTL), `DATABASE/database.MD` (payment_method write timing),
  `PAGES/design.MD` (list control), `how_to_run.MD` (return routes),
  `project.MD` (payment initiation wording), and this update log. Versions
  bumped to 2.83.0 where the document changed.
- Audit notes deliberately left out of scope (no change): an inactive pending
  purchase never expires (frontend recovery resumes it), the `failed` purchase
  status is currently unreachable, a dead `payments.Provider` constructor
  parameter, a non-transactional entitlement restore path, the exported
  `CreateEntitlement`/`RevokeEntitlement` surface, the published re-check at
  completion, PayPal return-context plumbing, and the hardcoded prices / Premium
  Level 2 card in frontend constants.

---

# Version 2.82.0

Title: Nutrition plans are now normalised, versioned and server-owned — generator v2 with an immutable plan lifecycle

Date: 2026-10-06

Summary:

v2.82.0 replaces the v2.81.0 single JSON plan snapshot with a normalised,
versioned plan domain and rewrites the generator around it. A delivered plan is
no longer an opaque document: the daily contract, every meal, every food and the
exclusions audit are persisted as rows, and each food is an immutable snapshot
taken from a server-owned food catalog that ships with the engine as code.
Because the plan is immutable and versioned, nothing can ever silently rewrite
what a client was shown, and the storage engine itself guarantees at most one
active plan per assignment.

The generator is upgraded to a fully deterministic search over that catalog. For
a given intake it always produces the same plan — the same fingerprint — which
makes regeneration idempotent and provably in place: a plan whose intake has not
changed is returned as it is, a failed run is retryable on the same revision, and
Plan delivery is now part of a completed purchase. The purchase service runs a
post-commit fulfilment hook that generates and stores the plan before the
checkout finishes; the hook is idempotent and its failures are logged and
deliberately swallowed, because no nutrition problem may ever unwind a verified
payment. A failure leaves the assignment retryable in place and never destroys a
previously delivered plan.

The API contract gains an explicit on-demand path,

```text
POST /api/v1/me/programs/{programID}/nutrition/generate
```

that is idempotent when the intake is unchanged and supersedes the delivered plan
when the intake revision has moved on. Supersession is atomic: the next plan
version is inserted, the previous active version is marked superseded and the
assignment completes at the new revision in one transaction with the assignment
row locked, so a concurrent run can neither observe a half-written version nor
race to deliver a competing plan. The claim itself is compare-and-swap, so two
simultaneous runs never both generate, and database constraints (
`uq_nutrition_plans_active`, `uq_nutrition_plans_version`) back the same
guarantee. A set of stored restrictions that cannot be satisfied now answers
`422 NUTRITION_RESTRICTIONS_UNSATISFIABLE` with a fixed sentence that names no
health data and alters nothing.

The plan document stays exclusively a read view. It exposes the version,
`engine_version`, `fingerprint`, `out_of_date` and the full plan body, with every
meal item as an immutable snapshot that shows the catalog code it came from, the
exact quantity and unit, its macros and any substitution note. The exclusions
audit is written from controlled server tokens (`reason_code`) and normalised
vocabulary entries (`token`), never from the free text a client typed, so no
sensitive health answer is ever duplicated outside the questionnaire row; free
text that no token resolves surfaces only as a generic caution and is never
echoed. The internal `failure_reason` of an assignment remains never-served.

On the frontend, the nutrition panel is a real renderer of the delivered
contract instead of summary tiles. It shows the daily targets, the day's meals as
a list of actual foods with quantity, unit and macros, substitution notes, the
exclusions audit, the server cautions, hydration and preparation guidance. A plan
the server marks `out_of_date` is still shown with the delivered content and a
stale banner that offers to update from the latest answers; a generation refusal
shows client-owned wording for the server reason code; and a failed refresh never
hides a delivered plan, because the page re-reads the authoritative state before
deciding what to render. A delivered plan is otherwise read-only: the only calls
to generation are the pending/failed build and retry actions and the defensive
stale-banner update.

The repository layer is now integration-tested against the real schema. New tests
cover the assignment state machine (idempotent pending, owner and program
scoping across owners, compare-and-swap claims rejecting in-flight and stale
runs, failure and retry) and the plan lifecycle (all rows persist and round-trip,
cross-owner reads fail closed, supersession retains the previous body, the
database rejects a second active plan, a stale questionnaire revision is refused,
and unclaimed or cross-owner completion is refused). The premium E2E test was
corrected to the real fulfilment contract: a completed purchase arrives with a
`completed` plan, regeneration is fingerprint-identical, and exactly one
assignment and one active plan row exist afterwards.

One defect was found and fixed during the work. Child-plan rows (meals, meal
items, exclusions) were linked and keyed inside the assignment service, splitting
ownership of a plan's identity across two layers; the repository now stamps
`plan_id` and `meal_id` and mints the IDs, so the service places formed plan
content and the persistence layer owns row identity. The retained
`nutrition_assignments.configuration` column is documented as deprecated leftover
from v2.81.0.

Main changes:

- Migration `000026_persist_nutrition_plans` adds `nutrition_plans`, with a
  `STORED` generated `active_plan` column enforcing one active plan per
  assignment (`uq_nutrition_plans_active`), `UNIQUE (assignment_id, version)`,
  `engine_version`, `fingerprint`, the normalised macro body, controlled
  `cautions` and soft deletes; `nutrition_plan_meals` with a `meal`/`snack`
  `CHECK` and unique positions; `nutrition_plan_meal_items` as immutable food
  snapshots with `catalog_code` provenance; and `nutrition_plan_exclusions` as
  the controlled-token audit trail. Four read views,
  `v_nutrition_plan`, `v_nutrition_plan_meal`, `v_nutrition_plan_meal_item` and
  `v_nutrition_plan_exclusion`, become the only read surface for a plan.
- The generator (`services/nutrition_assignment`, engine v2) is deterministic,
  catalog-backed, idempotent and supersede-aware, with compare-and-swap claims
  and a `NUTRITION_RESTRICTIONS_UNSATISFIABLE` path.
- Purchase fulfilment runs a post-commit hook that generates and persists the
  plan; failures are logged and swallowed so a payment is never undone.
- `POST /nutrition/generate` is the explicit on-demand endpoint (idempotent,
  supersede, `409 NUTRITION_NOT_READY`, `422` on unsatisfiable restrictions).
- The frontend nutrition panel (`premium_nutrition_plan.tsx`) renders the full
  delivered contract with a stale banner and client-owned failure wording, on
  both the Premium Level 1 page and Program Access.
- Repository integration tests for the assignment and plan lifecycle and a
  corrected premium E2E test.
- Documentation updated across `security.MD`, `arquitecture.MD`, `database.MD`,
  `design.MD`, `api.MD` and `how_to_run.MD`; `nutrition_assignments.configuration`
  is documented as deprecated, and the plan document replaces the old JSON
  snapshot contract.

---

# Version 2.81.0
