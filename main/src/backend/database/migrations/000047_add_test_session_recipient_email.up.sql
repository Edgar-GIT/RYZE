-- A Test Mode session can carry one administrator-supplied recipient address.
-- Personas have non-deliverable @ryze.local accounts, so this address is where
-- the normal purchase and lifecycle emails go while the session is active. It
-- is test data only: leaving Test Mode soft-deletes the session and the address
-- stops being used.
ALTER TABLE test_sessions ADD COLUMN recipient_email VARCHAR(255) NULL;
