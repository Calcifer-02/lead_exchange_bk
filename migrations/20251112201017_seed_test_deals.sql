-- +goose Up
-- +goose StatementBegin

INSERT INTO deals (deal_id, lead_id, seller_user_id, buyer_user_id, price, status)
VALUES
    -- PENDING deals (ожидают покупателя)
    (
        'd1a2b3c4-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
        'b5d7a10e-418d-42a3-bb32-87e90d4a7a24', -- Дом у моря (PUBLISHED)
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12', -- user2@m.c (seller)
        NULL, -- пока нет покупателя
        2500000,
        'PENDING'
    ),
    (
        'd2b3c4d5-6e7f-8a9b-0c1d-2e3f4a5b6c7e',
        'e1b88dcf-1225-4d0d-827f-4ea8fdf99664', -- Участок под застройку (PUBLISHED)
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', -- admin@m.c (seller)
        NULL, -- пока нет покупателя
        600000,
        'PENDING'
    ),
    -- ACCEPTED deals (приняты покупателем)
    (
        'd3c4d5e6-7f8a-9b0c-1d2e-3f4a5b6c7d8f',
        'a8b55f9d-32c2-4e1f-97c7-341f49b7c012', -- 3-комнатная квартира в центре (NEW)
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', -- user@m.c (seller)
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12', -- user2@m.c (buyer)
        800000,
        'ACCEPTED'
    ),
    -- COMPLETED deals (завершены, лид передан)
    (
        'd4d5e6f7-8a9b-0c1d-2e3f-4a5b6c7d8e9a',
        'c7d9e1ff-8a9e-4a4e-9b5c-b47c3fddf311', -- Квартира для инвестиций (PURCHASED)
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', -- user@m.c (seller)
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12', -- user2@m.c (buyer)
        420000,
        'COMPLETED'
    ),
    -- CANCELLED deals (отменены продавцом)
    (
        'd5e6f7a8-9b0c-1d2e-3f4a-5b6c7d8e9f0b',
        'b5d7a10e-418d-42a3-bb32-87e90d4a7a24', -- Дом у моря (PUBLISHED)
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12', -- user2@m.c (seller)
        NULL,
        3000000,
        'CANCELLED'
    ),
    -- REJECTED deals (отклонены покупателем)
    (
        'd6f7a8b9-0c1d-2e3f-4a5b-6c7d8e9f0a1c',
        'e1b88dcf-1225-4d0d-827f-4ea8fdf99664', -- Участок под застройку (PUBLISHED)
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', -- admin@m.c (seller)
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', -- user@m.c (buyer, но отклонил)
        700000,
        'REJECTED'
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE
FROM deals
WHERE deal_id IN
      ('d1a2b3c4-5e6f-7a8b-9c0d-1e2f3a4b5c6d',
       'd2b3c4d5-6e7f-8a9b-0c1d-2e3f4a5b6c7e',
       'd3c4d5e6-7f8a-9b0c-1d2e-3f4a5b6c7d8f',
       'd4d5e6f7-8a9b-0c1d-2e3f-4a5b6c7d8e9a',
       'd5e6f7a8-9b0c-1d2e-3f4a-5b6c7d8e9f0b',
       'd6f7a8b9-0c1d-2e3f-4a5b-6c7d8e9f0a1c');

-- +goose StatementEnd

