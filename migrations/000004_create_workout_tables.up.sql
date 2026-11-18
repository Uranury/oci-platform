CREATE TABLE exercises (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One "Bench Press" per user, case-insensitively.
CREATE UNIQUE INDEX idx_exercises_user_name ON exercises (user_id, lower(name));

CREATE TABLE workouts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT,
    notes        TEXT,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workouts_user_performed ON workouts (user_id, performed_at DESC);

CREATE TABLE workout_sets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id  UUID NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    -- RESTRICT: an exercise with logged sets can't be deleted out from under
    -- the history that references it.
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    reps        INTEGER NOT NULL CHECK (reps > 0),
    weight_kg   NUMERIC(6, 2) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workout_sets_workout ON workout_sets (workout_id, created_at);
CREATE INDEX idx_workout_sets_exercise ON workout_sets (exercise_id);
