-- +goose Up
ALTER TABLE runs ADD COLUMN gotap_metadata TEXT;

-- +goose Down
ALTER TABLE runs DROP COLUMN gotap_metadata;
