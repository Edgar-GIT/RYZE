# Version 2.2.44

Title: Stretch Our Vision hero + remodel Continuous Evolution

Date: 2026-08-05

Summary:

Stretched the hero `our_view.png` to the top of the page. Remodeled Continuous
Evolution to match the mock: `conti_bg.png` full-bleed background, left intro,
and a 2×3 grid of glowing cards with icon / title / copy / chevron.

Files modified:

- `main/src/resources/img/hero/conti_bg.png` (placeholder until real asset)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for hero top coverage and a Continuous Evolution section matching
the attached mock with `conti_bg.png`.

---

# Version 2.2.43

Title: Liven Our Vision page — quieter chips, no orbit rings

Date: 2026-08-05

Summary:

Made hero chips smaller with soft static shadows (no pulse). Removed planet
orbit lines from the roadmap. Boosted ambient light, card color, shadows and
hover motion across the page.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to stop the blinking chips, drop the orbit lines, and make the page
feel less dead.

---

# Version 2.2.42

Title: Restyle Our Vision hero capability chips

Date: 2026-08-05

Summary:

Rebuilt hero chips to match the neon glass mock: white labels, glowing icon
wells, cyan border bloom, staggered entrance and ambient pulse animation.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Previous chips were low-contrast and static; user asked for the mock style with
light and motion.

---

# Version 2.2.41

Title: Restore Our Vision hero capability chips

Date: 2026-08-05

Summary:

Added AI / Training / Nutrition / Recovery / Progress chips above the Our
Vision button in the hero, matching the provided block mock.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to put the capability blocks back above the hero CTA.

---

# Version 2.2.40

Title: Fix Our Vision hero crop, timeline nodes and lighting

Date: 2026-08-05

Summary:

Stopped cropping the bottom of `our_view.png` in the hero. Aligned roadmap
timeline dots to the rail. Increased page ambient light and enlarged / enriched
the bottom-left orbital planet graphic.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User reported clipped hero artwork, misaligned timeline nodes, and asked for
more light plus a stronger orbit visual.

---

# Version 2.2.39

Title: Redesign Our Vision roadmap to match mock

Date: 2026-08-05

Summary:

Rebuilt the “Where RYZE is going” section as a two-column layout with left
intro + orbital planet glow and a right vertical timeline of milestone cards
(icon, index, phase, title, chevron).

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Match the provided roadmap mock exactly.

---

# Version 2.2.38

Title: Simplify Our Vision hero copy

Date: 2026-08-05

Summary:

Removed the five capability badges under the hero lead. Kept only the Our
Vision jump link. Slightly reduced the hero section height so the background
image takes less vertical space.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked for a cleaner hero: text + CTA only, with a slightly smaller image.

---

# Version 2.2.37

Title: Replace 3D planet with our_view.png hero background

Date: 2026-08-05

Summary:

Removed the Three.js RYZE Planet component and all related dependencies.
Our Vision hero now uses `our_view.png` as a full-bleed background image,
matching Training / Start Faster section pattern, with left copy only.

Files modified:

- `main/src/frontend/src/components/ryze_planet/**` (removed)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The 3D planet path was unstable; ship a clean image-backed hero instead.

---

# Version 2.2.36

Title: Fit full planet in canvas, remove sparks, blink RYZE, shooting stars

Date: 2026-08-05

Summary:

Enlarged the planet canvas (~52rem) and pulled the camera back so the whole
sphere fits without clipping. Removed blinking surface squares. Added gold
shooting-star orbits and a 1s / 0.3s-fade RYZE wordmark blink. Violet rim
palette to match the reference.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Planet was clipped by a too-small frustum/canvas; sparks and missing stars
did not match the brief.

---

# Version 2.2.35

Title: Restore original Three.js rotating RYZE Planet (v1)

Date: 2026-08-05

Summary:

Restored the first-version 3D planet (commit `planet` / 91cd3b8): shader
sphere, slow rotation, atmosphere, hologram mark + RYZE wordmark, particles.
Removed EffectComposer bloom (it caused the opaque canvas square). Slightly
boosted lighting/surface so the ball stays readable.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

