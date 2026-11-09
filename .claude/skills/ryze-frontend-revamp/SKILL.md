---
name: ryze-frontend-revamp
description: Use when auditing or redesigning RYZE frontend pages, layouts, navigation, components, or visual systems while preserving protected areas, brand colors, mobile-first UX, and product behavior.
---

# RYZE frontend revamp

Before editing, read `docs/AI/frontend-revamp.md`, `docs/INSTRUCTIONS/design.MD`,
the relevant product/API docs, and inspect the route, page, imports, styles and
shared consumers. Map protected-area dependencies before changing shared code.

Use the installed `frontend-design`, `ui-ux-pro-max`, responsive, accessibility
and React skills selectively. Establish one coherent direction for the actual
fitness/nutrition product; do not copy generic SaaS patterns or redesign the
whole application in one unverified change.

Mandatory constraints:

- Start at 360–430px and verify narrow mobile before desktop adaptation.
- Preserve the official logo and documented RYZE brand colors; compatible
  neutrals and semantic tokens may improve hierarchy and contrast.
- No static background images outside login and registration. Functional images
  are permitted when they convey content.
- Never modify the footer, Exercise Library/details, Services page, or Home's
  “Find Your Next Program” section, directly or through shared styling.
- Treat Three.js as functional, lazy, fallback-capable and mobile-conscious;
  never add decorative 3D backgrounds or dependencies by implication.
- Preserve API contracts, auth, permissions, entitlements and workflows.

Implement in a small page group, reuse existing utilities, then verify keyboard,
reduced motion, loading/error/empty states, screenshots and all supported
viewports. Report any protected-area conflict instead of making an exception.
