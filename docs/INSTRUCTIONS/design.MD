# RYZE
# Design System

Version: 2.2.18

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
| Background | `#05070C` | `--bg-primary` |
| Secondary | `#09111B` | `--bg-secondary` |
| Surface | `#101826` | `--surface` |
| Elevated Surface | `#151F2E` | `--surface-elevated` / `--surface-hover` |

## Accents

| Token | Value | CSS variable |
|---|---|---|
| Primary Accent | `#2ED8FF` | `--accent` |
| Secondary Accent | `#1A84FF` | `--accent-secondary` |
| Gradient start | `#38E4FF` | part of `--accent-gradient` |
| Gradient end | `#187BFF` | part of `--accent-gradient` |
| Glow | `rgba(46, 216, 255, 0.25)` | `--accent-glow` |

Gradient:

`linear-gradient(135deg, #38E4FF, #187BFF)` → `--accent-gradient`

## Text

| Token | Value | CSS variable |
|---|---|---|
| Primary | `#F4F7FB` | `--text-primary` |
| Secondary | `#A6B2C7` | `--text-secondary` |
| Muted | `#6F7C91` | `--text-muted` |

## Borders

| Token | Value | CSS variable |
|---|---|---|
| Default | `rgba(255, 255, 255, 0.08)` | `--border` / `--border-color` |
| Hover | `rgba(46, 216, 255, 0.35)` | `--border-hover` |

---

# CSS Token Rules

Every UI surface must consume CSS variables.

Required token family:

```css
:root {
  --bg-primary
  --bg-secondary
  --surface
  --surface-hover
  --surface-elevated
  --border
  --border-hover
  --text-primary
  --text-secondary
  --text-muted
  --accent
  --accent-secondary
  --accent-gradient
  --accent-glow
}
```

Hardcoded palette hex values in component CSS should be avoided.
Prefer `var(--token)` so one change updates the whole product.

Exceptions:

- Official third-party brand marks (Gmail / Instagram / WhatsApp)
- Pure white / near-white used only for rare light button variants
- Soft `rgba(...)` washes derived from the official cyan / blue / near-black values

Never introduce:

- Teal / green-blue accents
- Competing purple / cream / newspaper aesthetics
- Random decorative SVG systems unrelated to the product

---

# Atmosphere Rules

Empty space may be filled only with:

- Very soft gradients
- Large radial lighting
- Subtle particles
- Ambient glow
- Depth / soft shadows

Do not add:

- Random circles
- Random dotted curves
- Random SVG lines
- Floating icon clusters
- Generated-looking graphic noise

---

# Platform Phone Mockup

The Start Faster phone is a premium product mockup, not a stretched screenshot.

Requirements:

- Large, sharp and fully visible
- Realistic perspective and soft blue ambient glow
- Realistic reflections and premium depth
- Complete UI kept inside the screen
- No cropped or simplified interface
- No floating marketing cards around the phone

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