User asked to bring back the original rotating 3D planet from the first
working version.

---

# Version 2.2.34

Title: Fix invisible planet — opaque SVG sphere

Date: 2026-08-05

Summary:

Rebuilt the planet as an opaque SVG ball with solid purple fill, rim light,
gold circuits and a CSS fallback background so the sphere cannot blend into
the page. Previous CSS gradients faded to page-black and made the ball
invisible.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Screenshot showed only RYZE text + haze because the sphere color matched the
background.

---

# Version 2.2.33

Title: Replace broken WebGL planet with CSS/SVG RYZE Planet

Date: 2026-08-05

Summary:

Removed the failing Three.js canvas (opaque/blurred square, invisible sphere)
and rebuilt the Our Vision planet as a reliable CSS/SVG composition matching
the reference: large violet sphere, gold circuit lines, RYZE blink, shooting
stars, nebula glow, and entrance animation.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx` (removed)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

WebGL path kept failing visually; the hero needs a guaranteed circular planet
aligned to the attached concept.

---

# Version 2.2.32

Title: Restore visible RYZE Planet sphere with reliable materials

Date: 2026-08-05

Summary:

Rebuilt the planet core with MeshStandardMaterial + panel texture so the
sphere is actually visible (custom shader was rendering near-black against
the page). Kept cyan/blue/violet site palette, orbital trails, shooting
stars, RYZE blink, and entrance animation. Removed broken bloom/shader path.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts` (removed)
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Users only saw a rectangular canvas with faint orbits because the sphere
silhouette matched the page background.

---

# Version 2.2.31

Title: Fix planet canvas box, site palette, RYZE blink and entrance

Date: 2026-08-05

Summary:

Removed EffectComposer bloom (it painted an opaque purple rectangle over the
transparent canvas and hid the sphere). Restored the visible planet with
site cyan/blue/violet colors, RYZE blink (1s cycle, 0.3s fade-out), and a
CSS entrance fade/scale for the planet.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The hero visual was broken by postprocessing alpha; palette and motion needed
to match the RYZE site language.

---

# Version 2.2.30

Title: Scale up RYZE Planet hero presence

Date: 2026-08-05

Summary:

Enlarged the Our Vision planet (container ~46rem, larger radius, closer
camera) and gave the visual column more grid space so the sphere dominates
the first viewport.

Files modified:

- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Planet still read too small against the reference; it needs hero-level scale.

---

# Version 2.2.29

Title: Refine planet size, shooting-star gold and RYZE wordmark

Date: 2026-08-05

Summary:

Enlarged and nudged the planet up/right; replaced gold burst lines with a
single shooting-star meteor; removed blinking surface sparks; restored a
clear intermittent gold “RYZE” wordmark via Troika Text.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Match the intended meteor motion and make the gold RYZE wordmark actually
visible.

---

# Version 2.2.28

Title: Tune RYZE Planet size, violet glow, gold streaks and wordmark

Date: 2026-08-05

Summary:

Made the Our Vision planet smaller and shifted left; switched glow to ultra
neon violet; slowed rotation to ~95s/turn; added occasional high-speed gold
surface streaks and a fluorescent gold “RYZE” wordmark that fades in and out.

Files modified:

- `main/src/frontend/src/components/ryze_planet/planet_scene.tsx`
- `main/src/frontend/src/components/ryze_planet/planet_shaders.ts`
- `main/src/frontend/src/components/ryze_planet/ryze_planet.module.css`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Refine the planet signature toward the requested neon-violet / gold pulse feel.

---

# Version 2.2.27

Title: Add real-time RYZE Planet 3D hero to Our Vision

Date: 2026-08-05

Summary:

Replaced the empty Our Vision hero visual with a lazy-loaded Three.js /
React Three Fiber planet: dark metallic tech sphere, ~50s rotation, Fresnel
cyan rim, subtle particles, soft bloom, and a centered holographic RYZE mark
+ wordmark. No orbiting modules in this phase.

Files modified:

