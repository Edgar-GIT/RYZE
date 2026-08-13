CREATE TABLE exercises (
    id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL DEFAULT NULL,
    target_muscles VARCHAR(255) NULL DEFAULT NULL,
    equipment VARCHAR(255) NULL DEFAULT NULL,
    difficulty VARCHAR(50) NULL DEFAULT NULL,
    video_url VARCHAR(500) NULL DEFAULT NULL,
    image_url VARCHAR(500) NULL DEFAULT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_name VARCHAR(255) GENERATED ALWAYS AS (IF(deleted_at IS NULL, name, NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_exercises_active_name (active_name),
    KEY idx_exercises_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
