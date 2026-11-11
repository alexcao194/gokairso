-- name: UpsertDeviceToken :exec
INSERT INTO device_tokens (user_id, token, platform, locale)
VALUES ($1, $2, $3, $4)
ON CONFLICT (token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    locale = EXCLUDED.locale,
    updated_at = now();

-- name: DeleteDeviceToken :exec
DELETE FROM device_tokens WHERE token = $1;

-- name: DeleteDeviceTokens :exec
DELETE FROM device_tokens WHERE token = ANY($1::text[]);

-- name: ListDeviceTokensByUser :many
SELECT token, platform, locale
FROM device_tokens
WHERE user_id = $1;
