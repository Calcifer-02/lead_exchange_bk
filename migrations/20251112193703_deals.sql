-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS deals
(
    deal_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id        UUID        NOT NULL REFERENCES leads(lead_id) ON DELETE CASCADE,
    seller_user_id UUID        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    buyer_user_id  UUID        REFERENCES users(user_id) ON DELETE SET NULL,
    price          INTEGER     NOT NULL,
    status         TEXT        NOT NULL DEFAULT 'PENDING',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS deals;

-- +goose StatementEnd


