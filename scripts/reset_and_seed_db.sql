-- Скрипт очистки БД и применения миграций для Render
-- Выполнить через psql или Render Shell
--
-- Подключение:
-- psql "postgresql://lead_exchange_bk_user:8m2gtTRBW0iAr7nY2Aadzz0VcZBEVKYM@dpg-d5ht8vi4d50c739akh2g-a.oregon-postgres.render.com/lead_exchange_bk"

-- =============================================
-- ЧАСТЬ 1: ПОЛНАЯ ОЧИСТКА БД
-- =============================================

-- Удаляем все таблицы
DROP TABLE IF EXISTS deals CASCADE;
DROP TABLE IF EXISTS properties CASCADE;
DROP TABLE IF EXISTS leads CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS goose_db_version CASCADE;

-- Удаляем расширения (пересоздадим)
DROP EXTENSION IF EXISTS vector CASCADE;

-- =============================================
-- ЧАСТЬ 2: СОЗДАНИЕ РАСШИРЕНИЙ
-- =============================================

CREATE EXTENSION IF NOT EXISTS vector;

-- =============================================
-- ЧАСТЬ 3: СОЗДАНИЕ ТАБЛИЦ (из миграций)
-- =============================================

-- 20251110162249_users.sql
CREATE TABLE IF NOT EXISTS users
(
    user_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT UNIQUE NOT NULL,
    password   TEXT        NOT NULL,
    first_name TEXT        NOT NULL,
    last_name  TEXT        NOT NULL,
    phone      TEXT,
    avatar_url TEXT,
    agency_name TEXT,
    role       TEXT        NOT NULL DEFAULT 'USER_ROLE_USER',
    status     TEXT        NOT NULL DEFAULT 'USER_STATUS_PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 20251112114136_leads.sql
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

-- 20251112193703_deals.sql
CREATE TABLE IF NOT EXISTS deals
(
    deal_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id        UUID        NOT NULL,
    property_id    UUID,
    seller_user_id UUID        NOT NULL,
    buyer_user_id  UUID        NOT NULL,
    status         TEXT        NOT NULL,
    amount         BIGINT,
    commission     BIGINT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 20251214194937_properties.sql
CREATE TABLE IF NOT EXISTS properties
(
    property_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    description     TEXT,
    address         TEXT        NOT NULL,
    property_type   TEXT        NOT NULL,
    area            DOUBLE PRECISION,
    price           BIGINT,
    rooms           INT,
    status          TEXT        NOT NULL,
    owner_user_id   UUID        NOT NULL,
    created_user_id UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 20251216120000_add_city_field.sql
ALTER TABLE leads ADD COLUMN IF NOT EXISTS city TEXT;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS property_type TEXT DEFAULT 'PROPERTY_TYPE_UNSPECIFIED';
ALTER TABLE properties ADD COLUMN IF NOT EXISTS city TEXT;

-- 20251214222218_add_embeddings.sql + 20260111000000_update_embedding_dimensions.sql
-- ВАЖНО: Используем размерность 1024 для ai-forever/ru-en-RoSBERTa
ALTER TABLE leads ADD COLUMN IF NOT EXISTS embedding vector(1024);
ALTER TABLE properties ADD COLUMN IF NOT EXISTS embedding vector(1024);

-- Индексы для эмбеддингов
CREATE INDEX IF NOT EXISTS leads_embedding_idx ON leads USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS properties_embedding_idx ON properties USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- 20251217120000_add_fulltext_search.sql
ALTER TABLE properties ADD COLUMN IF NOT EXISTS search_vector tsvector;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Функции для автоматического обновления search_vector
CREATE OR REPLACE FUNCTION properties_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('russian', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('russian', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('russian', COALESCE(NEW.address, '')), 'C') ||
        setweight(to_tsvector('russian', COALESCE(NEW.city, '')), 'A');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION leads_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('russian', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('russian', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггеры
DROP TRIGGER IF EXISTS properties_search_vector_trigger ON properties;
CREATE TRIGGER properties_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description, address, city ON properties
    FOR EACH ROW
    EXECUTE FUNCTION properties_search_vector_update();

DROP TRIGGER IF EXISTS leads_search_vector_trigger ON leads;
CREATE TRIGGER leads_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description ON leads
    FOR EACH ROW
    EXECUTE FUNCTION leads_search_vector_update();

-- GIN индексы для полнотекстового поиска
CREATE INDEX IF NOT EXISTS idx_properties_search_vector ON properties USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_leads_search_vector ON leads USING GIN(search_vector);

-- =============================================
-- ЧАСТЬ 4: ТЕСТОВЫЕ ДАННЫЕ
-- =============================================

-- 20251110170546_seed_test_users.sql
-- Пароль для всех: password (bcrypt hash)
INSERT INTO users (user_id, email, password, first_name, last_name, phone, avatar_url, agency_name, role, status)
VALUES
    (
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'user@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q',
        'Поль',
        'Зователёв',
        '+79991112233',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'Best Realty',
        'USER_ROLE_USER',
        'USER_STATUS_ACTIVE'
    ),
    (
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'agent@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q',
        'Агент',
        'Недвижимов',
        '+79994445566',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'Prime Estate',
        'USER_ROLE_AGENT',
        'USER_STATUS_ACTIVE'
    ),
    (
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'admin@m.c',
        '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqB7xXN2dPFHzPVEoF2zQ5uXZ5m.q',
        'Админ',
        'Администратов',
        '+79992223344',
        'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png',
        'Admin Corp',
        'USER_ROLE_ADMIN',
        'USER_STATUS_ACTIVE'
    );

-- 20251112114545_seed_test_leads.sql
INSERT INTO leads (lead_id, title, description, requirement, contact_name, contact_phone, contact_email, status, owner_user_id, created_user_id, city)
VALUES
    (
        'a8b55f9d-32c2-4e1f-97c7-341f49b7c012',
        '3-комнатная квартира в центре',
        'Просторная квартира рядом с метро и парком',
        '{"roomNumber": 3, "preferredPrice": "8000000", "district": "Центральный"}',
        'Иван Петров',
        '+79991112233',
        'ivan.petrov@example.com',
        'PUBLISHED',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'Санкт-Петербург'
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
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'Санкт-Петербург'
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
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'Москва'
    ),
    (
        'e1b88dcf-1225-4d0d-827f-4ea8fdf99664',
        '2-комнатная квартира у метро',
        'Ищу 2-комнатную квартиру в Санкт-Петербурге, рядом с метро, бюджет до 12 млн',
        '{"roomNumber": 2, "preferredPrice": "12000000", "metro": "близко"}',
        'Мария Белова',
        '+79998889900',
        'm.belova@example.com',
        'PUBLISHED',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'Санкт-Петербург'
    );

-- 20251214210945_seed_test_properties.sql
INSERT INTO properties (property_id, title, description, address, property_type, area, price, rooms, status, owner_user_id, created_user_id, city)
VALUES
    (
        'd1a2b3c4-1234-5678-9abc-def012345678',
        '2-комнатная квартира на Невском',
        'Светлая квартира с видом на Невский проспект, евроремонт, рядом метро',
        'Санкт-Петербург, Невский проспект, 100',
        'APARTMENT',
        65.5,
        11500000,
        2,
        'PUBLISHED',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'Санкт-Петербург'
    ),
    (
        'd2b3c4d5-2345-6789-abcd-ef0123456789',
        '3-комнатная квартира в центре',
        'Просторная квартира в историческом центре, высокие потолки, парковка',
        'Санкт-Петербург, ул. Рубинштейна, 25',
        'APARTMENT',
        95.0,
        18000000,
        3,
        'PUBLISHED',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'Санкт-Петербург'
    ),
    (
        'd3c4d5e6-3456-789a-bcde-f01234567890',
        '2-комнатная квартира у метро Московская',
        'Уютная квартира, 5 минут до метро, тихий двор, балкон',
        'Санкт-Петербург, Московский проспект, 180',
        'APARTMENT',
        54.0,
        9800000,
        2,
        'PUBLISHED',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1',
        'Санкт-Петербург'
    ),
    (
        'd4d5e6f7-4567-89ab-cdef-012345678901',
        'Студия в новостройке',
        'Современная студия с отделкой, закрытый двор, подземный паркинг',
        'Санкт-Петербург, ул. Оптиков, 52',
        'APARTMENT',
        32.0,
        5500000,
        1,
        'PUBLISHED',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'aea6842b-c540-4aa8-aa1f-90b1b46aba12',
        'Санкт-Петербург'
    ),
    (
        'd5e6f7a8-5678-9abc-def0-123456789012',
        '1-комнатная квартира на Петроградке',
        'Квартира с ремонтом, рядом парк, школа, детский сад',
        'Санкт-Петербург, ул. Большая Пушкарская, 15',
        'APARTMENT',
        42.0,
        8200000,
        1,
        'PUBLISHED',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242',
        'Санкт-Петербург'
    );

-- =============================================
-- ЧАСТЬ 5: ПРОВЕРКА
-- =============================================

SELECT 'Users:' as table_name, count(*) as count FROM users
UNION ALL
SELECT 'Leads:', count(*) FROM leads
UNION ALL
SELECT 'Properties:', count(*) FROM properties
UNION ALL
SELECT 'Deals:', count(*) FROM deals;

-- Проверяем структуру эмбеддингов
SELECT table_name, column_name, udt_name
FROM information_schema.columns
WHERE column_name = 'embedding' AND table_schema = 'public';

