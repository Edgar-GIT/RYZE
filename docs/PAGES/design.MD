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

- Floating rounded rectangle (`border-radius: ~1.15rem`), soft cyan border glow
- Soft glass fill that becomes more opaque on scroll
- Brand left (with divider), navigation links centered, auth CTAs right
- Nav links show icon + uppercase label at rest
- On scroll, nav link labels collapse and only icons remain
- Active page uses cyan icon/text plus a glowing cyan underline
- Auth actions: outline “Log in” (with user icon) + gradient cyan “Start free”
  (with arrow)
- Never use boxed active navigation items as the sole active treatment

## Cards

- Rounded corners
- Dark glass effect
- Soft `rgba(255,255,255,0.08)` border
- Soft cyan glow on hover
- Slight lift on hover

## Service cards

- Compact glass panels with logo mark, badge, feature list and full-width CTA
- Featured program uses solid cyan badge + primary CTA
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

- Brand Logo (left)
- Centered links: Services, Contact, Feedback, Our Vision, Feed
- Right actions: Log in (outline pill) and Start free (solid cyan pill)

The logo always redirects to the Home page.
Log in routes to `/login`. Start free routes to `/register`.

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
6. Choose Your Program
7. Contact preview
8. Footer

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
- Left copy matching Training hierarchy with accent on `program.`
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

### Programs

Three product cards for Free, Premium and Elite using the shared modern
`ServiceCard` (logo mark, badge, features, right-aligned price, CTA). Premium
Level 1 is featured.

### Contact preview

Contact channels plus two CTAs side by side: Contact us and Join for Free.

### Footer

Contains:

- RYZE slogan logo
- Circular icon links for email, Instagram and WhatsApp
- Copyright

---

## Services

Displays every available product offered by the platform.

Catalog landing

- Full-bleed `bg_services.png` background
- Eyebrow “Catalog”, headline with accent on “categories.”
- Three highlight points: Built by experts / Instant access / Always evolving
- Three modern glass program cards (shared `ServiceCard`) with independent logos
  (`logo_card1.png`, `logo_card2.png`, `logo_card3.png`)
- Bottom trust row: Secure & private / Satisfaction guarantee / Support 24/7

Each card contains:

- Large floating program logo on the left (`logo_card1`–`3`), not boxed
- Badge, title and description to the right of the logo
- Three feature bullets with cyan circular checks
- Compact right-aligned price above the CTA
- Call To Action (primary on featured Premium Level 1, outline on others)

Each product redirects to its own page.

The same `ServiceCard` language and `bg_services.png` background are used on
the homepage programs preview.

### Free

Displays generic training packs.

Includes:

- Search
- Filters
- Four-column responsive grid

---

### Premium

Displays training packs together with automatically assigned nutrition programs.

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

Acts as the platform's social / complaints area.

Digital Complaints Book and related feedback flows live here.

---

## Feed

Public activity feed for RYZE community updates.

Currently routes to an under-development placeholder (`/feed`).

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
`Get in touch` preview with short copy, three compact channel buttons and two
CTAs side by side: `Contact us` → `/contact` and `Join for Free` → free program.

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
2. Get started — full-bleed `how_bg.png` background with left intro
   (eyebrow “Get started”, headline with accent on “minutes.”, short lead)
   and a bottom “How RYZE works” panel: five numbered steps connected by a
   glowing rail (Sign up → goals → AI program → track → weekly improvement).
3. Fitness operating system — full-bleed `evo_man_bg.png` background with
   left intro (eyebrow, headline with accent on “operating system”, lead,
   Explore RYZE CTA) and a bottom bordered 2×4 feature panel of compact
   icon+copy cells (Adaptive Programs, AI Coaching, Nutrition, Progress,
   Devices, Accountability, Community, Results).
4. Our vision — full-bleed `future_bg.png` background with two-column
   roadmap: left intro with phase tags (Today / Next / Soon / Later /
   Horizon), right vertical timeline with numbered milestone cards.
5. Continuous evolution — full-bleed `conti_bg.png` background with left
   intro and a 2×3 grid of glowing capability cards (icon well, title,
   description, chevron).
