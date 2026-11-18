-- name: CreateWorkout :one
INSERT INTO workouts (user_id, name, notes, performed_at, template_id)
VALUES (@user_id, @name, @notes, @performed_at, @template_id)
RETURNING *;

-- name: GetWorkout :one
SELECT * FROM workouts WHERE id = @id;

-- name: ListWorkoutsByUser :many
SELECT * FROM workouts WHERE user_id = @user_id ORDER BY performed_at DESC;

-- name: UpdateWorkout :one
UPDATE workouts SET name = @name, notes = @notes, performed_at = @performed_at, updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING updated_at;

-- name: DeleteWorkout :execrows
DELETE FROM workouts WHERE id = @id;
