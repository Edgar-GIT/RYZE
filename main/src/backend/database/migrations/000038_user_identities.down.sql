DROP TABLE IF EXISTS user_identities;

ALTER TABLE users
    DROP COLUMN password_set;
