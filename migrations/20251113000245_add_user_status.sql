-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
ADD COLUMN status TEXT NOT NULL DEFAULT 'ACTIVE';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
DROP COLUMN IF EXISTS status;

-- +goose StatementEnd

