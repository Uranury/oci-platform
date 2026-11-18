-- Exercise library: system defaults (user_id NULL, seeded by slug from
-- internal/seed at startup) next to private custom exercises, plus
-- categories, tracking types and soft delete.
--
-- The value lists in the CHECK constraints mirror internal/models/
-- exercise_enums.go (TestEnumsMatchMigrations keeps them in sync). Plain
-- TEXT + CHECK rather than native enum types: a later migration can change
-- a list by dropping and re-adding one constraint.

ALTER TABLE exercises ALTER COLUMN user_id DROP NOT NULL;

-- Existing rows are all custom exercises logged as weight × reps; they get
-- neutral categories the owner can edit afterwards.
ALTER TABLE exercises
    ADD COLUMN slug              TEXT,
    ADD COLUMN movement_pattern  TEXT        NOT NULL DEFAULT 'other',
    ADD COLUMN primary_muscle    TEXT        NOT NULL DEFAULT 'other',
    ADD COLUMN secondary_muscles TEXT[]      NOT NULL DEFAULT '{}',
    ADD COLUMN mechanic          TEXT        NOT NULL DEFAULT 'compound',
    ADD COLUMN equipment         TEXT        NOT NULL DEFAULT 'other',
    ADD COLUMN tracking_type     TEXT        NOT NULL DEFAULT 'weight_reps',
    ADD COLUMN archived_at       TIMESTAMPTZ,
    ADD COLUMN updated_at        TIMESTAMPTZ NOT NULL DEFAULT now();

-- Defaults were only for the backfill; the app always sets these.
ALTER TABLE exercises
    ALTER COLUMN movement_pattern DROP DEFAULT,
    ALTER COLUMN primary_muscle   DROP DEFAULT,
    ALTER COLUMN mechanic         DROP DEFAULT,
    ALTER COLUMN equipment        DROP DEFAULT,
    ALTER COLUMN tracking_type    DROP DEFAULT;

ALTER TABLE exercises
    -- System rows have a slug and no owner; custom rows the reverse.
    ADD CONSTRAINT exercises_owner_slug_check CHECK ((user_id IS NULL) = (slug IS NOT NULL)),
    ADD CONSTRAINT exercises_name_check CHECK (btrim(name) <> '' AND char_length(name) <= 100),
    ADD CONSTRAINT exercises_movement_pattern_check CHECK (movement_pattern IN (
        'horizontal_push', 'vertical_push', 'horizontal_pull', 'vertical_pull',
        'squat', 'hinge', 'lunge', 'carry', 'core', 'other')),
    ADD CONSTRAINT exercises_primary_muscle_check CHECK (primary_muscle IN (
        'chest', 'lats', 'upper_back', 'shoulders', 'biceps', 'triceps', 'forearms',
        'quads', 'hamstrings', 'glutes', 'calves', 'abs', 'other')),
    ADD CONSTRAINT exercises_secondary_muscles_check CHECK (secondary_muscles <@ ARRAY[
        'chest', 'lats', 'upper_back', 'shoulders', 'biceps', 'triceps', 'forearms',
        'quads', 'hamstrings', 'glutes', 'calves', 'abs', 'other']::TEXT[]),
    ADD CONSTRAINT exercises_mechanic_check CHECK (mechanic IN (
        'compound', 'isolation')),
    ADD CONSTRAINT exercises_equipment_check CHECK (equipment IN (
        'barbell', 'dumbbell', 'machine', 'cable', 'bodyweight', 'kettlebell', 'band', 'other')),
    ADD CONSTRAINT exercises_tracking_type_check CHECK (tracking_type IN (
        'weight_reps', 'bodyweight_reps', 'duration', 'distance'));

-- Names are unique per owner, case-insensitively, among non-archived
-- exercises (so an archived "Bench" doesn't block a new one). System and
-- custom names are separate namespaces: a user may have their own "Bench
-- press" next to the default.
DROP INDEX idx_exercises_user_name;
CREATE UNIQUE INDEX idx_exercises_custom_name ON exercises (user_id, lower(name))
    WHERE user_id IS NOT NULL AND archived_at IS NULL;
CREATE UNIQUE INDEX idx_exercises_system_name ON exercises (lower(name))
    WHERE user_id IS NULL AND archived_at IS NULL;
CREATE UNIQUE INDEX idx_exercises_slug ON exercises (slug);

-- Sets: what is measured depends on the exercise's tracking type, so every
-- measurement is nullable; the app validates the combination
-- (models.WorkoutSet.ValidateFor). Existing sets keep reps and weight.
ALTER TABLE workout_sets
    ALTER COLUMN reps DROP NOT NULL,
    ALTER COLUMN weight_kg DROP NOT NULL,
    ALTER COLUMN weight_kg DROP DEFAULT,
    ADD COLUMN duration_seconds INTEGER CHECK (duration_seconds > 0),
    ADD COLUMN distance_m NUMERIC(8, 2) CHECK (distance_m > 0),
    ADD CONSTRAINT workout_sets_measured_check
        CHECK (reps IS NOT NULL OR duration_seconds IS NOT NULL OR distance_m IS NOT NULL);
