# RYZE FITNESS
# API.MD

Version: 1.0.0

---

# PURPOSE

This document defines the complete REST API architecture used by the RYZE platform.

It is the single source of truth regarding:

- API architecture
- Endpoints
- Authentication
- Authorization
- Request structure
- Response structure
- Error handling
- Versioning
- Permissions
- Validation
- Uploads
- WebSockets

Every endpoint implemented in the backend must follow this document.

The API is expected to evolve during development.

Whenever a better architecture is identified, this document should be updated before implementation.

---

# API PHILOSOPHY

The API is designed with the following priorities:

- Simplicity
- Consistency
- Security
- Performance
- Scalability
- Maintainability

The API should always remain predictable.

Similar operations must behave similarly.

Responses should always follow the same structure.

Breaking changes should be avoided whenever possible.

When breaking changes become necessary, a new API version should be created.

---

# API STYLE

RYZE uses a RESTful architecture.

Resources should be represented by nouns rather than verbs.

Examples:

Good

```
GET /packs

GET /users

GET /notifications
```

Bad

```
GET /getPacks

POST /createPack

GET /retrieveNotifications
```

HTTP methods must represent the intended action.

GET

Read data.

POST

Create resources.

PUT

Replace a complete resource.

PATCH

Update part of a resource.

DELETE

Soft delete whenever applicable.

---

# API VERSIONING

Every endpoint is versioned.

Current version:

```
/api/v1
```

Examples

```
GET /api/v1/packs

POST /api/v1/auth/login

PATCH /api/v1/users/me
```

Future breaking changes should create:

```
/api/v2
```

instead of modifying v1 behavior.

---

# DATA FORMAT

All communication uses:

```
application/json
```

except:

- File uploads
- Image uploads
- PDF uploads

Those use:

```
multipart/form-data
```

Character encoding:

```
UTF-8
```

---

# RESPONSE FORMAT

Every endpoint must return the same response structure.

Successful response

```json
{
    "success": true,
    "message": "Operation completed successfully.",
    "data": {}
}
```

Failed response

```json
{
    "success": false,
    "message": "Validation failed.",
    "error": {
        "code": "VALIDATION_ERROR",
        "details": []
    }
}
```

Responses should always be easy to parse.

---

# HTTP STATUS CODES

The API should use proper HTTP status codes.

200

Successful request.

201

Resource created successfully.

204

Successful request without response body.

400

Invalid request.

401

Authentication required.

403

Permission denied.

404

Resource not found.

409

Conflict.

422

Validation error.

429

Too many requests.

500

Internal server error.

---

# AUTHENTICATION

Authentication uses:

JWT (JSON Web Tokens)

Authorization header:

```
Authorization: Bearer <token>
```

The backend is responsible for:

- Token generation
- Token validation
- Token expiration
- Refresh tokens
- Logout invalidation

Passwords are never stored in plain text.

Passwords must always be hashed using a secure algorithm.

Google OAuth authentication is also supported.

---

# AUTHORIZATION

Every authenticated user has exactly one primary role.

Supported roles:

- Client
- Trainer
- Technical Administrator
- Management Administrator

Permissions are determined exclusively by the backend.

Frontend restrictions are visual only and must never replace backend authorization.

---

# ROLE PERMISSIONS

## Public

Accessible without authentication.

Examples:

- Landing page
- Public trainer profiles
- Generic packs
- Authentication endpoints

---

## Authenticated User

Accessible after login.

Examples:

- Personal profile
- Purchases
- Notifications
- Messages
- Current pack

---

## Client

Additional permissions:

- Purchase packs
- Complete questionnaires
- Upload measurements
- View progress
- Share profile
- Leave feedback

---

## Trainer

Additional permissions:

- Create packs
- Edit packs
- Manage own packs
- View associated clients
- Access trainer dashboard

---

## Technical Administrator

Responsible for system administration.

Permissions include:

- User management
- Maintenance
- Logs
- Configuration
- Infrastructure

Technical administrators do not manage business content unless explicitly permitted.

---

## Management Administrator

Responsible for business management.

Permissions include:

- Pack approval
- Trainer approval
- Statistics
- Marketplace management
- Promotions
- Business reports

---

# REQUEST VALIDATION

Every request must be validated.

Validation occurs in the backend.

Never trust client-side validation.

Invalid requests must return descriptive validation errors.

---

# PAGINATION

Collection endpoints support pagination.

Example:

```
GET /packs?page=1&limit=20
```

Default page size:

20

Maximum page size:

100

The backend may adjust these limits in future versions.

---

# SORTING

Resources support sorting whenever applicable.

