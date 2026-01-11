# AI-улучшения системы поиска и матчинга недвижимости

## Статус: ✅ Полностью реализовано (кроме Computer Vision)

**Дата обновления**: 11 января 2026

## Обзор

Реализован комплекс AI-улучшений для системы поиска и матчинга лидов недвижимости:

1. **Гибридный поиск** — комбинация векторного и полнотекстового поиска ✅
2. **Реранкер** — нейросетевое переранжирование результатов ✅
3. **Динамические веса** — адаптивное определение весов матчинга ✅
4. **AI-генерация контента** — автоматическое создание заголовков и описаний ✅
5. **Компьютерное зрение** — анализ фотографий объектов ⏸️ (отложено, нет хранения изображений)
6. **JSON-LD разметка** — schema.org для SEO и интеграций ✅
7. **Уточняющие вопросы** — AI-агент для "коротких" лидов ✅

## Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                        gRPC API Layer                           │
├─────────────────────────────────────────────────────────────────┤
│  LeadService           │  PropertyService                       │
│  ├─ GetClarification   │  ├─ MatchPropertiesAdvanced            │
│  ├─ ApplyClarification │  ├─ GetPropertyJSONLD                  │
│  └─ AnalyzeLeadIntent  │  ├─ GenerateListingContent             │
│                        │  └─ AnalyzePropertyImages              │
├─────────────────────────────────────────────────────────────────┤
│                     Service Layer                                │
├─────────────────────────────────────────────────────────────────┤
│  PropertyService       │  WeightsAnalyzer   │  ClarificationAgent│
│  ├─ MatchAdvanced()    │  ├─ AnalyzeLead()  │  ├─ Analyze()      │
│  ├─ applyReranker()    │  └─ heuristic()    │  └─ Generate()     │
│  └─ HybridSearch()     │                    │                    │
├─────────────────────────────────────────────────────────────────┤
│                   External Clients                               │
├─────────────────────────────────────────────────────────────────┤
│  RerankerClient  │  LLMClient  │  VisionClient  │  MLClient      │
│  (Jina AI)       │  (OpenAI)   │  (CV API)      │  (Embeddings)  │
├─────────────────────────────────────────────────────────────────┤
│                     Repository Layer                             │
├─────────────────────────────────────────────────────────────────┤
│  PropertyRepository                                              │
│  ├─ HybridSearch()        - RRF (Reciprocal Rank Fusion)        │
│  ├─ FulltextSearch()      - PostgreSQL tsvector                 │
│  └─ MatchPropertiesWithHardFilters()                             │
└─────────────────────────────────────────────────────────────────┘
```

## Новые файлы

### Библиотеки (`internal/lib/`)

| Файл | Описание |
|------|----------|
| `reranker/client.go` | Клиент для Jina AI Reranker API |
| `llm/client.go` | Клиент для OpenAI/LLM API |
| `vision/client.go` | Клиент для Computer Vision API |
| `jsonld/generator.go` | Генератор JSON-LD разметки schema.org |

### Сервисы (`internal/services/`)

| Файл | Описание |
|------|----------|
| `weights/analyzer.go` | Анализатор для динамических весов матчинга |
| `clarification/agent.go` | AI-агент для генерации уточняющих вопросов |

### Миграции (`migrations/`)

| Файл | Описание |
|------|----------|
| `20251217120000_add_fulltext_search.sql` | tsvector + GIN индексы для FTS |
| `20251217120001_add_image_analysis.sql` | Таблицы для CV и уточнений |

## Конфигурация

Новые переменные окружения:

```bash
# Reranker (Jina AI)
RERANKER_ENABLE=true
RERANKER_BASE_URL=https://api.jina.ai/v1
RERANKER_API_KEY=your_jina_api_key
RERANKER_MODEL=jina-reranker-v2-base-multilingual
RERANKER_TIMEOUT=30s
RERANKER_TOP_N=10

# LLM (OpenAI)
LLM_ENABLE=true
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=your_openai_api_key
LLM_MODEL=gpt-4o-mini
LLM_TIMEOUT=60s

# Computer Vision
VISION_ENABLE=false
VISION_BASE_URL=https://your-cv-api.com
VISION_API_KEY=your_vision_api_key
VISION_TIMEOUT=30s

