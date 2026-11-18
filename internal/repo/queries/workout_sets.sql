-- name: CreateWorkoutSet :one
WITH inserted AS (
    INSERT INTO workout_sets (workout_id, exercise_id, reps, weight_kg, duration_seconds, distance_m)
    VALUES (@workout_id, @exercise_id, @reps, @weight_kg, @duration_seconds, @distance_m)
    RETURNING *
)
SELECT i.id, i.workout_id, i.exercise_id, i.reps, i.weight_kg, i.duration_seconds,
    i.distance_m, i.created_at,
    e.name AS exercise_name, e.tracking_type AS exercise_tracking_type
FROM inserted i JOIN exercises e ON e.id = i.exercise_id;

-- The join resolves archived exercises too, so history keeps its names.
-- name: GetWorkoutSet :one
SELECT sqlc.embed(s), e.name AS exercise_name, e.tracking_type AS exercise_tracking_type
FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id
WHERE s.id = @id;

-- name: ListWorkoutSets :many
SELECT sqlc.embed(s), e.name AS exercise_name, e.tracking_type AS exercise_tracking_type
FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id
WHERE s.workout_id = @workout_id
ORDER BY s.created_at;

-- Scoped to the workout so a set ID from another workout never matches.
-- name: UpdateWorkoutSet :one
WITH updated AS (
    UPDATE workout_sets ws SET exercise_id = @exercise_id, reps = @reps, weight_kg = @weight_kg,
        duration_seconds = @duration_seconds, distance_m = @distance_m
    WHERE ws.id = @id AND ws.workout_id = @workout_id
    RETURNING ws.*
)
SELECT u.id, u.workout_id, u.exercise_id, u.reps, u.weight_kg, u.duration_seconds,
    u.distance_m, u.created_at,
    e.name AS exercise_name, e.tracking_type AS exercise_tracking_type
FROM updated u JOIN exercises e ON e.id = u.exercise_id;

-- name: DeleteWorkoutSet :execrows
DELETE FROM workout_sets WHERE id = @id;

-- For each exercise, the sets from the user's most recent workout before
-- @before that included it ("last time"), excluding @exclude_workout_id.
-- Archived exercises are kept so their history still reads right.
-- name: ListPreviousSets :many
WITH latest AS (
    SELECT DISTINCT ON (s.exercise_id) s.exercise_id, w.id AS workout_id
    FROM workout_sets s JOIN workouts w ON w.id = s.workout_id
    WHERE w.user_id = @user_id AND w.id <> @exclude_workout_id AND w.performed_at < @before
    ORDER BY s.exercise_id, w.performed_at DESC, w.id
)
SELECT sqlc.embed(s), e.name AS exercise_name, e.tracking_type AS exercise_tracking_type,
    w.performed_at AS workout_performed_at
FROM latest l
JOIN workout_sets s ON s.exercise_id = l.exercise_id AND s.workout_id = l.workout_id
JOIN exercises e ON e.id = s.exercise_id
JOIN workouts w ON w.id = s.workout_id
ORDER BY s.exercise_id, s.created_at;
