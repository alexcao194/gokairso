-- name: ListHolidays :many
SELECT id, name, name_en, type, date_type, day, month, year, is_public_holiday, description, created_at, updated_at
FROM holidays
WHERE (year IS NULL) OR (year >= sqlc.arg(start_year)::int AND year <= sqlc.arg(end_year)::int);
