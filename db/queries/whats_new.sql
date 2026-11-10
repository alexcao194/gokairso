-- name: ListWhatsNewReleases :many
SELECT id, version, published_at
FROM whats_new_releases
ORDER BY published_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: ListWhatsNewReleaseNotesByReleaseIDs :many
SELECT id, release_id, type, content, position
FROM whats_new_release_notes
WHERE release_id = ANY(sqlc.arg(release_ids)::uuid[])
ORDER BY release_id, position ASC;
