# RYZE
# Design System

Version: 2.2.20

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

# Footer

The current footer visual treatment is intentionally preserved unless an
explicit redesign is requested.
