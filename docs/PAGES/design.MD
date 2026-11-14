# RYZE
# Design Specification

Version: 2.104.0

---

# Design Philosophy

RYZE follows a premium AI-powered SaaS design language.

The platform should always feel:

- Elegant
- Premium
- Futuristic
- Dark
- Minimal
- High contrast
- Technology-first
- Clean
- Apple Vision Pro + Nothing + Raycast + Linear + Vercel quality

This is not a generic fitness website aesthetic.

It is a dark technology product interface with athletic purpose.

The Hero section is the single visual source of truth for the entire website.
Every other page and section must match that language — never the other way
around.

Token rules and the official palette live in `docs/INSTRUCTIONS/design.MD`.
This file focuses on page composition and component behaviour.

---

# Brand Identity

The RYZE brand represents progress, discipline and innovation.

Its visual identity combines near-black surfaces, electric cyan accents and
dark glass panels with soft ambient blue glow.

The logo should always be used consistently across the platform.

---

# Color Palette

Official palette. Do not invent alternate blues, teals or greys.

Background

- Primary Background: `#05070B` → `--bg-primary`
- Secondary Background: `#0A1018` → `--bg-secondary`
- Elevated Surface: `#101826` → `--surface`
- Cards: `#141F2D` → `--card`
- Card Hover: `#182638` → `--card-hover` / `--surface-hover`
- Navbar: `rgba(8, 12, 18, 0.78)` with `backdrop-filter: blur(20px)` → `--navbar-bg`

Primary Accent

- Primary Accent: `#18D8FF` → `--accent`
- Secondary Accent: `#2A8CFF` → `--accent-secondary`

Accent Gradient (primary CTAs)

- `linear-gradient(90deg, #18D8FF, #2A8CFF)` → `--accent-gradient`

Glow

- Glow: `rgba(24, 216, 255, 0.35)` → `--accent-glow`
- Strong Glow: `rgba(24, 216, 255, 0.5)` → `--accent-glow-strong`
- Soft Glow: `rgba(24, 216, 255, 0.16)` → `--accent-glow-soft`

Text

- Primary: `#FFFFFF` → `--text-primary`
- Secondary: `#B8C4D8` → `--text-secondary`
- Muted: `#7F8EA3` → `--text-muted`
- Disabled: `#4D5A6C` → `--text-disabled`

Borders

- Default: `rgba(255, 255, 255, 0.08)` → `--border` / `--border-color`
- Hover: `rgba(24, 216, 255, 0.35)` → `--border-hover`
- Focused: `#18D8FF` → `--accent`

CSS custom properties in `main/src/frontend/src/styles/global.css` are the
runtime source of truth for these tokens.

Every component must consume these variables. Hardcoded palette colors should
disappear from UI CSS except documented exceptions.

Never use:

- Old teal / green-blue accents
- Rainbow graph colors
- Hard grey cardboard surfaces
- Harsh black-only shadows without ambient cyan glow
- Random decorative circles, dotted curves, SVG lines or floating icon clutter

Icons may use white, muted grey or electric cyan only (except official third-
party brand marks such as Gmail / Instagram / WhatsApp).

---

# Visual Style

- Darker overall surfaces aligned with the Hero
- Living atmospheric background (soft cyan nebulae, vignette, subtle motion)
- Sections use translucent washes so atmosphere shows through
- Soft cyan glows around interactive elements and ambient washes
- Premium lighting on buttons via the accent gradient
- Cards as dark glass panels with shared surface tokens
- Soft ambient blue glows instead of harsh black shadows
- Spacing defines sections more than hard separators
- Never use a flat, static single-color page fill
- Section transitions must feel seamless — no obvious palette jumps
- Empty space is filled with soft gradients, radial lighting, subtle particles,
  ambient glow and depth — nothing else

---

# Typography

Primary Font

Google Sans

Fallback

- Inter
- Roboto
- Arial
- sans-serif

Hierarchy rules:

- Small uppercase eyebrows with wide tracking
- Strong short headlines
- Short supporting descriptions
- Prefer visual product demos over long paragraphs

---

# Design Principles

## Mobile First

Every interface must be designed for mobile devices before desktop.

## Product Over Brochure

Major homepage sections should demonstrate the platform through UI previews,
dashboards, charts and interactive product surfaces.

## Consistency

Buttons, cards, forms, spacing and typography should behave consistently
throughout the entire application.

When scrolling, the user should never feel that a section was built by another
designer. The whole website must read as one coherent premium product.

## Simplicity

Every screen should have one primary objective.

## Accessibility

The interface should provide sufficient color contrast, large clickable areas
and clear navigation.

## Performance

Animations should enhance the experience without negatively affecting
performance.

---

# Components

## Buttons

Primary

- Cyan / blue gradient (`#18D8FF` → `#2A8CFF`)
- Soft cyan glow
- High contrast text

Secondary

- Transparent
- Thin light border
- Cyan hover border / soft fill

## Navbar

- Floating rounded rectangle (`border-radius: ~1.15rem`), soft cyan border glow
- Soft glass fill that becomes more opaque on scroll
- Brand left (with divider), navigation links evenly distributed across the
  flexible middle column, auth CTAs right
- Nav links show icon + uppercase label at rest
- On scroll, nav link labels collapse and only icons remain
- At 1360px and below the labels collapse to icons at rest as well; the
  icon-only group remains centred, so the six links and the auth actions never
  overlap and the Start free pill always remains inside the bar
- Active page uses cyan icon/text plus a glowing cyan underline
- Auth actions: outline “Log in” (with user icon) + gradient cyan “Start free”
  (with arrow)
- Never use boxed active navigation items as the sole active treatment
- Language: one PT | EN toggle (2026-11-14). The whole pill is a single button:
  a click anywhere switches the language and a cyan light slides to the
  language in use with a slight spring; ← / → pick a side; its accessible name
  says the current language and the one it switches to. The same toggle (with a
  language icon) appears on the profile and account-recovery pages.

## Cards

- Rounded corners
- Dark glass effect
- Soft `rgba(255,255,255,0.08)` border
- Soft cyan glow on hover
- Slight lift on hover

## Service cards

- Rectangular image-backed cards using `rcc1.png` (generic marketplace) and
  `rcc2.png` (Premium Level 1) as full-bleed backgrounds, covered by a readable
  dark gradient overlay
- Content is minimal: compact right-aligned price above a full-width CTA
- Featured program (`Premium Level 1`) is taller (different aspect ratio),
  keeps the stronger cyan glow, and shows a floating "Featured" mini
  rectangle above the top edge
- Premium Level 2 carries a purple "Personalized" tag in the same place: Premium
  Level 1 is the most chosen route, Premium Level 2 the most personalized one
  (owner decision, 2026-11-11)
- Hover: lift + cyan border glow
- Used on `/services` only; Home presents the same categories with the
  image-free category overview described under Home
- Protected: the cards' markup and styles stay unchanged when the page around
  them is redesigned

## Inputs

- Dark surfaces
- Thin borders
- Cyan focus ring

## Icons

- White by default
- Accent icons use cyan

---

# Animations

- Smooth transitions: 200–300ms
- Buttons gently glow
- Cards slightly lift
- No exaggerated motion

---

# Section Backgrounds

