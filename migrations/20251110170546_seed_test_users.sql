-- +goose Up
-- +goose StatementBegin

-- Тестовые данные
INSERT INTO users (user_id, email, password_hash, first_name, last_name, phone, agency_name, avatar_url, role, status)
VALUES ('8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'user@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q', -- пароль: password
        'Поль', 'Зователёв',
        '+79991112233',
        'Best Realty',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'USER_ROLE_USER',
        'USER_STATUS_ACTIVE'),
       ('aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'agent@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q', -- пароль: password
        'Агент', 'Недвижимов',
        '+79994445566',
        'Prime Estate',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'USER_ROLE_AGENT',
        'USER_STATUS_ACTIVE'),
       ('f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'admin@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q', -- пароль: password
        'Админ', 'Администратов',
        '+79992223344',
        'Admin Corp',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'USER_ROLE_ADMIN',
        'USER_STATUS_ACTIVE');

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
