-- +goose Up
ALTER TABLE posts ADD COLUMN IF NOT EXISTS short_description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE posts DROP COLUMN IF EXISTS short_description;