# RYZE FITNESS
# SECURITY.MD

Version: 1.0.0

---

# PURPOSE

This document defines the security standards that every component of the RYZE platform must follow.

Security is considered a core requirement, not an optional feature.

Every implementation must prioritize:

- Confidentiality
- Integrity
- Availability
- Privacy
- Traceability
- Maintainability

Security decisions should always favor long-term protection over short-term convenience.

Whenever a more secure solution exists with acceptable complexity, it should be preferred.

This document is expected to evolve throughout the lifetime of the project.

AI assistants are encouraged to improve the security architecture whenever a better solution is identified.

Every modification must also be documented inside `update_logs.MD`.

---

# SECURITY PHILOSOPHY

RYZE follows a defense-in-depth approach.

Security should never depend on a single protection layer.

Instead, multiple independent layers should exist.

Examples:

- Client validation
- Server validation
- Database validation
- Permission validation
- Audit logging
- Secure defaults
- Least privilege
- Rate limiting
- Monitoring

If one layer fails, another layer should still prevent exploitation.

---

# GENERAL PRINCIPLES

Every implementation should follow these principles.

## Never trust user input.

Every value received from:

- Forms
- APIs
- URLs
- Headers
- Cookies
- Query parameters
- WebSockets
- Uploaded files

must always be considered malicious until validated.

---

## Server validation is mandatory.

Frontend validation exists only to improve user experience.

Every validation must also exist on the backend.

The backend is the only trusted authority.

---

## Least Privilege

Users should only have access to resources they actually need.

Permissions should always be as restrictive as possible.

Never expose administrative functionality to regular users.

---

## Secure by Default

Every feature should start in its most secure state.

Security should never depend on user configuration.

Examples:

- Private profiles by default when appropriate.
- HTTPS only.
- Secure cookies.
- Strong password hashing.
- Authentication required unless explicitly public.

---

## Fail Securely

Unexpected situations should always fail safely.

Never expose:

- Stack traces
- SQL queries
- Internal paths
- Server configuration
- Secrets
- API keys

Generic error messages should be shown to users.

Detailed errors should only appear in logs.

---

## Auditability

Critical actions must always be traceable.

Examples:

- Login
- Logout
- Password reset
- Email change
- Role change
- Pack creation
- Trainer approval
- Payment
- Refund
- Account deletion
- Permission modifications

Whenever possible, store:

- User
- Timestamp
- IP
- User-Agent
- Action
- Result

---

# AUTHENTICATION

Authentication identifies who the user is.

RYZE supports:

- Email + Password
- Google OAuth

Future providers may be added without changing the existing authentication architecture.

---

## Password Hashing

Passwords are never stored in plain text.

Passwords are never encrypted.

Passwords are hashed.

RYZE uses:

Argon2id

with secure parameters.

Each password must use:

- Random salt
- Independent hash

Passwords cannot be recovered.

Forgotten passwords must always be reset.

---

## Password Requirements

Minimum length:

6 characters

Recommended:

12+

Password should include a combination of:

- Uppercase letters
- Lowercase letters
- Numbers
- Symbols

Weak passwords should be rejected.

Future versions may include password strength estimation using libraries such as zxcvbn.

---

## Password Reset

Passwords can only be reset through verified email ownership.

The reset process must use:

- Single-use token
- Short expiration time
- HTTPS only

After password reset:

- Previous reset tokens become invalid.
- Existing sessions may optionally be revoked.
- User receives an email notification.

---

## Email Verification

New accounts should verify ownership of the email address.

Verification links should:

- Expire automatically
- Be single use
- Use cryptographically secure tokens

---

## Google Authentication

Google OAuth should never replace internal account management.

Instead:

Google authentication creates or links a local RYZE account.

Every user always owns a local account.

This allows:

- Future provider migration
- Account recovery
- Multiple authentication providers

without changing database architecture.

---

## Sessions

RYZE uses JWT authentication.

Architecture:

Access Token

Short lifetime.

Refresh Token

Long lifetime.

Refresh tokens should be stored securely.

Expired access tokens should never be renewed without a valid refresh token.

---

## Token Security

JWT tokens must:

- Have expiration dates
- Include issuer
- Include audience
- Include subject
- Include issued time
- Include expiration
- Be cryptographically signed

Never store sensitive user information inside JWT payloads.

JWT should only contain the minimum information necessary.

