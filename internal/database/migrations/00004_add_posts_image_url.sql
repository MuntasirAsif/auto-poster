-- +goose Up
ALTER TABLE posts ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE posts DROP COLUMN IF EXISTS image_url;