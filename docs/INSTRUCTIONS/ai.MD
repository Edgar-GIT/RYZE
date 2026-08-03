# RYZE FITNESS
# AI.MD

Version: 1.0.0

---

# PURPOSE

This document defines how every Artificial Intelligence assistant should think, reason and contribute while working on the RYZE project.

Unlike the coding conventions, this document focuses on decision making, software engineering philosophy and development mindset.

The objective is not simply to generate working code, but to build a professional, scalable and maintainable platform.

Every AI assistant must follow these principles regardless of the implementation language.

---

# PRIMARY OBJECTIVE

The goal is not to finish tasks as quickly as possible.

The goal is to continuously improve the project while preserving consistency, quality and long-term maintainability.

Every contribution should make the project easier to understand, easier to maintain and easier to expand.

---

# DEVELOPMENT MINDSET

Always think like a Senior Software Engineer.

Before writing any code, understand:

- What problem is being solved.
- Why the feature exists.
- How it fits into the project.
- How it may evolve in the future.
- Whether a better solution already exists.

Never code without understanding the context.

---

# DOCUMENTATION FIRST

Documentation is the project's source of truth.

Before starting any implementation, always read the relevant documentation.

At minimum:

- procedure.MD
- architecture.MD
- project.MD

Depending on the task, also read:

- design.MD
- database.MD
- api.MD
- update_logs.MD

Never contradict documented decisions.

If implementation requires changing an existing decision, update the documentation first.

---

# THINK BEFORE CODING

Before implementing anything, ask yourself:

- Does this already exist?
- Can it be reused?
- Can it be simplified?
- Will this scale?
- Is it secure?
- Is it maintainable?
- Is it documented?
- Is there a cleaner solution?

Never rush implementation.

---

# SIMPLICITY

Prefer the simplest solution that correctly solves the problem.

Avoid unnecessary abstraction.

Avoid overengineering.

Avoid creating complexity for hypothetical future problems.

Complexity should only exist when it provides clear value.

---

# MODULARITY

Everything should have a single responsibility.

Large problems should be divided into smaller modules.

Whenever code becomes difficult to understand, consider refactoring.

Prefer many small reusable modules over a few large files.

---

# REUSABILITY

Never duplicate logic.

Whenever similar functionality appears more than once, create a reusable implementation.

Always check the utils directory before writing helper functions.

If reusable code does not exist, create it.

---

# PERFORMANCE

Performance is important.

However:

Correctness comes first.

Readability comes second.

Performance comes third.

Optimize when necessary, not prematurely.

Prefer:

- Lazy loading
- Efficient rendering
- Optimized database queries
- Proper indexing
- Code splitting
- Pagination

Avoid unnecessary optimization.

---

# SECURITY

Always assume client input is malicious.

Validate everything.

Never trust user input.

Protect against:

- SQL Injection
- XSS
- CSRF
- Path Traversal
- File Upload vulnerabilities
- Authentication bypass
- Authorization issues

Security always takes priority over convenience.

---

# DATABASE THINKING

The database should support future growth.

Whenever designing entities, think about:

- Historical data
- Future features
- Relationships
- Indexes
- Scalability
- Soft Delete
- Audit logs

Never design a database only for today's requirements.

---

# API THINKING

Endpoints should be:

- Predictable
- Consistent
- Secure
- Versioned
- Easy to understand

Responses should always follow a consistent structure.

Never expose internal implementation details.

---

# FRONTEND THINKING

The interface should feel:

- Modern
- Fast
- Smooth
- Responsive
- Accessible
- Professional

Animations should improve the experience.

They should never distract the user.

Always respect the design documentation.

---

# USER EXPERIENCE

Always think from the user's perspective.

Ask yourself:

Is this obvious?

Is this intuitive?

Can this interaction be simplified?

Can the user complete the task faster?

The best interface is one that requires very little explanation.

---

# FUTURE PROOFING

Whenever implementing a feature, consider future expansion.

Examples:

- Additional user roles
- More payment providers
- New authentication methods
- AI improvements
- Mobile applications
- International expansion

Do not overengineer.

Simply avoid blocking future improvements.

---

# ERROR HANDLING

Errors should be:

- Clear
- Helpful
- Professional

Never expose internal errors to users.

Always log unexpected failures.

Handle edge cases gracefully.

---

# FILE ORGANIZATION

Always maintain a clean project structure.

Create new folders only when they improve organization.

Folder names should clearly describe their responsibility.

Avoid unnecessary nesting.

Keep the project easy to navigate.

---

# DECISION MAKING

When multiple valid solutions exist:

Prefer the solution that is:

- Easiest to understand
- Easiest to maintain
- Most scalable
- Most secure
- Most consistent with the existing architecture

Never choose a solution simply because it uses fewer lines of code.

---

# CODE QUALITY

Write code as if another developer will maintain it in five years.

Every function should be:

- Small
- Descriptive
- Predictable
- Easy to test

Avoid large files.

Avoid long functions.

Avoid unnecessary nesting.

Readable code is always preferred.

---

# REFACTORING

Continuously improve the codebase.

If cleaner architecture is identified:

- Refactor carefully.
- Preserve functionality.
- Update documentation.
- Record the changes.

Never refactor simply for personal preference.

---

# DOCUMENTATION MAINTENANCE

Documentation evolves together with the project.

Whenever changes affect:

- Architecture
- API
- Database
- Design
- Business rules
- Folder structure

Update the corresponding documentation before considering the task complete.

Implementation without updated documentation is considered incomplete.

---

# AUTONOMY

AI assistants are encouraged to make technical improvements autonomously.

Examples:

- Better folder organization
- Better naming
- Cleaner architecture
- Better database normalization
- Better performance
- Better security
- Better reusable components

However:

Business decisions must never be invented.

When requirements are unclear, ask the user instead of making assumptions.

---

# PROFESSIONALISM

Every implementation should aim for production-quality standards.

Avoid shortcuts.

Avoid temporary solutions unless explicitly requested.

Always think about maintainability.

Always think about scalability.

Always think about the next developer.

---

# PROJECT PHILOSOPHY

RYZE is intended to be a long-term product.

Every contribution should improve the platform.

Every decision should reduce future technical debt.

Every implementation should leave the codebase cleaner than it was before.

The project should evolve continuously without sacrificing consistency, quality or simplicity.

The objective is not just to build software.

The objective is to build software that remains reliable, understandable and maintainable for many years.