Examples:

```
?sort=name

?sort=created_at

?sort=price
```

Order:

```
?order=asc

?order=desc
```

---

# SEARCH

Endpoints may provide full-text search.

Example:

```
GET /packs?q=hypertrophy
```

Search behavior depends on the endpoint.

---

# FILTERING

Endpoints may expose resource-specific filters.

Examples:

```
GET /packs?difficulty=beginner

GET /packs?category=strength

GET /packs?trainer=sandro

GET /packs?equipment=home
```

Filters may be combined.

---

# FILE UPLOADS

Uploads use:

```
multipart/form-data
```

Supported upload types include:

- Profile pictures
- Trainer certificates
- Feedback images
- Pack resources
- Temporary PDFs

Binary files are stored on disk.

The database stores metadata only.

Uploads must be validated before storage.

Validation includes:

- MIME type
- File extension
- Maximum size
- Virus scanning (future)
- Image optimization

Temporary files should be removed automatically.

---

# WEBSOCKETS

Real-time communication uses Gorilla WebSocket.

WebSockets are reserved for events requiring immediate updates.

Examples:

- Chat messages
- Notifications
- Typing indicators
- System announcements
- Live dashboard updates

The API should continue functioning normally if WebSockets are unavailable.

---

# RATE LIMITING

Rate limiting protects the API against abuse.

Limits may differ depending on the endpoint.

Examples:

Authentication endpoints

More restrictive.

Read-only endpoints

Less restrictive.

Authenticated users may receive higher limits than anonymous visitors.

---

# CACHING

Caching should be implemented whenever beneficial.

Typical cache candidates include:

- Public packs
- Public trainer profiles
- Categories
- Static resources

Sensitive user information must never be cached publicly.

---

# ERROR HANDLING

Errors should always be consistent.

Error responses must:

- Clearly describe the problem.
- Avoid exposing internal implementation.
- Be easy to understand.
- Include machine-readable error codes whenever appropriate.

Unexpected server errors should always be logged.

---

# SECURITY

Every endpoint must follow secure development practices.

Including:

- JWT validation
- Permission verification
- SQL Injection prevention
- XSS prevention
- Input validation
- Output sanitization
- Rate limiting
- Secure headers

Security checks should never rely on the frontend.

---

# API EVOLUTION

The API is expected to evolve throughout the project's lifetime.

AI assistants are encouraged to improve:

- Endpoint organization
- Response structure
- Validation
- Security
- Performance
- Naming
- Documentation

...whenever those improvements are justified.

Breaking changes should be avoided whenever possible.

Whenever API behavior changes, both this document and `update_logs.MD` must be updated before implementation.

The API should remain predictable, RESTful and consistent across every version.

# AUTHENTICATION ENDPOINTS

Authentication is handled using JWT tokens.

Users may authenticate using:

- Email and password
- Google OAuth

The backend is responsible for validating credentials and generating access tokens.

---

## Login

```
POST /api/v1/auth/login
```

Authenticates a user using email and password.

Public endpoint.

---

## Admin Authentication

Administrator access uses two sequential steps. The first authenticates the
credentials, the second completes the second factor (the admin access code).
Only the second step issues the admin session cookie.

```
POST /api/v1/admin/auth/login
```

Step 1 — authenticates a platform administrator using a configured username
and password.

Request body:

```json
{
  "username": "...",
  "password": "..."
}
```

Only these two fields are accepted. Administrators are configuration-only
accounts defined by `ADMIN_1_USERNAME`/`ADMIN_1_PASSWORD` and
`ADMIN_2_USERNAME`/`ADMIN_2_PASSWORD` in `.env`; they are never stored in the
database. Missing, too-short or duplicated admin credentials fail at server
startup. Credential changes take effect on the next restart.

An unknown username or a wrong password fails with the same HTTP 401
`INVALID_CREDENTIALS`; callers cannot distinguish which case occurred. Empty
username/password or malformed JSON returns HTTP 400 `VALIDATION_ERROR`. On
success the endpoint returns HTTP 200 with an empty `data` object and sets a
short-lived temporary HttpOnly cookie `ryze_admin_stage_token` (`SameSite=Lax`,
`Secure` per `COOKIE_SECURE`, lifetime 5 minutes). This cookie only proves that
step 1 succeeded; it is not an authenticated admin session and is rejected by
the admin token validator.

Public endpoint.

---

```
POST /api/v1/admin/auth/verify
```

Step 2 — completes admin authentication with the access code. Requires the
`ryze_admin_stage_token` cookie issued by step 1.

Request body:

```json
{
  "access_code": "..."
}
```

