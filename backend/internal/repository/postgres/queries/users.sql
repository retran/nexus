-- Copyright 2025 Andrew Vasilyev
-- SPDX-License-Identifier: APACHE-2.0

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByKratosID :one
SELECT * FROM users
WHERE kratos_identity_id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUsersByRole :many
SELECT * FROM users
WHERE role = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateUser :one
INSERT INTO users (
  kratos_identity_id,
  email,
  name,
  picture,
  role
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO users (
  kratos_identity_id,
  email,
  name,
  picture,
  role
) VALUES (
  $1, $2, $3, $4, $5
)
ON CONFLICT (kratos_identity_id) DO UPDATE
SET
  email = EXCLUDED.email,
  name = EXCLUDED.name,
  picture = EXCLUDED.picture,
  updated_at = NOW()
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
  name = COALESCE($2, name),
  picture = COALESCE($3, picture),
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserRole :one
UPDATE users
SET
  role = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountUsersByRole :one
SELECT COUNT(*) FROM users
WHERE role = $1;
