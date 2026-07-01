-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;

-- name: CreateUser :one
INSERT INTO users (email, phone, name, avatar_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
    email = COALESCE($2, email),
    phone = COALESCE($3, phone),
    name = COALESCE($4, name),
    avatar_url = COALESCE($5, avatar_url),
    updated_at = now()
WHERE id = $1
RETURNING *;
