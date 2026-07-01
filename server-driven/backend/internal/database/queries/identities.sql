-- name: GetIdentityByProviderID :one
SELECT * FROM identities WHERE provider = $1 AND provider_id = $2;

-- name: GetIdentitiesByUserID :many
SELECT * FROM identities WHERE user_id = $1;

-- name: CreateIdentity :one
INSERT INTO identities (user_id, provider, provider_id, email, name)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