---

## Session Revocation

The backend must be capable of invalidating sessions.

Examples:

- Password change
- Account disabled
- User logout
- Security incident

Password changes revoke every previously issued access token: the account's
`session_version` is incremented and access tokens embed the session version
they were issued for, so the authentication middleware rejects any token
carrying an older version.

Account deletion follows the same mechanism: soft-deleting an account
increments its `session_version` in the same operation, so every previously
issued access token is immediately rejected even though the database row is
preserved for the soft-delete/reactivation lifecycle.

Administrators disabling a user account through the user-management foundation
(`PATCH /api/v1/admin/users/{id}/disable`) trigger the exact same mechanism:
the repository soft-delete sets `deleted_at` and increments `session_version`
atomically, so all of the user's active sessions are revoked immediately and
the user can no longer log in, while the account data remains intact and
recoverable.

Future implementations should support token blacklisting or rotation.

---

## Administrator Authentication

Platform administrators are configuration-only accounts defined through
`ADMIN_1`/`ADMIN_2` environment credentials (username, password and access
code); they are never stored in the database and no registration or recovery
flow exists for them. Invalid configurations fail at server startup.

Admin authentication requires two factors completed in two stages:

1. Credentials (`username` + `password`). Success issues only a short-lived
   temporary stage state (`ryze_admin_stage_token`, 5 minute lifetime). This
   state is not an authenticated session and is rejected by the admin token
   validator.
2. Access code (`ADMIN_1_ACCESS_CODE` / `ADMIN_2_ACCESS_CODE`). Only a valid
   access code issued against the current stage state produces the admin
   session cookie `ryze_admin_access_token`.

Access codes are configuration secrets with a minimum length of 8 characters;
they are never stored in the database, embedded in tokens, logged or returned
by the API. Wrong credentials, wrong access codes, expired stage state and
unknown administrators all map to the same indistinguishable HTTP 401, so a
failed login never reveals which stage failed or whether an administrator
exists.

Admin login uses the same JWT infrastructure as user login but issues a
distinct token kind. User tokens, admin tokens and the temporary stage tokens
are distinguished by a `kind` claim and are mutually rejected, so a compromised
user session can never authenticate as an administrator, an admin token can
never access user endpoints, and a leaked stage state can never be used as an
admin session.

The admin session uses a dedicated HttpOnly cookie (`ryze_admin_access_token`)
that differs from the user cookie (`ryze_access_token`) so the two session
kinds cannot be confused in the same browser. All other cookie hardening
(`SameSite=Lax`, `Secure` per `COOKIE_SECURE`, JWT only inside the cookie,
credentials never returned by the API) applies identically to admin sessions.

Admin-protected routes are guarded by a dedicated admin authentication
middleware (`middleware.AdminAuthenticate`). It accepts only the final admin
session cookie validated through the token service as an `admin`-kind token;
stage tokens, user tokens and any invalid or misconfigured token are rejected,
and the resolved identity must be one of the configured administrator
identities (`ADMIN_1`/`ADMIN_2`). Every failure returns the same
indistinguishable HTTP 401 `AUTHENTICATION_REQUIRED`. The middleware validates
tokens through the token service only and never parses JWTs manually or
touches the database, the JWT secret or the admin credentials.

Future authorization layers must enforce admin permissions on the backend and
must never rely on client-provided roles or hidden admin interfaces.

---

## Account Recovery

Account recovery should always require proof of ownership.

Recovery must never rely on:

- Username
- Display name
- Personal information

Recovery should always occur through verified email ownership.

Future versions may include:

- Recovery codes
- Two-factor authentication
- Hardware security keys

---

# AUTHORIZATION

Authentication answers:

"Who is the user?"

Authorization answers:

"What can this user do?"

Both are mandatory.

---

## Roles

Each account owns exactly one primary role.

Supported roles:

- Client
- Trainer
- Technical Administrator
- Management Administrator

Administrators and trainers may also purchase programs like normal clients.

Permissions are determined by role.

Administrator roles are never stored in the database and are never supplied by
the client. The role of an administrator is always derived server-side from the
configured administrator identity (ADMIN_1/ADMIN_2) established by the admin
authentication flow.

---

## Permission Validation

Every protected endpoint must validate permissions.

Frontend restrictions alone are never sufficient.

The backend must independently verify every action.

