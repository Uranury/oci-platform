-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: CreateUser :exec
INSERT INTO users (id, username, email, password) VALUES (@id, @username, @email, @password);

-- The account deletion, in this order inside one transaction: sets and
-- template rows point at exercises with ON DELETE RESTRICT, which a single
-- cascading DELETE FROM users would trip over.

-- name: DeleteUserWorkouts :exec
DELETE FROM workouts WHERE user_id = @user_id;

-- name: DeleteUserTemplates :exec
DELETE FROM workout_templates WHERE user_id = @user_id;

-- name: DeleteUser :execrows
-- Cascades to custom exercises, refresh tokens and bodyweights.
DELETE FROM users WHERE id = @id;
