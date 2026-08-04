# RYZE
# Design Specification

Version: 2.2.6

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
- Apple + Linear + Vercel quality

This is not a generic fitness website aesthetic.

It is a dark technology product interface with athletic purpose.

---

# Brand Identity

The RYZE brand represents progress, discipline and innovation.

Its visual identity combines near-black blue-tinted surfaces, electric cyan
accents and dark glass panels.

The logo should always be used consistently across the platform.

---

# Color Palette

Primary Background

- `#070A0F`
- `#090D14`
- `#0B1018`

Secondary Surfaces / Cards

- `#101722`
- `#111B27`
- `#162130`

Cards should be only slightly brighter than the page.

Avoid grey backgrounds.

Everything should keep a subtle blue atmosphere.

Primary Accent

- `#1FCBFF`
- `#2EBEFF`
- `#27B6FF`
- `#00C8FF`

Accent Gradient

- `#32D7FF` → `#1E8CFF`

Text

- Primary: `#FFFFFF`
- Secondary: `#B8C3D1`
- Muted: `#8794A8`

Borders

- `rgba(255, 255, 255, 0.08)`

Hover

- Increase brightness
- Soft cyan glow on interactive elements only

Never use saturated colors other than the blue / cyan accent.

---

# Visual Style

- Darker overall surfaces
- Living atmospheric background (soft cyan nebulae, vignette, subtle motion)
- Sections use translucent washes so atmosphere shows through
- Soft cyan glows only around interactive elements and ambient washes
- Premium lighting on buttons
- Cards as dark glass panels
- Blue-tinted shadows instead of flat black shadows
- Spacing defines sections more than hard separators
- Never use a flat, static single-color page fill

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

- Cyan / blue gradient (`#32D7FF` → `#1E8CFF`)
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

- Very dark background (`#070A0F` / `#090D14`)
- Slightly lighter surface (`#0B1018` / `#101722`)

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
- About Us
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

- Left column copy with headline, ecosystem description and CTA
- Right column device composition:
  - One almost-3D tilted phone (`rotateY(-30deg)`) with realistic proportions and
    RYZE UI inside
  - Exactly three floating glass panels, no connector lines:
    - Nutrition on the left (lower, mid/bottom of the phone)
    - This week on the upper right
    - Recovery on the lower right
  - Slow float motion only
  - On mobile, panels stack under the phone or hide; they never overlay the device
  - Meaningful product signals (weekly heatmap, macro rings, recovery score)
    instead of crowded orbit layouts or generic KPI dashboards

Other major sections after Platform should:

- Use a small title
- Use a strong headline
- Use a very short description
- Immediately show a visual product demo

Preferred demo surfaces include:

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

## About Us

Explains:

- The RYZE story
- Mission
- Vision
- Values

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
