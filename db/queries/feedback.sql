-- name: CreateCustomerFeedback :one
INSERT INTO customer_feedbacks (workspace_id, creator_id, feedback_text, content, estimate)
VALUES (sqlc.arg(workspace_id), sqlc.arg(creator_id), sqlc.arg(feedback_text), sqlc.arg(content), sqlc.arg(estimate))
RETURNING id, created_at, updated_at, workspace_id, creator_id, feedback_text, content, estimate;

-- name: ListCustomerFeedbacks :many
SELECT
    cf.id, cf.created_at, cf.updated_at, cf.workspace_id,
    cf.creator_id, u.alias AS creator_alias, u.name AS creator_name, u.avatar AS creator_avatar,
    cf.feedback_text, cf.content, cf.estimate
FROM customer_feedbacks cf
JOIN users u ON u.id = cf.creator_id
WHERE cf.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR cf.created_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR cf.created_at <= sqlc.narg(to_time)::timestamptz)
  AND (
      sqlc.narg(member_id)::uuid IS NULL OR EXISTS (
          SELECT 1 FROM customer_feedback_beneficiaries cfb 
          WHERE cfb.feedback_id = cf.id AND cfb.user_id = sqlc.narg(member_id)::uuid
      )
  )
ORDER BY cf.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetCustomerFeedback :one
SELECT
    cf.id, cf.created_at, cf.updated_at, cf.workspace_id,
    cf.creator_id, u.alias AS creator_alias, u.name AS creator_name, u.avatar AS creator_avatar,
    cf.feedback_text, cf.content, cf.estimate
FROM customer_feedbacks cf
JOIN users u ON u.id = cf.creator_id
WHERE cf.id = $1;

-- name: UpdateCustomerFeedback :one
UPDATE customer_feedbacks
SET
    updated_at    = now(),
    feedback_text = CASE WHEN sqlc.arg(feedback_text_present)::boolean THEN sqlc.arg(feedback_text)::text  ELSE feedback_text END,
    content       = CASE WHEN sqlc.arg(content_present)::boolean THEN sqlc.arg(content)::text  ELSE content       END,
    estimate      = CASE WHEN sqlc.arg(estimate_present)::boolean THEN sqlc.arg(estimate)::float ELSE estimate      END
WHERE id = $1
RETURNING id, created_at, updated_at, workspace_id, creator_id, feedback_text, content, estimate;

-- name: DeleteCustomerFeedback :exec
DELETE FROM customer_feedbacks WHERE id = $1;

-- name: UpsertFeedbackBeneficiary :one
INSERT INTO customer_feedback_beneficiaries (feedback_id, user_id, weight)
VALUES (sqlc.arg(feedback_id), sqlc.arg(user_id), sqlc.arg(weight))
ON CONFLICT (feedback_id, user_id) DO UPDATE SET weight = EXCLUDED.weight
RETURNING id, feedback_id, user_id, weight;

-- name: ListFeedbackBeneficiaries :many
SELECT cfb.id, cfb.feedback_id, cfb.user_id, cfb.weight,
       u.alias AS user_alias, u.name AS user_name, u.avatar AS user_avatar
FROM customer_feedback_beneficiaries cfb
JOIN users u ON u.id = cfb.user_id
WHERE cfb.feedback_id = $1;

-- name: DeleteFeedbackBeneficiaries :exec
DELETE FROM customer_feedback_beneficiaries WHERE feedback_id = $1;

-- name: SumFeedbackEstimateByBeneficiary :many
SELECT
    cfb.user_id,
    COALESCE(SUM(cf.estimate * cfb.weight / totals.total_weight), 0)::float AS total
FROM customer_feedbacks cf
JOIN customer_feedback_beneficiaries cfb ON cfb.feedback_id = cf.id
JOIN (
    SELECT feedback_id, SUM(weight) AS total_weight
    FROM customer_feedback_beneficiaries
    GROUP BY feedback_id
) totals ON totals.feedback_id = cf.id
WHERE cf.workspace_id = sqlc.arg(workspace_id)::uuid
  AND cf.created_at >= sqlc.arg('from')
  AND cf.created_at <= sqlc.arg('to')
  AND (sqlc.narg(member_id)::uuid IS NULL OR cfb.user_id = sqlc.narg(member_id)::uuid)
GROUP BY cfb.user_id;
