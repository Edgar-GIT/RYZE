# RYZE
# Design Specification

Version: 2.2.20

---

# Design Philosophy

RYZE follows a premium AI-powered SaaS design language.

The platform should always feel:

- Elegant
- Premium
- Futuristic
- Dark
- Minimal
- High contrast
- Technology-first
- Clean
- Apple Vision Pro + Nothing + Raycast + Linear + Vercel quality

This is not a generic fitness website aesthetic.

It is a dark technology product interface with athletic purpose.

The Hero section is the single visual source of truth for the entire website.
Every other page and section must match that language — never the other way
around.

Token rules and the official palette live in `docs/INSTRUCTIONS/design.MD`.
This file focuses on page composition and component behaviour.

---

# Brand Identity

The RYZE brand represents progress, discipline and innovation.

Its visual identity combines near-black surfaces, electric cyan accents and
dark glass panels with soft ambient blue glow.

The logo should always be used consistently across the platform.

---

# Color Palette

Official palette. Do not invent alternate blues, teals or greys.

Background

- Primary Background: `#05070B` → `--bg-primary`
- Secondary Background: `#0A1018` → `--bg-secondary`
- Elevated Surface: `#101826` → `--surface`
- Cards: `#141F2D` → `--card`
- Card Hover: `#182638` → `--card-hover` / `--surface-hover`
- Navbar: `rgba(8, 12, 18, 0.78)` with `backdrop-filter: blur(20px)` → `--navbar-bg`

Primary Accent

- Primary Accent: `#18D8FF` → `--accent`
- Secondary Accent: `#2A8CFF` → `--accent-secondary`

Accent Gradient (primary CTAs)

- `linear-gradient(90deg, #18D8FF, #2A8CFF)` → `--accent-gradient`

Glow

- Glow: `rgba(24, 216, 255, 0.35)` → `--accent-glow`
- Strong Glow: `rgba(24, 216, 255, 0.5)` → `--accent-glow-strong`
- Soft Glow: `rgba(24, 216, 255, 0.16)` → `--accent-glow-soft`

Text

- Primary: `#FFFFFF` → `--text-primary`
- Secondary: `#B8C4D8` → `--text-secondary`
- Muted: `#7F8EA3` → `--text-muted`
- Disabled: `#4D5A6C` → `--text-disabled`

Borders

- Default: `rgba(255, 255, 255, 0.08)` → `--border` / `--border-color`
- Hover: `rgba(24, 216, 255, 0.35)` → `--border-hover`
- Focused: `#18D8FF` → `--accent`

CSS custom properties in `main/src/frontend/src/styles/global.css` are the
runtime source of truth for these tokens.

Every component must consume these variables. Hardcoded palette colors should
disappear from UI CSS except documented exceptions.

Never use:

- Old teal / green-blue accents
- Rainbow graph colors
- Hard grey cardboard surfaces
- Harsh black-only shadows without ambient cyan glow
- Random decorative circles, dotted curves, SVG lines or floating icon clutter

Icons may use white, muted grey or electric cyan only (except official third-
party brand marks such as Gmail / Instagram / WhatsApp).

---

# Visual Style

- Darker overall surfaces aligned with the Hero
- Living atmospheric background (soft cyan nebulae, vignette, subtle motion)
- Sections use translucent washes so atmosphere shows through
- Soft cyan glows around interactive elements and ambient washes
- Premium lighting on buttons via the accent gradient
- Cards as dark glass panels with shared surface tokens
- Soft ambient blue glows instead of harsh black shadows
- Spacing defines sections more than hard separators
- Never use a flat, static single-color page fill
- Section transitions must feel seamless — no obvious palette jumps
- Empty space is filled with soft gradients, radial lighting, subtle particles,
  ambient glow and depth — nothing else

---

# Typography

Primary Font

Google Sans

Fallback

- Inter
- Roboto
- Arial
- sans-serif

Hierarchy rules:

- Small uppercase eyebrows with wide tracking
- Strong short headlines
- Short supporting descriptions
- Prefer visual product demos over long paragraphs

---

# Design Principles

## Mobile First

Every interface must be designed for mobile devices before desktop.

## Product Over Brochure

Major homepage sections should demonstrate the platform through UI previews,
dashboards, charts and interactive product surfaces.

## Consistency

Buttons, cards, forms, spacing and typography should behave consistently
throughout the entire application.

When scrolling, the user should never feel that a section was built by another
designer. The whole website must read as one coherent premium product.

## Simplicity

Every screen should have one primary objective.

## Accessibility

The interface should provide sufficient color contrast, large clickable areas
and clear navigation.

## Performance

Animations should enhance the experience without negatively affecting
performance.

---

# Components

## Buttons

Primary

- Cyan / blue gradient (`#18D8FF` → `#2A8CFF`)
- Soft cyan glow
- High contrast text

Secondary

- Transparent
- Thin light border
- Cyan hover border / soft fill

## Navbar