Permission validation must occur:

- Before executing business logic
- Before database modifications
- Before file access
- Before administrative operations

---

## Resource Ownership

Users should only access resources they own.

Examples:

A client:

- Cannot view another client's purchases.
- Cannot edit another client's measurements.
- Cannot access another client's notifications.

Ownership validation must always occur server-side.

---

## Administrative Permissions

Administrative functionality should require explicit authorization.

Examples:

- User management
- Pack approval
- Trainer approval
- Statistics
- Audit logs
- Database maintenance

Administrative endpoints should never rely solely on hidden frontend interfaces.

The backend must always enforce permissions.

---

## Administrator Authorization

The admin authorization layer is implemented as a reusable backend middleware
(`middleware.RequireAdminRole`) that runs strictly after the admin
authentication middleware (`middleware.AdminAuthenticate`).

Role model:

- ADMIN_1 is the **Technical Administrator**
  (`TECHNICAL_ADMINISTRATOR`).
- ADMIN_2 is the **Management Administrator**
  (`MANAGEMENT_ADMINISTRATOR`).

Granted permissions:

- Shared by both roles: users, trainers, statistics.
- Technical Administrator only: system, infrastructure, technical
  configuration, security, development.
- Management Administrator only: plans, finance, marketing.

Security rules:

- Authorization derives **only** from the authenticated admin identity stored
  in the request context by `AdminAuthenticate`. Roles and permissions
  supplied by the client (JSON, headers, query parameters) are never accepted.
- The Technical Administrator is **not** a super administrator: it never
  automatically receives management business permissions (plans, finance,
  marketing).
- The role model and its permission mapping live in
  `middleware/adminroles` and are the single reusable authorization source.
- Admin authorization failure returns HTTP 403 `FORBIDDEN` with a generic
  message that never reveals the required roles, the caller's role or any
  internal detail. Authentication failures are handled by the
  authentication middleware and remain HTTP 401 `AUTHENTICATION_REQUIRED`.
- The authorization middleware fails closed: mounted without
  `AdminAuthenticate` (no admin identity in context) it still denies the
  request.

Implemented under the shared `users` permission: the admin user-management
foundation (list users, get user, soft-delete user) is enforced by
`RequireAdminRole(TECHNICAL_ADMINISTRATOR, MANAGEMENT_ADMINISTRATOR)`, so both
administrator roles may manage regular user accounts. Target user identifiers
are never accepted from the client as ownership proof: a soft-delete targets
only the account resolved by the validated route identifier and always operates
server-side. The management response never exposes password hashes, session
versions or deletion markers.

---

## Future Authorization

The authorization architecture should remain flexible.

Future improvements may include:

- Fine-grained permissions
- Permission groups
- Feature flags
- Temporary permissions
- Delegated administration

The database should be designed to support these features without major structural changes.

---

# API SECURITY

The backend is the only trusted authority.

Every request received by the API should be considered potentially malicious until validated.

Security must be applied consistently across every endpoint.

Authentication alone is not sufficient.

Authorization, validation and auditing must always accompany authentication.

---

# REQUEST VALIDATION

Every incoming request must be validated.

Validation includes:

- Required fields
- Data types
- Maximum length
- Minimum length
- Numeric ranges
- Enum values
- Date formats
- UUID format
- Email format
- URL format
- Phone number format
- Uploaded file validation

Unexpected fields should be ignored or rejected depending on the endpoint.

Never trust frontend validation.

---

# INPUT SANITIZATION

Every string received from users should be sanitized before processing.

Examples include:

- Leading/trailing spaces
- Invisible Unicode characters
- HTML tags
- Dangerous characters
- Duplicate whitespace

Sanitization should never replace validation.

Both are required.

---

# OUTPUT SANITIZATION

Every response returned by the API should avoid exposing internal information.

Never expose:

- SQL queries
- Stack traces
- Server paths
- Internal IDs
- Secrets
- API keys
- Password hashes
- Internal configuration

Only expose information required by the client.

---

# SQL INJECTION

All database operations must use parameterized queries.

Never concatenate SQL strings.

Examples of forbidden practices:

- String concatenation
- Dynamic SQL without parameters
- Direct interpolation

The ORM or query builder must always generate parameterized statements.

Search queries should also use parameters.

Database permissions should follow least privilege.

---

# XSS (Cross Site Scripting)

