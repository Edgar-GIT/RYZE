# RYZE frontend revamp guide

The owner has approved a major visual redesign, but it must remain a coherent
RYZE product rather than a generic SaaS template. The official logo and current
RYZE brand colors are protected foundations. The redesign may change layout,
typography, spacing, tokens, navigation, components, interaction patterns and
compatible neutrals, but must not replace the brand identity.

The verified foundation in `docs/INSTRUCTIONS/design.MD` and
`main/src/frontend/src/styles/global.css` is: backgrounds `#05070B`, `#0A1018`,
`#101826`; cards/surfaces `#141F2D` and `#182638`; accents `#18D8FF` and
`#2A8CFF`; text `#FFFFFF`, `#B8C4D8`, `#7F8EA3`; default border
`rgba(255, 255, 255, 0.08)`; and hover border `rgba(24, 216, 255, 0.35)`.
Use CSS variables and compatible tonal variations rather than inventing a
competing primary palette or recoloring the logo.
The logo assets are under `main/src/resources/img/logo/` and are mapped by
`main/src/frontend/src/constants/brand_assets.ts`; inspect those assets before
proposing token changes.

## Non-negotiable constraints

- Design mobile-first at 360–430px, support narrower screens, then adapt to
  tablet and desktop. Primary actions must be obvious and touch-friendly.
- Keep UX straightforward, premium, distinctive, readable and fast to navigate.
- Do not use static background images, textures, decorative background video or
  image-based heroes outside login and registration. Functional exercise media,
  avatars and useful product images are allowed.
- Preserve authentication, permissions, API contracts, purchases, entitlements,
  workout execution and role-specific workflows.
- Use Three.js only for useful 3D interaction. Preserve the existing exercise
  demonstrations, lazy-load heavy functionality, provide WebGL fallback,
  respect reduced motion, and account for mobile GPU/battery use.

## Protected-area inventory

On 2026-11-10 the owner narrowed the original protection so these areas could
follow the new public design. What remains protected must not change, including
indirect effects from shared components, global tokens or CSS:

| Area | Still protected | May follow the public design (owner-approved 2026-11-10) |
| --- | --- | --- |
| Website footer — `components/footer/` (consumed by `components/layout/layout.tsx`) | Layout, link columns and their links, the slogan image, copy | Surface colour, lines, link/heading colours, typeface |
| Exercise Library and details — `pages/exercise_library_page/`, `pages/exercise_detail_page/`, `components/exercise_library/` (including the Three.js demo/viewer and tests) | Layout, components, behaviour, media and the 3D viewer | Colours only, through page-scoped token overrides in each page's `.page` rule, plus the library title typeface |
| Services page — `pages/services_page/` | The service cards: `components/service_card/` markup and styles, and the card grid (the owner added a purple "Personalized" tag variant for Premium Level 2 on 2026-11-11) | Page background (the former static background image is removed), intro, loading state, benefits row; the original showcase cards appear while nothing is published |
| Home “Find Your Next Program” — `components/programs_overview_section/` | The card background images (`advert1–3`), until the owner replaces them | Everything else: redesigned on 2026-11-11 as the Home's main call to action and moved right after the hero |
| Login and registration — `pages/login_page/` | Form fields, validation, messages, API calls and redirects | Everything visual (fully redesigned; no background image now) |

Verify the protected parts against a pristine copy (see the verification
methods below) after any change near them. Other pages follow the no-static-
background rule. Before editing shared layout, navigation, global styles or
tokens, inspect all protected consumers. Use isolated variants when necessary.

## Staged workflow

### Phase 1 — audit

Inspect routes, roles, page hierarchy, workflows, reusable components, CSS
architecture, API clients, mobile behavior, accessibility and performance. Read
the relevant product and design documentation. Capture baseline screenshots of
protected areas at the same browser/viewports that will be used for comparison.

### Phase 2 — visual direction

