# RYZE

Version: 1.0.0

---

# Overview

RYZE is a fully online fitness platform designed to automate the creation, distribution and sale of training and nutrition programs.

The platform aims to remove the complexity of getting started in fitness by providing structured, accessible and personalized guidance through a modern digital experience.

Rather than replacing personal trainers, RYZE provides a platform where trainers can scale their work, reach more clients and gradually benefit from automation powered by Artificial Intelligence.

The project is built around one core principle:

> Make professional fitness guidance accessible, simple and automated.

---

# Mission

RYZE exists to make professional fitness guidance available to everyone, regardless of experience.

By combining technology, automation and fitness knowledge, the platform reduces the time and complexity normally required for someone to start training with confidence.

---

# Vision

The long-term vision is to become a modern fitness ecosystem where users can
easily discover, purchase and follow high-quality training and nutrition programs
while certified trainers can publish and manage their own content.

Artificial Intelligence will progressively automate repetitive tasks, allowing
trainers to focus on quality rather than manual work.

The public `/our-vision` page communicates this direction as a technology-first
vision surface focused on the platform and roadmap, not founder biographies.

---

# Current Status

The project has moved from planning into active frontend implementation.

The public visual frontend is under development first. Backend, authentication,
database migrations and purchase flows are documented but not implemented yet.

---

# Team

RYZE is currently developed by two founders.

## Edgar

Responsible for:

- Software Architecture
- Backend Development
- Frontend Development
- Infrastructure
- Database
- Security
- DevOps

Technical decisions are primarily led by Edgar.

---

## Sandro

Responsible for:

- Fitness
- Nutrition
- Marketing
- Advertising
- Business Strategy

Business decisions are discussed and approved jointly.

---

# Target Audience

RYZE is designed for anyone interested in improving their physical condition through structured training.

Although the platform is suitable for every fitness level, its primary audience consists of beginners and intermediate users who need clear guidance and motivation.

The initial marketing strategy focuses mainly on women between the ages of 20 and 40.

Since the majority of users are expected to access the platform using smartphones, the entire experience is designed following a Mobile First approach.

---

# Business Model

RYZE generates revenue through the sale of digital training and nutrition packs.

RYZE currently offers three different program categories, each designed for a different level of personalization.

## RYZE Free

The Free Program is designed to help new users get started with structured training at no cost.

Users can browse a catalog of generic training packs and activate one program at a time.

Each program is pre-built and intended for common goals such as muscle gain, fat loss or general fitness.

Features include:

- One active training program
- Generic workout program
- Exercise demonstrations
- PDF export
- No nutrition program
- No personalization
- No coach interaction
- Limited access duration

---

## RYZE Premium

Premium Level 1 combines a structured training program with an automatically generated nutrition program.

Before purchasing, the user completes a short questionnaire describing their fitness goals and basic personal information.

Examples include:

- Build muscle
- Lose fat
- Improve endurance
- Maintain weight

Based on the questionnaire, the RYZE Recommendation Engine automatically selects the most appropriate nutrition program.

This allows users to receive both a workout program and a nutrition program immediately after purchase, without waiting for manual preparation.

Features include:

- Personalized training pack
- Automatically assigned nutrition program
- Immediate delivery
- Exercise demonstrations
- PDF export

---

## RYZE Elite

Premium Level 2 provides the highest level of personalization available on the platform.

After purchasing the program, the client completes a comprehensive questionnaire covering every relevant aspect of their physical condition, lifestyle and objectives.

Information may include:

- Personal goals
- Daily routine
- Previous training experience
- Nutrition habits
- Allergies
- Medical conditions
- Injuries
- Sleep quality
- Stress levels
- Available training time
- Equipment availability
- Additional lifestyle information

Initially, every Premium Level 2 program is manually created by a certified trainer, ensuring that both the training and nutrition programs are specifically tailored to that individual.

