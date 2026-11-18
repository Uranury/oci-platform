-- Lossy: sets that only fit the new tracking types, sets on system
-- exercises, and the system exercises themselves are deleted, and archived
-- custom exercises come back as normal ones (this fails if one now clashes by
-- name with a live exercise of the same user; rename it first).
DELETE FROM workout_sets WHERE reps IS NULL
    OR exercise_id IN (SELECT id FROM exercises WHERE user_id IS NULL);
UPDATE workout_sets SET weight_kg = 0 WHERE weight_kg IS NULL;

ALTER TABLE workout_sets
    DROP CONSTRAINT workout_sets_measured_check,
    DROP COLUMN distance_m,
    DROP COLUMN duration_seconds,
    ALTER COLUMN weight_kg SET DEFAULT 0,
    ALTER COLUMN weight_kg SET NOT NULL,
    ALTER COLUMN reps SET NOT NULL;

DELETE FROM exercises WHERE user_id IS NULL;

DROP INDEX idx_exercises_slug;
DROP INDEX idx_exercises_system_name;
DROP INDEX idx_exercises_custom_name;
CREATE UNIQUE INDEX idx_exercises_user_name ON exercises (user_id, lower(name));

ALTER TABLE exercises
    DROP CONSTRAINT exercises_tracking_type_check,
    DROP CONSTRAINT exercises_equipment_check,
    DROP CONSTRAINT exercises_mechanic_check,
    DROP CONSTRAINT exercises_secondary_muscles_check,
    DROP CONSTRAINT exercises_primary_muscle_check,
    DROP CONSTRAINT exercises_movement_pattern_check,
    DROP CONSTRAINT exercises_name_check,
    DROP CONSTRAINT exercises_owner_slug_check,
    DROP COLUMN updated_at,
    DROP COLUMN archived_at,
    DROP COLUMN tracking_type,
    DROP COLUMN equipment,
    DROP COLUMN mechanic,
    DROP COLUMN secondary_muscles,
    DROP COLUMN primary_muscle,
    DROP COLUMN movement_pattern,
    DROP COLUMN slug,
    ALTER COLUMN user_id SET NOT NULL;