Only `access_code` is accepted. The access code is a configuration-only secret
(`ADMIN_1_ACCESS_CODE`/`ADMIN_2_ACCESS_CODE` in `.env`, minimum 8 characters),
never stored in the database. Missing, too-short or invalid access codes fail
at server startup.

A missing, invalid or expired stage cookie, an unknown administrator or a wrong
access code all fail with the same HTTP 401 `INVALID_CREDENTIALS`; callers
cannot distinguish which case occurred. Empty `access_code` or malformed JSON
returns HTTP 400 `VALIDATION_ERROR`. On success the endpoint returns HTTP 200
with an empty `data` object, clears the `ryze_admin_stage_token` cookie and
sets the admin session HttpOnly cookie `ryze_admin_access_token`
(`SameSite=Lax`, `Secure` per `COOKIE_SECURE`, same lifetime as the user access
token). The admin cookie name differs from the user `ryze_access_token` cookie
so the two session kinds cannot be confused. The admin JWT carries the
administrator identity (`ADMIN_1`/`ADMIN_2`) as its subject and is only
accepted by admin token validation; user, admin and stage tokens are mutually
rejected. No password, access code, username, JWT, JWT secret or internal
details are ever returned.

Public endpoint.

Routes that require administrator access are protected by the admin
authentication middleware, which accepts only the `ryze_admin_access_token`
cookie validated as an `admin`-kind token carrying a configured admin identity
(`ADMIN_1`/`ADMIN_2`). Stage tokens, user tokens and any invalid token are
rejected with HTTP 401 `AUTHENTICATION_REQUIRED`.

---

## Register

```
POST /api/v1/auth/register
```

Creates a new client account.

Public endpoint.

Accounts are inactive until email verification is completed.

If the email belongs to a soft-deleted (previously deleted) account, the same
registration restores that account instead of creating a duplicate: the
original UUID and creation date are kept, the names are updated and the
password is replaced by the new password hash. The endpoint still returns
HTTP 201 and the same response structure in both cases. Registering an email
that belongs to an active account returns HTTP 409
`EMAIL_ALREADY_REGISTERED`.

---

## Logout

```
POST /api/v1/auth/logout
```

Invalidates the current session.

Authentication required.

---

## Change Password

```
POST /api/v1/auth/change-password
```

Replaces the password of the authenticated user after verifying the current
password.

Request body:

```json
{
  "current_password": "...",
  "new_password": "..."
}
```

Only these two fields are accepted. The user identity is resolved exclusively
from the authenticated session; client-provided identifiers are never accepted.

The current password must be verified, otherwise the request fails with
HTTP 401 `INVALID_CREDENTIALS`. The new password replaces the stored
`password_hash` and the account's `session_version` is incremented, revoking
every previously issued access token. On success the response clears the
`ryze_access_token` cookie (HTTP 200, empty `data`) and the user must
authenticate again. Empty passwords return HTTP 400 `VALIDATION_ERROR`.

Authentication required.

---

## Delete Account

```
POST /api/v1/auth/delete-account
```

Soft-deletes the authenticated user's account after verifying the current
password.

Request body:

```json
{
  "password": "..."
}
```

Only this single field is accepted. The user identity is resolved exclusively
from the authenticated session; client-provided identifiers are never accepted.

The account is never physically removed: the row is preserved with its UUID,
email and `created_at` unchanged and `deleted_at` populated. The account's
`session_version` is incremented in the same operation, revoking every
previously issued access token, and the `ryze_access_token` cookie is cleared.
A soft-deleted account can no longer log in; re-registering with the same
email reactivates the original row (same UUID) instead of creating a duplicate.

A wrong password, an unknown or already-deleted user and an unverifiable stored
hash all fail with the same HTTP 401 `INVALID_CREDENTIALS`. An empty password
or malformed JSON returns HTTP 400 `VALIDATION_ERROR`. On success the response
returns HTTP 200 with an empty `data` object. No password, hash, JWT, JWT
secret or internal details (`deleted_at`, database columns) are ever returned.

Authentication required.

---

## Refresh Token

```
POST /api/v1/auth/refresh
```

Generates a new JWT access token.

Authentication required.

---

## Forgot Password

```
POST /api/v1/auth/forgot-password
```

Sends a password recovery email.

Public endpoint.

---

## Reset Password

```
POST /api/v1/auth/reset-password
```

Updates the user's password using a recovery token.

Public endpoint.

---

## Verify Email

```
POST /api/v1/auth/verify-email
```

Verifies the user's email address.

Public endpoint.

---

## Resend Verification

