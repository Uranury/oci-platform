-- When a workout was finished, for session duration. NULL = unknown: not
-- finished yet, or logged after the fact (see WorkoutService.FinishWorkout).
ALTER TABLE workouts
    ADD COLUMN finished_at TIMESTAMPTZ,
    ADD CONSTRAINT workouts_finished_after_start
        CHECK (finished_at IS NULL OR finished_at >= performed_at);