Public website pages (Home, Our Vision, Contact, Feedback, Services,
login/registration and the public under-development placeholders) use the
public website layer defined in `docs/INSTRUCTIONS/design.MD` ("Public Website
Layer"): one continuous logo-navy page with a repeating rhythm of soft blue
light, transparent sections (no seams or colour jumps), the app's blue glass
cards, and the light-blue wave panel as the only bright surface. No background
images and no reliance on the animated global background. The Exercise Library
and detail pages apply the same navy surfaces through page-scoped token
overrides. The rules below still describe the remaining chrome pages.

Alternate between:

- Very dark background (`#05070B` / `#0A1018`)
- Slightly lighter surface (`#101826` / `#141F2D`)

Use soft radial cyan washes for atmosphere.

Section fills should stay translucent enough for the global animated background
to read through.

Do not use obvious separators.

Spacing and tone shifts should define sections.

---

# Footer

The footer layout, link columns, slogan image and copy are preserved. Since the
owner's 2026-11-10 revision its colours and type follow the public design
(logo navy with blue light, a thin cyan top line, cyan column headings,
Archivo). Do not change its layout or content unless explicitly requested.

---

# Navigation

The public navigation bar contains:

- Brand Logo (left)
- Distributed links: Services, Exercise Library, Contact, Feedback, Our Vision,
  Feed
- Right actions: Log in (outline pill) and Start free (solid cyan pill)

The six links occupy the flexible middle column and are spread evenly between
the brand and the auth actions, so the group stays balanced instead of leaving
a gap before the first link. At `1360px` and below the labels collapse and the
icon-only links remain centred; the auth actions stay right-aligned.

The logo always redirects to the Home page.
Log in routes to `/login`. Start free routes to `/register`.

Mobile navigation accessibility (no visual change): the 36px menu button has an
invisible 44px touch target, the collapsed panel uses `visibility: hidden` so
its links are not reachable with Tab while closed, and Escape closes the open
panel and returns focus to the menu button.

Navbar behaviour is defined in the Components section above and must be followed.

---

# Floating Actions

Depending on the user's role, floating action buttons may be displayed.

Examples include:

Customer Support

Available only for authenticated clients.

Notifications

Available for every authenticated user.

---

# Public Pages

## Home

Purpose

Present the platform by showing the product and encouraging users to begin.

Sections

1. Hero — the opening chapter of the Home story
2. Find your next program — right after the hero, so visitors reach the
   programs first (owner decision, 2026-11-11)
3. The other story chapters (Train, Fuel, Evolve, Get started; desktop
   Three.js, stacked on mobile)
4. Category overview (API-backed)
5. Purchase journey ("From discovery to your first session")
6. Contact preview
7. Footer (layout protected; colours follow the public design)

The Home route mounts `HomeStory` (which places the `ProgramsOverviewSection`
it receives right after the opening chapter), `CategoryOverview`,
`PurchaseJourney` and `ContactPreview` inside a `PageWrapper` carrying the
public website layer class. The Training, Nutrition
and Progress showcase components remain available as design building blocks but
are not mounted.

### Home story

`components/home_story/` tells RYZE in five chapters whose copy lives in
`story_content.ts`:

1. Identity — label "Training & nutrition platform", headline "Train
   anywhere. Progress everywhere." with "anywhere." and "everywhere." in cyan
   (owner revision 2026-11-10: the headline matches the globe), the line "Free
   training plans, premium programs and personalized coaching in one platform
   that goes wherever you train. Follow every session and track your progress
   from your phone or your computer.", CTAs "Start
   training free" → `/services/generic-program` and "Explore plans" →
   `/services`, and the three entry chips (Free programs, Premium plans,
   Personalized coaching).
2. Train — "Every session, already planned.": weeks and days, sets,
   reps, load, rest and tempo, the live session timer with set logging,
   workout history and progress.
3. Fuel — "Nutrition that fits the plan.": calorie and macro targets,
   meals with foods and portions, Premium Level 1 delivered after purchase,
   Premium Level 2 prepared by a RYZE trainer.
4. Evolve — "One platform, every screen.": catalogue search and
   filters, Exercise Library demonstrations, My Programs with history and
   progress.
5. Get started — "Your next level starts here." with the same two CTAs.

Every chapter sits on the same navy. In Train and Fuel, three cards lifted out
of the phone screen are set out on its left, one per feature point (week
schedule, set logging, rest timer; daily target, protein, a meal with foods).
Each card is joined to the screen element it shows (owner revision,
2026-11-11): the element is outlined on the phone screen, a rounded leader runs
from the phone's edge at that element's height to the card, and a light pulse
travels along it once drawn. Cards are ordered like their elements, so the
leaders never cross. Phones show two of them (set and rest timer; protein and
meal) overlapping the phone edges. In Evolve the laptop and phone stand side by
side under the badge "Synced · phone & laptop" (no connecting line), centred
over the pair with a soft breathing light, and the Earth leaves the frame. The laptop and the Evolve phone use large-type screens
designed for the size at which they appear. Product visuals are illustrative RYZE
screens designed in the app's style after the owner's phone reference, with
sample data and existing features only (`main/src/resources/img/showcase/`:
today's training session, the day's nutrition plan, the programme page), and
labelled "Illustrative RYZE screens with sample data."

Desktop (≥1024px wide, ≥680px tall, fine pointer, WebGL2, no reduced-motion
preference) pins one 3D stage behind the hero, the programs and a 500svh
chapter track, with the five story steps on its right edge (owner revision,
2026-11-11; the order hero → programs → chapters is the owner's and stays):

1. Hero — the story steps (01 RYZE, 02 Train, 03 Fuel, 04 Evolve, 05 Start)
   are visible and clickable; step 1 is current. As the hero scrolls away its
   copy fades and the Earth travels to the upper right corner — part of the
   journey, with no moment where the Earth stands alone.
2. Programs — they pass over the stage while the Earth waits in the corner
   (tucked a little further in, clear of the section header); the steps step
   aside (hidden and out of the tab order) while the programs fill the screen.
3. Chapters — the track starts while the programs are still leaving (its top
   at 55% of the viewport), so the phone flies in and the Train copy appears
   right behind them; the current step follows the chapter on screen and each
   step scrolls to its chapter. The stage keeps its height in the flow (the
   hero pulls itself up over it), so it is released and scrolls away exactly
   when the track ends, before the next section.

Scroll progress drives one Three.js scene (`story_scene.ts`, dynamically
imported):
the Earth with real continents and an atmosphere, RYZE routes leaving Lisbon
(light pulses travel each route) and the official icon on a badge above
Lisbon — the brand travelling the world (2026-11-14: land drawn as a glowing
dot matrix, twinkling city lights on the night side, a sun glint on the oceans,
thin procedural clouds and a brighter atmosphere; no orbit ring); a modelled phone (titanium frame,
glass, island, buttons, camera module, blue edge light) that flies in with the
training screen and spins, showing the logo on its back, to reveal the
nutrition screen; a laptop (black keys with backlit legends, Touch ID and arrow
keys, glass trackpad, opening scoop, hinge, display notch) that rises with the
logo on its closed lid and opens square-on to the viewer on the programme
screen (2026-11-14: the closed lid rests above the keys instead of cutting
through them; each key size has its own rounded cap, a cyan backlight shows
between the keys, speaker grilles flank the keyboard and the legends are
sharper), with the phone standing beside it; then the Earth returns for
the call to action. The stage backdrop stays in the navy family (only the
blue light moves) and fades into the page at its base, the chapter copy fades
in its scroll window (every chapter stays in the accessibility tree; keyboard
focus inside a faded chapter jumps it into view).
Device screens are sampled one mip level sharper than default to keep small
text crisp.
An SVG globe is shown until the scene is ready. Rendering runs only while the story is on screen and stops on
navigation; any scene failure switches to the stacked layout.

Mobile, tablet, reduced motion and browsers without WebGL2 get the stacked
layout: each chapter is a normal section with the SVG globe (identity: the same
continents and routes, animated pulses) or phone and laptop frames showing the
same screens. Three.js is never downloaded there.

### Programs

The program highlights are read from the published catalogue through
`useServiceCatalog`. While no program is published (empty catalogue or an
unavailable API), the section shows the original goal cards instead of an
empty state — Build Muscle → Premium Level 1, Lose Fat → Premium Level 2, Train
for Free → the training-plan marketplace — so Home always invites people in
(owner decision, 2026-11-10). With published programs, nothing is hardcoded: The section
shows at most three cards: one Premium Level 2 service first, then one Premium
Level 1 package, then published generic programs, then additional Premium
packages when needed. Each card uses the API name and description and links to
its own package; a loading state shows while the catalogue loads.

Presentation (owner revision, 2026-11-11): the Home's main call to action. Label
"Start training today", heading "Find your next program." and "See all plans" →
`/services`; tall full-photo cards (the current `advert1–3` images) with an icon,
the title, three lines of description and a primary CTA on every card; lift,
glow and photo zoom on hover. Only free programs carry a tag ("Free"): the
owner removed the Premium Level 1/2 tags because the goal cards and their
images do not describe the levels (Premium Level 1 is a hypertrophy training
plan with nutrition, not personalized; Premium Level 2 is personalized). Below
768px the cards form a swipeable row (keyboard tab stop only at those widths).
The section is transparent with a soft blue light behind the cards; on desktop
it sits between the two 3D stages, never over them.

### Category overview

API-backed through `useServiceCatalog` and `describeCatalogPrice`; hidden while
the catalogue is loading, failed or empty. Heading "Pick how much guidance you
want." One glass card per family with at least one published offer; more
personalization means more blue light: training plans (plain), Premium Level 1
(ring-blue glow) and Premium Level 2 (cyan edge and glow). Each shows a personalization meter (1–3 rising bars), the
category name, a one-line description and three "included" items summarising
`project.MD`, the catalogue-derived price and the existing CTA label and
destination. From 768px the cards stand in one row with rising heights.

### Purchase journey

"From discovery to your first session." with "See all training
plans" → `/services/generic-program` and "See how RYZE works" → `/our-vision`,
and five steps on a cyan rail for training plans and Premium Level 1: choose a
plan, check the details (level, duration, weekly frequency), activate (free)
or check out (paid), open My Programs, train and log. A note explains that
Premium Level 2 adds an intake after purchase and a trainer prepares the
programme and nutrition before they appear in My Programs.

### Contact preview

A light-blue wave panel on the navy page: "Questions, partnerships, or ready to
start?", the existing supporting line, CTAs "Join for Free" →
`/services/generic-program` and "Contact us" → `/contact`, and light channel
cards (Email, Instagram, WhatsApp) with their address or handle.

---

## Services

Displays every available product offered by the platform.

Catalog landing

- The public navy page with blue light behind the cards (the former
  `bg_services.png` background image is no longer used)
- Label "Services" and headline "Choose how you want to RYZE." in Archivo,
  "RYZE" in cyan; the headline wraps on phones so the cards fit the screen
- "RYZE" in the headline carries a light gradient (ice → cyan → ring blue)
- Four items in an app-style glass panel with dividers and cyan icon tiles,
  describing what the product does today (owner revision 2026-11-11 replaced
  the former "AI-powered programs that adapt to you" and "Smart analytics"
  claims): Planned for you (weeks and training days in order), Every detail
  set (sets, reps, load, rest and tempo per exercise), Log as you go (live
  session timer with set-by-set logging), See your progress (workout history
  after every session)
- A route guide in the login page's split panel (owner request, 2026-11-11:
  "apply the login look where we want attention"): on the navy half, "Which
  one is for you?" with three linked rows — Just training ("Ready-made
  training plans, including free ones." → `/services/generic-program`),
  Training and nutrition ("Premium Level 1: a hypertrophy programme with a
  nutrition plan, delivered right after purchase." →
  `/services/premium-level-1`), Built around you ("Premium Level 2: a RYZE
  trainer prepares your programme and nutrition plan after an intake." →
  `/services/premium-level-2`); on the light-blue wave half, "New to RYZE?",
  "Start free today.", a line inviting a free account and a first session, and
  "Create free account" → `/register` and "Log in" → `/login`. The halves
  stack on phones.
- One image-backed category card per available product family, using
  `rcc1.png` for the generic training-plan marketplace and `rcc2.png` for
  Premium Level 1 and Premium Level 2, each animating in with a staggered
  fade-up on page load
- A loading state while the catalogue loads; when no offer is published (empty
  catalogue or unavailable API) the original showcase cards are shown — Generic
  Program "FREE", Premium Level 1 "14,49€" and Premium Level 2 "19,49€" with
  `rcc1/2/3.png` — as before the catalogue became API-driven (owner decision,
  2026-11-10)

Each card contains:

- Full-bleed category background image with a dark gradient overlay for
  readability
- Compact right-aligned price derived from the published catalogue (`Free`,
  exact price, `From <cheapest>`, or `Free and paid`)
- Call To Action (primary on featured Premium Level 1, outline on generic)
- Featured Premium Level 1 card is taller, glows stronger and carries a
  floating "Featured" tag above its top edge

The generic card opens the training-plan marketplace, the Premium Level 1 card
opens the Premium package catalogue and the Premium Level 2 card opens the
trainer-delivered service page. With published offers, no product price or
offer copy is hardcoded and each card is rendered only when its family has at
least one published offer; the showcase prices above appear only while nothing
is published.

`ServiceCard` is used only by the Services page; the cards are protected and
unchanged except for the owner-added tags: purple "Personalized" on Premium
Level 2 and mint "Free" on the generic training plans (2026-11-14).

Showroom (`components/service_showroom/`, 2026-11-14, rebuilt the same day
after owner feedback): the cards stay flat and crisp at all times — no 3D
rotation at rest, no blur on the card in focus. With a precise pointer from
1024px each card stands on a real 3D pedestal drawn by a lazily loaded WebGL
stage behind the cards (`showroom_scene.ts`: slim two-tier navy pedestals with
a matte top, a cyan light ring on the top edge, a pool of light under the card,
a soft glow on the ground and a soft light behind the card; no floor, no
spotlights). The stage is placed from the cards' layout positions and drawn on
demand (arrival and focus transitions only, never a permanent loop). Under
each pedestal a placard states the route's goal in one look ("Just training —
Ready-made training plans, including free ones"; "Training and nutrition — A
programme with a nutrition plan, right after purchase"; "Built around you — A
RYZE trainer prepares your programme and nutrition"). The cards arrive from
the left, the depth and the right while their pedestals rise and light up.
Hovering or keyboard-focusing a card lifts and enlarges it from its base (2D
only), brightens its pedestal, steps the other cards back (dimmed, no blur) and
floats out four bubbles with what the route includes (generic: ready-made
plans, weeks and days in order, set-by-set logging, history and progress;
Premium Level 1: training with a nutrition plan, available right after
purchase, daily calorie and macro targets, hypertrophy programme; Premium
Level 2: complete intake, a RYZE trainer prepares the plan, training and
nutrition built around you, follow the request until delivery). Phones,
tablets and touch screens keep stacked cards (arriving from alternating
sides) with the goal placard and the facts as chips under each card. Reduced
motion removes the arrivals and drifting. Home no longer renders
`ServiceCard`; it uses the image-free category overview described under Home.

### Free

Leads into the Generic Training Plans Marketplace (see below) — the searchable,
filterable catalog of free/generic training plans.

### Generic Training Plans Marketplace

Reached from the free program CTA on the Services area
(`/services/generic-program`). A product-like, Lyfta-inspired browsing
experience built on the RYZE dark/cyan identity — not a promotional landing
page. The existing navbar is untouched; all marketplace content starts below it.

Layout hierarchy:

```
NAVBAR
SEARCH (sticky)                          N programs
TITLE + SUBTITLE
FILTER RESULTS COUNT (when filters active)

[FILTERS]  TRAINING PLANS
 (left)     Find a program built around your goals, schedule and experience.

            All Training Plans
            ← [ CARD ][ CARD ][ CARD ] →
               (click arrow → smooth fade-in, next group of 3)

            ← [ CARD ][ CARD ][ CARD ] →
               (click arrow → next group of 3)

            Recently Updated
            ← [ CARD ][ CARD ][ CARD ] →
            ...
            ── Showing up to 12 programs per section · N match ──
```

- **Featured coach programs** (v2.111.0) — when no filter is active and at
  least one trainer program is published, a highlighted glass zone at the top
  of the rows ("Featured coach programs", "The most viewed plans from RYZE
  coaches…") shows up to six trainer programs ranked by views, as the same
  program cards in the same carousel row. Under each card a pill with the
  coach's initials avatar and name links to the coach profile. The zone is
  independent of the filters and stays visible when the platform catalogue is
  empty (the "No training plans have been published yet." message is then
  hidden).
- **Coach profile** (`/coaches/:trainerId`, v2.111.0) — public page: back link
  to Training plans; a glass hero with a large initials avatar (cyan gradient,
  ring glow), "Verified RYZE coach" badge, the coach's name, "Coach since
  <month year>" and two stat pills (programs, views); then "Programs" with
  every published plan as program cards (2 columns on phones, 3 from 720px, 4
  from 1100px). Unknown or non-public coaches get "This coach has no public
  profile." Only the registered name is shown.
- **Program detail attribution** — a trainer-owned program shows a
  "Created by" pill (initials avatar + name) under its title, linking to the
  coach profile. Opening the page records one view (deduplicated by the API).
  The trainer's own profile page (`/trainer/profile`) has an "Open public
  profile" button.
- **Search bar** — sticky pill under the navbar. Stronger at the top of the
  page (more opaque, subtle cyan border + soft RYZE glow); becomes
  translucent/subtle after scrolling so it blends into the page. Contains a
  search icon (placeholder "Search training plans..."), a results count at the
  right side reflecting the current filter state (e.g. "N programs" with no
  filters, "0 programs" when a filter is active, hidden on mobile) and a
  `Filters` button on mobile only.
- **Live program search** — while the user types, a debounced query (220ms)
  resolves against the real public catalog (`services/program_search.ts`, backed
  by `fetchMarketplacePrograms` with a `q` term) and renders a dropdown panel
  below the pill. Each result row shows the plan name plus a small metadata
  line (Calendar icon + `N weeks`, Repeat icon + `M days/week`, plus the
  training-type · level category) and links to the program detail page. When the
  query matches nothing, a single "No results found" row is shown; while a
  query is pending, a "Searching..." row is shown; network failures resolve to
  an empty state. The dropdown closes on outside click or Escape and when the
  input is cleared.
- **Page heading** — sits outside the filter/content grid, immediately above the
  layout. Displays "Training plans" as a confident heading with a muted
  supporting line beneath it. A small filter-results indicator
  ("0 results for your filters — N filters active") appears inline below the
  heading whenever any filters are selected; it disappears when all filters are
  cleared. This keeps the heading visually separated from the first row with
  enough breathing room.
- **Filter sidebar (desktop)** — compact vertical filter panel, sticky beneath
  the navbar/search area, with its own independent vertical scroll
  (`max-height` capped to the viewport below the search bar, wheel scroll stays
  inside the panel and does not drag the whole page). Plain square checkboxes
  that fill in RYZE cyan when selected. Groups: Training Type (Hypertrophy,
  Strength, HYROX, CrossFit, Fat Loss, At Home), Training Frequency
  (1–7 days / week), Level (Beginner, Intermediate, Advanced), Plan Duration
  (1–2 / 3–4 / 5–8 / 9–12 weeks). A small "Clear all" pill at the bottom resets
  the selection. Typography is small and compact (uppercase muted group
  headings, thin dividers); no boxes around option rows.
- **Collapsible desktop filters** — the sidebar has two states. EXPANDED shows
  the "Filters" title, a collapse control and all filter groups. COLLAPSED
  reduces it to a slim vertical "Filters" control (rotated label + icon) that
  re-expands on click. The program area grows to fill the freed space, and the
  grid column transitions smoothly. On tablet widths the sidebar starts
  collapsed by default; on mobile it is replaced by the drawer.
- **Program area** — vertical list of `ProgramRow` sections flowing naturally
  beneath the page heading and filter-results indicator. Each row is
  independently stateful. After the last row, a footer displays a divider and a
  small centered line stating that each section shows up to 12 programs and how
  many programs match the active filters. The marketplace never claims the
  whole catalog was exhausted when a row is intentionally bounded.
- **Program rows** — static page-based carousels. Each row shows EXACTLY three
  large cards side by side with a circular arrow centered vertically on each
  side (← back, forward →). Cards are NOT scrollable: no horizontal track, no
  drag, no `scrollBy`. Browsing works purely by page index — each row keeps its
  own `currentPage` state, and arrows swap the visible set via `slice`. The CSS
  grid is `grid-template-columns: auto 1fr auto` where the left/right arrow
  columns are `auto` with `align-self: center` (arrows vertically centered
  against the card column), and the center card column is `1fr` containing
  three equal slots in a sub-grid (`repeat(3, minmax(0, 1fr))`). When the page
  changes, a direction-aware slide animation plays: the current cards slide out
  (forward → exit left, back → exit right), then the new set slides in from
  the opposite edge. This two-phase animation uses four CSS keyframes
  (`slideInForward`, `slideOutForward`, `slideInBack`, `slideOutBack`) at 400ms
  eased, giving a literal swipe effect. Both arrows are disabled at the
  boundaries (back at page 0, forward at the last page). Each row is
  independent: advancing "All Training Plans" never moves any other row. Rows
  are backed by the real published catalog: All Training Plans → default
  catalog order, Recently Updated → `updated_at` descending, New & Noteworthy →
  `created_at` descending, then category rows: Build Strength → `Strength`,
  Build Muscle → `Hypertrophy`, Fat Loss → `Fat Loss`, Train at Home → `At Home`,
  Performance & Conditioning → `HYROX` + `CrossFit`. Every row is deterministic:
  discovery rows fall back to the same catalog order when the catalog is empty.
  No row title or description implies sales, popularity or trend data that RYZE
  does not collect. Row title is large and
  confident, with an optional small muted supporting line for discovery rows.
- **Program cards** — large portrait covers (4:5 image area) that visually
  dominate the page: dark placeholder with a subtle centered RYZE logo and
  faint cyan lighting until real imagery exists, program title below
  (prominent, readable), then a metadata row showing the real schedule
(Calendar icon + `N weeks`, Repeat icon + `M days/week`) and the category
   (`training_type · level`, falling back to the audience level when the training
   type is unset). A price line communicates the current sale price: "Free" in
   cyan for free programs, the EUR amount in prominent type for paid ones
   (formatted from integer minor units). No downloads, no ratings and no CTAs;
   the only badge is the decorative "Free" on a free program's cover. Each card links to the program detail page. Hover: gentle lift, subtle image
   scale, brighter border and faint cyan shadow — restrained, not glowy. Cards
   are sized purely by the CSS grid (no JavaScript measurement); all cards in a
   row share an equal height. Inside an active Test Mode session every
   purchasable premium card is forced to render as "Free" (the price is zeroed
   for display; the underlying program data is never mutated). For signed-in
   clients the marketplace resolves ownership on mount (fire-and-forget
   `GET /me/entitlements`): an owned program swaps its price line for a cyan
   accent "Owned" pill and its card links straight to
   `/services/my-programs/:programId`, bypassing the detail page. A guest or an
   unavailable entitlements call simply renders no owned state.
- **No horizontal overflow** — the page must NEVER scroll horizontally. Card
  columns use `minmax(0, 1fr)`, there is no scrollable track and no fixed card
  width; `overflow-x: clip` is applied defensively at the application level
  (`body`, `#root`) and on the marketplace section, but the layout itself
  genuinely fits the container.
- **Vertical flow** — the page flows naturally from navbar → search → header →
  rows → footer, all in normal document flow. The marketplace section has no
  forced viewport height, so the document ends exactly after the footer with no
  empty scroll space below it.
- **Infinity scroll concept** — the marketplace is structured as endless
  vertical page scrolling; no numbered pagination and no load-more button.
- **Colour (owner request, 2026-11-11: "more life, same layout")** — the
  public design's colours through page-scoped tokens: logo navy with blue
  light around the page, glass filter panel and arrows, a navy search pill
  with a cyan edge, cyan filter headings and checkboxes, the "Training plans."
  title and row titles in Archivo (row titles with a short cyan light bar),
  and a cyan "Free" price. Card covers are brand gradients (a cyan horizon
  under a blue sky, in four tones chosen from the program id) with a glowing
  logo and, for free programs, a cyan "Free" badge on the cover. The layout,
  rows, carousels and behaviour are unchanged.
- **Responsive** — desktop: three large cards per row with arrows on both sides,
  vertically centered against the card column, beside the sticky, scrollable,
  collapsible filter rail in the wide (1500px) program area. Tablet: sidebar
  starts collapsed so three large cards are preserved. Mobile: "Filters" button
  opens the existing slide-in drawer; a single large card at a time is shown
  next to the forward arrow so covers never become tiny; the carousel mechanism
  (button-controlled page swap with direction-aware slide animation) is
  identical at every breakpoint and the page never scrolls horizontally. All
  respects `prefers-reduced-motion`.

Components: `SearchBar`, `FilterSidebar`, `ProgramRow`, `ProgramCard`,
`CarouselArrow` under `components/`, the search resolver in
`services/program_search.ts`, the API client in `services/marketplace_api.ts`,
row/filter data in `constants/marketplace.ts`.
The marketplace uses the shared `Container` in its `wide` sizing.
The page sends the active filters, sort and a 12-program limit to the public
catalog for every row (`scope=generic`). Filtering, sorting and pagination are
server-side; the client only intersects a category row's training types with the
selected types. The total comes from the unrestricted first row, so the result
count is never derived from the bounded cards on screen. Home, Services and the
Premium catalogue page through the same API until every page is loaded, because
their aggregate price/highlight logic needs the complete returned set.

### Generic Program Detail

Reached from any marketplace card or search result
(`/services/generic-program/:programId`). A focused, content-first page showing
one published generic program. The public page is metadata-only: the programme
structure is never rendered here. Free programmes use an activation panel; paid
programmes expose a single in-page purchase panel with no separate checkout
page. The public metadata always remains viewable, but activation and payment
require authentication.

- **Top bar** — a back Button (arrow in a dark pill) that returns to the
  marketplace (`/services/generic-program`).
- **Header** — the program name as the page title, the description as a muted
  supporting line, and a set of soft cyan chips: the current sale price
  (prominent chip for paid programs, muted "Free" chip for free ones),
  `training_type`, `level`, `N weeks` and `M days/week` (each chip is hidden
  when its value is absent). Inside an active Test Mode session a paid program
  shows an amber "Free" price chip plus a small "Test Mode" chip, signalling
  that its price is zeroed for the session.
- **Purchase panel** (paid programs only) — one card below the header whose
  content depends on the checkout state:
  - `checking` — brief spinner while the page resolves ownership.
  - `guest` — price, "Sign in to purchase this training plan." and a "Sign in"
    button linking to `/login`. Buyable-elsewhere styling, deliberately not
    locked: the program content remains readable.
  - `owned` — "You have access" success treatment ("You already own this
    training plan") with a primary "Open program" button into Program Access
    and a back Button to the marketplace.
  - `ready` — the price and a primary "Buy now" CTA. When the backend advertises
    more than one payment method (`GET /api/v1/payments/methods`), a small
    radio list ("Payment method") sits between the note and the CTA with the
    available methods (Card, MB WAY, PayPal — whichever the backend advertises,
    in the order it returns them), the first one pre-selected; the note reads
    "One-time purchase. Secure checkout." rather than naming a provider. When
    exactly one method is available the selector is omitted and the method is
    simply used. When no provider is configured the panel shows an amber
    "Checkout unavailable" note ("No payment provider is currently configured.
    Please try again later.") with no buy action. The selector is fully
    data-driven: adding or removing a provider method requires no frontend
    change, and no method is hard-coded in the UI.
  - `creating` / `negotiating` — spinner states while the purchase is created
    and the provider approval URL is being prepared.
  - `redirecting` — spinner while the browser is sent to the provider.
  - `processing` — spinner shown on return; the page calls the capture endpoint
    with the order token, waits, and only then shows success.
  - `success` — success treatment ("Purchase complete · you now have access")
    with a primary "Open program" button and a "My Programs" link.
  - `cancelled` — muted note ("Payment cancelled"), the panel returns to
    `ready`.
  - `error` — muted error with a Retry button; a cancelled order and a failed
    capture (purchase still pending) are presented as retryable, never as paid.
    This is the expected state when a wallet method (MB WAY) is approved in its
    app and Stripe has not reported the payment yet: the retry re-runs the
    server-side verification, and the verified webhook completes the purchase
    regardless of what the browser saw.
  - Active Test Mode variant — for a paid program, the panel swaps to an amber
    "Get plan in Test Mode" CTA at price "Free" with a note that the purchase
    completes instantly with no payment. A `409 DUPLICATE_ENTITLEMENT` resolves
    to the `owned` state (the persona already owns the program).
- **Free activation panel** (free programmes only) — resolves the client's Free
  activations. A guest sees "Sign in to activate" with a Sign in button; a
  signed-in unactivated client sees "Activate free plan"; an activated client
  sees "Open program". Activation navigates to Program Access on success. A
  `401` during activation redirects to `/login`. The panel never renders
  structure.
- **Locked structure** — one explanatory section states that weeks, days,
  exercises, prescriptions, set logging, rest timers and progress unlock after
  activation or purchase. It contains no public workout data.
- **States** — loading (muted centered state), generic error with a Retry
  button, and "Program not found" for unknown, draft or removed programs (the
  backend returns one indistinguishable 404 for all of them).
- **Responsive** — the header and panels scale down gracefully;
  `prefers-reduced-motion` is respected.

The panel resolves ownership by fetching the user's entitlements (401 → guest
state); return URLs carry `purchase_id` + `token` (capture) and `status`
(optional cancelled flag). Success is never inferred from the redirect alone —
the panel only shows "success" after the server-verified capture responds. The
selected payment method is sent at initiation and is never re-asked on return:
the backend resolved the provider and recorded the method when the checkout
started, so the browser return only triggers the capture.

Non-goals for this stage: reviews, social features, trainer attribution or
related-program recommendations.

---

### My Programs

The authenticated client-facing listing of every program the client can enter:
entitled programs, trainer assignments and activated Free programs. It follows
the palette, motion and typography of the other authenticated pages.

- **Route** — `/services/my-programs` ("Train" in the member navigation),
  rendered inside the member app shell (see "Member Area" below) without the
  public chrome. The global navigation is never swapped for an authenticated
  state outside of it.
- **Data** — `GET /api/v1/me/entitlements` plus `GET /api/v1/me/free-programs`,
  401 → redirect to `/login`. A Free activation is merged into the same list and
  marked with a distinct "Free" chip; the card never claims purchase metadata.
- **States** — loading (spinner), generic error with a Retry button, and an
  empty state ("You have no programs yet") linking to the marketplace.
- **Cards** — one card per accessible program: name, description and the generic
  calendar chips (`training_type`, `level`, `duration_weeks`,
  `frequency_per_week`), plus an "Open program" affordance navigating to
  `/services/my-programs/:programId`. No price and no entitlement metadata are
  shown.
- **Entry point** — "Train" in the member navigation, the home dashboard's
  current plan and the "Start workout" action card.

---

### Program Access

The access-protected program detail ("client program detail"). It renders the
safe structure for any program the client may enter through an entitlement,
trainer assignment or Free activation. It is only reachable from My Programs.

- **Route** — `/services/my-programs/:programId`, inside the member app shell.
  From 1100px the plan structure and a sticky side column (session banner,
  shortcuts) sit side by side.
- **Data** — `GET /api/v1/me/programs/:programId`; 401 → redirect to
  `/login`.
- **States** — loading (spinner), generic error with a Retry button, and a
  generic "no longer available" state for the indistinguishable 404.
- **Layout** — a "My Programs" back Button, the program heading with name,
  description and the calendar chips, then the training structure (weeks → days
  → exercises → prescription set table). There is no purchase surface here.
- **Actions** — an active session banner offers Resume. Every workout row has a
  Start action (or Resume when that workout owns the active session); starting a
  different workout while one is live surfaces the server conflict instead of
  discarding the live session. History and Progress links are available from
  this page.
- **Shared structure** — `ProgramStructure` is access-only and is not reused by
  the public marketplace detail, which is metadata-only.

---

### Workout Session

The interactive execution surface for one prescribed workout.

- **Route** — `/services/workout-sessions/:sessionId`, a full-screen player
  outside the member shell (no navigation while training). From 1100px a sticky
  overview column shows the progress ring, the elapsed clock, the exercise list
  with jump links to each exercise block, and Finish.
- **Data** — `GET /api/v1/me/workout-sessions/:sessionId`; 401 → redirect to
  `/login`.
- **Layout** — programme name, week/day position, elapsed time, completed-set
  progress, then one block per exercise. Each set row pairs the immutable
  prescription (reps, weight, RIR/RPE, tempo) with editable actual reps,
  weight, RIR/RPE and a complete/skip control. Completed and skipped states are
  visually distinct; disabled states are never implied by colour alone.
- **Rest timer** — when a completed set returns `rest_ends_at`, the UI computes
  the remaining duration from that absolute server timestamp. It never
  decrements a local countdown, so backgrounding the tab cannot drift.
- **Controls** — pause/resume, complete and abandon. Complete is the primary
  action; abandon is destructive and asks for confirmation. A finished session
  shows a read-only summary and the duration is frozen.
- **Errors** — a foreign/unknown session shows the generic not-found state; a
  transition conflict instructs the client to refresh rather than retry blindly.

---

### Workout History and Progress

- **Routes** — `/services/workout-history` and `/services/progress`, inside
  the member app shell ("History" and "Progress" in its navigation). History
  lists sessions in two columns from 1100px.
- **History data** — `GET /api/v1/me/workout-sessions?limit=50`; 401 →
  redirect to `/login`. Cards show programme, week/day, date, a correct status
  badge (`In progress`, `Paused`, `Completed`, `Abandoned`), set progress,
  duration and skipped sets. Details opens the session page.
- **Progress data** — `GET /api/v1/me/statistics`; 401 → redirect to `/login`.
  Shows whether a current programme exists, its completion percentage, unique
  workouts completed versus total, last workout date and all-time completed
  sessions, abandoned sessions, completed/skipped sets and active time.
- **States** — loading, generic retryable error and an empty state that links
  back to My Programs.

---

### Purchase History

The authenticated client-facing order history. It continues the account area
palette and typography of the member area.

- **Route** — `/account/purchases`, inside the member app shell ("Purchases"
  in its secondary navigation).
- **Data** — `GET /api/v1/me/purchases`, 401 → redirect to `/login`. The
  endpoint returns a safe history DTO (never commission or payout fields) with
  a nested program summary plus server-computed `test` and `access` flags.
- **Layout** — a confident heading ("Purchase History") with a muted supporting
  line, then a vertical list of entry cards, newest first. Each entry shows the
  program name (with its description when present), a Calendar icon + purchase
  date, the amount formatted from minor units (always currency-formatted, so a
  zero-price Test purchase reads as `€0.00` rather than "Free"), and right-side
  metadata:
  - status pill — green "Completed", amber "Pending" or red "Failed"
    (the only statuses the backend emits);
  - a neutral pill naming the payment method used (Card / MB WAY / PayPal) for
    entries
    that carry a `payment_method`; omitted for Test purchases, which never reach
    a payment provider, and for pending purchases created before payment
    initiation;
  - a cyan "Test purchase" pill for Test Mode entries;
  - a primary "Open program" button linking to
    `/services/my-programs/:programId` only when `status === completed` and
    `access === true`;
  - a muted "No longer available" note for completed purchases whose program
    was retired or deleted (history persists, access is not).
- **States** — loading (spinner), generic error with a Retry button, empty
  state ("You have not purchased any programs yet") linking to the marketplace.
- **Responsive** — entries stack vertically on narrow screens with the
  metadata moving under the program text.

---

### Premium Level 1 and Premium Level 2

The public Premium Level 1 page is documented under "Premium Level 1 Page" below.
The Premium Level 2 page is documented under "Premium Level 2 Page" below. Each
page reads only its own product family; neither embeds the other family or a
future Elite counterpart.

---

## Exercise Library

Glass style (owner request 2026-11-14, layout unchanged): the page sits on
fixed blue/cyan light; the search and filter toolbar is frosted glass with a
top highlight and inset fields; exercise cards are translucent glass with a
brighter top edge, a reflection over the thumbnail and a cyan-tinted body;
hover lifts the card, lights its edge and sweeps a sheen across it; category
tags are glass pills. The detail page uses translucent surface tokens on the
same light. Filter labels are translated in Portuguese.

The public, canonical catalogue of platform exercises. It is reachable from the
public navbar and is not part of the purchase flow: every entry is readable
without an account, and the catalogue is identical for every visitor.

Routes: `/exercises` (list) and `/exercises/:exerciseId` (detail). The navbar
link sits between Services and Contact.

List page

- Heading "Every movement, one catalogue" with a short explanatory line.
- Search field (debounced 250 ms) that targets the server search endpoint and
  a clear control when a query is present.
- Five server-side filters — Body part, Muscle, Equipment, Difficulty and
  Category — plus a "Clear N filters" control when any filter or the query is
  active.
- A polite result bar (`aria-live`) that reports the visible range and total,
  or that no exercise matches.
- Responsive card grid loaded in batches of 10 with infinite scroll: an
  intersection sentinel requests the next batch as the visitor approaches the
  end, appends it without duplicates, and reports loading/retry/end states.
  Cards use only the static `image_url` (never the animated GIF or video) and
  fill their square media area with `object-fit: cover`, so the image never
  leaves empty space inside the card; the 180×180 source is therefore upscaled
  on wider cards. The attribution caption sits over the bottom of the image,
  and the grid drops to two columns on small phones. Each card also shows the
  exercise name, the meta line (primary muscle · equipment · difficulty) and
  the body-part tag.
- States: loading, error with Retry, and an empty state whose primary action
  clears the search and filters. States never fabricate catalogue content.

Detail page

- An "Exercise Library" back link and the exercise name as the page heading.
- Reuses the shared exercise preview body in a split layout: the description,
  metadata and instructions stay on the left, while the media sits in a
  dedicated right-hand column. The card is capped at `66rem` so the two columns
  stay proportional. Below `860px` the preview stacks with the media first.
- Only the detail page plays the animated demonstration (`animation_url`), in
  that right-hand column with a maximum width of `18rem`; the 180×180 source is
  upscaled to that width rather than cropped. The same preview body keeps its
  stacked layout in the library dialog and the administrative editor.
- 404 (`EXERCISE_NOT_FOUND`) renders an honest "does not exist or is no longer
  available" state with a link back to the library; other failures offer Retry.
- Opens scrolled to the top.

Media behaviour

- Media is only rendered while `media_available` is true; otherwise the shared
  media placeholder is shown, never a broken asset or invented demonstration.
- The animated demonstration (`animation_url`) plays only on the detail page;
  videos are only rendered when the stored reference is an actual video file.
  Media is never labelled as video or 3D when it is not.
- `media_attribution` is displayed wherever media is shown.
- Lazy loading and an unavailable/error fallback keep the page usable when
  media delivery is disabled server-side.

3D demonstrations

- Entries carrying a supported `three_d_key` show an "Open 3D view" control
  inside the detail page. The viewer is a separate, lazily imported Three.js
  chunk, so the library loads and works without it.
- The viewer renders a procedural animated demonstration with orbit drag, wheel
  zoom, play/pause and 0.5× / 1× / 1.5× playback. It never replaces the
  catalogue entry, instructions or media; 3D is additive.
- Without WebGL the page falls back to an honest unavailable message instead of
  a blank canvas.

---

## Feedback

Acts as the platform's social / complaints area.

Digital Complaints Book and related feedback flows live here.

Member feedback (migration 000036): below the stories, a "We would love to
hear your feedback!" card with "Leave feedback". Opening it checks the account
(only then, so anonymous visitors make no authenticated request): signed out →
"Log in" / "Create account"; no plan → "Feedback opens once you have a plan.
Free training plans count too." with "Find a plan"; a submission awaiting
review → a thank-you note; otherwise a 1–5 star picker (radio inputs, keyboard
accessible) and a message box (10–1000 characters, counter, "Shown with your
first name and last initial."). After sending: "Thank you! Your feedback will
appear here after a quick review." Beside it (stacked on phones), "Other
feedback from our athletes" lists approved feedback (initial avatar, public
name, date, stars, message) with "Show more", and an empty state inviting the
first feedback. The Management Administrator reviews submissions on
`/admin2/feedback` (Pending / Approved / Rejected tabs, Approve and Reject).

---

## Feed

Public activity feed for RYZE community updates.

Currently routes to an under-development placeholder (`/feed`). The shared
`UnderDevelopmentPage` placeholder (also used by the `/services/:serviceSlug`
fallback and the catch-all route) is a navy section with a cyan status label,
heading, description, "Back Home" → `/`, "View Services" → `/services` and the
SVG globe.

---

## Contact

A navy page on the public website layer (no background image), in the login
page's look (owner request, 2026-11-11: "more life"): one split card. The
light-blue wave half carries the label "Contact", the display heading "Talk to
RYZE." ("RYZE." in royal), the supporting description and light channel cards
(Email, Instagram, WhatsApp) with an icon tile, the address or handle and an
outward arrow; external cards announce that they open in a new tab. The navy
half carries the form. On phones the bright half comes first.

- Email opens the mail client
- Instagram opens the official profile
- WhatsApp opens the official community invite

Form column: the navy half of the card, with dark fields, titled `Write to RYZE` with Name, Email, Subject,
Message and Attachment (optional, hint "PNG, JPEG, WebP or PDF, up to 1 MB.");
primary CTA `Send Message`; helper text `We usually reply within 24 hours.`.
Input borders meet 3:1 non-text contrast; the hidden file input is named
"Attachment (optional) Choose file" and shows its focus on the "Choose file"
label; the `role="status"` result message stays mounted (visually hidden while
empty) so success and error text is announced.

There is no modal, popup or background blur overlay for contact.

The form submits multipart data to `POST /api/v1/contact`. The backend owns
validation and delivery; the client only pre-checks required fields and the
1 MiB attachment limit. Success is shown only when the API confirms the SMTP
server accepted the message. A missing SMTP configuration or delivery failure
shows the server's unavailable message instead of a false success. The
attachment is never stored by RYZE.

The Home page must not duplicate the full Contact page. It uses a compact
preview with short copy, three channel buttons and two CTAs: `Join for Free` →
`/services/generic-program` and `Contact us` → `/contact`.

---

## Our Vision

Route: `/our-vision`

Purpose: sell the RYZE platform vision — not founder biographies.

Legacy `/about-us` redirects to `/our-vision`. No background images.

Sections:

1. Hero — "Thousands of fitness apps. All the same. RYZE isn't." ("RYZE
   isn't." in cyan); lead "Most fitness platforms sell the same static plans
   and stop there. RYZE brings training, nutrition and real trainers into one
   platform — and keeps evolving, with AI-generated programs next on our
   roadmap."; "On this page" links (`#why-ryze`, `#how-it-works`,
   `#vision-roadmap`, `#vision-evolution`); and a wall of identical greyed-out
   "12-week plan" apps with RYZE in the middle as a light-blue wave tile (the
   login page's bright panel, its icon on a navy tile), listing training and
   nutrition in one plan, plans prepared by real trainers, every set logged and
   every week tracked, and AI-generated programs marked "Next" (owner revision,
   2026-11-11; AI generation is on the roadmap, not live — see
   `docs/INSTRUCTIONS/project.MD`).
   From 768px with WebGL and full motion (owner request 2026-11-13, "more WOW,
   maybe 3D"), the wall becomes a Three.js stage (`vision_hero_visual.tsx` +
   lazily imported `vision_scene.ts`): an endless tunnel of the same greyed-out
   "12-week plan" cards streams past the camera (one instanced mesh, speed dust,
   fog) while the RYZE card stays still in front, circled by two cyan rings
   with travelling sparks and a pulsing halo; the card floats and tilts with
   the mouse like a hologram (glare follows the pointer) and the camera
   parallaxes. The scene pauses off screen and in hidden tabs, caps the pixel
   ratio, and falls back to the flat wall on context loss. Phones, reduced
   motion and no-WebGL keep the flat wall, which now has the tiles lighting up
   in sequence and a glint crossing the RYZE card (CSS only; reduced motion
   stops both). Phones never download Three.js.
1b. Why RYZE (`#why-ryze`) — "Others stop at a plan. RYZE keeps going.": a
   dashed "Typical fitness apps" column (one template for everyone; training
   or nutrition, rarely both; a plan you follow on your own; no record of what
   you actually did) against a RYZE column (programs filtered by goal, level,
   duration and weekly frequency; training and nutrition together in Premium
   plans; Premium Level 2 plans prepared by a RYZE trainer; every session
   logged with history and progress; AI-generated programs, "Next"). The RYZE
   column has a cyan top edge, a cyan glow and filled cyan check marks.
2. How RYZE works — "Start your journey in minutes. Transform for
   life." and five numbered glass step cards with gradient numbers.
3. Roadmap — "Where RYZE is going." with the milestones grouped by
   phase (Today, Next, Soon, Later, Horizon) on a rail; Today is highlighted
   and sends out a slow ring of light (static with reduced motion).
   Milestones are not links.
4. Continuous evolution — "Software is never finished.", "See what's
   next" → `#vision-roadmap`, six glass capability cards with cyan icon tiles,
   and a light-blue wave panel with the official `ryze_icon2.png` mark on a navy tile, "The
   future of fitness is adaptive." and "Join the journey" → `#vision-roadmap`.
5. Join — "Grow together with RYZE, join us now.", lead, "Join RYZE" →
   the official WhatsApp community (note "Opens the RYZE community on
   WhatsApp."), five community cards and the statement "RYZE is more than a
   platform. It's a movement."

Do not use long paragraphs, huge founder photos, or traditional “About Us”
brochure layouts. Founder/people content is deferred for a later release.

---

## Login / Register

Routes: `/login` (login), `/register` (register).

Purpose: authenticate existing accounts and create new ones. Both forms send
data to the backend (`POST /api/v1/auth/login` and
`POST /api/v1/auth/register`). Login stores the `ryze_access_token` HttpOnly
cookie in the browser; no JWT is ever handled in JavaScript. On success both
forms redirect to `/profile`, the authenticated profile page. Backend and
network errors are shown inside the card as a compact inline alert above the
submit button.

Single page component (`LoginPage`) rendering both forms — LOGIN and CREATE
ACCOUNT — in one card on the public navy page (no background image since the
owner's 2026-11-10 revision). The fields, validation messages, API calls and
redirects are unchanged.

The auth page is rendered **without** the site chrome: no navbar, no footer
and no animated background. The `Layout` receives `showChrome={false}` for
`/login` and `/register`.

Page structure

- The navy page with soft blue light; small RYZE logo (brand mark + wordmark)
  in the top-left corner, linking home.
- A discreet legal line below the card:
  `© 2026 RYZE · Privacy Policy · Terms of Service`.

Card (from 900px)

- Two halves: one form and the light-blue wave panel (official icon on a navy
  tile, an invitation to the other mode and an ink button). Login mode: login
  form left, panel right ("New to RYZE? Start your evolution." → "Create
  account"). Register mode: panel left ("Already a member? Welcome back." →
  "Log in"), register form right.
- Switching slides the panel across (~0.75s ease), the outgoing form fades
  and shifts, the incoming form takes the other half, and the panel copy
  crossfades. The in-form switch link is hidden at these widths.

Phones and tablets (below 900px)

- A compact light-blue wave header with the icon above the active form; the
  inactive form is not rendered visibly. The form's own switch link changes
  mode and the new form fades up.

Forms

- Login: "Welcome back." (continuation in cyan), "Log in to continue your
  journey.", Email or username, Password with show/hide, "Remember me" and
  "Forgot your password?", primary "Log in", divider "or continue with",
  Google button, "Don't have an account? Create one here".
- Register: "Create account.", "Join RYZE and start your evolution.", First
  and Last name (side by side from 420px), Email, Password and Confirm
  password with show/hide, the terms checkbox with Terms of Service and
  Privacy Policy links, primary "Create account", the Google button and
  "Already have an account? Log in".
- Fields: dark fill, `#4D6FA6` border (≈3.5:1), icon, cyan focus ring; errors in
  a compact inline alert above the submit button.

Accessibility

- The inactive form and the inactive panel copy are `inert` and hidden from
  assistive technology; after a switch, focus moves to the new form's heading.
- The previous one-second loading overlay between modes is gone; with
  `prefers-reduced-motion` the switch is immediate.

---

## Member Area

Owner request of 2026-11-14: the reserved user area is an app for both
desktop and phone, in the colours of the login and registration page, with
clear information about everything the member does. ADMIN_1 and ADMIN_2 are
not part of it.

### App shell

`components/app_shell/app_shell.tsx` (`AppShell`) is the shared layout of the
member and trainer areas.

- **Palette** — tokens scoped to `.shell` (never global): the logo navy page,
  royal glass panels, cyan as the light and the light-blue wave gradient
  (`--app-light`) reserved for the main action of a page; display headings use
  the RYZE display font.
- **Desktop (above 1180px)** — a 17rem sidebar: brand, primary navigation,
  secondary navigation, an action card in the light gradient and an optional
  switch to the other area. Content uses the full remaining width.
- **Tablet (768–1180px)** — the sidebar collapses to an icon rail with
  accessible labels.
- **Phone (767px and below)** — a top bar and a bottom tab bar with five tabs;
  the content stacks in one column.
- **Top bar** — PT | EN toggle, notification bell and the account avatar
  (linking to Account).
- **Loading** — `AppLoading`, a shimmer skeleton inside the shell; reduced
  motion removes the shimmer.

`components/app_shell/member_shell.tsx` (`MemberShell`) loads
`GET /api/v1/me` once for every member page (401 → `/login`) and shares it
through `useMember()` (`user`, `isTrainer`, `reload`). Navigation: Home, Train,
Progress, History; secondary: Training plans, Exercises, Purchases, Support,
Account. Phone tabs: Home, Train, Progress, History, Account. The action card
is "Start workout" (to Train); approved trainers also see a switch to the
trainer area. `routes/member_routes.tsx` is loaded lazily and keeps one shell
mounted while moving between member pages. The workout player and the
Premium Level 2 request page render outside the shell.

### Home (`/profile`)

The member's dashboard. Every value comes from existing endpoints
(`/me`, entitlements and Free activations, the plan's access detail,
`/me/workout-sessions` and `/me/statistics`); nothing is invented, and empty
data shows an honest empty state.

- **Header** — date, greeting and the current streak.
- **Next workout** (light wave panel) — the next workout of the current plan
  with Start (creates the session and opens the player), Continue when a
  session is live, a "plan complete" state, or a no-plan state linking to the
  training plans.
- **This week** — a day strip with the trained days and the week's sessions.
- **Totals** — completed workouts, training time, total volume and completed
  sets.
- **Activity** — sessions per week over the last 8 weeks.
- **Current plan** — completion ring and per-week bars, linking to the plan.
- **Recent workouts** and **personal bests** (the best completed set per
  exercise), plus shortcuts.
- **Layout** — a twelve-column grid on desktop, stacked below 1024px.
- An email-verification notice appears when the email is not verified.

### Account (`/account/settings`)

Profile details, language preference, sign-in methods, email verification,
password change, trainer status (apply or open the trainer area), sign-out and
account deletion. These panels moved from the old profile page with their
behaviour, API calls and confirmations unchanged. Two columns on desktop, one
on phones.

Logout calls `POST /api/v1/auth/logout` and redirects to `/login`. The JWT is
never read from JavaScript; the HttpOnly cookie is managed by the browser.

---

## Feedback page hero

Route: `/feedback`. No background image.

- Header: label "Feedback", "Real people. Real results." with "Real results."
  in cyan and the existing description, centred from 768px. The slogan image
  beside the headline and the result chips were removed (owner revisions).
  Life on load (owner request 2026-11-13, CSS only, no image): three brand
  lights drift slowly behind the header; label, title lines and lead rise in
  sequence; a light sweep crosses "Real results." every few seconds.
- Highlights band between header and stories: an endless, edge-faded ticker
  of word-only phrases from the stories ("A whole new level of confidence",
  "So much more energy", "Training with purpose"… — no figures), hidden from
  assistive technology because the full stories follow.
- Stories: app-style glass cards on the same navy, below the header (no
  overlap). Below 768px they form a swipeable, scroll-snapped row (keyboard tab stop only
  at these widths, inset focus ring, "Swipe to read more stories." hint); from
  768px a three-column grid. Each card: photo at 4:5 (lazy, intrinsic size),
  five stars ("Rated 5 out of 5 stars"), the quote in a `blockquote` and
  "Anonymous". Around the cards: a breathing cyan light band behind the row,
  a cyan light travelling round each card's edge (out of step per card), a
  staggered entrance, and on hover a lift with a cyan glow and a slight photo
  zoom. Each photo is a before | after split and carries "Before" / "After"
  labels on its halves. Reduced motion stops every animation.
- CTA (light-blue wave panel): "Your story starts with one session." and
  "Start your transformation" → `/register`.

Card copy (from the supplied assets `main/src/resources/img/feedbacks/f1.jpeg`,
`f2.jpeg`, `f3.jpeg`):

- "Over 16 weeks, I gained 12kg and a whole new level of confidence. The RYZE
  training and nutrition plan completely changed the way I approach fitness."
- "I lost 9kg and feel so much more energetic! I learned how to train with
  purpose and eat better. I feel stronger both inside and out."
- "RYZE helped me rebuild my confidence and feel better in my own body. The
  results came consistently and in a healthy, sustainable way."

---

# Trainer Area

Routes: `/trainer/apply`, `/trainer/dashboard`, `/trainer/clients`,
`/trainer/clients/:userID`, `/trainer/programs`, `/trainer/programs/:programId`,
`/trainer/premium-level-2`, `/trainer/premium-level-2/:requestID`,
`/trainer/profile`.

Since 2026-11-14 the trainer area uses the same app shell as the member area
(`AppShell`, see "Member Area"): the login palette, a sidebar on desktop, an
icon rail on tablets and a top bar with bottom tabs on phones, and the shimmer
`AppLoading` skeleton. `/trainer/apply` renders inside the member shell
because the applicant is not yet a trainer.

Access is decided exclusively by the backend: the shared session context loads
`GET /api/v1/trainer/access` on entry. Unauthenticated users are sent to
`/login`; authenticated non-trainers are sent to the apply page; approved
trainers see the trainer layout. The frontend never infers trainer status
client-side.

Entry point

- The member shell shows a switch to the trainer area for approved trainers,
  and Account (`/account/settings`) shows the trainer status: open the trainer
  area for approved trainers, "Become a Trainer" otherwise. A failed access
  request hides the trainer link; it never blocks the page.

Trainer layout

- `TrainerLayout` renders `AppShell` with Dashboard, Clients, Premium L2 and
  Programmes as primary navigation and Profile and Support as secondary
  navigation; links show an active state.
- The action card "New programme" opens `/trainer/programs?new=1`, which
  starts with the creation form open; a switch returns to the member area
  (`/profile`).

Apply page (`/trainer/apply`)

- Presentational states derived from the access payload: no application (the
  form), `PENDING` (waiting for review), `REJECTED` (reapply is allowed),
  or an existing trainer (link into the trainer area).
- The application collects no screening fields today; the backend endpoint
  accepts no body, so the page submits and explains the review step instead of
  inventing personal-data collection.
- While the application is `PENDING`, the review state includes an optional
  supporting-documents panel: a document-type select (Certification,
  Identification, Insurance, Other), a file picker restricted to PDF/JPEG/PNG,
  an upload action, and the list of attached documents with download and a
  two-step remove confirmation. The panel is presentational: the backend
  validates type, size and ownership, and the panel is only offered in the
  pending state. Server error codes are mapped to plain wording.

Dashboard (`/trainer/dashboard`)

- PT and EN. Data from the existing endpoints only: the trainer profile, the
  first clients, the own programmes, the Premium Level 2 requests and the
  public coach profile (view count, shown only when the profile is public).
- A light wave hero with "Add client" and "New programme", a public profile
  views card, tiles for active clients, published programmes, drafts and
  Premium Level 2 requests, then lists of recent clients, programmes (with
  Published/Draft status) and the Premium Level 2 queue.
- No fake statistics, revenue, ratings or placeholder charts.

Clients (`/trainer/clients`)

- Add-by-exact-email flow: look up the account, show the matched identity, then
  confirm. No fuzzy search and no user directory.
- One card list of active clients (name, email, relationship date) with
  pagination and an inline two-step remove confirmation.
- Each client row has a clear "View client" action. Server codes are mapped to
  plain wording (unknown email, already added, already removed, self-add); raw
  tokens are never displayed.

Client detail (`/trainer/clients/:userID`)

- The page loads the safe client profile, the training overview, the client's
  trainer assignments and the trainer's own programme list together; a missing
  or unrelated client shows one honest "Client not found" state, never a
  distinguishable reason.
- Header: avatar initial, name, email and relationship date, plus a back link to
  all clients.
- Four compact summary cards: current programme, programme completion, completed
  workouts and training time. Values are real API data only; empty states are
  explicit and no placeholder charts are shown.
- A live-workout card appears only when the client has an in-progress or paused
  session, with programme, week/workout position, elapsed time and status.
- Programme access is one card with three separate source groups: trainer
  assignments, purchases and Free activations. Each row shows the programme,
  grant date, optional expiry and an Active/Unavailable chip. Assignment and
  Free access become Unavailable when the programme is not published; purchases
  stay Active after unpublishing.
- The current trainer may assign one of their own programmes and remove their
  own assignment with an inline two-step confirmation. Another trainer's
  assignment is read-only and has no addressable action.
- Session history starts from the 10 most recent finished sessions and can load
  the paginated full list. Each row expands to show the snapshot exercises, the
  prescribed and actual reps/weight, effort and set status. Long tables scroll
  horizontally on small screens.
- Exercise history uses an exercise selector built from completed sessions and
  shows the prescribed and actual sets from that immutable snapshot.
- Legacy completion-only history is explained as a count and latest date, never
  rendered as if it had duration or set performance.
- The page notes that it reflects recorded training only and includes no
  nutrition, medical data or unrecorded activity. All expandable rows are real
  buttons with `aria-expanded`; color is never the only status signal.

Programs (`/trainer/programs`)

- One card list of the trainer's own programs (name, type, status, price) with
  pagination and a compact create form: name, description, type
  (free / premium / personalized), price in EUR. Free is fixed at 0,00;
  paid programs enforce the platform minimum. The form states that programs
  start as drafts.
- Success navigates to the program detail.

Program detail (`/trainer/programs/:programId`)

- Metadata card: edit name/description/type/price, publish (POST publish) and
  unpublish (back to draft).
- Builder card: weeks → workouts → exercises, each level with inline add,
  block-style up/down reorder buttons and delete. Exercises are picked from the
  existing exercise library dialog; already-added exercises are disabled in the
  picker. Workout exercises load lazily per workout.
- Danger zone: delete program with inline confirmation.
- Honest empty states at every level; no placeholder content.

Premium L2 (`/trainer/premium-level-2`)

- One card list of requests allocated to the signed-in trainer, with status and
  pagination. Only allocated requests are returned; the page never exposes a
  client directory.
- The detail page (`/trainer/premium-level-2/:requestID`) shows the offer, the
  client identity, the status and the full intake rendered with the shared staff
  catalog. It reuses the existing dark card language and the shared intake
  answer component.
- Training: attach one of the trainer's own draft programmes that contains at
  least one exercise, then review the attached programme. The attached
  programme is never published.
- Nutrition: the shared curated nutrition editor is reused with Level 2 copy;
  the trainer authors summary, hydration, cautions, 3–8 meals and their foods,
  and saves a versioned plan.
- Delivery: the action is enabled only when both a programme and a nutrition
  plan exist. A two-step confirmation explains that delivery releases the
  content and creates the client's entitlement. After delivery the page shows
  the delivered state and history and becomes read-only. Errors use plain
  wording derived from server codes; no raw token is shown.

Profile (`/trainer/profile`)

- View-only summary of the trainer identity (name, email, member since) with a
  note that trainer profile editing is not available yet. No invented editable
  fields.

---

# Client Dashboard

The dashboard is the client's main workspace.
Navigation

- Profile
- Settings
- Share Profile
- Calendar

Main Sections

Statistics

Shows:

- Workout frequency
- Weekly progress
- Strength progression
- Muscle split
- Charts

Measurements

Optional body measurements similar to Lyfta.

Current Program

Displays the active training and nutrition program.

Calendar

Upcoming workouts and events.

Settings

Account information and management (ex: changing password or email).

Profile Sharing

Allows the user to generate and copy a public profile link or share it directly
through supported applications.

---

# Trainer Dashboard

Each trainer has a personal public page.

The dashboard allows trainers to:

- Manage packs
- View customers
- View revenue
- View statistics

---

# Management Dashboard

Responsible for business management.

Functions include:

- Manage trainers
- Manage clients
- Create packs
- Edit packs
- Manage the exercise catalogue
- Marketplace management
- Platform statistics

---

# Technical Dashboard

## Admin Console

The admin console implements the Technical and Management dashboards as two
responsive shells: `/admin` for Technical Administrators (ADMIN_1) and
`/admin2` for Management Administrators (ADMIN_2). Each shell shares one
sidebar with the brand, the current administrator role, section groups and a
sign-out action, and hides public-facing navigation and footer.

Available to Technical Administrators (ADMIN_1):

- User Management
- Trainer Management
- Maintenance
- System Monitoring
- Infrastructure Management
- Technical Configuration
- Security & Authentication administration
- Development

Available to Management Administrators (ADMIN_2):

- Plans / Packs
- Exercise catalogue
- Finance
- Marketing
- Sales / Commerce
- Trainer Management
- Client Management
- Marketplace management
- Platform statistics

The respective page is only shown when the administrator role grants it; the
navigation is filtered by the server-provided role (`TECHNICAL_ADMINISTRATOR`
or `MANAGEMENT_ADMINISTRATOR`) and the sections not granted are never rendered.
The admin login page resolves the role after the second factor and redirects
ADMIN_1 to `/admin/dashboard` and ADMIN_2 to `/admin2/dashboard`.

### Routes

```
/admin/login
/admin/dashboard
/admin/users
/admin/trainers
/admin/trainer-applications
/admin/system
/admin/configuration
/admin/test-mode
/admin/development
/admin/audit
```

```
/admin2/dashboard
/admin2/plans
/admin2/plans/create
/admin2/plans/:programId
/admin2/exercises
/admin2/marketplace
/admin2/trainers
/admin2/clients
/admin2/sales
/admin2/analytics
/admin2/feedback
/admin2/settings
```

`/admin/login` renders the two-step administrator authentication only (no
sidebar shell). Every other `/admin` route renders inside the `AdminLayout`
shell, and every `/admin2` route inside the `Admin2Layout` shell; both resolve
the admin identity before rendering and redirect unauthenticated visitors to
`/admin/login`.

Pages implemented in the current cycle:

- **ADMIN_1 Dashboard** — live totals for users, trainers and pending trainer
  applications, plus a system overview card (current administrator, API and
  database reachability, session lifetime).
- **ADMIN_1 Users** — active/deleted tabs, page search, create/edit, password
  reset, disable/reactivate, pagination.
- **ADMIN_1 Trainers** — active/deleted tabs, disable/reactivate, pagination.
- **ADMIN_1 Trainer applications** — status filter (All / Pending / Approved /
  Rejected), approve/reject actions, and a per-row Review panel. The panel loads
  the application's evidence documents and decision history from the reviewer
  endpoints: documents can be downloaded (never edited), the history shows each
  transition, reviewer note and timestamp, and the reviewer can approve or
  reject with an optional decision note. The quick row actions remain
  reason-less shortcuts.
- **ADMIN_1 System, Technical configuration** — live values (v2.110.0).
  **Development** — placeholder page.
- **ADMIN_1 Audit log** (`/admin/audit`, v2.111.0) — newest-first list of
  every administrative change, evidence download and console sign-in, with
  "Everyone / ADMIN_1 · Technical / ADMIN_2 · Management" and "Any result /
  Done / Denied / Failed" filter chips, a readable action name over the
  method and route template, the administrator badge, date and time, result
  badge with HTTP status, target id and client address; 50 per page with
  Previous/Next; stacked cards on phones. Entries cannot be edited or
  deleted.
- **ADMIN_1 Test Mode** — the Technical-navigation page for entering/exiting
  Test Mode. Shows the current session state (Test Mode active as Test Client /
  Test Trainer, with an Exit control, or two persona option cards to enter),
  a configurable return path (default `/admin/test-mode`), and honest error
  states (e.g. "Test Mode is disabled on the server" when `TEST_MODE_ENABLED`
  is off, or a restricted message when the signed-in role is not the Technical
  Administrator). After entering, the backend redirects to the client site; the
  banner becomes the primary control surface.
- **ADMIN_2 Dashboard** — live platform totals (clients, trainers, pending
  applications, published programs), product mix bars and a recently published
  programs table.
- **ADMIN_2 Plans** — the platform-owned plan list with three product-family
  tabs, Generic, Premium Level 1 and Premium Level 2, each backed by its own
  family route (`/api/v1/programs/generic`,
  `/api/v1/programs/premium-level-1` and
  `/api/v1/programs/premium-level-2`). A Premium Level 2 row is an offer, not a
  training structure. The list
  always shows exactly one family at a time; the family is carried in the URL
  (`?family=`) and can never be invented by the client. Searchable (the search bar
  queries the real backend), with a pricing-category filter (All / Free / Premium
  / Personalized), a level filter and pagination. The pricing-category filter
  filters the commercial `type` and is labelled as such; it is not the product
  family. Each row shows the plan title, type, level, frequency, duration, price
  and last-updated date with Edit and Delete (soft delete, confirmed via dialog)
  actions. The list also exposes one right-aligned sales group per plan — Sales
  (`real_sales`: completed non-test purchases), Test (Test Mode completions) and
  Revenue (completed non-test) — fetched in a single batch call against the
  commerce endpoint (Management Administrator only). When the sales batch fails
  the numeric columns fall back to an em dash and the section subtitle adds
  "· Sales unavailable" without taking the plan list down. "New plan" opens the
  creation workspace in the active family.
- **ADMIN_2 Plan creation workspace** — a dedicated `/admin2/plans/create`
  flow to build a platform-owned plan: a product-family selector (Generic,
  Premium Level 1 or Premium Level 2), a 4-step wizard (basic info, program structure, marketplace,
  review & publish) with a live RYZE marketplace card preview and a two-pane
  exercise library picker. Weeks contain training days; a day holds structured
  exercise assignments — each assignment references one catalog exercise (never
  a copy), carries its position, an optional per-day tailored instruction, and
  its full set prescription (reps, weight, RIR, RPE, rest, tempo, set type).
  Clicking an assignment opens the detailed editor where the instructor tailors
  the instruction, edits or adds sets, replaces the exercise with a catalog
  alternative and removes it; assignments reorder within the day and remain
  immutable references so a catalog update is reflected everywhere. The
  Marketplace step also captures the optional audience metadata — Level
  (Beginner / Intermediate / Advanced), Training Type (Hypertrophy / Strength /
  HYROX / CrossFit / Fat Loss / At Home), Duration in weeks and Frequency in
  days per week. A Premium Level 1 package additionally requires an objective
  (Gain muscle / Lose fat / Strength & conditioning) and a nutrition mode. The
  curated mode opens the curated nutrition editor for the package summary,
  hydration guidance, cautions, meals, foods and macros. Objective and mode are
  fixed after creation; the selector is disabled in edit mode. A Premium Level
  2 offer skips the structure step and the audience fields, accepts only the
  `premium` pricing category and shows "Trainer delivered after purchase" in
  the preview: the service has no weeks and no nutrition content at the package
  level. The Publish & Save action talks to the family API: it maps the local draft to the create
  payload, disables the button until a change is dirty and while a request is in
  flight (`Saving…`), and surfaces the outcome as an inline notice — green on
  success, and per-code friendly errors (e.g. missing catalog exercise, duplicate
  exercise inside a workout, program not found, already-published conflict,
  validation details) instead of a raw failure. The same workspace opens in edit
  mode at `/admin2/plans/create?edit=<id>&family=<family>`, loads the existing
  program into the draft and saves changes transactionally through the update
  endpoint.
- **ADMIN_2 Premium L2 requests** — `/admin2/premium-level-2` is the request
  queue for the trainer-delivered service. Search (client name/email/offer),
  status filter and pagination come from the real API; the detail view shows the
  offer, client, status, allocated trainer, intake answers rendered with the
  shared staff catalog, delivery state and request history. ADMIN_2 allocates
  or changes an active trainer from the same view. The queue never shows intake
  answers.
- **ADMIN_2 Plan detail** — `/admin2/plans/:programId?family=<family>` opens a
  platform-owned plan from the real family detail endpoint and renders the
  complete structure in-tree (weeks → training days → exercises → sets with
  reps, weight, RIR, RPE, rest and tempo) next to a program-facts summary
  (product family, objective, nutrition mode, type, status, level, training
  type, frequency, duration, price, created, updated). A curated package also
  shows a read-only Curated nutrition section with its summary, dietary pattern,
  hydration, meals, foods, macros and cautions. Actions: Publish plan (draft →
  published via the guarded publish endpoint), Unpublish plan (published → draft
  via the full update with a draft status), View in marketplace (the Premium
  catalogue for a Premium Level 1 package, otherwise the public
  `/services/generic-program/:programId` detail page when published), Edit plan
  (builds on the plan workspace in edit mode) and Delete (soft delete with
  confirmation). While a plan is published a compact Sales section in the right
  rail shows Sales (`real_sales`), Revenue and Test purchases and a "View
  purchases" action that deep-links to `/admin2/sales?program_id=<id>`. A plan
  with any completed purchase is content-locked: the page states that its content
  and status are locked, and disables Publish/Unpublish, Edit and Delete.
  Existing clients keep access through their entitlement even if the catalogue
  state changes.
- **ADMIN_2 Exercises** — `/admin2/exercises`, the administrative surface of the
  canonical exercise catalogue (Management Administrator only). A debounced
  search, status filter (All / Draft / Published / Archived) and body-part
  filter drive the real admin endpoint; the table shows the exercise with its
  source provenance, body part + difficulty, equipment, publication status,
  last update and an Edit action, with pagination and loading/error/empty
  states. "New exercise" opens the editor modal; entries are created as drafts.
  The editor groups classification fields (body part, difficulty, movement
  category, equipment, muscles), content (description, instructions,
  one-step-per-line instruction steps) and relative media references
  (image/animation path, source media id, attribution), and exposes the
  publication status only when editing. Names are immutable after creation.
  Server errors are mapped to plain wording (duplicate name, validation,
  missing entry); creating or saving refreshes the list.
- **ADMIN_2 Marketplace** — published catalog with filter chips grouped by
  product type, price and currency.
- **ADMIN_2 Trainers (business)** — trainer profiles with a commission manager
  (default 20%, per-trainer override, max 10000 bps) backed by the platform
  commission endpoints.
- **ADMIN_2 Clients** — paginated member accounts (read-only analytics view).
- **ADMIN_2 Sales** — the commerce dashboard backed by the live `/admin/purchases*`
  endpoints (Management Administrator only). Six metric cards (Revenue with a
  "Completed sales · Test excluded" hint, Completed, Pending, Failed, Test
  purchases, Programs with sales), a filter toolbar (program select populated
  from the per-program endpoint, status, real/test mode, from/to date range and
  a reset control), an accessible transactions table (program + type with
  Retired/Test markers, customer name + email, currency-formatted amount,
  status badge, active/no-access badge, date, and a View action opening a
  purchase detail modal) with loading / error+retry / empty states and
  pagination, plus a per-program sales section (completed, revenue excluding
  test, pending, failed, test counts) with its own pagination. Test purchases
  never contribute to revenue. The page accepts an optional `program_id` URL
  query parameter as a deep link from the plan detail: it pre-selects the
  program filter on mount and guarantees that program appears in the dropdown
  even when it falls outside the top-100 option list.
- **ADMIN_2 Sales Reconciliation** — the settlement audit section on the Sales
  page backed by `GET /admin/purchases/reconciliation-events`
  (Management Administrator only, read-only). A scope select (Needs attention —
  the default, hiding resolved settlements — vs Complete trail) and an outcome
  select (All outcomes plus each recorded outcome code, client-owned wording:
  transient failure, uncertain delivery, amount/currency/method/reference
  mismatch, late approval, entitlement integrity), an accessible events table
  (outcome badge with settled/needs-attention/rejected tone, program + customer
  with a Test marker, currency-formatted amount, provider, provider event id in
  mono, recorded date, and a View action opening the same purchase detail modal
  as the transactions table) with loading / error+retry / empty states and
  pagination. The section is visibility-only: no action on an event can modify a
  purchase, entitlement or outcome, and provider event ids are displayed as
  opaque identifiers never passed anywhere.
- **ADMIN_2 Feedback** — honest unavailable state: feedback is not part of the
  product scope yet, so the page explains the gap instead of fabricating data.
- **ADMIN_2 Analytics** — real aggregates (total accounts, product mix, average
  paid price) with a placeholder reserved for purchase/revenue charts.
- **ADMIN_2 Settings** — read-only commercial configuration (currency, pricing
  floor, default commission, cap).

### UI Principles

- The console is content-focused: one table per primary action, no cards
  around cards, no decorative clutter on data screens.
- Actions use the existing reusable `Button` component and the shared design
  token palette; nothing is invented outside the design language.
- The sidebar collapses into a drawer with an overlay scrim on mobile.
- Data loads through the existing frontend services layer; the admin API
  endpoints are documented in `docs/INSTRUCTIONS/api.MD`.
- Unimplemented backend capabilities surface as honest empty states with a
  clear explanation; the console never fabricates business data.

### Test Mode Banner

A fixed, persistent control shown across the client site whenever the server
reports an active Test Mode session. It is the client-side counterpart to the
admin Test Mode page and the only always-reachable Exit affordance.

- **Placement** — fixed near the bottom-center of the viewport
  (`left: 50%; transform: translateX(-50%)`), width `min(42rem, 100vw - 2rem)`,
  high z-index so it floats above page content.
- **Appearance** — a dark amber pill card (amber border, amber icon chip, white
  text) distinct from the cyan design language: amber reads as "test/debug
  mode", never as a brand accent. A small Flask icon introduces the label
  "TEST MODE · Active as Test Trainer / Test Client", a one-line muted
  subtitle ("Purchases are completed instantly at no cost. No real payments
  are made."), an inline error line when an exit attempt fails, and a small
  danger "Exit Test Mode" button that hard-redirects to the stored return path
  on success.
- **Visibility** — rendered only while active; hidden inside admin shells and
  the authentication pages (`/login`, `/register`, `/admin/login`) so
  impersonating state never noise-crowds chrome and auth flows. It is not a
  modal — it never blocks content.
- **Source of truth** — the session state comes from `GET /api/v1/auth/test-mode`
  via the shared `useTestMode` context (the same one the marketplace and detail
  pages read to force prices to Free); the banner never guesses a persona.

---

## RYZE Feed

RYZE Feed uses a simple, immersive short-form video interface focused on consuming fitness content with minimal UI.

### Feed Layout

The Feed uses a full-screen vertical video layout.

Each video occupies the majority of the viewport and users navigate between videos by vertical scrolling/swiping.

The interface should remain visually clean and avoid unnecessary elements.

### Video

Each Feed item contains:

- Full-screen vertical video
- Automatic playback when the video becomes active
- Automatic pause when the video leaves the viewport
- Muted/unmuted video control
- Video playback should prioritize a smooth and uninterrupted experience

Videos should use a vertical format whenever possible.

### Creator Information

The active video displays a small creator section near the bottom of the screen containing:

- Trainer profile picture
- Trainer name
- Follow button
- Short video description/caption

The creator name should be clickable and lead to the trainer's public profile.

If the user already follows the trainer, the button should indicate the current following state.

### Actions

A small vertical action area is positioned on the right side of the video.

Initial actions:

- Like
- Share

The Like button displays the current like state and may display the number of likes.

Comments are intentionally not available.

### Navigation

Users navigate the Feed primarily through vertical scrolling/swiping.

The Feed should behave similarly to modern short-form video applications:

1. User opens RYZE Feed.
2. The first video automatically starts playing.
3. User scrolls vertically.
4. The current video stops playing.
5. The next video becomes active and starts playing.
6. The process continues indefinitely while content is available.

There should be no traditional pagination visible to the user.

### Follow Integration

Following a trainer is available directly from the video interface.

Following a trainer does not create a separate feed.

Instead, followed trainers receive increased relevance in the user's personalized Feed.

### UI Principles

The RYZE Feed should prioritize:

- Video content
- Minimal UI
- Fast navigation
- Mobile-first interaction
- Clear creator attribution
- Simple actions

The interface should avoid:

- Large cards
- Complex navigation
- Comment sections
- Excessive metadata
- Persistent menus covering the video
- Traditional social-media post layouts

### Responsive Design

The Feed is primarily designed for mobile devices.

On mobile:

- Video fills the viewport.
- Controls remain easily reachable with one hand.
- Creator information stays near the bottom.
- Actions remain vertically aligned on the right.

On desktop, the Feed may remain centered with a vertical-video aspect ratio rather than stretching the video across the entire screen.

### Visual Style

RYZE Feed should follow the existing RYZE visual identity:

- Dark interface
- RYZE blue/cyan accents
- Subtle borders and glow effects
- Minimal typography
- High contrast controls
- Smooth transitions

The Feed should feel modern and technological without becoming visually overloaded.

The content itself should remain the primary visual focus.

---

# Responsive Design

Every page must function correctly on:

- Mobile
- Tablet
- Desktop

Layouts should adapt naturally without changing the overall user experience.

---

# Future Improvements

Future versions may introduce:

- Additional themes
- Accessibility improvements
- Custom dashboards
- Advanced profile customization

# Inspiration

The following products should be used as design references.

## Linear

Reference for:

- Dark denseness done calmly
- Navigation
- Minimalism
- Product surfaces

## Raycast / Arc / Vercel / Nothing

Reference for:

- Premium technology branding
- Restraint
- Strong hierarchy
- Product-led presentation

## Lyfta

Reference for:

- Exercise pages
- Statistics
- Measurements
- Workout visualization

## Apple

Reference for:

- Showing the product instead of explaining it
- Simplicity
- Intentional spacing

## Stripe

Reference for:

- Landing clarity
- Pricing structure

---

# Premium Level 1 Page

Route: `/services/premium-level-1`. It must be declared before the dynamic
`/services/:serviceSlug` route, or the slug route will swallow it.

The selected package is a single scrolling detail page, not a wizard with its
own navigation. A legacy package follows product → questionnaire → checkout →
access; a curated package follows product → checkout → access. The user always
sees where they are. Without a selected package the same route is a catalogue of
every returned Premium Level 1 package.

## Package catalogue

Without a `program` query parameter, the page groups published Premium Level 1
packages by objective. Each objective card shows its icon, client label,
description and package count and opens
`/services/premium-level-1?objective=<value>`. The three objectives are Gain
muscle, Lose fat and Strength & conditioning, in that fixed order. An objective
with no packages is not shown.

A selected objective lists only its packages, headed by the objective label, a
package count and an "All objectives" action. Each package card shows the API
name and description, the Test Mode-aware price, available
duration/frequency/level metadata and a "View package" action. Selecting a
package navigates to
`/services/premium-level-1?program=<id>&objective=<value>` so the back link can
return to the active objective.

When the response also contains packages without a known objective (legacy rows
from before this catalogue model), they are listed under "Other packages" rather
than being hidden or forced into an objective. When no objective is known at
all, the page falls back to the flat package list. A provider return that omits
the query parameter selects the first package only long enough to resolve the
purchase, then replaces the URL with the purchased `program_id`. An unavailable
requested package renders the catalogue with an inline error instead of silently
opening another product.

With a selected package, the page offers "Back to <objective>" plus "All
objectives" when an objective is active, or "All objectives" when several
packages are available. A re-read after purchase keeps the objective context.

## Product presentation

Everything on this page is read from the backend. Name, description, price,
currency, duration, level, frequency, objective and product family come from the
public catalog scoped to `premium_level_1` and from the program detail response.
The page never hardcodes a product id, a price or an offer description.

Product copy states plainly what the package is and what it is not. A curated
package says that its nutrition side is a fixed, expert-defined plan curated for
the package objective and that every buyer receives the same content. A legacy
package says that its nutrition side is personalised automatically from the
questionnaire, so both halves are built from the same answers. Both modes state
that the package is not personal coaching, that no trainer reviews an intake
(for the curated mode there is no intake at all), and that it is not a medical or
clinical assessment.

The disclaimers are part of the product presentation, not fine print. A client
should not have to discover that Premium Level 1 is automated or predefined after
paying.

Price presentation follows the existing rule: Test Mode sessions display a zeroed
price, and the underlying product data is never mutated. Test Mode is marked with
the same accent treatment used elsewhere.

## Layout

Two columns on desktop, single column below 900px:

- **Main column** — the package explanation, then whichever step is active.
- **Aside** — a sticky summary card with the product name, price, a one-sentence
  description and the single primary action for the current step.

Below 900px the aside stops being sticky and the column order is preserved. The
page never scrolls horizontally.

## Questionnaire

A curated package has no questionnaire. Its primary action is "Continue to
checkout" and the page goes directly from product to checkout.

A legacy package renders the form from the server-declared catalog: field type,
label, help text, required flag, sensitive flag, options, numeric bounds and
entry caps. The client holds no hardcoded field list, so a new question renders
without a client release.

- Required fields are marked with an asterisk and carry an accessible
  "(required)" label for assistive technology only.
- Help text and sensitivity notes are linked to the control through
  `aria-describedby`, together with any field error.
- Multi-select options are pill-shaped toggles. When the server-declared entry cap
  is reached, remaining unchecked options become disabled. Nothing is ever selected
  on the buyer's behalf.
- List fields render as one free-typed entry per line in a multi-line control;
  blank lines are dropped and entries are trimmed.
- Rejected fields show client-owned wording for the server's reason code. A raw
  server token is never displayed, especially for a health-related field.
- Unsaved answers live in component state only. There is no browser persistence,
  because the answers are sensitive.

Saving advances to checkout. A server refusal to save because the purchase locked
the intake is not shown as an error: the page re-reads the authoritative state and
renders the locked view.

## Locked questionnaire

A legacy package replaces the questionnaire with a locked panel once a purchase
exists. The panel carries a lock icon and a plain explanation that the answers
are part of the delivered package.

There is intentionally no "edit again", "regenerate" or "start over" action. The
primary action becomes "Continue to checkout" or "Open Program Access". A delivered
package's intake is immutable, and offering an action that silently cannot work
would be worse than not offering it.

## Checkout

Checkout reuses the existing purchase and payment APIs through the shared
checkout hook. A legacy panel states that the questionnaire is saved; a curated
panel states that the predefined nutrition plan is included. Both state that the
purchase unlocks both halves of the package.

States are rendered explicitly rather than hidden: purchase confirmed, payment
cancelled with a resume action, payment failed with a retry action, sign-in
required, and checkout unavailable. Payment methods are radio options rendered
from the entitlements response.

A cancelled payment states plainly that no money was taken. On a legacy package
it also says that the questionnaire is still saved: the user has already invested
in a long form and should not be asked to redo it. A curated package has no form
to preserve.

In a Test Mode session a banner states that the purchase completes without
contacting a payment provider.

## Nutrition

The nutrition panel maps the server's assignment state directly and never invents
readiness. A curated package reads its completed plan through the same client
contract, but the copy says "nutrition plan" and never implies generation from
answers. A legacy package uses the generation wording and the build/retry actions.

- `pending` and `processing` — a preparing state with a refresh action, and a build
  action when `pending`. A curated pending state says the curated plan is being
  opened; a legacy pending state says the personalised programme is being built.
- `failed` — states that nothing is wrong with the purchase and that no action is
  required from the client, with a retry action, in a non-alarm note. A failed
  generation is never presented as a broken purchase, and a refused generation
  shows client-owned wording (`NUTRITION_RESTRICTIONS_UNSATISFIABLE` → a fixed
  "requirements cannot be met" sentence) rather than a raw server token.
- `completed` — the plan.

A completed plan is rendered from the delivered contract:

- Daily targets as a tile grid — daily calories, target versus maintenance, the
  macro split in grams and percentages, and fluids.
- The day's eating structure as a list of meals and snacks, each with its share of
  the day, its energy, and its actual foods as real entries (name, quantity, unit,
  macros). An entry that was substituted during planning shows the equivalent it
  may be replaced with; a substituted entry never hides what it originally was.
- The dietary guarantee as an exclusions audit for a legacy plan — the controlled
  reason and the resolved exclusion token. No free-text from the intake is ever
  shown. A curated plan has no intake and therefore no exclusions audit.
- The server's cautions in a highlighted note, and the preparation guidance and
  summary when present.

The panel is resilient to the safety contract too:

- A panel that reports completion but sends no plan renders as unavailable with a
  refresh action. Inventing readiness from a missing payload would be a lie the
  client cannot detect.
- When a legacy server marks the plan `out_of_date`, the panel keeps showing the
  delivered plan and adds a stale banner offering to update to the latest
  answers. This banner is defensive: the intake is locked for the life of the
  entitlement, so it should never appear in production. A curated plan never
  renders this banner.

A delivered plan is read-only. There is no manual regeneration or replacement
control anywhere in this flow: the only actions that call generation are the
build/retry actions of the legacy `pending` and `failed` states and the defensive
legacy stale-banner update. The panel closes by stating that the programme is
part of the package: fixed from the submitted answers for a legacy package, or
fixed package content for a curated package.

### Program Access

Program Access renders the same nutrition panel below the training structure for
Premium entitlements, with the same state, readiness and resilience rules.

# Premium Level 2 Page

Route: `/services/premium-level-2`, declared before `/services/:serviceSlug`.
It is a single scrolling detail page, not a wizard. Without a `program` query
parameter it lists every published Premium Level 2 offer; selecting one adds the
`program` parameter and shows the product.

## Product presentation

Name, description, price and currency come from the public `premium_level_2`
catalog and program detail response. The page never hardcodes a product id or
price. It states plainly that this is a manually prepared service: the client
buys first, completes a detailed intake, a certified trainer is allocated, and
the training and nutrition are authored individually before delivery. It states
that delivery is not immediate and is not an automated package. Test Mode keeps
the existing zeroed-price treatment without mutating product data.

## Purchase and next step

The page reuses `useProgramCheckout` and the existing payment/Test Mode flow.
After a completed purchase it resolves the service request and routes to
`/services/premium-level-2/requests/:requestID`; when the request is not yet
readable it keeps the purchase confirmation and a link to My Programs. The page
does not collect intake answers before purchase and does not invent a second
checkout.

# Premium Level 2 Request Page

Route: `/services/premium-level-2/requests/:requestID`. The owner is always
taken from the session; the id in the URL is only an address.

- Header shows the offer, status, allocated trainer and delivery date when
  present.
- `awaiting_intake` / `awaiting_trainer_assignment`: the catalog-driven intake
  form, grouped in sections, with required-field validation and a save action.
  Answers already saved are prefilled. The form explains that editing closes
  when a trainer is allocated.
- `in_progress`: read-only answers and a waiting state explaining that the
  trainer is preparing the programme and nutrition. No fake progress bar.
- `ready_for_delivery`: read-only answers and a waiting state explaining that
  the pack is being finalised.
- `delivered`: read-only answers, a link to the delivered training programme in
  Program Access and the delivered nutrition plan rendered from the read-only
  plan API. Draft nutrition is never shown.
- Errors are mapped from server codes (`PREMIUM_LEVEL_2_NOT_FOUND`,
  `PREMIUM_LEVEL_2_NOT_EDITABLE`, `NUTRITION_NOT_FOUND`) to client-owned
  wording. Answers are never written to browser storage.

# My Programs and Program Access

The Premium tier is not given its own account area.

- **My Programs** marks Premium entitlements with a distinct "Premium Level 1"
  or "Premium Level 2" chip. Premium Level 1 adds a "Training + nutrition" chip
  and labels the action "Open package". A Premium Level 2 entitlement links
  directly to its service request page, where intake, status and delivered
  nutrition live; before the request is available the card falls back to the
  service page. The empty state links to the generic marketplace, Premium Level
  1 and Premium Level 2.
- **Program Access** marks the owned program the same way and renders the nutrition
  panel below the training structure, for Premium entitlements only. A generic
  program never issues the nutrition request at all. The panel keeps its curated
  or legacy wording based on the package's objective, and a purchased program
  remains accessible even if it is later unpublished.

Both read the family from the backend response and never compare a hardcoded id.

## Responsive and accessibility

Single column below 900px, reduced padding below 640px. Every control is reachable
by keyboard with a visible focus ring, every icon is decorative and hidden from
assistive technology, state changes are announced through `role="alert"` for errors
and `role="note"` for cautions, and the required-field marker is duplicated
visually only.
