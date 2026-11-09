---
paths:
  - "main/src/frontend/src/**/*.ts"
  - "main/src/frontend/src/**/*.tsx"
  - "main/src/frontend/src/**/*.css"
---

# RYZE frontend rules

- Read `docs/INSTRUCTIONS/design.MD`, the relevant page/API documentation, and
  `docs/AI/frontend-revamp.md` before changing UI.
- Design mobile-first at 360–430px, then validate narrower mobile, tablet and
  desktop behavior. Use accessible labels, keyboard focus, touch targets,
  loading/error/empty states and reduced-motion behavior.
- Preserve existing API contracts, authentication, permissions, purchase and
  workout flows. Keep business decisions in the backend.
- The official RYZE logo and documented brand palette are protected foundations;
  redesign distribution and hierarchy freely, but do not replace the brand.
- Do not use static background images outside login and registration. Functional
  content imagery is allowed when it communicates real information.
- Three.js is for useful interaction only. Preserve existing exercise demos,
  lazy-load heavy code, provide WebGL fallback, and check the installed version.
- Do not globally restyle shared components until protected-area consumers are
  mapped. These are immutable: footer; Exercise Library/details; Services; and
  Home's “Find Your Next Program” section. Use isolated variants when needed.
- Prefer existing components and API utilities. Do not duplicate state or add a
  new state/data-fetching library without explicit approval.
- Use CSS Modules and the existing CSS architecture. Avoid broad global changes
  that can alter protected pages. Verify actual browser screenshots before
  calling a redesign responsive.
