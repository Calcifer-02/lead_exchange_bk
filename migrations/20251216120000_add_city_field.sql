-- +goose Up
-- +goose StatementBegin

-- Добавляем поле city в таблицу leads для жёсткой фильтрации по городу
ALTER TABLE leads ADD COLUMN IF NOT EXISTS city TEXT;

-- Добавляем поле city в таблицу properties для жёсткой фильтрации по городу
ALTER TABLE properties ADD COLUMN IF NOT EXISTS city TEXT;

-- Создаём индексы для быстрой фильтрации по городу
CREATE INDEX IF NOT EXISTS leads_city_idx ON leads (city) WHERE city IS NOT NULL;
CREATE INDEX IF NOT EXISTS properties_city_idx ON properties (city) WHERE city IS NOT NULL;

-- Составной индекс для типичных фильтров матчинга
CREATE INDEX IF NOT EXISTS properties_matching_idx ON properties (city, property_type, status)
    WHERE status = 'PUBLISHED';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS properties_matching_idx;
DROP INDEX IF EXISTS properties_city_idx;
DROP INDEX IF EXISTS leads_city_idx;

ALTER TABLE properties DROP COLUMN IF EXISTS city;
ALTER TABLE leads DROP COLUMN IF EXISTS city;

-- +goose StatementEnd


