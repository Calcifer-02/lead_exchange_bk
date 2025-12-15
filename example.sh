#!/bin/bash

# ============================================================================
# Примеры curl запросов для тестирования работы эмбеддингов
# ============================================================================
# 
# Копируйте команды и выполняйте их вручную в терминале
# 
# ============================================================================

cat << 'EOF'

============================================================================
0. Тест ML сервиса напрямую
============================================================================

curl -X POST "https://calcifer0323-matching.hf.space/prepare-and-embed" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Ищу 3-комнатную квартиру в центре",
    "description": "Нужна просторная квартира для семьи с детьми",
    "requirement": {"roomNumber": 3, "preferredPrice": "8000000", "district": "Центральный"},
    "price": 8000000,
    "rooms": 3,
    "district": "Центральный"
  }'

Ожидаемый результат: embedding размерности 384

============================================================================
1. Создание лида (embedding будет сгенерирован автоматически)
============================================================================

curl -X POST "http://localhost:8081/v1/leads" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "title": "Ищу 3-комнатную квартиру в центре",
    "description": "Нужна просторная квартира для семьи с детьми, рядом с метро",
    "requirement": "eyJyb29tTnVtYmVyIjozLCJwcmVmZXJyZWRQcmljZSI6IjgwMDAwMDAiLCJkaXN0cmljdCI6ItCm0LXQvdGC0YDQsNC70YzQvdGL0LkifQ==",
    "contact_name": "Иван Петров",
    "contact_phone": "+79991112233",
    "contact_email": "ivan@example.com"
  }'

Сохраните lead_id из ответа!
После создания подождите 5-10 секунд для генерации embedding.

============================================================================
2. Получение лида (проверка что embedding был создан)
============================================================================

# Замените {LEAD_ID} на реальный ID из шага 1
curl -X GET "http://localhost:8081/v1/leads/{LEAD_ID}" \
  -H "Authorization: Bearer test"

# Или только основные поля:
curl -X GET "http://localhost:8081/v1/leads/{LEAD_ID}" \
  -H "Authorization: Bearer test" | jq '.lead | {lead_id, title, status, created_at}'

============================================================================
3. Создание объекта недвижимости (embedding будет сгенерирован автоматически)
============================================================================

curl -X POST "http://localhost:8081/v1/properties" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "title": "3-комнатная квартира в центре Москвы",
    "description": "Просторная квартира с видом на парк, современный ремонт",
    "address": "г. Москва, ул. Тверская, д. 10, кв. 45",
    "property_type": "PROPERTY_TYPE_APARTMENT",
    "area": 85.5,
    "price": 15000000,
    "rooms": 3
  }'

Сохраните property_id из ответа!
После создания подождите 5-10 секунд для генерации embedding.

============================================================================
4. Создание дополнительных объектов для тестирования матчинга
============================================================================

# Квартира подходящая (3 комнаты)
curl -X POST "http://localhost:8081/v1/properties" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "title": "3-комнатная квартира у метро",
    "description": "Квартира для семьи, рядом метро, детский сад",
    "address": "г. Москва, ул. Ленинградская, д. 25, кв. 12",
    "property_type": "PROPERTY_TYPE_APARTMENT",
    "area": 90.0,
    "price": 12000000,
    "rooms": 3
  }'

# Квартира не подходящая (1 комната)
curl -X POST "http://localhost:8081/v1/properties" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "title": "Студия в центре",
    "description": "Маленькая студия для одного",
    "address": "г. Москва, ул. Арбат, д. 5, кв. 1",
    "property_type": "PROPERTY_TYPE_APARTMENT",
    "area": 35.0,
    "price": 8000000,
    "rooms": 1
  }'

# Дом (не подходит по типу)
curl -X POST "http://localhost:8081/v1/properties" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "title": "Загородный дом с участком",
    "description": "Двухэтажный дом, 6 соток, баня, гараж",
    "address": "Московская обл., д. Подмосковное, ул. Дачная, д. 15",
    "property_type": "PROPERTY_TYPE_HOUSE",
    "area": 180.0,
    "price": 25000000,
    "rooms": 5
  }'

После создания всех объектов подождите 10-15 секунд для генерации embedding.

============================================================================
5. Матчинг: поиск подходящих объектов для лида
============================================================================

# Замените {LEAD_ID} на реальный ID из шага 1
curl -X POST "http://localhost:8081/v1/properties/match" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "lead_id": "df4ccc33-47c6-4e93-9396-351cd54cd0ba",
    "filter": {
      "status": "PROPERTY_STATUS_PUBLISHED",
      "property_type": "PROPERTY_TYPE_APARTMENT",
      "min_rooms": 3,
      "max_rooms": 3,
      "min_price": 5000000,
      "max_price": 20000000
    },
    "limit": 10
  }'

# Матчинг без фильтров (все объекты):
curl -X POST "http://localhost:8081/v1/properties/match" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "lead_id": "df4ccc33-47c6-4e93-9396-351cd54cd0ba",
    "limit": 10
  }'

# Вывод только топ-5 с коэффициентом схожести:
curl -X POST "http://localhost:8081/v1/properties/match" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "lead_id": "df4ccc33-47c6-4e93-9396-351cd54cd0ba",
    "limit": 5
  }' | jq '.matches[] | "\(.similarity | tostring | .[0:5]) - \(.property.title) (\(.property.price) руб.)"'

============================================================================
6. Дополнительные полезные команды
============================================================================

# Получить все лиды:
curl -X GET "http://localhost:8081/v1/leads" \
  -H "Authorization: Bearer test"

# Получить все лиды с фильтром по статусу:
curl -X GET "http://localhost:8081/v1/leads?filter.status=LEAD_STATUS_PUBLISHED" \
  -H "Authorization: Bearer test"

# Получить все объекты недвижимости:
curl -X GET "http://localhost:8081/v1/properties" \
  -H "Authorization: Bearer test"

# Получить объект недвижимости по ID:
curl -X GET "http://localhost:8081/v1/properties/{PROPERTY_ID}" \
  -H "Authorization: Bearer test"

# Получить объекты с фильтром:
curl -X GET "http://localhost:8081/v1/properties?filter.status=PROPERTY_STATUS_PUBLISHED&filter.property_type=PROPERTY_TYPE_APARTMENT" \
  -H "Authorization: Bearer test"

# Обновить статус лида (опубликовать):
curl -X PATCH "http://localhost:8081/v1/leads/{LEAD_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "status": "LEAD_STATUS_PUBLISHED"
  }'

# Обновить статус объекта недвижимости (опубликовать):
curl -X PATCH "http://localhost:8081/v1/properties/{PROPERTY_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "status": "PROPERTY_STATUS_PUBLISHED"
  }'

============================================================================
Примечания:
============================================================================

1. Embedding генерируется асинхронно после создания лида/property
2. Подождите 5-10 секунд после создания перед проверкой embedding
3. Для матчинга нужен лид с уже сгенерированным embedding
4. Коэффициент схожести (similarity) от 0 до 1, где 1 - полное совпадение
5. Матчинг использует косинусное расстояние через pgvector
6. requirement для лида передаётся как base64 строка JSON

EOF
