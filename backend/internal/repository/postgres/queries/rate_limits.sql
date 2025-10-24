-- name: GetRateLimit :one
SELECT * FROM rate_limits
WHERE key = $1;

-- name: UpsertRateLimit :one
INSERT INTO rate_limits (key, attempts, reset_at)
VALUES ($1, 1, $2)
ON CONFLICT (key) DO UPDATE
SET attempts = rate_limits.attempts + 1,
    reset_at = CASE
        WHEN rate_limits.reset_at < NOW() THEN $2
        ELSE rate_limits.reset_at
    END
RETURNING *;

-- name: ResetRateLimit :exec
DELETE FROM rate_limits
WHERE key = $1;

-- name: CleanupExpiredRateLimits :exec
DELETE FROM rate_limits
WHERE reset_at < NOW();
