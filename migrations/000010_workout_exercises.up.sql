-- Each workout's own ordered exercise list, with the plan's targets copied
-- in when it starts from a template. The workout screen shows exactly this
-- list: exercises can be added before any set is logged, removed, and
-- reordered, and editing a template later no longer changes old workouts
-- (which plan-adherence stats need).
CREATE TABLE workout_exercises (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id              UUID NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    -- RESTRICT: deleting an exercise still in a workout archives it instead
    -- (ExerciseService.Delete), as with logged sets and templates.
    exercise_id             UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    -- Display order. Not unique: a reorder rewrites every position, and two
    -- concurrent appends landing on the same number only tie (broken by id).
    position                INTEGER NOT NULL CHECK (position >= 0),
    -- The plan, when it came from a template; all NULL for an exercise added
    -- during the workout.
    target_sets             INTEGER CHECK (target_sets > 0),
    target_reps_min         INTEGER CHECK (target_reps_min > 0),
    target_reps_max         INTEGER CHECK (target_reps_max > 0),
    target_duration_seconds INTEGER CHECK (target_duration_seconds > 0),
    target_distance_m       NUMERIC(8, 2) CHECK (target_distance_m > 0),
    notes                   TEXT,
    UNIQUE (workout_id, exercise_id),
    CHECK (target_reps_min <= target_reps_max)
);

CREATE INDEX idx_workout_exercises_workout ON workout_exercises (workout_id, position);
CREATE INDEX idx_workout_exercises_exercise ON workout_exercises (exercise_id);

-- Backfill existing workouts the way the workout screen used to derive
-- them: the template's plan (as it is now) in plan order, then anything
-- logged off-plan in the order first logged.
INSERT INTO workout_exercises (workout_id, exercise_id, position, target_sets, target_reps_min,
    target_reps_max, target_duration_seconds, target_distance_m, notes)
SELECT w.id, te.exercise_id, te.position, te.target_sets, te.target_reps_min,
    te.target_reps_max, te.target_duration_seconds, te.target_distance_m, te.notes
FROM workouts w
JOIN template_exercises te ON te.template_id = w.template_id
ON CONFLICT (workout_id, exercise_id) DO NOTHING;

INSERT INTO workout_exercises (workout_id, exercise_id, position)
SELECT x.workout_id, x.exercise_id,
    coalesce((SELECT max(we.position) + 1 FROM workout_exercises we WHERE we.workout_id = x.workout_id), 0)
        + row_number() OVER (PARTITION BY x.workout_id ORDER BY x.first_at, x.exercise_id) - 1
FROM (
    SELECT s.workout_id, s.exercise_id, min(s.created_at) AS first_at
    FROM workout_sets s
    GROUP BY s.workout_id, s.exercise_id
) x
WHERE NOT EXISTS (
    SELECT 1 FROM workout_exercises we
    WHERE we.workout_id = x.workout_id AND we.exercise_id = x.exercise_id
);
