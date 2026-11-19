-- name: CreateWorkout :one
INSERT INTO workouts (user_id, name, notes, performed_at, template_id)
VALUES (@user_id, @name, @notes, @performed_at, @template_id)
RETURNING *;

-- name: GetWorkout :one
SELECT * FROM workouts WHERE id = @id;

-- name: ListWorkoutsByUser :many
-- Newest first, each with a summary for the history list: how many sets
-- were logged and which exercises, in the order first logged.
SELECT sqlc.embed(w),
    (SELECT count(*) FROM workout_sets s WHERE s.workout_id = w.id)::int AS set_count,
    coalesce((
        SELECT array_agg(x.name ORDER BY x.first_at, x.name)
        FROM (
            SELECT e.name, min(s.created_at) AS first_at
            FROM workout_sets s JOIN exercises e ON e.id = s.exercise_id
            WHERE s.workout_id = w.id
            GROUP BY e.id, e.name
        ) x
    ), '{}')::text[] AS exercise_names
FROM workouts w
WHERE w.user_id = @user_id
ORDER BY w.performed_at DESC;

-- name: UpdateWorkout :one
UPDATE workouts SET name = @name, notes = @notes, performed_at = @performed_at,
    finished_at = @finished_at, updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING updated_at;

-- name: DeleteWorkout :execrows
DELETE FROM workouts WHERE id = @id;
