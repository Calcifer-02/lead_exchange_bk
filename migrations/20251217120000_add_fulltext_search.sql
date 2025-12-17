-- +goose Up
-- +goose StatementBegin

-- Добавляем tsvector колонку для полнотекстового поиска по properties
ALTER TABLE properties ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Создаём функцию для обновления search_vector
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

-- Создаём триггер для автоматического обновления search_vector
DROP TRIGGER IF EXISTS properties_search_vector_trigger ON properties;
CREATE TRIGGER properties_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description, address, city ON properties
    FOR EACH ROW
    EXECUTE FUNCTION properties_search_vector_update();

-- Создаём GIN индекс для быстрого полнотекстового поиска
CREATE INDEX IF NOT EXISTS idx_properties_search_vector ON properties USING GIN(search_vector);

-- Обновляем существующие записи
UPDATE properties SET search_vector =
    setweight(to_tsvector('russian', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('russian', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('russian', COALESCE(address, '')), 'C') ||
    setweight(to_tsvector('russian', COALESCE(city, '')), 'A')
WHERE search_vector IS NULL;

-- Добавляем tsvector колонку для leads
ALTER TABLE leads ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Создаём функцию для обновления search_vector у leads
CREATE OR REPLACE FUNCTION leads_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('russian', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('russian', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Создаём триггер для leads
DROP TRIGGER IF EXISTS leads_search_vector_trigger ON leads;
CREATE TRIGGER leads_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, description ON leads
    FOR EACH ROW
    EXECUTE FUNCTION leads_search_vector_update();

-- Создаём GIN индекс для leads
CREATE INDEX IF NOT EXISTS idx_leads_search_vector ON leads USING GIN(search_vector);

-- Обновляем существующие записи leads
UPDATE leads SET search_vector =
    setweight(to_tsvector('russian', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('russian', COALESCE(description, '')), 'B')
WHERE search_vector IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS properties_search_vector_trigger ON properties;
DROP FUNCTION IF EXISTS properties_search_vector_update();
DROP INDEX IF EXISTS idx_properties_search_vector;
ALTER TABLE properties DROP COLUMN IF EXISTS search_vector;

DROP TRIGGER IF EXISTS leads_search_vector_trigger ON leads;
DROP FUNCTION IF EXISTS leads_search_vector_update();
DROP INDEX IF EXISTS idx_leads_search_vector;
ALTER TABLE leads DROP COLUMN IF EXISTS search_vector;
-- +goose StatementEnd