As the platform evolves, this process may gradually incorporate Artificial Intelligence to assist trainers in generating personalized recommendations while maintaining professional oversight.

Since every program is created individually, delivery is not immediate.

Clients are informed of the estimated preparation time and receive access to their complete training and nutrition pack once it has been finalized.

Features include:

- Fully personalized workout program
- Fully personalized nutrition program
- Manual trainer review
- Individual program creation
- Exercise demonstrations
- PDF export
- Highest level of customization---

# Marketplace

RYZE includes a marketplace where certified trainers can publish their own training and nutrition packs.

Before joining the platform, every trainer must complete a validation process that may include:

- Curriculum Vitae
- Professional Certifications
- Identity Verification
- Manual Approval

Published packs are subject to a platform commission.

The commission structure may evolve as the platform grows.

---

# User Roles

The platform currently defines four user roles.

## Client

Clients can:

- Purchase packs
- Follow training programs
- Track progress
- View statistics
- Participate in the community
- Manage their personal profile

---

## Trainer

A trainer is an extension of an existing user account: approving a trainer
application creates a trainer profile linked to the applicant's **same** user
identity (same UUID, email and password) — it never creates a second account.

Trainers can:

- Create packs
- Edit packs
- Publish packs
- View statistics
- Manage clients
- Access trainer dashboards

Trainer applications are created by authenticated users through
`POST /api/v1/trainer/apply` and reviewed by both administrators.

Client management is a first-class trainer capability (under `trainer.clients`):
an active trainer can list their clients and add or remove the relationship with
any active user. The relationship has its own lifecycle and is independent of
the user account and the trainer profile (see the Trainer Clients sections in
`database.MD` and `api.MD`).

---

## Management Administrator

Management administrators are responsible for the operational side of the platform.

Responsibilities include:

- Managing trainers
- Managing clients
- Approving trainer applications
- Managing marketplace content
- Viewing business statistics
- Creating and editing platform packs

The Management Administrator holds a read-only view of user accounts (list and
view, under `users.read`). The full user lifecycle (create, update,
soft-disable, reactivate and administrative password reset, under
`users.manage`) belongs to the Technical Administrator.

Trainer application review (list, view, approve and reject, under
`trainer_applications.read` / `trainer_applications.manage`) is implemented as a
backend foundation shared by **both** administrators, matching the operational
responsibility of approving trainer applications.

---

## Technical Administrator

Technical administrators have unrestricted access to the platform's technical side.

They are responsible for:

- Platform maintenance
- Infrastructure
- Security
- User management
- Software development
- Database administration

The Technical Administrator does not act as a super administrator: management
business responsibilities (plans, finance, marketing) belong exclusively to
the Management Administrator and are never granted automatically to the
Technical Administrator.

User management (full lifecycle: create, update, soft-disable, reactivate and
administrative password reset) is implemented as a backend foundation under the
`users.manage` permission and is exclusive to the Technical Administrator. The
Management Administrator keeps a read-only view under `users.read`.

Trainer application review is a shared responsibility: both administrators hold
`trainer_applications.read` and `trainer_applications.manage` and can list,
view, approve and reject applications.

---

# Platform Features

The platform currently includes the following core features:

- Training programs
- Nutrition programs
- Exercise library
- Progress tracking
- Marketplace
- User authentication
- PDF export
- Notifications
- Community system
- Statistics dashboard
- Multi-language support
- Gamification

Additional features will continue to be introduced as the platform evolves.

---

## RYZE Feed

RYZE Feed is a vertical, short-form video feed designed to provide users with continuous educational and fitness-related content from RYZE trainers and creators.
Its not just another page like instagram, its for educational videos.
The experience is inspired by short-form video platforms such as Instagram Reels, but is intentionally focused on fitness, training, nutrition, recovery, and related educational content.

### Core Experience