```
POST /api/v1/auth/resend-verification
```

Resends the email verification link.

Public endpoint.

---

## Google Authentication

```
GET /api/v1/auth/google
```

Redirects the user to Google's authentication page.

---

## Google Callback

```
GET /api/v1/auth/google/callback
```

Processes Google's authentication response.

---

# USER ENDPOINTS

Endpoints related to user accounts.

Authentication required unless otherwise stated.

---

## Current User

```
GET /api/v1/users/me
```

Returns the authenticated user's information.

---

## Update Current User

```
PATCH /api/v1/users/me
```

Updates editable account information.

Examples:

- Display name
- Biography
- Location
- Language
- Preferences

---

## Delete Account

```
DELETE /api/v1/users/me
```

Performs a soft delete.

The account remains stored but becomes inactive.

---

## Change Password

```
PATCH /api/v1/users/me/password
```

Changes the current password.

Requires the current password.

---

## Upload Avatar

```
POST /api/v1/users/me/avatar
```

Uploads a profile picture.

Image optimization is performed automatically.

---

## Remove Avatar

```
DELETE /api/v1/users/me/avatar
```

Removes the current avatar.

---

## Upload Banner

```
POST /api/v1/users/me/banner
```

Uploads a profile banner.

---

## Remove Banner

```
DELETE /api/v1/users/me/banner
```

Removes the profile banner.

---

## Public Profile

```
GET /api/v1/users/{username}
```

Returns the public profile of any user.

Visibility depends on privacy settings.

---

## Public Packs

```
GET /api/v1/users/{username}/packs
```

Returns packs published by a trainer.

Public endpoint.

---

## Search Users

```
GET /api/v1/users/search
```

Searches public users.

Supports:

- Name
- Username
- Trainer
- Country

---

# USER SETTINGS

User preferences.

Authentication required.

---

## Get Settings

```
GET /api/v1/settings
```

Returns every configurable preference.

---

## Update Settings

```
PATCH /api/v1/settings
```

Updates preferences.

Examples:

- Language
- Notifications
- Privacy
- Units
- Public profile

---

## Notification Settings

```
PATCH /api/v1/settings/notifications
```

Updates notification preferences.

---

## Privacy Settings

```
PATCH /api/v1/settings/privacy
```

Updates profile visibility.

---

## Connected Accounts

```
GET /api/v1/settings/accounts
```

Returns connected authentication providers.

Example:

- Google

---

## Link Google Account

```
POST /api/v1/settings/accounts/google
```

Associates a Google account.

---

## Unlink Google Account

```
DELETE /api/v1/settings/accounts/google
```

Removes the Google association.

---

# NOTIFICATION ENDPOINTS

Notifications generated by the platform.

Authentication required.

---

## List Notifications

```
GET /api/v1/notifications
```

Returns recent notifications.

Supports pagination.

---

## Notification Count

```
GET /api/v1/notifications/count
```

Returns the number of unread notifications.

---

## Mark as Read

```
PATCH /api/v1/notifications/{id}/read
```

Marks a notification as read.

---

## Mark All as Read

```
PATCH /api/v1/notifications/read-all
```

Marks every notification as read.

---

## Delete Notification

```
DELETE /api/v1/notifications/{id}
```

Removes a notification from the interface.

---

# FILE UPLOAD ENDPOINTS

Handles media uploaded by users.

Authentication required.

---

## Upload Image

```
POST /api/v1/uploads/image
```

Uploads an image.

Supported uses:

- Avatar
- Banner
- Feedback
- Trainer resources

---

## Upload Temporary File

```
POST /api/v1/uploads/temp
```

Uploads a temporary file.

Files are automatically removed after expiration.

---

## Delete Temporary File

```
DELETE /api/v1/uploads/temp/{id}
```

Deletes a temporary upload.

---

## Upload Trainer Certificate

```
POST /api/v1/uploads/certificate
```

Uploads professional certification during trainer application.

---

# LOCALIZATION

---

## Supported Languages

```
GET /api/v1/localization/languages
```

Returns every supported language.

Current languages:

- English
- Portuguese

---

## Change Language

```
PATCH /api/v1/localization/language
```

Updates the user's preferred language.

If no preference exists, the backend automatically selects the language based on the user's approximate location.

---

# HEALTH CHECK

Public endpoints used for monitoring.

---

## API Status

```
GET /api/v1/health
```

Returns the current API status.

---

## API Version

```
GET /api/v1/version
```

Returns the running API version.

---

# NOTES

All endpoints described in this section represent the base user management layer of the RYZE platform.