- Floating pill shape (`border-radius: 999px`), never a full-width rectangle
- Soft glass fill that becomes more opaque on scroll
- Light refined nav text; hover brightens to white
- Enough top padding so the bar is never clipped
- Larger brand mark in the nav
- Active page indicated only by a glowing cyan underline
- Never use boxed active navigation items

## Cards

- Rounded corners
- Dark glass effect
- Soft `rgba(255,255,255,0.08)` border
- Soft cyan glow on hover
- Slight lift on hover

## Service cards

- Tall glass panels with image media, cyan badge, feature list and full-width CTA
- Hover: lift + cyan border glow
- Same card language on homepage preview and `/services`

## Inputs

- Dark surfaces
- Thin borders
- Cyan focus ring

## Icons

- White by default
- Accent icons use cyan

---

# Animations

- Smooth transitions: 200–300ms
- Buttons gently glow
- Cards slightly lift
- No exaggerated motion

---

# Section Backgrounds

Alternate between:

- Very dark background (`#05070B` / `#0A1018`)
- Slightly lighter surface (`#101826` / `#141F2D`)

Use soft radial cyan washes for atmosphere.

Section fills should stay translucent enough for the global animated background
to read through.

Do not use obvious separators.

Spacing and tone shifts should define sections.

---

# Footer

The current footer visual treatment is intentionally preserved.

Do not redesign the footer as part of general palette updates unless explicitly
requested.

---

# Navigation

The public navigation bar contains:

- Brand Logo
- Services
- Contact
- Feedback
- Our Vision
- Profile

The logo always redirects to the Home page.

Navbar behaviour is defined in the Components section above and must be followed.

---

# Floating Actions

Depending on the user's role, floating action buttons may be displayed.

Examples include:

Customer Support

Available only for authenticated clients.

Notifications

Available for every authenticated user.

---

# Public Pages

## Home

Purpose

Present the platform by showing the product and encouraging users to begin.

The homepage should feel like a guided product tour, not a marketing brochure.

Sections

1. Hero
2. Platform ecosystem
3. Training Tracking
4. Nutrition Intelligence
5. Progress Analytics
6. Choose Your Plan
7. Final CTA
8. Contact preview
9. Footer

### Hero

Simple full-bleed layout:

- Hero image covers the full hero width and height (`object-fit: cover`)
- Text content overlays the left side
- Soft bottom fade into the following page sections
- Headline, description, CTAs and trust indicators

The artwork asset (`main/src/resources/img/hero/hero.png`) already contains the
left empty space and right visual composition, so no special masking or
split-column image tricks are required.

### Product showcase sections

The Platform section is a dedicated visual moment after the hero.

It uses:

- Full-width section near `100vh` with a 45% / 55% two-column grid
- Left column with eyebrow, multi-line heading (`without waiting.` in accent
  gradient), short supporting paragraph, primary CTA and secondary link
- Horizontal four-step process under the CTAs with glowing circular icons and
  thin connectors
- Right column with the detailed premium phone mockup (original dense screen UI)
  sized to fit the section — not oversized — with soft blue glow and realistic
  glass reflection
- Atmosphere aligned with the Hero: deep black / navy, cyan nebula, tiny
  particles, faint planetary arc and subtle grid/noise
- No floating marketing cards around the phone
- No random decorative blobs or abstract shapes

Other major sections after Platform should:

- Use a small title
- Use a strong headline
- Use a very short description
- Prefer full-bleed section backgrounds when artwork assets exist
  (e.g. Training uses `training_sec_bg.png`)

The Training section uses:

- Full-bleed background image (`main/src/resources/img/hero/training_sec_bg.png`)
- Left copy: eyebrow, heading with accent on `follow.`, short description
- Vertical feature badges (icon tile + title + subtitle)
- Reveal animations on copy and badges

The Nutrition section uses:

- Atmospheric dark background (nebula / faint grid / particles)
- Left copy matching Training hierarchy with accent on `plan.`
- Vertical nutrition feature badges
- Right column: glass nutrition card (kcal ring, macros, recommendation)
- Reveal + subtle floating card motion

The Progress section uses:

- Same atmospheric treatment as Nutrition
- Left copy with accent on `see.` and vertical progress feature badges
- Right glass analytics card: strength chart, summary stats, metric tiles
- Reveal + subtle floating card motion

Preferred demo surfaces for remaining showcase sections include:

- Weekly activity charts
- Workout calendars
- Nutrition rings and macros
- Progress trend charts
- Floating UI windows

Do not use generic multi-card "Why RYZE" feature grids.
Do not use generic admin-dashboard placeholder layouts for Platform.

### Plans

Three product cards for Free, Premium and Elite. Each card shows a compact
right-aligned price above its CTA.

### Final CTA

Compact dark band with clear next actions.

### Footer

Contains:

- RYZE slogan logo
- Circular icon links for email, Instagram and WhatsApp
- Copyright

---

## Services

Displays every available product offered by the platform.

Landing Page

Three product cards.

Each card contains:

