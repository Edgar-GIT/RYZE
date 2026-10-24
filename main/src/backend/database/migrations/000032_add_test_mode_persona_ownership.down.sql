ALTER TABLE users
    DROP INDEX uq_users_active_test_persona,
    DROP CONSTRAINT chk_users_test_persona,
    DROP COLUMN active_test_persona,
    DROP COLUMN test_persona;
