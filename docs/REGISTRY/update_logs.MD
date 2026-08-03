# Version 1.1.4

Title: Footer slogan and email contact panel

Date: 2026-08-03

Summary:

Refined footer branding, enlarged the loading screen logo and added a visual
email contact panel.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/constants/brand_assets.ts`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added the existing `ryze_slogan.png` asset to the shared brand asset registry.
- Changed the footer brand image from the icon mark to the RYZE slogan logo.
- Kept footer social actions as three compact circular icon-only controls.
- Increased the loading screen brand mark and adjusted the rotating arrows to
  match the larger scale.
- Changed the email contact card to open a centered overlay panel with blurred
  backdrop, close controls and a structured visual form.
- Added Escape-key and backdrop-close behavior for the email panel.
- Updated design documentation to match the footer and contact interaction.
- Aligned package metadata with patch version `1.1.4`.

Reason:

The footer needed to use the full slogan asset, the loading screen logo was
still too small, and the contact email action needed to behave like a polished
frontend form interaction without introducing backend logic.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.3

Title: Logo rendering and footer simplification

Date: 2026-08-03

Summary:

Adjusted the brand logo rendering and simplified the footer layout according to
the latest visual direction.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Removed the visual wrapper, background, border, shadow and crop from the RYZE
  logo component.
- Rendered the original logo asset directly in the navbar, footer and loading
  screen.
- Simplified the footer to a logo plus three small circular social/contact icon
  buttons.
- Removed the footer contact cards and future social links label.
- Updated the design documentation footer specification.
- Aligned package metadata with patch version `1.1.3`.

Reason:

The RYZE logo asset should be displayed directly without an artificial container,
and the footer needed a cleaner minimal structure with only the brand and social
icons.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.2

Title: Navbar and footer visual correction

Date: 2026-08-03

Summary:

Corrected the navbar, hero first viewport and footer layout after visual review,
with a much larger rendered RYZE logo and a cleaner footer structure.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_mark/**`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.tsx`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/navbar/navbar.tsx`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Added a reusable `BrandMark` component that crops and zooms the original
  `ryze_icon.png` asset so the logo is visibly large instead of appearing tiny.
- Applied the enlarged brand mark to the navbar, footer and loading screen.
- Rebuilt the navbar layout with a cleaner full-width header, larger brand
  area, stronger navigation pill and larger profile button.
- Refined the hero first viewport spacing and visual balance below the navbar.
- Rebuilt the footer into a two-column brand/contact layout with structured
  contact rows and subtle future social placeholders.
- Increased footer contact icon sizes and improved footer alignment on desktop
  and mobile.
- Aligned package metadata with patch version `1.1.2`.

Reason:

The previous navbar and footer still looked visually weak because the logo asset
was rendered with too much surrounding image area and the footer contact/social
layout felt unbalanced. This change improves brand visibility, hierarchy and
overall polish while preserving the documented RYZE visual identity.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

# Version 1.1.1

Title: Frontend visual refinement

Date: 2026-08-03

Summary:

Improved the initial frontend visual presentation based on design review
feedback, with stronger brand scale, better navigation polish, higher button
contrast and correct branded contact icons.

Files modified:

- `.gitignore`
- `package.json`
- `package-lock.json`
- `main/src/frontend/src/components/brand_icon/**`
- `main/src/frontend/src/components/button/button.tsx`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.tsx`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/styles/global.css`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Increased the RYZE brand icon size in the navbar, footer and loading screen.
- Increased service and contact icon sizing for stronger visual hierarchy.
- Removed the large top gap above the hero by tightening hero and navbar
  spacing.
- Improved the navbar with a stronger branded active state, larger profile and
  menu icons, and a more polished glass surface.
- Added reusable branded SVG icons for Gmail, Instagram and WhatsApp.
- Replaced generic contact icons in the contact section and footer with the
  correct branded icons.
- Improved dark CTA button contrast by adding light and light-outline button
  variants.
- Refined the page background, footer and CTA band styling for a cleaner,
  premium RYZE aesthetic.
- Narrowed the root `how_to_run.MD` ignore rule so `docs/how_to_run.MD` can be
  tracked.
- Aligned package metadata with the project patch version.

Reason:

The first visual pass had weak brand scale, low-contrast CTA links on dark
sections, generic social icons and excessive spacing above the hero. These
adjustments improve clarity, accessibility and visual consistency while keeping
the documented RYZE color palette and frontend architecture.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- `npm audit --json` reports zero vulnerabilities.

---

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
