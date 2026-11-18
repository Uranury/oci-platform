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

-- name: DeleteWorkoutSet :execrows
DELETE FROM workout_sets WHERE id = @id;
