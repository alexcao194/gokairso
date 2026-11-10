-- name: GetUserByID :one
SELECT id, created_at, updated_at, deleted_at, name, alias, avatar,
       firebase_uid, email, email_verified, firebase_provider, last_sign_in_at,
       birthday, bio, phone, nickname
FROM users
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByAlias :one
SELECT id, created_at, updated_at, deleted_at, name, alias, avatar,
       firebase_uid, email, email_verified, firebase_provider, last_sign_in_at,
       birthday, bio, phone, nickname
FROM users
WHERE alias = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateUserProfile :one
UPDATE users
SET
    name = COALESCE(sqlc.narg(name), name),
    alias = COALESCE(sqlc.narg(alias), alias),
    avatar = CASE
        WHEN sqlc.narg(avatar)::text IS NULL THEN avatar
        ELSE NULLIF(sqlc.narg(avatar), '')
    END,
    birthday = CASE
        WHEN sqlc.arg(birthday_present)::boolean THEN sqlc.narg(birthday)
        ELSE birthday
    END,
    bio = CASE
        WHEN sqlc.arg(bio_present)::boolean THEN sqlc.narg(bio)
        ELSE bio
    END,
    phone = CASE
        WHEN sqlc.arg(phone_present)::boolean THEN sqlc.narg(phone)
        ELSE phone
    END,
    nickname = CASE
        WHEN sqlc.arg(nickname_present)::boolean THEN sqlc.narg(nickname)
        ELSE nickname
    END
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, name, alias, avatar,
          firebase_uid, email, email_verified, firebase_provider, last_sign_in_at,
          birthday, bio, phone, nickname;

-- name: UpdateFirebaseUser :one
UPDATE users
SET
    name = CASE WHEN $2 <> '' THEN $2 ELSE name END,
    avatar = CASE
        WHEN avatar IS NULL THEN NULLIF($3, '')
        ELSE avatar
    END,
    email = NULLIF($4, ''),
    email_verified = $5,
    firebase_provider = $6,
    last_sign_in_at = NOW(),
    deleted_at = NULL
WHERE firebase_uid = $1
RETURNING id, created_at, updated_at, deleted_at, name, alias, avatar,
          firebase_uid, email, email_verified, firebase_provider, last_sign_in_at,
          birthday, bio, phone, nickname;

-- name: InsertFirebaseUser :one
INSERT INTO users (name, alias, avatar, firebase_uid, email, email_verified, firebase_provider, last_sign_in_at)
VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, $7, NOW())
RETURNING id, created_at, updated_at, deleted_at, name, alias, avatar,
          firebase_uid, email, email_verified, firebase_provider, last_sign_in_at,
          birthday, bio, phone, nickname;
