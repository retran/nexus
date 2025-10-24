-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    user_id,
    event_type,
    ip_address,
    user_agent,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type')::audit_event_type)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAuditLogByID :one
SELECT * FROM audit_logs
WHERE id = $1;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type')::audit_event_type);