- `main/src/frontend/src/components/ryze_planet/**` (new)
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Our Vision needed a premium tech-first signature visual that communicates
RYZE as future fitness software, not a traditional gym landing page.

---

# Version 2.2.26

Title: Start Our Vision hero rebuild — new copy, clear right visual

Date: 2026-08-05

Summary:

Rewrote the Our Vision first section to match the reference: “THE FUTURE OF
FITNESS” eyebrow, accent “software.” headline, adaptive-training lead,
horizontal AI / Training / Nutrition / Recovery / Progress badges, and an Our
Vision jump link. Removed the orbit / node hero visual so the right side can
receive a new composition next.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/our_vision_page.tsx`
- `main/src/frontend/src/pages/our_vision_page/our_vision_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Full Our Vision page redo starts with locking the hero copy and clearing the
outdated right-side system diagram.

---

# Version 2.2.25

Title: Rebuild Progress section with analytics card and feature badges

Date: 2026-08-04

Summary:

Replaced the Progress showcase/demo with a dedicated section: left accent copy
and vertical badges, right glass progress card matching the attached chart /
summary / metric tiles design, with reveal and float animations.

Files modified:

- `main/src/frontend/src/components/progress_section/progress_section.tsx`
- `main/src/frontend/src/components/progress_section/progress_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Progress needed the same complete treatment as Training and Nutrition.

---

# Version 2.2.24

Title: Rebuild Nutrition section with accent copy, badges and glass card

Date: 2026-08-04

Summary:

Replaced the Nutrition showcase/demo with a dedicated section: left copy with
accent highlight and vertical feature badges (Training-style), right glass
nutrition card matching the attached kcal/macros/recommendation design, plus
reveal and float animations.

Files modified:

- `main/src/frontend/src/components/nutrition_section/nutrition_section.tsx`
- `main/src/frontend/src/components/nutrition_section/nutrition_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Nutrition needed the same richer left-column treatment as Training and a card
matching the provided reference.

---

# Version 2.2.23

Title: Rebuild Training section with background artwork and feature badges

Date: 2026-08-04

Summary:

Replaced the Training showcase/demo block with a full-bleed section using
`training_sec_bg.png`. Left column now matches the reference copy hierarchy and
vertical feature badges, with reveal animations kept.

Files modified:

- `main/src/frontend/src/components/training_section/training_section.tsx`
- `main/src/frontend/src/components/training_section/training_section.module.css`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Training needed the same artwork-driven treatment as Start Faster, with left
content matching the attached reference.

---

# Version 2.2.22

Title: Start Faster background image test without phone

Date: 2026-08-04

Summary:

Temporarily removed the phone mockup from Start Faster. Section now uses
`start_sec_bg.png` like the Hero full-bleed background. Kept copy, CTAs,
process badges and reveal animations. Shifted left content inward from the edge.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Layout/phone implementation was not matching the reference; this is a visual
test with the provided background asset.

---

# Version 2.2.21

Title: Fix Start Faster spacing, phone atmosphere and section palette continuity

Date: 2026-08-04

Summary:

