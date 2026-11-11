-- +goose Up
CREATE TABLE device_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token text NOT NULL,
    platform character varying(16) DEFAULT 'web' NOT NULL,
    locale character varying(8) DEFAULT 'vi' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT device_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT device_tokens_token_key UNIQUE (token),
    CONSTRAINT device_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_device_tokens_user_id ON device_tokens USING btree (user_id);

CREATE TRIGGER update_device_tokens_updated_at BEFORE UPDATE ON device_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TABLE device_tokens;