Additional modules such as Packs, Purchases, Payments, Workouts, Nutrition, Feedback, Chat, Trainer and Administration are documented in the following sections.

# PACK ENDPOINTS

Packs represent the commercial products available on the RYZE platform.

Every pack belongs to one of the supported categories:

- Generic
- Premium Level 1
- Premium Level 2

Only trainers and management administrators can create or modify packs.

---

## List Packs

```
GET /api/v1/packs
```

Returns every available pack.

Supports:

- Pagination
- Search
- Sorting
- Filtering

---

## Get Pack

```
GET /api/v1/packs/{slug}
```

Returns detailed information about a specific pack.

Public endpoint.

---

## Search Packs

```
GET /api/v1/packs/search
```

Searches available packs.

Supports:

- Name
- Difficulty
- Trainer
- Equipment
- Goal
- Category

---

## Featured Packs

```
GET /api/v1/packs/featured
```

Returns featured packs displayed on the homepage.

---

## Create Pack

```
POST /api/v1/packs
```

Creates a new pack.

Trainer or Management Administrator only.

---

## Update Pack

```
PATCH /api/v1/packs/{id}
```

Updates an existing pack.

Changes create a new version whenever necessary.

---

## Publish Pack

```
PATCH /api/v1/packs/{id}/publish
```

Publishes a draft pack.

---

## Unpublish Pack

```
PATCH /api/v1/packs/{id}/unpublish
```

Temporarily removes a pack from the marketplace.

---

## Delete Pack

```
DELETE /api/v1/packs/{id}
```

Soft deletes a pack.

Historical purchases remain untouched.

---

# PURCHASE ENDPOINTS

Handles every purchase performed on the platform.

---

## Checkout

```
POST /api/v1/purchases/checkout
```

Starts the checkout process.

---

## Purchase History

```
GET /api/v1/purchases
```

Returns every purchase performed by the authenticated user.

---

## Purchase Details

```
GET /api/v1/purchases/{id}
```

Returns detailed purchase information.

---

## Active Pack

```
GET /api/v1/purchases/current
```

Returns the currently active pack.

---

## Expired Packs

```
GET /api/v1/purchases/history
```

Returns previous packs already completed.

---

## Cancel Purchase

```
POST /api/v1/purchases/{id}/cancel
```

Cancels a purchase when allowed.

---

# PAYMENT ENDPOINTS

Responsible for payment processing.

---

## Payment Methods

```
GET /api/v1/payments/methods
```

Returns supported payment providers.

---

## Create Payment

```
POST /api/v1/payments
```

Creates a payment request.

---

## Payment Status

```
GET /api/v1/payments/{id}
```

Returns payment status.

---

## Retry Payment

```
POST /api/v1/payments/{id}/retry
```

Attempts payment again.

---

## Invoice

```
GET /api/v1/payments/{id}/invoice
```

Downloads the generated invoice.

---

## Refund

```
POST /api/v1/payments/{id}/refund
```

Processes a refund when applicable.

---

# QUESTIONNAIRE ENDPOINTS

Premium programs require questionnaires.

---

## List Questionnaires

```
GET /api/v1/questionnaires
```

Returns available questionnaires.

---

## Questionnaire Details

```
GET /api/v1/questionnaires/{id}
```

Returns a questionnaire.

---

## Submit Questionnaire

```
POST /api/v1/questionnaires/{id}
```

Stores questionnaire answers.

---

## Questionnaire History

```
GET /api/v1/questionnaires/history
```

Returns previously submitted questionnaires.

---

# NUTRITION ENDPOINTS

Nutrition programs associated with purchased packs.

---

## Current Nutrition Program

```
GET /api/v1/nutrition/current
```

Returns the user's current nutrition program.

---

## Nutrition History

```
GET /api/v1/nutrition/history
```

Returns previous nutrition programs.

---

## Download Nutrition PDF

```
GET /api/v1/nutrition/pdf
```

Downloads the nutrition program.

---

# WORKOUT ENDPOINTS

Training management.

---

## Current Workout

```
GET /api/v1/workouts/current
```

Returns today's workout.

---

## Workout History

```
GET /api/v1/workouts/history
```

Returns completed workouts.

---

## Workout Details

```
GET /api/v1/workouts/{id}
```

Returns a completed workout.

---

## Complete Workout

```
POST /api/v1/workouts/{id}/complete
```

Marks a workout as completed.

---

## Download Workout PDF

```
GET /api/v1/workouts/pdf
```

Downloads the workout program.

---

# EXERCISE ENDPOINTS

Exercise library.

---

## Exercise Library

```
GET /api/v1/exercises
```

Returns every exercise.

