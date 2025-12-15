-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS properties
(
    property_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    description     TEXT,
    address         TEXT        NOT NULL,
    property_type   TEXT        NOT NULL,
    area            NUMERIC(10, 2),
    price           INTEGER,
    rooms           INTEGER,
    status          TEXT        NOT NULL,
    owner_user_id   UUID        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_user_id UUID        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS properties;

-- +goose StatementEnd

