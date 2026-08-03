# RYZE
# Design Specification

Version: 2.1.1

---

# Design Philosophy

RYZE follows a modern, dark, technology-first design language with a controlled
galactic atmosphere.

The platform should always feel:

- Premium
- Alive
- Precise
- Clean
- Fast
- Professional
- Accessible
- Mobile First

The interface must avoid generic marketing templates and text-heavy brochure
layouts, while still allowing intentional cyan glow, nebula depth and glass
product surfaces.

Every public page should sell the product by showing the product.

---

# Brand Identity

The RYZE brand represents progress, discipline and innovation.

Its visual identity combines dark surfaces, restrained blue accents and
glass interfaces. The experience should feel closer to a premium software
product than a fitness brochure.

The logo should always be used consistently across the platform.

---

# Color Palette

Base Surfaces

- `#020617` — page background
- `#0A1224` — primary surface
- `#111827` — elevated surface
- `#155E75` — deep accent surface

Accent

- `#06B6D4` — primary highlight
- `#22D3EE` — interactive highlight / CTAs
- `#67E8F9` — soft accent text / glow

Neutral / Text

- `#FFFFFF` — primary text
- `#E2E8F0` — soft text
- `#94A3B8` — muted text
- `#64748B` — secondary neutral

The public visual identity is near-black navy with high-contrast white text and
electric cyan accents on interactive elements.

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

Avoid unnecessary components or information overload.

## Accessibility

The interface should provide sufficient color contrast, large clickable areas
and clear navigation.

## Performance

Animations should enhance the experience without negatively affecting
performance.

---

# Components

The interface is primarily composed of reusable components.

These include:

- Buttons
- Cards
- Glass panels
- Inputs
- Navigation bars
- Product showcase sections
- Product demo surfaces
- Statistics cards
- Charts
- Badges
- Notifications
- Dialogs
- Tooltips

Every component should maintain the same dark, restrained visual language.

---

# Animations

Animations should feel subtle and intentional.

Examples include:

- Fade In
- Slide Up
- Soft hover elevation
- Progressive navbar glass transition
- Animated navigation underline
- Chart / progress entrance
- Smooth page transitions

Animations should never delay user interaction.

Avoid flashy looping shine effects and heavy glow animations.

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

## Navbar behaviour

At the very top of the page:

- Fully integrated / nearly transparent
- No heavy border or opaque pill
- Clean logo + links + profile layout

As the user scrolls:

- Background opacity increases
- Soft blur / glass appears
- Transition remains smooth

## Active page indicator

Do not use boxed / pill active states for desktop navigation items.

The active page is indicated by a thin cyan underline beneath the item.

The underline should animate smoothly between navigation items.

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
2. Platform Dashboard
3. Training Tracking
4. Nutrition Intelligence
5. Progress Analytics
6. Choose Your Plan
7. Testimonials
8. Final CTA
9. Contact
10. Footer

### Hero

Simple full-bleed layout:

- Hero image covers the full hero width and height (`object-fit: cover`)
- Text content overlays the left side
- Soft bottom fade into the following page sections
- Headline, description, CTAs and trust indicators

The artwork asset already contains the left empty space and right visual
composition, so no special masking or split-column image tricks are required.

### Product showcase sections

Each major section after the hero should:

- Use a small title
- Use a strong headline
- Use a very short description
- Immediately show a visual product demo

Preferred demo surfaces include:

- Glass dashboards
- Weekly activity charts
- Workout calendars
- Nutrition rings and macros
- Progress trend charts
- Floating UI windows

Do not use generic multi-card "Why RYZE" feature grids.

### Plans

Three product cards for Free, Premium and Elite.

### Testimonials

Short quotes with concrete outcomes, presented inside compact glass boxes with
subtle accent borders.

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

Contains the platform's contact form.

The form should remain simple and easy to complete.

The email contact card opens a focused overlay panel with background blur and a
direct-message form. At the current frontend stage this interaction remains
visual only and does not send data.

The email panel includes a visual file upload control. File validation and
delivery remain backend responsibilities and are not implemented in the initial
frontend-only stage.

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
