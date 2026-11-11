# RYZE
# Design System

Version: 2.104.0

---

# Source of Truth

The **Hero section** is the official visual reference for the entire website.

Every page, section and component must feel like the same product as the Hero.
Do not invent alternate palettes, decorative systems or competing visual styles.

Runtime tokens live in:

`main/src/frontend/src/styles/global.css`

Page-level layout rules also live in:

`docs/PAGES/design.MD`

When those documents disagree with older notes, this palette and the Hero win.

---

# Official Color Palette

## Backgrounds

| Token | Value | CSS variable |
|---|---|---|
| Background | `#05070B` | `--bg-primary` |
| Secondary | `#0A1018` | `--bg-secondary` |
| Elevated Surface | `#101826` | `--surface` |
| Cards | `#141F2D` | `--card` |
| Card Hover | `#182638` | `--card-hover` / `--surface-hover` |

## Accents

| Token | Value | CSS variable |
|---|---|---|
| Primary Accent | `#18D8FF` | `--accent` |
| Secondary Accent | `#2A8CFF` | `--accent-secondary` |
| Gradient | `linear-gradient(90deg, #18D8FF, #2A8CFF)` | `--accent-gradient` |
| Glow | `rgba(24, 216, 255, 0.35)` | `--accent-glow` |

## Text

| Token | Value | CSS variable |
|---|---|---|
| Primary | `#FFFFFF` | `--text-primary` |
| Secondary | `#B8C4D8` | `--text-secondary` |
| Muted | `#7F8EA3` | `--text-muted` |

## Borders

| Token | Value | CSS variable |
|---|---|---|
| Default | `rgba(255, 255, 255, 0.08)` | `--border` / `--border-color` |
| Hover | `rgba(24, 216, 255, 0.35)` | `--border-hover` |

---

# CSS Token Rules

Every UI surface must consume CSS variables.

Hardcoded palette hex values in component CSS should be avoided.
Prefer `var(--token)` so one change updates the whole product.

Exceptions:

- Official third-party brand marks (Gmail / Instagram / WhatsApp)
- Pure white used only for rare light button variants
- Soft `rgba(...)` washes derived from the official cyan / blue / near-black values
- Footer visual treatment (intentionally preserved)

Apply these tokens across Hero, Services, Contact, Our Vision, pricing cards,
dashboard previews, forms, modals, inputs, navigation and every shared section.

Never introduce:

- Teal / green-blue accents
- Competing purple / cream / newspaper aesthetics
- Random decorative SVG systems unrelated to the product

---

# Atmosphere Rules

Empty space may be filled only with:

- Very soft gradients
- Large radial lighting / cyan nebula
- Subtle particles
- Ambient glow
- Depth / soft shadows
- Faint planetary arcs and extremely subtle grid/noise

Do not add:

- Random circles
- Random dotted curves
- Random SVG clutter
- Floating icon clusters
- Large abstract blobs

---

# Platform Phone Mockup

Since v2.104.0 the public Home presents illustrative app screens, designed after
the owner's phone reference (`main/src/resources/img/hero/start_sec_bg.png`), on
a 3D phone and laptop (desktop) or CSS device frames (mobile); see "Public
Website Layer". The rules below describe the earlier illustrated phone and apply
only if it is reintroduced.

The Start Faster phone is a premium product mockup, not a stretched screenshot.

Requirements:

- Detailed original screen UI kept intact (session, week, exercises, metrics,
  navigation, streak, ready gauge)
- Sized to fit the section — never oversized relative to the layout
- Realistic perspective and soft blue ambient glow
- Realistic reflections and premium depth
- Complete UI kept inside the screen
- No cropped or simplified interface
- No floating marketing cards around the phone
- Process steps under the copy and one faint planetary arc are part of the
  official Start Faster composition

---

# Consistency Checklist

Before shipping UI work, scroll the site and ask:

1. Does this section share the Hero palette?
2. Are backgrounds, accents and text using global tokens?
3. Would a visitor think another designer built this block?

If yes to 3, fix it before merge.

---

# Public Website Layer

The public pages in the main navigation (Home, Our Vision, Contact, Feedback,
Services, login/registration and the public under-development placeholders) use
an opt-in layer in `main/src/frontend/src/components/public_ui/`. The global
tokens above are unchanged; the footer, the Exercise Library and the Home program
section follow the same look within the owner's limits (see
`docs/AI/frontend-revamp.md`).

## One principle

The website is built from the same materials as the app:

- one continuous logo-navy page; sections are transparent, and the page carries
  a repeating rhythm of soft blue light, so there are no seams or colour jumps
  between sections;
- cards are the app's blue glass surfaces (`surface`: a translucent royal-to-navy
  gradient with a thin light-blue border);
- cyan is the light: headline continuations, primary buttons, icons, focus;
- one bright element only: the light-blue wave panel (`toneLight` + `lightPanel`
  + `Waves`), used for calls to action and the authentication side panel.
- the login page's look is the model for moments that should draw attention
  (owner, 2026-11-11): the split panel (`split` + `splitPane` + `splitLight`
  in `public_ui`) joins a navy glass half to the light-blue wave half in one
  card. It is used on Services (route guide and "Start free today"), and the
  wave panel's colours light the RYZE tile in the Our Vision hero. Use it for
  conversion and comparison moments, not as a section background.

## Verified logo colours

Sampled from the official artwork in `main/src/resources/img/logo/` (the logo
is authoritative; the layer uses these values):

| Role | Value | Where in the logo |
|---|---|---|
| Navy | `#02041F` | Outlines |
| Royal blue | `#072A77` | Ring shading |
| Ring blue | `#0481DC` | The broken ring |
| Z cyan | `#00CBFF` (bright `#50E2FE`) | The Z and slogan dots |
| Chrome | `#E4E9F1` | The R and the wordmark |

