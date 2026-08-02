Purpose & context

Edgar is building RYZE (ryzecoaching.com), a fully online platform
for automated sale of training and nutrition packs.
The central value proposition is automation:
clients get a plan quickly and without needing to wait
for availability from a gym or professional.

Team:

Edgar — technical development (frontend, backend, infrastructure)
Sandro — fitness, advertising, and marketing
50/50 profit split; decisions by consensus

Target audience: Primarily women aged 20–40 with some training experience but difficulty progressing. Open to all levels, with a focus on beginners. Primarily mobile usage.

Business model — three product tiers:

Free — automatic, no personalization
Personalized — plan chosen via search and filters
Premium — most personalized: client fills a detailed questionnaire (goals, allergies, routine, training history, sleep, stress, medical conditions, injuries, etc.) and receives a dedicated plan

Pricing & billing:

Plans offered as Monthly (6 weeks) or Quarterly (12 weeks, 2 continuous packs, ~25€ discount)
Portugal pricing: free to ~30€; international pricing TBD
Manual renewal (not automatic recurring subscription)
Invoices issued by email on client request (legal compliance to be confirmed)

Marketplace model (scope/phase TBD): Third-party coaches can join as partners, paying a commission to offer their own packs. Requires submission of professional certification for validation. Clients can purchase from multiple coaches (no exclusive 1-to-1 relationship).

Support model: Automated — no default direct person-to-person contact. Support via email or in-platform chat, 24h target response time. WhatsApp client community also exists.

Goals:

~1,000 clients as initial target
~9,000€/month revenue as success milestone
~30% churn rate as acceptable threshold (to be validated)
Acquisition via word-of-mouth and digital marketing

Legal & compliance: Injury liability disclaimer, 14-day right of withdrawal, online complaints book, written image authorization for marketing. Health data collected in the Premium questionnaire requires heightened GDPR care (likely Article 9 special category data, potential DPIA required).

Current state

Edgar is in the planning and architecture phase — no production code written yet. The project is well-defined at the concept level, with the following actively in motion:

Comprehensive pre-build planning underway (a 220+ question checklist across 16 categories was generated, with 🔴 blocking items flagged)
Living project documentation is a stated goal (architecture, technical decisions, DB schema) so any AI or new collaborator can get up to speed without repeating context

Critical open decision: The exact mechanism for generating Premium personalized plans (rules/templates vs. AI vs. manual preparation by a trainer) is unresolved and architecturally blocking — it affects backend design, data flows, and potentially the marketplace model.

Other open questions:

Scope and launch phase of the multi-trainer marketplace
Legal compliance of invoice-on-request billing model
Validation of the 30% churn target

On the horizon

Resolving the Premium plan generation mechanism (critical path item)
Defining international pricing
Clarifying marketplace launch scope and phasing
Confirming legal invoicing compliance in Portugal (AT-certified invoice series, SAF-T implications)
GDPR/DPIA assessment for health data collected in questionnaires

Key learnings & principles

The Premium questionnaire collects data that likely qualifies as health data under GDPR Article 9, requiring stronger protections and potentially a Data Protection Impact Assessment (DPIA) — this is a blocking concern, not a post-launch consideration
"Native WebSockets" in Go requires a library (e.g., gorilla/websocket) — the Go standard library does not implement the WebSocket protocol
Analysis paralysis is a real risk with large planning checklists; distinguishing "blocking now" from "phase-dependent" questions is important to maintain momentum
MySQL for chat/interaction history requires proper indexing and a defined retention policy to avoid performance degradation at scale
Client-side i18n (react-i18next) and SEO-friendly URL-prefixed routes are separate concerns that need to be explicitly bridged

Approach & patterns

Edgar prefers front-loading planning to avoid mid-project ambiguity, especially around sensitive data handling
Documentation-first mindset: wants a central living document so context doesn't need to be re-established in future sessions
Decisions made by consensus between Edgar and Sandro; technical decisions are Edgar's domain

Tools & resources

Frontend: React + TypeScript, PWA (installable as mobile app; no native app in this phase)
Backend: Go
Database: MySQL
Internationalization: react-i18next with automatic browser language detection (PT/EN)
Design: Google Sans font; color palette 
#092936, 
#105773, 
#599DB1, 
#5699A1, 
#DEDEDE, 
#FFFFFF; futuristic mid-range aesthetic
Community: WhatsApp client community

Site structure

Navbar: Brand icon (Home), Serviços, Contactos, Feedback, About Us, Profile
Fixed floating buttons: Customer support bubble (clients only), notification bell
Home: Motivational headline + 2 CTAs ("Adere já grátis" / "Começa já"), services section, contact, footer with WhatsApp/Instagram
Serviços: Landing with 3 cards (Free/Personalized/Premium) → dedicated pages with search/filters (Free & Personalized) or questionnaire + plan preparation screen (Premium)
Feedback: Comments + digital complaints book, social feed style (posts, images, stats, badges)
Contactos: Contact form
About Us: Brand story and origin
Perfil (roles: Client, Trainer, Technical Admin, Management Admin):
Client: Progress stats (charts, workouts, weekly muscle split), optional body measurements, current plan, schedule, profile sharing
Trainer: Personal page, associated clients, paying clients list with revenue
Technical Admin: General management (CRUD users), maintenance
Management Admin: Pack creation, trainer/client search and filtering, general transaction and usage stats
Gamification: Missions, badges, and progress notifications to maintain motivation and track evolution over time
