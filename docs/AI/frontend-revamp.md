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

These areas must not change, including indirect effects from shared components,
global tokens or CSS:

| Protected area | Verified source locations and dependencies |
| --- | --- |
| Entire website footer | `main/src/frontend/src/components/footer/footer.tsx`, `footer.module.css`; consumed by `components/layout/layout.tsx` |
| Exercise Library and exercise details | `pages/exercise_library_page/`, `pages/exercise_detail_page/`, and `components/exercise_library/` including the existing Three.js demo/viewer/media styles and tests; routes `/exercises` and `/exercises/:exerciseId` |
| Services page | `pages/services_page/services_page.tsx`, `services_page.module.css`; route `/services` and its service presentation/assets |
| Home “Find Your Next Program” section | `components/programs_overview_section/programs_overview_section.tsx`, `.module.css`, imported by `pages/home_page/home_page.tsx` |

The Services page currently contains an existing static background asset. That
protected implementation wins over the general background-image rule: do not
modify it as part of the revamp. Other pages must follow the no-static-
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

## Before implementation

Inspect the installed Three.js version (`three@0.186.1`) and actual direct
Three.js implementation before applying external examples. RYZE has no verified
React Three Fiber or Drei dependency. Never upgrade dependencies or modify
protected source/assets for design consistency without explicit owner approval.