- Image
- Name
- Description
- Main Features
- Compact right-aligned price above the CTA
- Call To Action

Each product redirects to its own page.

### Free

Displays generic training packs.

Includes:

- Search
- Filters
- Four-column responsive grid

---

### Premium

Displays training packs together with automatically assigned nutrition plans.

Includes:

- Search
- Filters
- Four-column responsive grid

---

### Elite

Displays the personalization questionnaire.

After submission, the user receives a preparation status page before being
redirected to the client dashboard.

---

## Feedback

Acts as the platform's social area.

Contains two sections.

Community

Facebook-inspired feed where users can:

- Create posts
- Upload images
- View achievements
- Share progress
- Comment

Digital Complaints Book

Official complaint submission page.

---

## Contact

The Contact page is a single elegant two-column experience.

Left column:

- Large heading (`Talk to RYZE.`)
- Short supporting description
- Compact interactive contact rows (Email, Instagram, WhatsApp)
- Email opens the mail client
- Instagram opens the official profile
- WhatsApp opens the official community invite

Right column:

- Embedded contact form inside a premium dark glass panel
- Title: `Write to RYZE`
- Fields: Name, Email, Subject, Message, Attachment
- Primary CTA: `Send Message`
- Helper text: `We usually reply within 24 hours.`

There is no modal, popup or background blur overlay for contact.

At the current frontend stage the form remains visual only and does not send
data. File validation and delivery remain backend responsibilities.

The Home page must not duplicate the full Contact page. It uses a compact
`Get in touch` preview with short copy, three compact channel buttons and a
`Contact us` CTA that navigates to `/contact`.

---

## Our Vision

Route: `/our-vision`

Purpose: sell the RYZE platform vision — not founder biographies.

The page must feel like a modern technology vision surface (Apple / Linear /
Vercel style): minimal copy, strong hierarchy, generous whitespace, premium
motion and dark futuristic atmosphere.

Legacy `/about-us` redirects to `/our-vision`.

Sections:

1. The future of fitness — large left-aligned hero over `our_view.png`
   background (same full-bleed image pattern as Training / Start Faster):
   eyebrow, thesis headline with accent on “software.”, short lead, horizontal
   capability chips (AI / Training / Nutrition / Recovery / Progress), and an
   Our Vision jump link. No stock gym photography.
2. Our mission — concise statement plus 3–5 icon cards (Accessible,
   Affordable, Professional, Simple, Technology First).
3. Why RYZE — dense responsive feature grid of premium glass cards with icon,
   title, one sentence, hover lift and soft glow.
4. Our vision — two-column roadmap: left intro + orbital planet glow,
   right vertical timeline with numbered milestone cards (icon, index, phase
   label, title, chevron).
5. Continuous evolution — floating capability cards instead of paragraphs.
6. Technology first — modular software visual (connected product modules).
7. The people behind RYZE — small closing section with subtle founder cards
   (initials / circular marks, name, role, one sentence). Platform remains the
   hero.

Do not use long paragraphs, huge founder photos, or traditional “About Us”
brochure layouts.

---

# Client Dashboard

The dashboard is the client's main workspace.

Navigation

- Profile
- Settings
- Share Profile
- Calendar

Main Sections

Statistics

Shows:

- Workout frequency
- Weekly progress
- Strength progression
- Muscle split
- Charts

Measurements

Optional body measurements similar to Lyfta.

Current Plan

Displays the active training and nutrition plan.

Calendar

Upcoming workouts and events.

Settings

Account information and management (ex: changing password or email).

Profile Sharing

Allows the user to generate and copy a public profile link or share it directly
through supported applications.

---

# Trainer Dashboard

Each trainer has a personal public page.

The dashboard allows trainers to:

- Manage packs
- View customers
- View revenue
- View statistics

---

# Management Dashboard

Responsible for business management.

Functions include:

- Manage trainers
- Manage clients
- Create packs
- Edit packs
- Marketplace management
- Platform statistics

---

# Technical Dashboard

Responsible for platform administration.

Functions include:

- User Management
- Maintenance
- System Monitoring
- Infrastructure Management

---

# Responsive Design

Every page must function correctly on:

- Mobile
- Tablet
- Desktop

Layouts should adapt naturally without changing the overall user experience.

---

# Future Improvements

Future versions may introduce:

- Additional themes
- Accessibility improvements
- Custom dashboards
- Advanced profile customization

# Inspiration

The following products should be used as design references.

## Linear

Reference for:

- Dark denseness done calmly
- Navigation
- Minimalism
- Product surfaces

## Raycast / Arc / Vercel / Nothing

Reference for:

- Premium technology branding
- Restraint
- Strong hierarchy
- Product-led presentation

## Lyfta

Reference for:

- Exercise pages
- Statistics
- Measurements
- Workout visualization

## Apple

Reference for:

- Showing the product instead of explaining it
- Simplicity
- Intentional spacing

## Stripe

Reference for:

- Landing clarity
- Pricing structure
