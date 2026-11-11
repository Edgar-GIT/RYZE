-- Generic Program Builder: platform-owned catalogue programs (trainer_id NULL)
-- now carry an optional marketplace training-type vocabulary. training_type
-- uses the documented marketplace vocabulary (Hypertrophy | Strength | HYROX |
-- CrossFit | Fat Loss | At Home) and is validated by the service layer; NULL
-- means the program does not advertise a specific training type.
ALTER TABLE programs
    ADD COLUMN training_type VARCHAR(40);

CREATE INDEX idx_programs_training_type ON programs (training_type);
