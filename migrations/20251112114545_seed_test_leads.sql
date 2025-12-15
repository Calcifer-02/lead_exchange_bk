-- +goose Up
-- +goose StatementBegin

INSERT INTO leads (lead_id, title, description, requirement, contact_name, contact_phone, contact_email, status, owner_user_id, created_user_id)
VALUES
    (
        'a8b55f9d-32c2-4e1f-97c7-341f49b7c012',
        '3-комнатная квартира в центре',
        'Просторная квартира рядом с метро и парком',
        '{"roomNumber": 3, "preferredPrice": "8000000", "district": "Центральный"}',
        'Иван Петров',
        '+79991112233',
        'ivan.petrov@example.com',
        'NEW',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1'
    ),
    (
        'b5d7a10e-418d-42a3-bb32-87e90d4a7a24',
        'Дом у моря',
        'Двухэтажный дом с видом на залив',
        '{"rooms": 5, "preferredPrice": "25000000", "region": "Приморский"}',
        'Ольга Сидорова',
        '+79995557788',
        'olga.sid@example.com',
        'PUBLISHED',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12'
    ),
    (
        'c7d9e1ff-8a9e-4a4e-9b5c-b47c3fddf311',
        'Квартира для инвестиций',
        'Новая квартира в развивающемся районе',
        '{"roomNumber": 1, "preferredPrice": "4200000", "yield": "7%"}',
        'Дмитрий Котов',
        '+79993334455',
        'd.kotov@example.com',
        'PURCHASED',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1'
    ),
    (
        'e1b88dcf-1225-4d0d-827f-4ea8fdf99664',
        'Участок под застройку',
        '15 соток в черте города, коммуникации рядом',
        '{"area": "15 соток", "preferredPrice": "6000000", "purpose": "строительство"}',
        'Мария Белова',
        '+79998889900',
        'm.belova@example.com',
        'PUBLISHED',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242'
    ),
    (
        'f5e7a5a1-2c13-4e3e-a6c5-57a1c82f7792',
        'Квартира с панорамным видом',
        'Вид на город, дизайнерский ремонт, премиум класс',
        '{"roomNumber": 4, "preferredPrice": "15000000", "view": "панорамный"}',
        'Пётр Иванов',
        '+79992223344',
        'p.ivanov@example.com',
        'DELETED',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242'
    );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE
FROM leads
WHERE lead_id IN
      ('a8b55f9d-32c2-4e1f-97c7-341f49b7c012',
       'b5d7a10e-418d-42a3-bb32-87e90d4a7a24',
       'c7d9e1ff-8a9e-4a4e-9b5c-b47c3fddf311',
       'e1b88dcf-1225-4d0d-827f-4ea8fdf99664',
       'f5e7a5a1-2c13-4e3e-a6c5-57a1c82f7792');

-- +goose StatementEnd