The global `#18D8FF` is close to the Z cyan; the global `#2A8CFF` and the
charcoal `#05070B` do not appear in the logo.

## Scope rules

- A page opts in by passing `public_ui.module.css`'s `page` class to
  `PageWrapper`. It declares the `--r-*` custom properties and paints the navy
  page with its light rhythm, never inherited colour, font or descendant rules,
  because pages render protected cards (Services, Home programs) inside it.
- Typography and colour come from two tone classes placed on public elements
  only, never on an ancestor of a protected card: `toneNavy` (text on the navy
  page; no background) and `toneLight` (ink text on the light-blue panel, royal
  continuations, ≥4.5:1 for small text across the gradient).
- Do not introduce other section colours (paper, white, cyan or royal bands):
  the owner rejected them as making parts of the site look unrelated.
- No background images, textures or decorative background video. Gradients
  built from the logo colours are allowed.
- `PublicAction` (primary cyan, ink navy, ghostOnDark, ghostOnLight, quiet) is
  separate from the shared `Button`, which protected pages use.
- Form fields on navy: dark fill, `#4D6FA6` border (≈3.5:1), cyan focus ring;
  the same treatment on the contact form and the authentication page.

## Typography

Archivo (variable, SIL Open Font License, self-hosted in
`main/src/resources/fonts/archivo/` as "RYZE Archivo"; the CSP allows
`font-src 'self'`). Expanded widths echo the RYZE wordmark:

- Display: width 125%, weight 820, line height 1.02, `clamp(2.2rem, 11.2vw, 6.25rem)`, from
  1024px `clamp(3.5rem, 5.6vw, 5.75rem)`
- Heading: width 120%, weight 800, line height 1.05, `clamp(2.05rem, 7.6vw, 4.25rem)`, from
  1024px `clamp(2.4rem, 4.2vw, 3.9rem)`
- Title, lead (≤38ch), body and caption at width 100–112%
- Labels: small uppercase, wide tracking, with a short rule, in the tone's
  highlight colour
- Headlines carry a brand-coloured continuation (`hl`), restored from the
  original site ("Train smarter. Progress further.", "Real results.")

## 3D and product visuals

- Desktop Home only: one Three.js scene (`components/home_story/story_scene.ts`)
  driven by scroll, shown on two pinned stages that share one canvas (it moves
  to whichever stage is on screen): the opening (the copy fades while the
  Earth turns to the centre, then the stage scrolls away with the page) and the
  device chapters (the stage scrolls into view with the phone already in place
  and scrolls away after "Get started"). The programs sit between the stages in
  normal page flow, at least one viewport tall, so no 3D is ever behind them.
  The scene is loaded with a dynamic import after a capability check
  (`hooks/use_immersive_story.ts`: ≥1024×680, fine pointer, WebGL2, no
  reduced-motion preference). Procedural geometry only; default GPU preference;
  pixel ratio capped at 1.5 and no MSAA on high-DPI screens; shaders compiled
  before the first frame; full frame rate only while scroll or pointer moves
  (about 30 fps for the idle motion); rendering only while on screen; full
  disposal on unmount and on any creation failure, which falls back to the
  stacked layout. Objects scale with the stage aspect ratio.
- Scene content: the Earth with real continents (Natural Earth land mask,
  `main/src/resources/img/globe/`) and RYZE routes leaving Lisbon with light
  pulses and the official icon on a badge at home; a modelled phone (titanium
  frame, glass, island, buttons, camera module, the logo on the back, blue edge
  light as in the owner's reference) and laptop (black keys with backlit
  legends, glass trackpad, hinge, display notch, the logo on the lid). The
  laptop screen is a 2560×1600 image of a 1024×640 layout with large type, and
  horizontal parts barely reflect the studio environment (seen at a grazing
  angle they would mirror white). Train and Fuel show three cards lifted out of
  the phone screen (`callout_*.webp`, shared with the mobile layout); Evolve
  places laptop and phone side by side under a "Synced" badge with a soft
  breathing light (no line between the devices). Each Train and Fuel card is
  joined to the screen element it shows: the element is outlined on the phone
  screen, and a rounded leader runs from the phone's edge at that element's
  height to the card; outline, leader and card arrive in that order, and a
  light pulse then travels along each leader. Cards are ordered like their
  elements, so leaders never cross. Screens and cards are sampled one mip level sharper (map
  LOD bias) and the laptop/Evolve-phone screens use large type for the size at
  which they appear. Each stage's backdrop dissolves into the page where the
  stage meets it, so the page's light continues without an edge. The backdrop
  stays in the navy family through every chapter. All materials, including the
  hidden diagram and badge ones, are compiled before the first frame.
- Mobile, tablet, reduced motion and no-WebGL use lightweight equivalents
  (`components/brand_globe/`: an SVG globe with the same routes;
  `components/showcase/`: phone and laptop frames). They never download Three.js.
- Device screens are illustrative RYZE screens designed in the app's style with
  sample data (`main/src/resources/img/showcase/`) and are labelled as such.
  They only show existing features. Do not show sample prices.

## Motion

Scroll-driven chapter transitions on the desktop Home story; subtle reveals
elsewhere through the shared `Reveal`; hover lifts on cards. Everything
respects `prefers-reduced-motion` (the 3D story is replaced by the stacked
layout).

---

# Footer

The footer keeps its layout, link columns, slogan image and copy. Since the
owner's 2026-11-10 revision its surface, lines and type follow the public
design: logo navy with blue light, a thin cyan line along the top, cyan column
headings and Archivo.
