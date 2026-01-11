-- +goose Up
-- +goose StatementBegin

-- Обновление размерности эмбеддингов с 384 на 1024 для модели ai-forever/ru-en-RoSBERTa
-- ВНИМАНИЕ: Это изменит тип колонки и очистит существующие данные эмбеддингов!
-- Эмбеддинги нужно будет пересчитать через ML сервис

-- Обновляем колонку embedding в таблице leads
ALTER TABLE leads ALTER COLUMN embedding TYPE vector(1024);

-- Обновляем колонку embedding в таблице properties
ALTER TABLE properties ALTER COLUMN embedding TYPE vector(1024);

-- Пересоздаём индексы для оптимизации с новой размерностью
DROP INDEX IF EXISTS leads_embedding_idx;
DROP INDEX IF EXISTS properties_embedding_idx;

CREATE INDEX leads_embedding_idx ON leads USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX properties_embedding_idx ON properties USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Возврат к старой размерности 384
DROP INDEX IF EXISTS leads_embedding_idx;
DROP INDEX IF EXISTS properties_embedding_idx;

ALTER TABLE leads ALTER COLUMN embedding TYPE vector(384);
ALTER TABLE properties ALTER COLUMN embedding TYPE vector(384);

CREATE INDEX leads_embedding_idx ON leads USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX properties_embedding_idx ON properties USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- +goose StatementEnd