User-generated content must never be trusted.

Every displayed value should be escaped before rendering.

Examples:

- Comments
- User names
- Profile descriptions
- Feedback
- Messages

Dangerous HTML should never be accepted unless explicitly supported.

Markdown support, if introduced in the future, should be sanitized before rendering.

---

# STORED XSS

Data stored in the database should never be assumed safe.

Every output must still be escaped when displayed.

Escaping should occur at render time.

---

# REFLECTED XSS

Query parameters, URLs and search terms should never be rendered directly into HTML.

Every reflected value must be escaped.

---

# DOM XSS

Frontend code should avoid directly injecting HTML.

Avoid:

- innerHTML
- document.write()

Prefer:

- textContent
- React rendering

React's default escaping should always be preserved.

Dangerous HTML rendering should only be allowed after strict sanitization.

---

# CSRF (Cross Site Request Forgery)

Authentication mechanisms should minimize CSRF risk.

Whenever cookies are used:

- SameSite should be enabled.
- CSRF tokens should be considered.
- Secure cookies should be enforced.

State-changing requests should never rely solely on browser cookies.

---

# SSRF (Server Side Request Forgery)

The backend should never fetch arbitrary URLs provided by users.

Whenever remote requests are necessary:

- Validate protocol.
- Validate destination.
- Block internal IP ranges.
- Block localhost.
- Block metadata endpoints.
- Restrict redirects.
- Apply request timeout.

Allowlist approaches should be preferred.

---

# REMOTE CODE EXECUTION

The backend should never execute user input.

Never execute:

- Uploaded files
- Shell commands
- User-generated scripts

Whenever external processes are required:

- Validate parameters
- Escape arguments
- Limit permissions

---

# COMMAND INJECTION

Never concatenate shell commands.

Avoid:

exec()

system()

bash -c

Use native libraries whenever possible.

If external commands are unavoidable:

- Validate arguments
- Escape parameters
- Restrict execution

---

# PATH TRAVERSAL

User-controlled file paths must never be trusted.

Reject:

../

..\\

Absolute paths

Encoded traversal attempts

Only allow files inside approved storage directories.

---

# HTTP SECURITY HEADERS

The backend should configure modern security headers.

Recommended headers include:

- Strict-Transport-Security
- X-Content-Type-Options
- X-Frame-Options
- Referrer-Policy
- Permissions-Policy
- Content-Security-Policy

Header configuration should evolve according to browser standards.

---

# CONTENT SECURITY POLICY (CSP)

The project should use a restrictive Content Security Policy.

Avoid:

unsafe-inline

unsafe-eval

Scripts should originate only from trusted sources.

Future third-party integrations should be explicitly allowlisted.

---

# CORS

Cross-Origin Resource Sharing should be restrictive.

Never use:

Access-Control-Allow-Origin: *

Production environments should explicitly define allowed origins.

Allowed methods should be limited.

Allowed headers should be limited.

Credentials should only be enabled when required.

---

# HTTPS

All production traffic must use HTTPS.

HTTP requests should automatically redirect to HTTPS.

Sensitive information should never travel through unencrypted connections.

HSTS should be enabled in production.

---

# RATE LIMITING

Every public endpoint should support rate limiting.

Examples:

Authentication

Password reset

Registration

Search

Comments

Feedback

Uploads

Rate limits may differ depending on endpoint sensitivity.

Administrative endpoints should use stricter limits.

---

# BRUTE FORCE PROTECTION

Authentication endpoints should detect repeated failures.

Possible protections include:

- Temporary delays
- Temporary lockouts
- Progressive backoff
- CAPTCHA after repeated failures

Password spraying attacks should also be considered.

---

# DENIAL OF SERVICE (DoS)

Endpoints should avoid expensive operations.

Every request should define:

- Maximum payload size
- Maximum execution time
- Maximum pagination size

Database queries should always be optimized.

Heavy computations should run asynchronously whenever possible.

---

# DISTRIBUTED DENIAL OF SERVICE (DDoS)

Infrastructure should be prepared for DDoS mitigation.

Recommended protections include:

- Cloudflare
- Reverse Proxy
- Load Balancer
- Web Application Firewall (WAF)
- Global Rate Limiting
- IP Reputation
- Geographic Filtering (when appropriate)

Application-level protections should complement infrastructure protections.

---

# BOT PROTECTION

