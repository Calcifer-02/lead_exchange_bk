-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS leads
(
    lead_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    description     TEXT,
    requirement     JSONB       NOT NULL,
    contact_name    TEXT        NOT NULL,
    contact_phone   TEXT        NOT NULL,
    contact_email   TEXT,
    status          TEXT        NOT NULL,
    owner_user_id   UUID        NOT NULL,
    created_user_id UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE leads;

-- +goose StatementEnd
