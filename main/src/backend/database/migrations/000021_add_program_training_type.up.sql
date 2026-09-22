-- Generic Program Builder: platform-owned catalogue programs (trainer_id NULL)
-- now carry an optional marketplace training-type vocabulary. training_type
-- uses the documented marketplace vocabulary (Hypertrophy | Strength | HYROX |
-- CrossFit | Fat Loss | At Home) and is validated by the service layer; NULL
-- means the program does not advertise a specific training type.
ALTER TABLE programs
    ADD COLUMN training_type VARCHAR(40) NULL DEFAULT NULL AFTER frequency_per_week,
    ADD KEY idx_programs_training_type (training_type);