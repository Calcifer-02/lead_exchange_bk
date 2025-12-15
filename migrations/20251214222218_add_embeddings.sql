-- +goose Up
-- +goose StatementBegin

-- Устанавливаем расширение pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- Добавляем колонку embedding в таблицу leads
ALTER TABLE leads ADD COLUMN IF NOT EXISTS embedding vector(384);

-- Добавляем колонку embedding в таблицу properties
ALTER TABLE properties ADD COLUMN IF NOT EXISTS embedding vector(384);

-- Создаём индексы для быстрого поиска по косинусному расстоянию
CREATE INDEX IF NOT EXISTS leads_embedding_idx ON leads USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS properties_embedding_idx ON properties USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS properties_embedding_idx;
DROP INDEX IF EXISTS leads_embedding_idx;

ALTER TABLE properties DROP COLUMN IF EXISTS embedding;
ALTER TABLE leads DROP COLUMN IF EXISTS embedding;

-- Расширение vector не удаляем, так как оно может использоваться другими таблицами

-- +goose StatementEnd

