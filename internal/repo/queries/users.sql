-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: CreateUser :exec
INSERT INTO users (id, username, email, password) VALUES (@id, @username, @email, @password);
