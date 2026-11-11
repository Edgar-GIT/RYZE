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