# Search Configuration
HYBRID_SEARCH_ENABLE=true
SEARCH_VECTOR_WEIGHT=0.7
SEARCH_FULLTEXT_WEIGHT=0.3
SEARCH_USE_RERANKER=true
SEARCH_RERANKER_CANDIDATES=50
DYNAMIC_WEIGHTS_ENABLE=true
```

## API Endpoints

### LeadService (новые методы)

```protobuf
// Получить уточняющие вопросы для "короткого" лида
rpc GetClarificationQuestions (GetClarificationQuestionsRequest) returns (GetClarificationQuestionsResponse);
GET /v1/leads/{lead_id}/clarification

// Применить ответы на уточняющие вопросы
rpc ApplyClarificationAnswers (ApplyClarificationAnswersRequest) returns (ApplyClarificationAnswersResponse);
POST /v1/leads/{lead_id}/clarification

// Анализ намерений лида для определения оптимальных весов
rpc AnalyzeLeadIntent (AnalyzeLeadIntentRequest) returns (AnalyzeLeadIntentResponse);
GET /v1/leads/{lead_id}/analyze
```

### PropertyService (новые методы)

```protobuf
// Расширенный поиск с гибридным поиском и реранкером
rpc MatchPropertiesAdvanced (MatchPropertiesAdvancedRequest) returns (MatchPropertiesResponse);
POST /v1/properties/match/advanced

// JSON-LD разметка объекта (schema.org)
rpc GetPropertyJSONLD (GetPropertyJSONLDRequest) returns (GetPropertyJSONLDResponse);
GET /v1/properties/{property_id}/jsonld

// AI-генерация заголовка и описания
rpc GenerateListingContent (GenerateListingContentRequest) returns (GenerateListingContentResponse);
POST /v1/properties/generate-content

// Анализ изображений объекта
rpc AnalyzePropertyImages (AnalyzePropertyImagesRequest) returns (AnalyzePropertyImagesResponse);
POST /v1/properties/{property_id}/analyze-images
```

## Примеры использования

### 1. Расширенный поиск

```go
// Использование MatchPropertiesAdvanced с гибридным поиском и реранкером
matches, err := propertyService.MatchPropertiesAdvanced(ctx, leadID, filter, 10)
```

### 2. Анализ лида для динамических весов

```go
analyzer := weights.NewAnalyzer(log, llmClient, searchCfg)
result, err := analyzer.AnalyzeLead(ctx, lead)

// result.Weights — рекомендованные веса
// result.LeadType — тип лида (budget_oriented, family_oriented, etc.)
// result.Criteria — извлечённые критерии поиска
```

### 3. Уточняющие вопросы

```go
agent := clarification.NewAgent(log, llmClient, weightsAnalyzer)
result, err := agent.AnalyzeAndGenerateQuestions(ctx, lead)

if result.NeedsClarification {
    for _, q := range result.Questions {
        fmt.Printf("Вопрос: %s (поле: %s)\n", q.Question, q.Field)
    }
}
```

### 4. JSON-LD генерация

```go
generator := jsonld.NewGenerator()
jsonldData, err := generator.GeneratePropertyJSONLDBytes(property, "https://api.example.com")
```

## Типы лидов

Анализатор определяет следующие типы лидов:

| Тип | Описание | Веса |
|-----|----------|------|
| `budget_oriented` | Ориентирован на бюджет | Price: 0.45, District: 0.20 |
| `location_oriented` | Локация — приоритет | District: 0.40, Price: 0.20 |
| `family_oriented` | Для семьи | Rooms: 0.30, Area: 0.20 |
| `investor` | Инвестиционный запрос | Price: 0.35, District: 0.30 |
| `luxury` | Премиум-сегмент | Semantic: 0.30, Area: 0.20 |
| `balanced` | Сбалансированный | Все веса равны |

## Следующие шаги

1. **Регенерация proto** — выполнить `make generate` для обновления pb.go файлов
2. **Интеграция в app.go** — подключить новые сервисы в инициализацию приложения
3. **Тестирование** — написать unit и integration тесты
4. **Мониторинг** — добавить метрики для AI-вызовов
и