# Version 1.1.0

Title: Initial frontend implementation

Date: 2026-08-03

Summary:

Implemented the initial visual frontend for the RYZE platform using the
documented React, TypeScript, Vite, React Router and CSS Modules stack.

Files modified:

- `.gitignore`
- `index.html`
- `package.json`
- `package-lock.json`
- `tsconfig.json`
- `vite.config.ts`
- `main/src/frontend/src/**`
- `main/src/utils/class_names.ts`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added the Vite React application entry point.
- Implemented global styling tokens based on the documented RYZE palette.
- Added reusable frontend components for buttons, cards, layout, navbar,
  footer, loading screen, section titles, hero, service cards, CTA bands and
  contact content.
- Implemented the Home page with navbar, hero, services preview, CTA sections,
  contact section and footer.
- Implemented React Router routes for Home, Services, Contact, Feedback,
  About Us, Profile and service detail paths.
- Added professional under development screens for routes not fully implemented
  yet.
- Added the reusable global loading screen with the RYZE logo and two rotating
  arrows.
- Used existing logo assets from `main/src/resources/img/logo/` without moving
  or duplicating them.
- Added developer run documentation.
- Documented the current frontend folder structure.

Reason:

The platform needed a production-quality initial frontend foundation before
backend integration begins, while preserving the documented visual identity,
responsive behavior and reusable component architecture.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---
