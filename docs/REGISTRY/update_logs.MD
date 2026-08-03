# Version 2.1.9

Title: Dark contrast palette and navbar polish

Date: 2026-08-03

Summary:

Aligned the global palette to a deeper navy / electric cyan system and cleaned
the navbar into a transparent integrated bar with stronger text and button
contrast.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/animated_background/animated_background.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/footer/footer.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The reference hero look depends on near-black surfaces, white text and cyan CTAs.
The navbar also needed to feel integrated instead of heavily framed.

---

# Version 2.1.8

Title: Rebuild hero as background artwork

Date: 2026-08-03

Summary:

Rebuilt the hero from scratch with a single container, left-side content only,
and an absolute oversized artwork background layer starting mid-viewport with a
left fade mask.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Previous approaches kept treating the illustration as layout content. The
intended design is immersive background artwork like Linear / Apple / Vercel.

---

# Version 2.1.7

Title: Oversized hero background artwork

Date: 2026-08-03

Summary:

Reworked the hero illustration into an oversized CSS background layer using the
premium SaaS pattern: absolute right-anchored artwork with `background-size:
cover`, intentional overflow and left-edge masking.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Fitting the full image inside a container made the artwork look embedded. The
hero needs immersive oversized background artwork like Linear / Raycast /
Vercel landing pages.

---

# Version 2.1.6

Title: Atmospheric hero illustration blend

Date: 2026-08-03

Summary:

Converted the hero artwork into an absolute right-side atmospheric layer with
left-edge CSS masking so it blends into the page instead of appearing as an
embedded image.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Reason:

The illustration was still reading as a separate image block. It needs to feel
like integrated SaaS landing artwork that never pushes the text column.

---

# Version 2.1.5

Title: Hero split layout for illustration

Date: 2026-08-03

Summary:

Rebuilt the hero as a balanced 45/55 content/illustration grid so the artwork
stays on the right, scales with `object-fit: contain`, and never overlaps text.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Treating the illustration as a full-bleed cover background caused cropping and
overlap. The artwork needs a dedicated right column that grows wider on large
screens while preserving the full composition.

---

# Version 2.1.4

Title: Right-aligned contained hero artwork

Date: 2026-08-03

Summary:

Scaled the hero artwork down, locked it to the right side with full visibility,
kept the left side as atmosphere only, and cleared the navbar overlap zone.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

Cover-cropping was hiding the right composition and pushing important artwork
under the navbar. The visual needs to stay fully visible, upright and smaller.

---

# Version 2.1.3

Title: Smaller hero image and seamless page blend

Date: 2026-08-03

Summary:

Reduced the hero image scale/height and faded it into a deeper navy page
atmosphere so the cut into the sections below is no longer abrupt.

Files modified:

- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/animated_background.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero artwork ended in a hard horizontal seam against the rest of the site.
The page needed a smaller hero treatment and a continuous galactic surface.

---

# Version 2.1.2

Title: Full-bleed hero image layout

Date: 2026-08-03

Summary:

Corrected the hero so `test4.png` spans the full hero width, with copy on the
left and the image subject visible on the right, and removed the floating cards.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/components/hero/hero.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero image already contains the composed product visual. It should act as a
full-bleed canvas, not as a cropped card with extra floating UI overlays.

---

# Version 2.1.1

Title: Galactic hero composition and richer atmosphere

Date: 2026-08-03

Summary:

Rebuilt the hero into a free-floating brand composition with orbiting glass
cards, enriched the galactic background atmosphere and boxed the athlete
testimonials.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/hero/**`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`

Changes made:

- Removed the rectangular frame around the hero image and treated `test4.png`
  as a free-floating brand centerpiece.
- Added floating glass product cards for training, volume, readiness, nutrition
  and recovery around the hero mark.
- Updated hero copy, CTA labels and trust indicators to match the product-led
  galactic direction.
- Increased cyan/blue atmospheric depth in the global palette and background.
- Added boxed glass testimonial cards.
- Updated design documentation for the hero composition and richer atmosphere.

Reason:

The previous hero still looked boxed and flat. The site needed more life,
color and a technological / galactic product presence without leaving the RYZE
blue palette.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.

---

# Version 2.1.0

Title: Darker product-led homepage redesign

Date: 2026-08-03

Summary:

Moved the public visual identity to a darker premium technology language,
redesigned navbar behaviour and replaced the generic "Why RYZE" feature cards
with product-demo showcase sections.

Files modified:

- `package.json`
- `package-lock.json`
- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/navbar/**`
- `main/src/frontend/src/components/hero/**`
- `main/src/frontend/src/components/showcase_section/**`
- `main/src/frontend/src/components/product_demos/**`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/components/section_title/section_title.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/testimonials_section/**`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/REGISTRY/update_logs.MD`

Removed:

- `main/src/frontend/src/components/why_choose_ryze/**`
- `main/src/frontend/src/components/coaches_section/**`
- `main/src/frontend/src/components/faq_section/**`
- empty `main/src/frontend/src/components/hero_scene/`
- empty `main/src/core/`

Changes made:

- Updated global tokens to a darker surface system with blue as the primary
  accent.
- Softened the animated background and reduced flashy gradient treatment.
- Redesigned the navbar so it is nearly transparent at the top and gradually
  gains opacity, blur and darker glass while scrolling.
- Replaced boxed active navigation states with an animated accent underline.
- Rebuilt the homepage flow as a product tour: dashboard, training, nutrition,
  progress, plans, testimonials, CTA and contact.
- Added reusable showcase and product-demo components that visually demonstrate
  the platform instead of explaining it with feature cards.
- Tightened hero copy, buttons, cards, glass panels and testimonials for a
  calmer premium look.
- Synchronized design, architecture and project status documentation.
- Aligned package metadata with version `2.1.0`.

Reason:

The previous landing still felt like a generic AI marketing page. RYZE needed a
recognizable darker product identity and a homepage that sells through the
interface itself.

Additional notes:

- No backend logic was implemented.
- `npm run build` succeeds.
- Temporary hero testing imagery remains until final brand assets are chosen.

---

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
