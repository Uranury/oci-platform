-- name: UpsertBodyweight :one
INSERT INTO bodyweights (user_id, day, weight_kg)
VALUES (@user_id, @day, @weight_kg)
ON CONFLICT (user_id, day) DO UPDATE SET weight_kg = excluded.weight_kg, updated_at = now()
RETURNING *;

-- name: ListBodyweights :many
-- Newest day first.
SELECT * FROM bodyweights
WHERE user_id = @user_id
ORDER BY day DESC
LIMIT @max_rows;

-- name: DeleteBodyweight :execrows
DELETE FROM bodyweights WHERE user_id = @user_id AND day = @day;
