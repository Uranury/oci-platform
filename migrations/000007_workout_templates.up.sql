-- Workout templates: a named, reusable, ordered plan of exercises with
-- optional targets. Starting a workout from one only links it
-- (workouts.template_id); no sets are pre-created, so skipped sets never
-- turn into fake history.

CREATE TABLE workout_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 100),
    notes      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_workout_templates_user_name ON workout_templates (user_id, lower(name));

-- Which targets apply depends on the exercise's tracking type; the app
-- validates the combination (models.TemplateExercise.ValidateFor).
CREATE TABLE template_exercises (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id             UUID NOT NULL REFERENCES workout_templates (id) ON DELETE CASCADE,
    -- RESTRICT: deleting an exercise used by a template archives it instead
    -- (ExerciseService.Delete), the same as for logged sets.
    exercise_id             UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    position                INTEGER NOT NULL CHECK (position >= 0),
    target_sets             INTEGER NOT NULL CHECK (target_sets > 0),
    target_reps_min         INTEGER CHECK (target_reps_min > 0),
    target_reps_max         INTEGER CHECK (target_reps_max > 0),
    target_weight_kg        NUMERIC(6, 2) CHECK (target_weight_kg >= 0),
    target_duration_seconds INTEGER CHECK (target_duration_seconds > 0),
    target_distance_m       NUMERIC(8, 2) CHECK (target_distance_m > 0),
    notes                   TEXT,
    UNIQUE (template_id, position),
    CHECK (target_reps_min <= target_reps_max)
);

CREATE INDEX idx_template_exercises_exercise ON template_exercises (exercise_id);

-- Where a workout came from. SET NULL: deleting a template keeps history.
ALTER TABLE workouts
    ADD COLUMN template_id UUID REFERENCES workout_templates (id) ON DELETE SET NULL;

CREATE INDEX idx_workouts_template ON workouts (template_id, performed_at DESC)
    WHERE template_id IS NOT NULL;
