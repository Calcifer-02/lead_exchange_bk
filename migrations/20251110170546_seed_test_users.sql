-- +goose Up
-- +goose StatementBegin

-- Тестовые данные
INSERT INTO users (user_id, email, password_hash, first_name, last_name, phone, agency_name, avatar_url, role)
VALUES ('8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'user@m.c',
        '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', -- пароль: password
        'Поль', 'Зователёв',
        '+79991112233',
        'Best Realty',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'USER'),
       ('aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'user2@m.c',
        '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', -- пароль: password
        'ПольДва', 'ЗователёвДва',
        '+79991112244',
        'Worst Realty',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'USER'),
       ('f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'admin@m.c',
        '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', -- пароль: password
        'Админ','Нистраторов',
        '+79992223344',
        'Lead Exchange HQ',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'ADMIN');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE
FROM users
WHERE user_id IN
      ('8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
       'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
       'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242');

-- +goose StatementEnd