---

## Exercise Details

```
GET /api/v1/exercises/{id}
```

Returns complete exercise information.

Includes:

- Description
- Images
- Videos
- Target muscles
- Equipment

---

## Exercise Search

```
GET /api/v1/exercises/search
```

Searches exercises.

---

# MEASUREMENT ENDPOINTS

Body measurements.

---

## Current Measurements

```
GET /api/v1/measurements
```

Returns current measurements.

---

## Add Measurement

```
POST /api/v1/measurements
```

Stores a new measurement.

---

## Update Measurement

```
PATCH /api/v1/measurements/{id}
```

Updates a measurement.

---

## Measurement History

```
GET /api/v1/measurements/history
```

Returns historical evolution.

---

# ACHIEVEMENT ENDPOINTS

Gamification.

---

## Achievements

```
GET /api/v1/achievements
```

Returns unlocked achievements.

---

## Badges

```
GET /api/v1/badges
```

Returns earned badges.

---

## Missions

```
GET /api/v1/missions
```

Returns active missions.

---

## Events

```
GET /api/v1/events
```

Returns active events.

---

## Streak

```
GET /api/v1/streak
```

Returns the user's current workout streak.

---

# FEEDBACK ENDPOINTS

Community feedback.

---

## Feed

```
GET /api/v1/feedback
```

Returns the public feed.

---

## Create Post

```
POST /api/v1/feedback
```

Creates a new post.

---

## Update Post

```
PATCH /api/v1/feedback/{id}
```

Updates a post.

---

## Delete Post

```
DELETE /api/v1/feedback/{id}
```

Soft deletes a post.

---

## Like Post

```
POST /api/v1/feedback/{id}/like
```

Likes a post.

---

## Remove Like

```
DELETE /api/v1/feedback/{id}/like
```

Removes a like.

---

## Comments

```
GET /api/v1/feedback/{id}/comments
```

Returns every comment.

---

## Create Comment

```
POST /api/v1/feedback/{id}/comments
```

Creates a comment.

---

## Delete Comment

```
DELETE /api/v1/comments/{id}
```

Removes a comment.

---

# CHAT ENDPOINTS

Messaging between clients and trainers.

---

## Conversations

```
GET /api/v1/chat
```

Returns every conversation.

---

## Conversation

```
GET /api/v1/chat/{id}
```

Returns a conversation.

---

## Messages

```
GET /api/v1/chat/{id}/messages
```

Returns conversation messages.

---

## Send Message

```
POST /api/v1/chat/{id}/messages
```

Sends a new message.

---

## Delete Message

```
DELETE /api/v1/chat/messages/{id}
```

Soft deletes a message.

---

## Upload Chat Image

```
POST /api/v1/chat/upload
```

Uploads an image inside a conversation.

---

This section defines every endpoint related to the platform's core functionality, including packs, purchases, workouts, nutrition, feedback, messaging and user progression.

Administrative endpoints, trainer management and system operations are documented in the final section.

# TRAINER ENDPOINTS

Endpoints available only to approved trainers.

---

## Trainer Dashboard

```
GET /api/v1/trainer/dashboard
```

Returns the trainer's dashboard.

Includes:

- Revenue
- Clients
- Published packs
- Statistics
- Ratings

---

## My Packs

```
GET /api/v1/trainer/packs
```

Returns every pack created by the trainer.

---

## Create Pack

```
POST /api/v1/trainer/packs
```

Creates a new training pack.

Initially created as a draft.

---

## Update Pack

```
PATCH /api/v1/trainer/packs/{id}
```

Updates one of the trainer's packs.

---

## Delete Pack

```
DELETE /api/v1/trainer/packs/{id}
```

Soft deletes one of the trainer's packs.

---

## Published Clients

```
GET /api/v1/trainer/clients
```

Returns every client associated with the trainer.

---

## Client Details

```
GET /api/v1/trainer/clients/{id}
```

Returns detailed client information available to the trainer.

---

## Trainer Statistics

```
GET /api/v1/trainer/statistics
```

Returns statistics including:

- Revenue
- Active clients
- Sales
- Pack popularity
- Reviews

---

## Trainer Revenue

```
GET /api/v1/trainer/revenue
```

Returns financial statistics.

---

# TRAINER APPLICATION

Users may apply to become trainers.

---

## Apply

```
POST /api/v1/trainer/application
```

Submits a trainer application.

Includes:

- Curriculum Vitae
- Certificates
- Personal information

---

## Application Status

```
GET /api/v1/trainer/application
```

Returns the application's current status.

---

## Cancel Application

```
DELETE /api/v1/trainer/application
```

