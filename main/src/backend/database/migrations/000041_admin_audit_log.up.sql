-- Administrative audit log.
--
-- One row per administrative action: every state-changing request an ADMIN_1 or
-- ADMIN_2 session makes (create, update, publish, delete, approve, reject,
-- reply, pricing, commission, Test Mode), every evidence-document download and
-- every second-factor sign-in attempt. Denied attempts (the role lacks the
-- permission) are recorded too.
--
-- Rows describe the action, never its content: the route template, the target
-- identifier from the path, the HTTP status and the outcome. Request bodies,
-- passwords, access codes, notes and personal answers are never stored.
--
-- Rows are immutable: a trigger rejects UPDATE and DELETE, so nobody (not even
-- the application) can rewrite history. There is no soft delete and no
-- updated_at.
CREATE TABLE admin_audit_logs (
    id VARCHAR(36) NOT NULL,
    admin_id VARCHAR(16) NOT NULL,
    action VARCHAR(64) NOT NULL,
    method VARCHAR(8) NOT NULL,
    route VARCHAR(160) NOT NULL,
    target_id VARCHAR(64),
    status_code SMALLINT NOT NULL,
    outcome VARCHAR(12) NOT NULL,
    client_ip VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT admin_audit_logs_pkey PRIMARY KEY (id),
    CONSTRAINT chk_admin_audit_logs_admin CHECK (admin_id IN ('ADMIN_1', 'ADMIN_2')),
    CONSTRAINT chk_admin_audit_logs_outcome CHECK (outcome IN ('success', 'denied', 'failed')),
    CONSTRAINT chk_admin_audit_logs_status CHECK (status_code BETWEEN 100 AND 599)
);

CREATE INDEX idx_admin_audit_logs_created ON admin_audit_logs (created_at DESC, id);
CREATE INDEX idx_admin_audit_logs_admin_created ON admin_audit_logs (admin_id, created_at DESC);

CREATE FUNCTION ryze_reject_audit_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'admin_audit_logs rows are immutable';
END;
$$;

CREATE TRIGGER trg_admin_audit_logs_immutable BEFORE UPDATE OR DELETE ON admin_audit_logs
    FOR EACH ROW EXECUTE FUNCTION ryze_reject_audit_change();