The platform should detect abusive automated traffic.

Possible techniques:

- Rate limiting
- Behavioral analysis
- Honeypot fields
- CAPTCHA
- Email verification
- Device fingerprinting (future)

Legitimate search engines should remain unaffected whenever possible.

---

# FILE UPLOAD SECURITY

File uploads are one of the highest-risk attack vectors within the platform.

Every uploaded file must be treated as malicious until proven safe.

Validation must never rely solely on:

- File extension
- MIME type
- Browser information
- Client-side validation

The backend is solely responsible for determining whether a file is safe.

---

# ALLOWED FILE TYPES

Only explicitly allowed file types may be uploaded.

Initially supported:

Images

- PNG
- JPG
- JPEG
- WEBP

Documents

- PDF

Future file types must be explicitly approved before implementation.

Everything else must be rejected.

---

# FILE SIZE LIMITS

Every upload must have a maximum allowed size.

Examples:

Profile Pictures

10 MB

Exercise Images

15 MB

Documents

25 MB

Temporary uploads exceeding the limit should be rejected immediately.

---

# FILE SIGNATURE VALIDATION

Never trust the file extension.

Never trust the MIME type reported by the browser.

Every uploaded file must be validated using its binary signature (magic bytes).

Example:

A file named:

image.png

that actually contains:

Executable code

must be rejected.

The real content of the file must always match the declared format.

---

# IMAGE VALIDATION

Every uploaded image must be decoded by the backend.

If the decoder fails, the upload should be rejected.

This prevents:

- Corrupted images
- Invalid image headers
- Fake image files
- Binary payloads disguised as images

Only successfully decoded images may be accepted.

---

# IMAGE RE-ENCODING

Uploaded images should never be stored exactly as received.

Instead:

The backend should decode the image and generate a completely new image file.

Benefits include:

- Removes hidden metadata
- Removes malformed chunks
- Removes embedded payloads
- Standardizes image format
- Reduces attack surface

The stored file should always be generated by the server.

Never keep the original uploaded image.

---

# IMAGE METADATA

Metadata should be removed whenever possible.

Examples:

- GPS coordinates
- Device model
- Camera information
- Author
- Editing software
- Hidden comments

Only image pixels should be preserved.

---

# IMAGE COMPRESSION

Images should be automatically optimized.

Benefits:

- Smaller storage
- Faster loading
- Lower bandwidth
- Better performance

Compression should preserve acceptable visual quality.

---

# RANDOMIZED FILENAMES

Uploaded filenames should never be preserved.

Instead generate:

UUID

or

Cryptographically secure random filename.

Never expose original filenames inside storage.

---

# STORAGE

Uploads should never be stored inside the public frontend.

Storage should remain outside publicly accessible directories whenever possible.

Access should occur through authenticated API endpoints.

Direct filesystem exposure should be avoided.

---

# TEMPORARY FILES

Temporary files should automatically expire.

Examples:

- PDF generation
- Image processing
- Upload cache

Automatic cleanup jobs should periodically remove expired files.

---

# EXECUTABLE FILES

Executable files should never be accepted.

Examples:

- EXE
- DLL
- BAT
- CMD
- SH
- APK
- MSI
- COM

These files should always be rejected.

---

# SCRIPT FILES

Scripts should never be accepted.

Examples:

- JS
- PHP
- ASP
- JSP
- PY
- GO
- EXE wrappers

Regardless of extension, executable content should always be rejected.

---

# DOUBLE EXTENSIONS

Reject files using multiple extensions.

Examples:

image.png.exe

photo.jpg.php

avatar.webp.bat

Only one valid extension should exist.

---

# ARCHIVES

Compressed archives should initially be rejected.

Examples:

- ZIP
- RAR
- 7Z
- TAR
- GZIP

Future support should include:

- Archive inspection
- Zip bomb detection
- Recursive validation

---

# MALWARE DETECTION

Future versions should integrate malware scanning.

Recommended:

ClamAV

or another enterprise-grade antivirus engine.

Every uploaded file should be scanned before becoming available.

Files flagged as suspicious should:

- Be quarantined
- Never become publicly accessible
- Generate audit logs

---

# DATABASE SECURITY

The database should never be directly exposed to the internet.

Access should only occur through the backend.

Database credentials should never be hardcoded.

Least privilege should always be applied.

Every service should receive only the permissions it requires.