Withdraws the application.

---

# TECHNICAL ADMINISTRATION

Accessible only by Technical Administrators.

---

## Dashboard

```
GET /api/v1/admin/dashboard
```

Returns system statistics.

---

## Users

```
GET /api/v1/admin/users
```

Returns every registered user.

Supports filtering and pagination.

---

## User Details

```
GET /api/v1/admin/users/{id}
```

Returns complete account information.

---

## Update User

```
PATCH /api/v1/admin/users/{id}
```

Updates user information.

---

## Disable User

```
PATCH /api/v1/admin/users/{id}/disable
```

Soft disables an account.

---

## Enable User

```
PATCH /api/v1/admin/users/{id}/enable
```

Reactivates a disabled account.

---

## Audit Logs

```
GET /api/v1/admin/logs
```

Returns administrative audit logs.

---

## System Settings

```
GET /api/v1/admin/settings
```

Returns global platform settings.

---

## Update System Settings

```
PATCH /api/v1/admin/settings
```

Updates global configuration.

---

## Maintenance Mode

```
PATCH /api/v1/admin/maintenance
```

Enables or disables maintenance mode.

---

# MANAGEMENT ADMINISTRATION

Accessible only by Management Administrators.

---

## Dashboard

```
GET /api/v1/management/dashboard
```

Returns business statistics.

---

## Marketplace Statistics

```
GET /api/v1/management/statistics
```

Returns marketplace analytics.

---

## Trainers

```
GET /api/v1/management/trainers
```

Returns every registered trainer.

---

## Pending Applications

```
GET /api/v1/management/trainer-applications
```

Returns trainer applications awaiting review.

---

## Approve Trainer

```
PATCH /api/v1/management/trainer-applications/{id}/approve
```

Approves a trainer application.

---

## Reject Trainer

```
PATCH /api/v1/management/trainer-applications/{id}/reject
```

Rejects a trainer application.

---

## Platform Revenue

```
GET /api/v1/management/revenue
```

Returns platform-wide revenue.

---

## Sales

```
GET /api/v1/management/sales
```

Returns sales statistics.

---

## Promotions

```
GET /api/v1/management/promotions
```

Returns active promotions.

---

## Create Promotion

```
POST /api/v1/management/promotions
```

Creates a promotion.

---

## Update Promotion

```
PATCH /api/v1/management/promotions/{id}
```

Updates a promotion.

---

## Delete Promotion

```
DELETE /api/v1/management/promotions/{id}
```

Disables a promotion.

---

# SCHEDULER ENDPOINTS

Reserved for internal system operations.

These endpoints are not publicly accessible.

Examples:

- Notification cleanup
- Temporary upload cleanup
- Scheduled emails
- Cache refresh
- Statistics generation
- Ranking updates
- Event activation
- Event expiration
- Pack expiration
- Backup generation

---

# WEBSOCKET EVENTS

The platform exposes several real-time events.

Examples include:

```
notification:new

notification:read

chat:new_message

chat:typing

chat:message_deleted

trainer:application_status

purchase:completed

pack:published

mission:completed

achievement:unlocked

event:started

event:ended

system:announcement
```

Additional events may be introduced as the platform evolves.

---

# API DOCUMENTATION

Every endpoint should include:

- Description
- Authentication requirements
- Required permissions
- Request body
- Response body
- Possible errors
- Validation rules

API documentation should remain synchronized with the implementation.

---

# DEPRECATION POLICY

Deprecated endpoints should not be removed immediately.

Instead:

- Mark the endpoint as deprecated.
- Document the replacement endpoint.
- Maintain compatibility until the next major API version.

Breaking changes should only occur between major API versions.

---

## RYZE Feed API

The RYZE Feed API provides endpoints for consuming Feed content, managing Likes and Follows, and allowing authorized trainers to publish and manage their own content.

All endpoints must follow the existing RYZE API conventions, authentication requirements, versioning strategy, validation rules, and error-handling standards.

### Feed Consumption

#### Get Feed

