-- name: ListWebhooks :many
SELECT id, workspace_id, name, url, token, enabled, failure_count, last_status, last_error, last_delivered_at,
       created_by, created_at, updated_at
FROM webhooks
WHERE workspace_id = $1
ORDER BY created_at;

-- name: ListEnabledWebhooks :many
SELECT id, url, token
FROM webhooks
WHERE workspace_id = $1 AND enabled;

-- name: CountWebhooks :one
SELECT count(*) FROM webhooks WHERE workspace_id = $1;

-- name: GetWebhook :one
SELECT id, workspace_id, name, url, token, enabled, failure_count, last_status, last_error, last_delivered_at,
       created_by, created_at, updated_at
FROM webhooks
WHERE id = $1 AND workspace_id = $2;

-- name: CreateWebhook :one
INSERT INTO webhooks (workspace_id, name, url, token, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, workspace_id, name, url, token, enabled, failure_count, last_status, last_error, last_delivered_at,
          created_by, created_at, updated_at;

-- name: UpdateWebhook :one
UPDATE webhooks
SET name = $3, url = $4, token = $5, enabled = $6,
    failure_count = CASE WHEN $6 THEN 0 ELSE failure_count END
WHERE id = $1 AND workspace_id = $2
RETURNING id, workspace_id, name, url, token, enabled, failure_count, last_status, last_error, last_delivered_at,
          created_by, created_at, updated_at;

-- name: DeleteWebhook :execrows
DELETE FROM webhooks WHERE id = $1 AND workspace_id = $2;

-- name: RecordWebhookSuccess :exec
UPDATE webhooks
SET failure_count = 0, last_status = $2, last_error = '', last_delivered_at = now()
WHERE id = $1;

-- name: RecordWebhookFailure :one
UPDATE webhooks
SET failure_count = failure_count + 1, last_status = $2, last_error = $3, last_delivered_at = now(),
    enabled = enabled AND failure_count + 1 < sqlc.arg(disable_after)::int
WHERE id = $1
RETURNING enabled;

-- name: InsertWebhookDelivery :exec
INSERT INTO webhook_deliveries (webhook_id, event_id, event_type, success, status_code, error, attempts, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: PruneWebhookDeliveries :exec
DELETE FROM webhook_deliveries
WHERE webhook_id = $1 AND created_at < now() - interval '7 days';

-- name: ListWebhookDeliveries :many
SELECT id, event_id, event_type, success, status_code, error, attempts, duration_ms, created_at
FROM webhook_deliveries
WHERE webhook_id = $1
ORDER BY created_at DESC
LIMIT 50;