---

# SENSITIVE DATA

Sensitive information should always receive additional protection.

Examples include:

- Password hashes
- Authentication tokens
- Email verification tokens
- Password reset tokens
- Payment identifiers
- Personal questionnaires
- Medical information
- Nutrition data

Sensitive information should never appear in logs.

---

# ENCRYPTION

Passwords:

Argon2id

Sensitive tokens:

Cryptographically secure random values.

Future versions may encrypt particularly sensitive database fields.

Encryption keys should never be stored inside the repository.

---

# ENVIRONMENT VARIABLES

Secrets should always be stored using environment variables.

Examples:

- JWT Secret
- Database Password
- SMTP Keys
- Google OAuth Secrets
- API Keys
- Cloud Credentials

Never commit secrets to Git.

Never expose secrets through frontend code.

---

# LOGGING

Security-related events should always be logged.

Examples:

- Login
- Failed login
- Password reset
- Email verification
- Upload rejection
- Permission denial
- Suspicious request
- Administrative actions

Logs should never contain:

- Passwords
- Tokens
- Secrets
- Personal questionnaire answers

---

# AUDIT LOGS

Critical actions should generate immutable audit records.

Examples:

- Role changes
- Trainer approval
- Pack publication
- Payment refund
- Account deactivation
- Administrative modifications

Audit records should never be editable.

---

# PRIVACY

RYZE should follow GDPR principles.

Collected information should be:

- Necessary
- Relevant
- Protected
- Traceable

Users should always be informed about stored personal information.

Future implementations should support:

- Data export
- Account recovery
- Right to rectification
- Right to restriction
- Right to portability

Whenever legally permitted.

---

# MONITORING

The platform should continuously monitor its own health.

Important metrics include:

- CPU usage
- Memory usage
- Disk usage
- API response times
- Active users
- Failed requests
- Login failures
- Upload failures
- Payment failures
- Database latency

Future versions may integrate dedicated monitoring solutions.

Examples:

- Prometheus
- Grafana
- Loki
- OpenTelemetry

---

# BACKUPS

The database should be backed up automatically.

Recommended strategy:

Daily Incremental Backup

Weekly Full Backup

Monthly Archive Backup

Backups should be encrypted before storage.

Backups should never remain on the same machine as the production server.

Periodic restoration tests should be performed to verify backup integrity.

---

# DISASTER RECOVERY

The platform should be capable of recovering from:

- Database corruption
- Server failure
- Hardware failure
- Accidental deletion
- Software bugs
- Security incidents

Recovery procedures should prioritize:

1. Data integrity
2. Service availability
3. User communication

---

# DEPENDENCY SECURITY

Third-party dependencies should be minimized.

Before adding any dependency, verify:

- Active maintenance
- Community adoption
- Security history
- License compatibility
- Necessity

Unused dependencies should be removed.

Dependencies should be updated regularly to receive security patches.

---

# THIRD-PARTY SERVICES

Every external service should be considered a potential security risk.

Examples:

- Google OAuth
- Resend
- Google Maps
- Payment providers
- AI providers

Only the minimum amount of required data should be shared.

Sensitive information should never be transmitted unless strictly necessary.

API keys should never be exposed to the frontend.

---

# PAYMENT SECURITY

Payment processing should always use certified payment providers.

RYZE should never store:

- Credit card numbers
- CVV codes
- Banking credentials

Payment information should remain entirely under the payment provider's responsibility.

Only transaction identifiers and payment status should be stored.

---

# EMAIL SECURITY

Emails should always originate from verified domains.

Authentication methods should include:

- SPF
- DKIM
- DMARC

Sensitive actions requiring email confirmation include:

- Password reset
- Email verification
- Email address change
- Security alerts

Email links should always expire.

---

# WEBSOCKET SECURITY

Every WebSocket connection must be authenticated before becoming active.

Permission validation should continue after connection establishment.

Incoming messages must be validated exactly like REST requests.

Connections should automatically close after prolonged inactivity.

Rate limiting should also apply to WebSocket traffic.

---

# CACHE SECURITY

Cached information should never expose sensitive data.

Authentication tokens should never be cached publicly.

Private information should only be cached in secure server-side storage when necessary.

Expired cache entries should be removed automatically.

---

## RYZE Feed Security

RYZE Feed introduces additional security requirements because trainers can upload and publish media content and users can interact with that content.