Increased page gutters so Start Faster copy and phone are no longer glued to the
edges. Centered the phone in its column with cinematic glow, stars and planetary
arc matching the phone reference. Enriched the phone status/nav chrome. Removed
hard section tone jumps so the site scrolls as one continuous palette.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/*`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- Shared cards/CTA/contact/showcase CSS modules
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Layout spacing and inter-section contrast still broke visual continuity.

---

# Version 2.2.20

Title: Pixel-match Start Faster + migrate official RYZE palette

Date: 2026-08-04

Summary:

Rebuilt Start Faster to match the attached reference (~100vh, 45/55 grid,
gradient headline, process steps, hero-like atmosphere). Restored the detailed
phone screen UI at a correct non-oversized scale with premium glow and
perspective. Migrated the global token system to the new Hero palette across the
site (footer preserved).

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/*`
- Shared component/page CSS modules (palette migration)
- `docs/INSTRUCTIONS/design.MD`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The section diverged from the approved mockup, the phone was oversized, and the
site still needed the final official palette applied consistently.

---

# Version 2.2.19

Title: Match Start Faster section to reference mockup

Date: 2026-08-04

Summary:

Rebuilt the Start Faster / phone section to match the attached reference:
accent headline phrase, process steps, thin connecting arc, and a larger
front-facing premium phone with circular ready gauge, glass reflection and
soft blue ambient glow.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The live phone section diverged from the approved reference composition.

---

# Version 2.2.18

Title: Official Hero token system + restore premium Platform phone

Date: 2026-08-04

Summary:

Restored the Start Faster phone as a large premium product mockup and removed
the random decorative graphics. Introduced the official global CSS token system
from the Hero palette and migrated shared UI surfaces to consume it. Updated
design, project and procedure documentation so future work follows one palette.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- Shared component/page CSS modules (token migration)
- `docs/INSTRUCTIONS/design.MD`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/INSTRUCTIONS/arquitecture.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform pass was a visual regression, and the site still felt like
multiple palettes. Hero is now the only official design reference.

---

# Version 2.2.17

Title: Unify global palette to Hero design system

Date: 2026-08-04

Summary:

Replaced the legacy teal/blue-grey token set with the Hero-derived futuristic
palette across global CSS and shared UI surfaces. Updated design documentation
as the new visual source of truth. Footer styling was intentionally preserved.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- Multiple component/page CSS modules (palette sync)
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/procedure.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The Hero already defined the premium futuristic identity, but later sections
still used the old palette and broke visual continuity.

---

# Version 2.2.16

Title: Rebuild Start Faster around phone atmosphere + page reveals

Date: 2026-08-04

Summary:

Removed floating Platform cards and made the phone the sole hero, surrounded by
atmospheric grid/glow/trails and tiny decorative fragments. Shortened Start
Faster copy, added a secondary link, and introduced shared scroll reveals across
Home and Services.

Files modified:

- `main/src/frontend/src/components/platform_section/**`
- `main/src/frontend/src/components/reveal/reveal.tsx` (new)
- `main/src/frontend/src/components/showcase_section/showcase_section.tsx`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/pages/services_page/services_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Floating cards still felt AI-generated and competed with the phone; Home and
Services needed the same premium scroll motion language as Our Vision.

---

# Version 2.2.15

Title: Replace About Us with Our Vision page

Date: 2026-08-04

Summary:

Replaced the About Us placeholder with a premium `/our-vision` page focused on
platform vision, mission, roadmap and technology. Updated navigation/footer
labels, redirected `/about-us`, and synchronized design/project/run docs.

Files modified:

- `main/src/frontend/src/pages/our_vision_page/**` (new)
- `main/src/frontend/src/routes/app_routes.tsx`
- `main/src/frontend/src/constants/navigation.ts`
- `main/src/frontend/src/components/footer/footer.tsx`
- `docs/PAGES/design.MD`
- `docs/INSTRUCTIONS/project.MD`
- `docs/how_to_run.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The old About Us framing no longer matched RYZE as a technology company
reinventing fitness; the public page needed to sell vision, not founders.

---

# Version 2.2.14

Title: Fix Start Faster card orbit layout around tilted phone

Date: 2026-08-04

Summary:

Rebuilt the Platform visual composition as a 3-column CSS grid so Nutrition,
This week and Recovery stay anchored beside the phone instead of drifting with
absolute edge positioning. Increased phone tilt to `rotateY(-40deg)`.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Absolute left/right card placement made the panels look scattered and poorly
aligned relative to the centered phone.

---

# Version 2.2.13

Title: Rebalance Start Faster phone and floating cards

Date: 2026-08-04

Summary:

Increased the phone `rotateY` to `-30deg`, moved Nutrition to the lower-left of
the device, kept This week and Recovery on the right, and removed all connector
lines. Mobile stacks or hides the cards so they never overlay the phone.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The visual needed a stronger 3D phone angle and more balanced card weight on
both sides of the device, without connector lines.

---

# Version 2.2.12

Title: Rebuild Start Faster visual around a 3D phone

Date: 2026-08-04

Summary:

Replaced the crowded Platform visual with a tilted almost-3D phone (realistic
proportions and RYZE UI inside) and three emerging glass panels: This week,
Nutrition and Recovery. Soft dashed connectors and slow float motion remain.

Files modified:

- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous right-side composition still felt busy; the section needed a
clearer product focal point with only the strongest supporting panels.

---

# Version 2.2.11

Title: Enable WhatsApp community contact link

Date: 2026-08-04

Summary:

Activated WhatsApp across contact surfaces with the official community invite
URL. Updated homepage preview chips, Contact page channel row and footer
Resources link. Removed disabled/coming-soon states.

Files modified:

- `main/src/frontend/src/constants/contact.ts`
- `main/src/frontend/src/components/contact_preview/contact_preview.tsx`
- `main/src/frontend/src/components/contact_preview/contact_preview.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/footer/footer.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

WhatsApp community is live and should be reachable from every contact entry
point.

---

# Version 2.2.10

Title: Show plan price above service card CTA

Date: 2026-08-04

Summary:

Added a compact right-aligned price above each service card CTA, using the
existing `ServicePlan.price` field. Applies to both homepage plans preview and
`/services`.

Files modified:

- `main/src/frontend/src/components/service_card/service_card.tsx`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Plan pricing needed to be visible on cards without competing with the CTA.

---

# Version 2.2.9

Title: Remove homepage Athletes testimonials section

Date: 2026-08-04

Summary:

Removed the Athletes/testimonials section from the homepage and deleted the
unused `testimonials_section` component. Updated Home page section order in
design docs.

Files modified:

- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `main/src/frontend/src/components/testimonials_section/**` (removed)
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The testimonials block was no longer wanted on the homepage product tour.

---

# Version 2.2.8

Title: Harmonize homepage palette after Platform redesign

Date: 2026-08-04

Summary:

Aligned the remaining homepage surfaces and interactive components with the new
blue/cyan brand palette and deep background continuity. Kept footer structure
and styling unchanged while improving cross-section visual cohesion.

Files modified:

- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/showcase_section/showcase_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/contact_preview/contact_preview.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

After the Platform and global palette overhaul, several homepage blocks still
used legacy hardcoded tones, creating visible inconsistency between sections.

---

# Version 2.2.7

Title: Rebuild Platform into a hierarchical mobile-first system

Date: 2026-08-04

Summary:

Reworked the homepage Platform section away from the decorative orbit/mind-map
layout into a clearer mobile-first composition: goals bar, primary phone
surface, and grouped support panels for training, nutrition, recovery,
progress, PRs and coach feedback.

Files modified:

- `main/src/frontend/src/components/platform_section/platform_section.tsx`
- `main/src/frontend/src/components/platform_section/platform_section.module.css`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.tsx`
- `main/src/frontend/src/components/platform_section/ecosystem_visual.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform section still behaved like a decorative mental map with
weak hierarchy, fragile layout rules and poor mobile behavior.

---

# Version 2.2.6

Title: Platform ecosystem visualization redesign

Date: 2026-08-04

Summary:

Replaced the generic Platform dashboard demo with a premium ecosystem
visualization: floating phone, glass orbit cards, connection lines, parallax and
slow motion. Updated Platform messaging and removed DashboardDemo.

Files modified:

- `main/src/frontend/src/components/platform_section/**` (new)
- `main/src/frontend/src/components/product_demos/dashboard_demo.tsx` (removed)
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous Platform section read like an AI-generated admin dashboard and
failed to communicate that RYZE is a connected fitness operating system.

---

# Version 2.2.5

Title: Point image imports to hero and plan_cards

Date: 2026-08-04

Summary:

Updated hero and service plan image imports after removing `img/testing` and
moving assets to `img/hero` and `img/plan_cards`.

Files modified:

- `main/src/frontend/src/components/hero/hero.tsx`
- `main/src/frontend/src/constants/services.ts`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Asset paths broke after the testing folder was deleted and images were renamed
and relocated.

---

# Version 2.2.4

Title: Premium embedded Contact experience

Date: 2026-08-04

Summary:

Rebuilt Contact as a dense two-column SaaS layout with compact channel rows and
an embedded glass form (no modal). Home now uses a compact Get in touch preview.
Fixed the mismatched top strip on `/contact`.

Files modified:

- `main/src/frontend/src/components/contact_section/**`
- `main/src/frontend/src/components/contact_form/**` (new)
- `main/src/frontend/src/components/contact_preview/**` (new)
- `main/src/frontend/src/components/email_contact_panel/**` (removed)
- `main/src/frontend/src/pages/contact_page/**`
- `main/src/frontend/src/pages/home_page/home_page.tsx`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The previous contact cards + modal felt like a generic template and created a
visible top color seam on the Contact page.

---

# Version 2.2.3

Title: Living dark atmospheric background

Date: 2026-08-04

Summary:

Added stronger cyan nebulae, soft vignette shadows, breathing glow and translucent
section washes so the dark UI feels alive instead of flat/static. Footer unchanged.

Files modified:

- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/section_wrapper/section_wrapper.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

Solid opaque section fills were covering the animated atmosphere and making the
page feel dead.

---

# Version 2.2.2

Title: Rounded pill navbar and larger brand mark

Date: 2026-08-04

Summary:

Changed the navbar from a rectangular strip to a floating pill bar and enlarged
the navigation brand mark.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

The rectangular navbar felt too rigid; a rounded capsule and a stronger icon
match the premium SaaS navigation pattern.

---

# Version 2.2.1

Title: Full visual consistency pass

Date: 2026-08-04

Summary:

Finished the SaaS consistency pass: thinner navbar, soft section tone blends,
premium service cards, and purged remaining teal/grey-blue accents from demos,
panels and secondary pages. Footer left unchanged.

Files modified:

- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/section_wrapper/**`
- `main/src/frontend/src/components/showcase_section/**`
- `main/src/frontend/src/components/service_card/**`
- `main/src/frontend/src/components/product_demos/product_demos.module.css`
- `main/src/frontend/src/components/email_contact_panel/email_contact_panel.module.css`
- `main/src/frontend/src/components/loading_screen/loading_screen.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.tsx`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/pages/home_page/**`
- `main/src/frontend/src/pages/services_page/services_page.module.css`
- `main/src/frontend/src/pages/under_development_page/under_development_page.module.css`
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`

Reason:

After adopting the official dark cyan identity, leftover teal accents and
inconsistent section/card treatments still broke the Linear/Apple/Vercel feel.

---

# Version 2.2.0

Title: Official dark cyan SaaS visual identity

Date: 2026-08-03

Summary:

Fixed the clipped navbar, reduced the brand mark size, and adopted the official
near-black / electric cyan design system across the product UI while preserving
the existing footer.

Files modified:

- `main/src/frontend/src/styles/global.css`
- `main/src/frontend/src/components/navbar/navbar.module.css`
- `main/src/frontend/src/components/brand_mark/brand_mark.module.css`
- `main/src/frontend/src/components/button/button.module.css`
- `main/src/frontend/src/components/animated_background/**`
- `main/src/frontend/src/components/glass_panel/glass_panel.module.css`
- `main/src/frontend/src/components/card/card.module.css`
- `main/src/frontend/src/components/cta_band/cta_band.module.css`
- `main/src/frontend/src/components/testimonials_section/testimonials_section.module.css`
- `main/src/frontend/src/components/service_card/service_card.module.css`
- `main/src/frontend/src/components/contact_section/contact_section.module.css`
- `main/src/frontend/src/components/hero/hero.module.css`
- `main/src/frontend/src/components/footer/footer.module.css` (restored)
- `docs/PAGES/design.MD`
- `docs/REGISTRY/update_logs.MD`
- `package.json`
- `package-lock.json`

Reason:

The hero established the official RYZE SaaS look. The rest of the product needed
to match that darker high-contrast cyan system, and the navbar needed room so it
is no longer cut off at the top.

---

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