```http
GET /api/v1/feed

Returns a paginated set of Feed content personalized for the authenticated user.

The response should provide the information required to render the Feed without requiring unnecessary additional requests.

Conceptual response data:

feed item
├── id
├── video
├── thumbnail
├── title
├── description
├── trainer
│   ├── id
│   ├── name
│   └── profile_image
├── likes_count
└── user_liked

The endpoint should support incremental loading suitable for infinite scrolling.

The pagination mechanism must be stable enough to prevent excessive duplication or skipping of content while the user is consuming the Feed.

Feed Content
Get Feed Item
GET /api/v1/feed/:id

Returns a single published Feed item and its associated creator information.

Unpublished, hidden, deleted, or otherwise inaccessible content must not be returned to normal users.

Likes
Like Feed Item
POST /api/v1/feed/:id/like

Creates a Like for the authenticated user.

The operation must be idempotent from the client's perspective.

A user must not be able to create duplicate Likes for the same Feed item.

Unlike Feed Item
DELETE /api/v1/feed/:id/like

Removes the authenticated user's Like from the Feed item.

The operation should remain safe if the user has not previously Liked the content.

Trainer Following
Follow Trainer
POST /api/v1/trainers/:id/follow

Creates a Follow relationship between the authenticated user and the specified trainer.

The operation must prevent duplicate Follow relationships.

Unfollow Trainer
DELETE /api/v1/trainers/:id/follow

Removes the authenticated user's Follow relationship with the specified trainer.

Get Trainer Follow State

The API should provide the current Follow state where required by Feed and Trainer interfaces.

The exact endpoint may be:

GET /api/v1/trainers/:id/follow

or incorporated into the Trainer response.

The implementation should avoid unnecessary requests when the state can already be returned as part of another response.

Trainer Content Management

Only authorized trainers may create and manage their own RYZE Feed content.

Create Feed Content
POST /api/v1/trainer/feed

Creates a new Feed content record.

The initial content should not automatically become publicly available if video processing, validation, moderation, or other required processing has not completed.

Upload Video
POST /api/v1/trainer/feed/:id/video

Uploads or associates the video media with the Feed content.

The upload must pass all RYZE media security and validation requirements before becoming available for playback.

Update Feed Content
PATCH /api/v1/trainer/feed/:id

Allows the owning trainer to modify editable metadata such as:

Title
Description
Category/tags when supported
Visibility/status when permitted

A trainer must not be able to modify content belonging to another trainer.

Delete Feed Content
DELETE /api/v1/trainer/feed/:id

Removes the trainer's Feed content according to the project's established deletion strategy.

Deleted content must immediately stop appearing in normal Feed results.

Authorization

Feed permissions are separated into two primary groups.

Authenticated Users

Users may:

View Feed content
Like content
Remove Likes
Follow trainers
Unfollow trainers
View public trainer information
Authorized Trainers

Trainers may additionally:

Create Feed content
Upload videos
Edit their own content
Publish content when permitted
Hide their own content
Remove their own content

A trainer must never be able to modify another trainer's content.

Content Visibility

The Feed API must distinguish between content states.

Normal Feed requests should only return content that is:

Published
Valid
Available for the current user
Not deleted
Not hidden

Administrative or trainer-management endpoints may access additional states according to their authorization level.

Personalization

Feed retrieval may use:

Trainer follows
Video Likes
Content relevance
Content freshness
Popularity
Future recommendation signals

The API contract should not expose or depend on the internal ranking formula.

The ranking system may evolve independently while maintaining the same Feed API contract.

Error Handling

The API must use the standard RYZE error response format.

Examples of relevant errors include:

401 Unauthorized
403 Forbidden
404 Feed content not found
409 Relationship already exists
422 Invalid content

The exact status codes must follow the global RYZE API conventions.

Security

All Feed API endpoints must follow the security requirements defined in security.md.

Particular attention must be given to:

Authentication
Authorization
Trainer ownership checks
Rate limiting
Input validation
Upload validation
File type validation
File size limits
Malicious media detection
Abuse prevention
Resource enumeration protection

Feed APIs must never trust client-provided trainer IDs, user IDs, ownership information, file metadata, or permission claims.

---

# FUTURE EXPANSION

The API has been designed to support future features without major architectural changes.

Possible future additions include:

- Native mobile applications
- Public developer API
- GraphQL gateway
- AI endpoints
- Advanced analytics
- Additional payment providers
- Multi-currency support
- Wearable integrations
- Health platform integrations
- External trainer APIs

The existing architecture should be extended whenever possible instead of being replaced.

---

# FINAL NOTES

This document defines the official API specification of the RYZE platform.

The API is expected to evolve throughout development.

AI assistants may introduce:

- New endpoints
- New parameters
- Additional response fields
- Better naming
- Improved security
- Improved validation
- Better endpoint organization

...provided that these changes improve the platform's architecture and remain consistent with the project's documentation.

Whenever the API changes:

- Update `api.MD`.
- Update `update_logs.MD`.
- Maintain backward compatibility whenever possible.
- Avoid breaking changes unless a new API version is introduced.

The RYZE API should always prioritize consistency, simplicity, security, scalability and long-term maintainability.
