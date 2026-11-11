-- Trainer-application evidence documents and reviewer decision history
-- (v2.94.0).
--
-- Reviewing an application needs two things the schema could not express:
-- the evidence a human reviewer looks at, and a durable record of who decided
-- what and why. Both are added here.
--
-- trainer_application_documents stores one row per uploaded evidence file. The
-- binary content lives outside the database in the private evidence store
-- (never under a web root); the row keeps only the metadata a reviewer needs
-- plus the opaque storage key. Files are validated by magic bytes and size
-- before a row exists, and only an allow-listed content type can be stored, so
-- a document row can never point at an executable or unvalidated payload.
-- Rows are soft-deleted so a mistaken upload can be withdrawn without
-- destroying the review trail.
--
-- trainer_application_decisions is the append-only decision history: one row
-- per approve or reject action, recording the reviewing administrator identity,
-- the status transition and an optional reason. It is never updated or
-- soft-deleted: a review decision is an audit fact. The admin identity is the
-- configured administrator id (ADMIN_1/ADMIN_2), not a database user: platform
-- administrators are configuration-only identities, so the applicant can never
-- be the reviewer and no decision can be self-issued.
CREATE TABLE trainer_application_documents (
    id VARCHAR(36) NOT NULL,
    application_id VARCHAR(36) NOT NULL,
    kind VARCHAR(40) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    byte_size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT trainer_application_documents_pkey PRIMARY KEY (id),
    CONSTRAINT uq_trainer_application_documents_storage_key UNIQUE (storage_key),
    CONSTRAINT fk_trainer_application_documents_application
        FOREIGN KEY (application_id) REFERENCES trainer_applications (id),
    CONSTRAINT chk_trainer_application_documents_kind
        CHECK (kind IN ('certification', 'identification', 'insurance', 'other')),
    CONSTRAINT chk_trainer_application_documents_content_type
        CHECK (content_type IN ('application/pdf', 'image/jpeg', 'image/png')),
    CONSTRAINT chk_trainer_application_documents_byte_size
        CHECK (byte_size > 0)
);

CREATE INDEX idx_trainer_application_documents_application ON trainer_application_documents (application_id);

CREATE TABLE trainer_application_decisions (
    id VARCHAR(36) NOT NULL,
    application_id VARCHAR(36) NOT NULL,
    admin_id VARCHAR(40) NOT NULL,
    from_status VARCHAR(20) NOT NULL,
    to_status VARCHAR(20) NOT NULL,
    reason VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT trainer_application_decisions_pkey PRIMARY KEY (id),
    CONSTRAINT fk_trainer_application_decisions_application
        FOREIGN KEY (application_id) REFERENCES trainer_applications (id),
    CONSTRAINT chk_trainer_application_decisions_from_status
        CHECK (from_status IN ('PENDING', 'APPROVED', 'REJECTED')),
    CONSTRAINT chk_trainer_application_decisions_to_status
        CHECK (to_status IN ('PENDING', 'APPROVED', 'REJECTED'))
);

CREATE INDEX idx_trainer_application_decisions_application ON trainer_application_decisions (application_id, created_at);