All Feed functionality must follow the general security requirements defined in this document. The following rules are specific to the RYZE Feed.

### Content Ownership and Authorization

All trainer Feed operations must enforce server-side authorization.

The backend must never trust:

- `trainer_id` supplied by the client
- `user_id` supplied by the client
- ownership information supplied by the client
- client-side role information
- client-side permissions

The authenticated identity must come from the validated authentication context.

A trainer may only modify or delete Feed content that they own, unless an explicitly authorized administrative role is performing the operation.

### Video Upload Security

Video uploads must be treated as untrusted input.

The server must validate uploaded media independently of the filename and client-provided MIME type.

Validation should include:

- File size
- Actual file type
- Container format
- Codec compatibility
- File structure
- Duration
- Resolution
- Encoding characteristics
- Potentially malformed media
- Potentially malicious content

Client-provided extensions such as `.mp4`, `.mov`, `.webm`, or similar must never be considered proof that the file is safe.

### Media Processing Isolation

Uploaded media should be processed in an isolated environment where practical.

Media processing must not execute uploaded files as native applications.

Video processing components must be treated as security-sensitive dependencies because malformed media can target vulnerabilities in codecs, parsers, or processing libraries.

Processing workers should have:

- Minimal filesystem permissions
- Minimal network access
- No unnecessary application credentials
- Restricted access to application secrets
- Resource limits
- CPU and memory limits
- Execution time limits

A compromised media-processing process must not provide direct access to the main application or database.

### File Storage

Uploaded videos must not be stored in locations where they can be interpreted as executable server-side code.

User-uploaded media should be stored separately from application source code.

Storage access should use generated identifiers rather than trusting user-provided filenames.

Original filenames must not be used as authoritative storage paths.

### File Validation

The system must validate the actual content of uploaded files.

At minimum:

