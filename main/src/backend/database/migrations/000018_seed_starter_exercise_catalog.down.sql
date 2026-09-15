DELETE FROM exercise_alternatives
WHERE exercise_id IN (SELECT id FROM exercises WHERE id LIKE 'd91e2f00-%');

DELETE FROM exercises
WHERE id LIKE 'd91e2f00-%';