- Users browse videos through a vertically scrollable feed.
- Videos are presented one at a time and optimized for visual consumption.
- The feed is primarily discovery-oriented rather than a traditional social network.
- Trainers can publish educational and informative fitness content.
- Content may cover topics such as:
  - Training techniques
  - Hypertrophy
  - Exercise execution
  - Nutrition
  - Recovery
  - Fitness education
  - General training advice
- Users can like videos.
- Comments are not supported.
- RYZE Feed does not provide a user-facing viewing history.
- The experience does not depend on traditional post feeds or long-form social interactions.

### Trainer Following

Users can follow trainers.

Following a trainer influences feed personalization, increasing the frequency at which content from followed trainers is presented to the user.

Following does not restrict the user to only that trainer's content. The feed may continue to include content from other trainers and creators.

### Feed Personalization

RYZE Feed should progressively personalize content based on user interactions.

Relevant signals may include:

- Trainers followed by the user
- Videos liked by the user
- General interaction with content
- Content categories and topics
- Trainer/content relevance

The exact recommendation algorithm is an implementation detail and may evolve independently from the product specification.

### Content Ownership

RYZE Feed content is published by authorized trainers or other roles explicitly permitted by the platform.

Each video belongs to its publishing creator and must maintain a relationship with that creator for attribution, management, and future personalization.

### Product Constraints

RYZE Feed intentionally excludes:

- Comments
- Public discussion threads
- User-created posts
- User-facing viewing history
- Traditional social-media timelines
- Direct messaging between users and trainers through the feed

The initial goal is to provide a focused educational content experience rather than reproduce a complete social network.

---

# Community

RYZE includes a community area available to users who have previously acquired at least one training pack.

The community encourages motivation and engagement through:

- Posts
- Images
- Comments
- Progress sharing

Certified trainers may also participate in community discussions.

The community is not intended to replace professional coaching.

---

# Gamification

Consistency is one of the main objectives of the platform.

To encourage long-term engagement, RYZE incorporates several gamification systems, including:

- Achievement badges
- Streaks
- Missions
- Events
- Progress notifications
- Milestone rewards

These systems are designed to improve user motivation without interfering with the training experience.

---

# Artificial Intelligence

Artificial Intelligence represents one of the most important long-term components of the project.

Initially, AI will support internal development.

Future releases will integrate AI directly into the platform, allowing automatic generation of personalized training and nutrition programs based on user questionnaires.

The long-term objective is to significantly reduce manual work while maintaining high-quality recommendations.

---

# Internationalization

RYZE is designed for an international audience.

The initial release supports:

- Portuguese
- English

Additional languages may be introduced in future versions.

---

# Design Philosophy

RYZE follows a modern, clean and futuristic design language.

The public website uses one official visual system. The Hero section is the
source of truth. Every other section must share the same near-black / electric
cyan palette and token set so the product feels coherent end to end.

The platform prioritizes:

- Simplicity
- Accessibility
- Performance
- Mobile First
- Consistency
- Fast navigation

All visual guidelines are documented in:

- `docs/INSTRUCTIONS/design.MD` — palette, tokens and consistency rules
- `docs/PAGES/design.MD` — page composition and component behaviour

---

# Technical Documentation

This document intentionally focuses on the business and functional aspects of the project.

All technical decisions are documented separately.

- `architecture.MD` — Technologies, frameworks, project structure and infrastructure.
- `docs/INSTRUCTIONS/design.MD` — Official palette, CSS tokens and visual rules.
- `docs/PAGES/design.MD` — User interface, branding, user experience and page specifications.
- `procedure.MD` — Development workflow, coding rules and AI instructions.
- `update_logs.MD` — Complete history of project modifications.

---

# Future Vision

RYZE is designed to evolve continuously.

Future releases may introduce:

- AI-generated programs
- Advanced marketplace features
- Additional payment providers
- Two-Factor Authentication
- Improved trainer tools
- New gamification systems
- Expanded analytics

Every new feature should remain aligned with the project's original philosophy:

> **Make professional fitness guidance accessible, automated and simple for everyone.**
