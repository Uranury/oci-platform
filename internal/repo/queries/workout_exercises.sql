-- name: ListWorkoutExercises :many
-- The workout's exercises in display order, with the exercise's name,
-- tracking type and whether it has been archived since.
SELECT sqlc.embed(we), e.name AS exercise_name, e.tracking_type AS exercise_tracking_type,
    (e.archived_at IS NOT NULL)::bool AS exercise_archived
FROM workout_exercises we
JOIN exercises e ON e.id = we.exercise_id
WHERE we.workout_id = @workout_id
ORDER BY we.position, we.id;

-- name: AppendWorkoutExercise :execrows
-- Adds the exercise at the end of the workout, with its plan when it has
-- one (restoring a removed planned exercise); no-op if it's already there.
INSERT INTO workout_exercises (workout_id, exercise_id, position, target_sets, target_reps_min,
    target_reps_max, target_duration_seconds, target_distance_m, notes)
SELECT @workout_id, @exercise_id, coalesce(max(position) + 1, 0),
    sqlc.narg(target_sets)::int, sqlc.narg(target_reps_min)::int, sqlc.narg(target_reps_max)::int,
    sqlc.narg(target_duration_seconds)::int, sqlc.narg(target_distance_m)::numeric, sqlc.narg(notes)::text
FROM workout_exercises WHERE workout_id = @workout_id
ON CONFLICT (workout_id, exercise_id) DO NOTHING;

-- name: CopyTemplateIntoWorkout :exec
-- The plan, frozen into a workout started from the template.
INSERT INTO workout_exercises (workout_id, exercise_id, position, target_sets, target_reps_min,
    target_reps_max, target_duration_seconds, target_distance_m, notes)
SELECT @workout_id, te.exercise_id, te.position, te.target_sets, te.target_reps_min,
    te.target_reps_max, te.target_duration_seconds, te.target_distance_m, te.notes
FROM template_exercises te
WHERE te.template_id = @template_id
ON CONFLICT (workout_id, exercise_id) DO NOTHING;

-- name: SetWorkoutExercisePosition :execrows
UPDATE workout_exercises SET position = @position
WHERE workout_id = @workout_id AND exercise_id = @exercise_id;

-- name: DeleteWorkoutExercise :execrows
DELETE FROM workout_exercises WHERE workout_id = @workout_id AND exercise_id = @exercise_id;

-- name: DeleteWorkoutExerciseSets :execrows
DELETE FROM workout_sets WHERE workout_id = @workout_id AND exercise_id = @exercise_id;
