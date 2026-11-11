-- +goose Up
ALTER TABLE users ADD COLUMN locale character varying(8) DEFAULT 'vi' NOT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN locale;
