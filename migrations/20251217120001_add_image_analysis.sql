-- +goose Up
-- +goose StatementBegin

-- Таблица для хранения анализа изображений объектов
CREATE TABLE IF NOT EXISTS property_images (
    image_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id UUID NOT NULL REFERENCES properties(property_id) ON DELETE CASCADE,

    -- URL или путь к изображению
    image_url TEXT,
    storage_path TEXT,

    -- Результаты анализа CV
    detected_features JSONB DEFAULT '[]'::jsonb,
    room_type VARCHAR(50),
    quality_score FLOAT,
    view_type VARCHAR(50),
    brightness FLOAT,
    tags JSONB DEFAULT '{}'::jsonb,

    -- Визуальные признаки для эмбеддинга (16 float значений)
    visual_features vector(16),

    -- Метаданные анализа
    analysis_confidence FLOAT,
    analyzed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Индекс для быстрого поиска по property_id
CREATE INDEX IF NOT EXISTS idx_property_images_property_id ON property_images(property_id);

-- Индекс для поиска по типу комнаты
CREATE INDEX IF NOT EXISTS idx_property_images_room_type ON property_images(room_type);

-- Добавляем колонку для агрегированных визуальных признаков в properties
ALTER TABLE properties ADD COLUMN IF NOT EXISTS visual_features vector(16);
ALTER TABLE properties ADD COLUMN IF NOT EXISTS visual_assessment TEXT;
ALTER TABLE properties ADD COLUMN IF NOT EXISTS average_quality_score FLOAT;

-- Таблица для хранения истории уточняющих вопросов
CREATE TABLE IF NOT EXISTS lead_clarifications (
    clarification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL REFERENCES leads(lead_id) ON DELETE CASCADE,

    -- Сгенерированные вопросы
    questions JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Ответы пользователя
    answers JSONB DEFAULT '{}'::jsonb,

    -- Приоритет и статус
    priority VARCHAR(20) DEFAULT 'medium',
    status VARCHAR(20) DEFAULT 'pending', -- pending, answered, skipped

    -- Оценка качества лида до и после
    quality_score_before FLOAT,
    quality_score_after FLOAT,

    -- Метаданные
    created_at TIMESTAMP DEFAULT NOW(),
    answered_at TIMESTAMP,

    CONSTRAINT chk_priority CHECK (priority IN ('high', 'medium', 'low')),
    CONSTRAINT chk_status CHECK (status IN ('pending', 'answered', 'skipped'))
);

-- Индекс для поиска по lead_id
CREATE INDEX IF NOT EXISTS idx_lead_clarifications_lead_id ON lead_clarifications(lead_id);

-- Индекс для поиска pending clarifications
CREATE INDEX IF NOT EXISTS idx_lead_clarifications_status ON lead_clarifications(status) WHERE status = 'pending';

-- Таблица для кэширования JSON-LD разметки
CREATE TABLE IF NOT EXISTS property_jsonld_cache (
    property_id UUID PRIMARY KEY REFERENCES properties(property_id) ON DELETE CASCADE,
    jsonld_data JSONB NOT NULL,
    generated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS property_jsonld_cache;
DROP TABLE IF EXISTS lead_clarifications;
DROP TABLE IF EXISTS property_images;
ALTER TABLE properties DROP COLUMN IF EXISTS visual_features;
ALTER TABLE properties DROP COLUMN IF EXISTS visual_assessment;
ALTER TABLE properties DROP COLUMN IF EXISTS average_quality_score;
-- +goose StatementEnd