Use the installed official `frontend-design` plugin and `ui-ux-pro-max` search
intelligently. Establish one direction with brand-color roles, typography,
spacing, semantic tokens, navigation, mobile interaction, accessibility and
functional-image rules. Do not generate many disconnected variations or persist
unverified design intelligence.

### Phase 3 — foundations

Create only the manageable shared primitives and tokens needed by the approved
scope. Keep protected consumers on their existing implementation when a shared
change would alter them. Do not introduce a new framework or styling system.

### Phase 4 — implementation groups

Work in small coherent groups: public pages other than protected Services;
authentication; marketplace/purchase; client training; history/progress;
trainer; then ADMIN_2. Verify each group before moving on. Do not redesign the
entire frontend in one unverified operation.

### Phase 5 — visual QA

Check small mobile, standard mobile, large mobile, tablet and desktop in a real
browser. Verify screenshots, keyboard/focus, contrast, touch targets, reduced
motion, no horizontal overflow, loading/error/empty states and preserved API
flows. Compare protected-area baselines after every significant milestone.

## Progress

- 2026-11-10 (owner revision) — The owner asked for one consistent site: no
  more switching between navy, paper, cyan and royal sections. The public layer
  now has a single principle — the website is built from the app's materials:
  one continuous logo-navy page with a repeating rhythm of soft blue light
  (sections are transparent, so they cannot show seams), the app's blue glass
  cards, cyan as the light, and one recurring bright element, the light-blue
  wave panel (`Waves` in `components/public_ui/`). The footer, Services,
  Exercise Library, "Find Your Next Program" and login/registration were
  aligned within the limits in the table above. The desktop 3D story was
  rebuilt: a real-continent Earth (Natural Earth land mask) with RYZE routes
  leaving Lisbon, a modelled phone and laptop, and illustrative app screens in
  the style of the owner's phone reference. Phones get an SVG globe with the
  same routes (`components/brand_globe/`). The globe assets are produced by
  `scripts/generate_globe_assets.py` from Natural Earth's public-domain
  `ne_50m_land` and `ne_110m_land` GeoJSON files.

- 2026-11-10 — Group 1 (public pages other than protected Services) is
  implemented: Home (except "Find Your Next Program"), Our Vision, Contact,
  Feedback and the public `UnderDevelopmentPage`. A first, mostly dark and
  minimal pass was rejected by the owner and replaced. The accepted direction:
  verified logo colours with alternating navy, royal, cyan, paper and white
  sections; self-hosted Archivo (expanded) typography; brand-coloured headline
  continuations restored from the original site; real app screens captured
  with sample data; and, on desktop Home only, a scroll-driven Three.js story
  approved by the owner as product storytelling (not a background). The navbar
  received invisible accessibility fixes only. Authentication,
  marketplace/purchase, client, trainer and ADMIN_2 groups are not started.

Verification methods used for Group 1 (reuse for later groups):

- Protected areas: serve a pristine `git archive HEAD` copy beside the working
  tree with the same API mocks; compare every descendant's computed style
  (including `::before`/`::after`) and relative geometry for protected
  elements (for example each service card and each Home program card). Force a
  fresh layout before reading styles. Element screenshots alone are unreliable
  when a section's page offset changes (sub-pixel anti-aliasing, fixed navbar).
- 3D: run headless Chromium with `--use-angle=swiftshader
  --enable-unsafe-swiftshader`; count `drawElements`/`drawArrays` calls to prove
  the loop renders while visible, stops off screen and stops after navigation;
  confirm no Three.js request below 1024px, with reduced motion or without
  WebGL.
- Each scene instance must own its canvas: React StrictMode mounts effects
  twice in development, and disposing a shared canvas (`forceContextLoss`)
  kills the live instance.
- Very tall full-page screenshots can show compositing repeats in headless
  Chromium; review long pages with viewport-sized captures.

## Before implementation

Inspect the installed Three.js version (`three@0.186.1`) and actual direct
Three.js implementation before applying external examples. RYZE has no verified
React Three Fiber or Drei dependency. Never upgrade dependencies or modify
protected source/assets for design consistency without explicit owner approval.