6. Join the movement — full-bleed `join_bg.png` background. Fully centered
   composition on the central vertical axis: eyebrow ("The journey
   continues"), two-line headline with accent on "join us now.", short
   description, neon outline CTA ("JOIN RYZE →" to `/register`), a wide
   single glass panel with five numbered feature items (community / training
   / passion / evolution / journey), and a bottom statement ("RYZE is more
   than a platform." with side lines, then "It's a movement."). No left or
   right columns, no extra cards, no navigation.

Do not use long paragraphs, huge founder photos, or traditional “About Us”
brochure layouts. Founder/people content is deferred for a later release.

---

## Login / Register

Routes: `/login` (login), `/register` (register).

Purpose: authenticate existing accounts and create new ones. Both forms send
data to the backend (`POST /api/v1/auth/login` and
`POST /api/v1/auth/register`). Login stores the `ryze_access_token` HttpOnly
cookie in the browser; no JWT is ever handled in JavaScript. On success both
forms redirect to `/profile` (still a placeholder screen at this stage).
Backend and network errors are shown inside the card as a compact inline alert
above the submit button.

Single page component (`LoginPage`) with two interchangeable forms — LOGIN and
CREATE ACCOUNT — on the same compact floating card. The background must never
change during the transition; only the card content swaps.

The auth page is rendered **without** the site chrome: no navbar, no footer
and no animated background. The `Layout` receives `showChrome={false}` for
`/login` and `/register`.

Page structure

- Full-viewport background. The image
  `main/src/resources/img/login_create/background.png` is applied as a CSS
  `background-image` (never an `<img>` element) with `background-size: cover`,
  `background-position: center center` and `background-repeat: no-repeat`. The
  original aspect ratio is always preserved; the viewport crops the image
  naturally at its edges, never stretched or deformed.
- Small RYZE logo (brand mark + wordmark) fixed in the top-left corner,
  linking home.
- A discreet legal line pinned to the bottom center:
  `© 2026 RYZE · Privacy Policy · Terms of Service`.

Card

- Compact, content-sized floating card, ~380–440px wide
- Dark/translucent background, thin cyan/blue border, `border-radius: 20px`
- Very subtle blue glow; `backdrop-filter: blur`
- Centered horizontally, slightly above the vertical center
- The space background stays clearly visible around the card
- No traditional footer inside or below the card

Login form

- Title "Welcome back" (white, bold, centered)
- Subtitle "Log in to continue your journey."
- Email field with envelope icon (placeholder "Email or username")
- Password field with lock icon and show/hide eye toggle (placeholder
  "Password")
- Options row: "Remember me" checkbox left, "Forgot your password?" cyan
  link right
- Primary button "Log in" (full width, cyan gradient, `border-radius: 8px`)
- Divider "or continue with" with side lines + white circle with the official
  Google G
- Footer link "Don't have an account? Create one here" (cyan)

Register form

- Title "Create account", subtitle "Join RYZE and start your evolution."
- Fields: First name, Last name, Email, Password, Confirm password (all
  identical size/style; password fields have eye toggles)
- Terms checkbox: "I agree to the Terms of Service and the Privacy Policy"
  (cyan links)
- Primary button "Create account", same Google divider, footer "Already have
  an account? Log in" (cyan)

Login ⇄ Register transition

The swap is not a page change nor a `display: none` swap. It shows the shared
RYZE `LoadingScreen` overlay for ~1 second and then swaps the form content,
which staggers in (each block fades up ~8px, ~50ms apart — title, subtitle,
fields, terms, button, Google, footer link). The card and background never
move.

Respects `prefers-reduced-motion` (direct swap, no loading overlay).

---

## Feedback page hero

Route: `/feedback`.

The Feedback page currently contains only the hero section (testimonial
preview). It sits directly below the navbar.

Background

- Solid near-black (`#000001`) fallback behind `feedback_bg.png`
  (`main/src/resources/img/feedbacks/feedback_bg.png`), applied as a full
  `background-image` with `cover`/center

Composition

- Top row: two columns. Left column holds the copy (blue uppercase eyebrow
  "FEEDBACK", large bold headline "Real people. / Real results." with
  "Real results." in bright RYZE blue and a subtle glow, soft gray
  description) anchored to the bottom. Right column shows the RYZE slogan
  logo (`BRAND_ASSETS.slogan`), bottom-aligned.
- Below: three testimonial cards in a single row (stack on small screens)
- Bottom: centered CTA

Testimonial cards

- Equal width/height, rounded corners (`18px`), very dark translucent surface,
  thin blue border (`rgba(42, 140, 255, 0.28)`) and a subtle blue edge glow
- Image occupying the upper portion (aspect ratio 4:5, `object-fit: cover`,
  rounded top corners, no filters)
- Body: five individual blue stars (`★★★★★`), white "Anonymous" label, muted
  testimonial copy with a blue quotation mark

Card copy (from the supplied assets `main/src/resources/img/feedbacks/f1.jpeg`,
`f2.jpeg`, `f3.jpeg`):

- "Over 16 weeks, I gained 12kg and a whole new level of confidence. The RYZE
  training and nutrition plan completely changed the way I approach fitness."
- "I lost 9kg and feel so much more energetic! I learned how to train with
  purpose and eat better. I feel stronger both inside and out."
- "RYZE helped me rebuild my confidence and feel better in my own body. The
  results came consistently and in a healthy, sustainable way."

CTA

- Centered below the cards: "START MY TRANSFORMATION" with an arrow icon,
  linking to `/register`
- Blue/cyan outline style, rounded, subtle glow — must not dominate the cards

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

Current Program

Displays the active training and nutrition program.

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

## RYZE Feed

RYZE Feed uses a simple, immersive short-form video interface focused on consuming fitness content with minimal UI.

### Feed Layout

The Feed uses a full-screen vertical video layout.

Each video occupies the majority of the viewport and users navigate between videos by vertical scrolling/swiping.

The interface should remain visually clean and avoid unnecessary elements.

### Video

Each Feed item contains:

- Full-screen vertical video
- Automatic playback when the video becomes active
- Automatic pause when the video leaves the viewport
- Muted/unmuted video control
- Video playback should prioritize a smooth and uninterrupted experience

Videos should use a vertical format whenever possible.

### Creator Information

The active video displays a small creator section near the bottom of the screen containing:

- Trainer profile picture
- Trainer name
- Follow button
- Short video description/caption

The creator name should be clickable and lead to the trainer's public profile.

If the user already follows the trainer, the button should indicate the current following state.

### Actions

A small vertical action area is positioned on the right side of the video.

Initial actions:

- Like
- Share

The Like button displays the current like state and may display the number of likes.

Comments are intentionally not available.

### Navigation

Users navigate the Feed primarily through vertical scrolling/swiping.

The Feed should behave similarly to modern short-form video applications:

1. User opens RYZE Feed.
2. The first video automatically starts playing.
3. User scrolls vertically.
4. The current video stops playing.
5. The next video becomes active and starts playing.
6. The process continues indefinitely while content is available.

There should be no traditional pagination visible to the user.

### Follow Integration

Following a trainer is available directly from the video interface.

Following a trainer does not create a separate feed.

Instead, followed trainers receive increased relevance in the user's personalized Feed.

### UI Principles

The RYZE Feed should prioritize:

- Video content
- Minimal UI
- Fast navigation
- Mobile-first interaction
- Clear creator attribution
- Simple actions

The interface should avoid:

- Large cards
- Complex navigation
- Comment sections
- Excessive metadata
- Persistent menus covering the video
- Traditional social-media post layouts

### Responsive Design

The Feed is primarily designed for mobile devices.

On mobile:

- Video fills the viewport.
- Controls remain easily reachable with one hand.
- Creator information stays near the bottom.
- Actions remain vertically aligned on the right.

On desktop, the Feed may remain centered with a vertical-video aspect ratio rather than stretching the video across the entire screen.

### Visual Style

RYZE Feed should follow the existing RYZE visual identity:

- Dark interface
- RYZE blue/cyan accents
- Subtle borders and glow effects
- Minimal typography
- High contrast controls
- Smooth transitions

The Feed should feel modern and technological without becoming visually overloaded.

The content itself should remain the primary visual focus.

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