```text
Client Upload
     │
     ▼
Size Validation
     │
     ▼
File Signature / Type Detection
     │
     ▼
Media Parsing
     │
     ▼
Security Validation
     │
     ▼
Processing / Transcoding
     │
     ▼
Safe Storage
     │
     ▼
Publication

Files that fail validation must never become publicly accessible.

Malicious Media

A valid-looking media extension does not guarantee that the content is safe.

The security system must account for:

Malformed video containers
Parser exploitation
Codec vulnerabilities
Embedded unexpected data
Polyglot files
Files with misleading extensions
Resource-exhaustion media
Decompression/resource bombs

Where appropriate, uploaded videos should be transcoded into controlled output formats rather than serving the original uploaded binary directly.

Thumbnails and Generated Media

Generated thumbnails and preview images must also be treated as untrusted media until safely generated.

The system should preferably generate thumbnails from the processed video rather than trusting trainer-uploaded thumbnail files.

If custom thumbnails are supported later, they must pass the same media security validation requirements as other uploaded images.

Feed Content Abuse

RYZE Feed must include mechanisms to prevent abusive content from becoming an uncontrolled public distribution channel.

The architecture should support:

Content hiding
Content removal
Trainer restrictions
Administrative moderation
Content status changes
Abuse/rate controls
Future reporting functionality

Moderation functionality may evolve independently from the initial Feed implementation.

Likes

Like endpoints must be protected against abuse.

The backend must enforce:

One Like per user per Feed item
Unique database constraints
Authentication
Rate limiting
Server-side validation

Like counts must not be accepted from the client as authoritative values.

Trainer Following

Follow endpoints must enforce:

Authentication
Valid user identity
Valid trainer identity
Unique user/trainer relationship
Rate limiting
Server-side validation

A user must not be able to create duplicate Follow relationships through repeated or concurrent requests.

Feed Enumeration Protection

Feed APIs must avoid exposing unnecessary internal information.

The system should not expose:

Internal storage paths
Database identifiers that are not intended for public use
Processing infrastructure details
Internal moderation information
Private trainer data
Hidden or unpublished content

Public Feed responses should contain only the information required by the client.

Rate Limiting

Feed interactions must be protected against excessive automated requests.

Rate limiting should be considered for:

Feed retrieval
Likes
Follows
Video uploads
Content creation
Content modification
Content deletion

Upload and content-management operations should have significantly stricter limits than normal Feed browsing.

Resource Protection

Video processing and uploads can consume significant resources.

The system must enforce limits for:

Maximum upload size
Maximum video duration
Maximum resolution
Maximum processing time
Maximum concurrent processing jobs
Maximum storage consumption where applicable

These limits must be enforced server-side.

Authentication

Users must be authenticated before performing:

Likes
Follows
Trainer content management
Video uploads

Public Feed access may be allowed only if explicitly defined by the product's access model.

Authentication and authorization must never depend solely on frontend route protection.

XSS and Content Injection

Trainer-provided metadata such as:

Titles
Descriptions
Tags

must be treated as untrusted input.

The application must prevent stored XSS and other injection attacks.

User-generated text must be safely encoded when rendered.

HTML provided by trainers must not be rendered as trusted content unless an explicit, secure sanitization pipeline exists.

SQL Injection

All Feed database operations must use the project's parameterized database access mechanisms.

User-provided values must never be concatenated directly into SQL queries.

This applies particularly to:

Feed search/filtering
Trainer identifiers
Pagination
Sorting
Content identifiers
Likes
Follows
Authorization Testing

The Feed backend must include tests for horizontal privilege escalation.

Examples:

Trainer A cannot edit Trainer B's video.
Trainer A cannot delete Trainer B's video.
User A cannot modify User B's Like.
User A cannot modify User B's Follow relationship.
Unpublished content cannot be retrieved through public Feed endpoints.
Deleted content cannot be retrieved through normal Feed endpoints.
Logging and Auditing

Security-relevant Feed events should be auditable where appropriate.

Potential events include:

Trainer content creation
Video upload
Content publication
Content hiding
Content deletion
Administrative moderation
Trainer restrictions

Logs must not contain sensitive user information or authentication secrets.

Security by Default

RYZE Feed must follow a deny-by-default security model.

New Feed capabilities must not automatically become publicly accessible.

Any new content type, upload mechanism, interaction, or trainer capability must explicitly define:

Authentication requirements
Authorization requirements
Input validation
Rate limiting
Storage requirements
Moderation requirements
Audit requirements

---

# PRIVACY

Privacy is a core requirement of the platform.

Only collect information that is necessary for platform functionality.

Users should always remain in control of their personal information whenever legally possible.

Personal information should never be shared with third parties without explicit necessity.

Medical and nutrition-related information should receive additional protection due to its sensitive nature.

---

# INCIDENT RESPONSE

Whenever suspicious activity is detected, the platform should respond appropriately.

Possible actions include:

- Logging the incident
- Rate limiting
- Temporary account lock
- Session invalidation
- Administrator notification
- User notification
- Temporary IP blocking

Responses should be proportional to the detected threat.

---

# SECURITY CHECKLIST

Before every production deployment, verify:

Authentication is functioning correctly.

Authorization has been validated.

Passwords use Argon2id.

JWT configuration is secure.

Secrets are stored outside the repository.

HTTPS is enabled.

Security headers are configured.

Rate limiting is active.

SQL Injection protections are verified.

XSS protections are verified.

CSRF protections are verified.

Uploaded files are validated.

Images are decoded and re-encoded.

Executable files are rejected.

Malware scanning is operational (if enabled).

Audit logging is functioning.

Backups are working.

Database credentials use least privilege.

Error messages do not expose internal information.

Unused endpoints have been removed.

Unused dependencies have been removed.

Production environment variables are configured.

---

# FUTURE IMPROVEMENTS

Future versions of RYZE may introduce additional security mechanisms, including:

- Two-Factor Authentication (2FA)
- Passkeys (WebAuthn)
- Hardware Security Keys
- Device Trust Management
- Session Management Dashboard
- Login History
- Suspicious Login Detection
- Geographic Login Detection
- AI-assisted Threat Detection
- Automatic Intrusion Detection
- Web Application Firewall (WAF)
- Real-time Malware Analysis
- Security Dashboard for Administrators

The database and backend architecture should remain flexible enough to support these features without requiring major structural changes.

---

# FINAL PRINCIPLES

Security is never considered "finished".

Every new feature introduced into the platform must be evaluated from a security perspective before implementation.

Whenever a more secure solution is identified, AI assistants are encouraged to propose it, provided it does not unnecessarily increase project complexity.

Security should evolve continuously alongside the platform.

The goal of RYZE is not only to protect data, but to build a secure, trustworthy and maintainable ecosystem capable of supporting long-term growth.
