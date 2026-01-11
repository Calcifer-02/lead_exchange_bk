# Скрипт для тестирования пайплайна создания лида и генерации эмбеддинга
# Запуск: .\test_embedding_pipeline.ps1

$API_URL = "https://lead-exchange-bk.onrender.com"
$ML_URL = "https://calcifer0323-matching.hf.space"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ТЕСТИРОВАНИЕ ПАЙПЛАЙНА ЭМБЕДДИНГОВ" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Шаг 0: Проверяем ML сервис
Write-Host "`n[ШАГ 0] Проверка ML сервиса..." -ForegroundColor Yellow
try {
    $mlInfo = Invoke-RestMethod -Uri "$ML_URL/model-info" -Method GET -TimeoutSec 30
    Write-Host "✓ ML сервис доступен" -ForegroundColor Green
    Write-Host "  Модель: $($mlInfo.model_name)" -ForegroundColor Gray
    Write-Host "  Размерность: $($mlInfo.dimensions)" -ForegroundColor Gray
} catch {
    Write-Host "✗ ML сервис недоступен: $_" -ForegroundColor Red
    exit 1
}

# Шаг 1: Авторизация
Write-Host "`n[ШАГ 1] Авторизация..." -ForegroundColor Yellow
try {
    $loginBody = '{"email":"admin@m.c","password":"password"}'
    $loginResponse = Invoke-RestMethod -Uri "$API_URL/v1/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
    $token = $loginResponse.token
    Write-Host "✓ Авторизация успешна" -ForegroundColor Green
    Write-Host "  Token: $($token.Substring(0, 50))..." -ForegroundColor Gray
} catch {
    Write-Host "✗ Ошибка авторизации: $_" -ForegroundColor Red
    exit 1
}

$headers = @{ "Authorization" = "Bearer $token" }

# Шаг 2: Создаём тестовый лид
Write-Host "`n[ШАГ 2] Создание тестового лида..." -ForegroundColor Yellow
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$leadBody = @{
    title = "ТЕСТ_$timestamp - 2к квартира в СПб"
    description = "Тестовый лид для проверки эмбеддингов. Ищу 2-комнатную квартиру в Санкт-Петербурге, центр или Московский район. Бюджет до 12 млн рублей."
    contactName = "Тест Тестович"
    contactPhone = "+79991234567"
    contactEmail = "test_$timestamp@test.com"
    city = "Санкт-Петербург"
} | ConvertTo-Json -Depth 3

try {
    $createResponse = Invoke-RestMethod -Uri "$API_URL/v1/leads" -Method POST -Headers $headers -Body $leadBody -ContentType "application/json"
    $leadId = $createResponse.lead.leadId
    Write-Host "✓ Лид создан успешно" -ForegroundColor Green
    Write-Host "  Lead ID: $leadId" -ForegroundColor Gray
    Write-Host "  Title: $($createResponse.lead.title)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Ошибка создания лида" -ForegroundColor Red
    Write-Host "  Статус: $($_.Exception.Response.StatusCode)" -ForegroundColor Red
    try {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Host "  Ответ: $($reader.ReadToEnd())" -ForegroundColor Red
    } catch {}
    exit 1
}

# Подождём немного, чтобы эмбеддинг сгенерировался
Write-Host "`n  Ожидаем 5 секунд для генерации эмбеддинга..." -ForegroundColor Gray
Start-Sleep -Seconds 5

# Шаг 3: Проверяем лид через API
Write-Host "`n[ШАГ 3] Проверка созданного лида через API..." -ForegroundColor Yellow
try {
    $getLeadResponse = Invoke-RestMethod -Uri "$API_URL/v1/leads/$leadId" -Method GET -Headers $headers
    Write-Host "✓ Лид получен через API" -ForegroundColor Green
    Write-Host "  Title: $($getLeadResponse.lead.title)" -ForegroundColor Gray
    Write-Host "  City: $($getLeadResponse.lead.city)" -ForegroundColor Gray
    Write-Host "  Status: $($getLeadResponse.lead.status)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Ошибка получения лида: $_" -ForegroundColor Red
}

# Шаг 4: Тестируем ML сервис напрямую
Write-Host "`n[ШАГ 4] Тестирование ML сервиса напрямую..." -ForegroundColor Yellow
$mlTestBody = @{
    title = "ТЕСТ - 2к квартира в СПб"
    description = "Тестовый лид для проверки эмбеддингов"
} | ConvertTo-Json

try {
    $mlResponse = Invoke-RestMethod -Uri "$ML_URL/prepare-and-embed" -Method POST -Body $mlTestBody -ContentType "application/json" -TimeoutSec 60
    Write-Host "✓ ML сервис вернул эмбеддинг" -ForegroundColor Green
    Write-Host "  Размерность: $($mlResponse.dimensions)" -ForegroundColor Gray
    Write-Host "  Длина массива: $($mlResponse.embedding.Length)" -ForegroundColor Gray
    Write-Host "  Первые 5 значений: $($mlResponse.embedding[0..4] -join ', ')" -ForegroundColor Gray
} catch {
    Write-Host "✗ Ошибка ML сервиса: $_" -ForegroundColor Red
}

# Шаг 5: Пробуем переиндексировать лид
Write-Host "`n[ШАГ 5] Переиндексация лида..." -ForegroundColor Yellow
try {
    $reindexResponse = Invoke-RestMethod -Uri "$API_URL/v1/leads/$leadId/reindex" -Method POST -Headers $headers -ContentType "application/json"
    Write-Host "✓ Переиндексация успешна" -ForegroundColor Green
    Write-Host "  Message: $($reindexResponse.message)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Ошибка переиндексации" -ForegroundColor Red
    try {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $errorBody = $reader.ReadToEnd()
        Write-Host "  Ответ: $errorBody" -ForegroundColor Red
    } catch {
        Write-Host "  $_" -ForegroundColor Red
    }
}

# Шаг 6: Пробуем матчинг
Write-Host "`n[ШАГ 6] Тестирование матчинга..." -ForegroundColor Yellow
$matchBody = @{
    leadId = $leadId
    limit = 5
    filter = @{
        status = "PROPERTY_STATUS_PUBLISHED"
    }
} | ConvertTo-Json -Depth 3

try {
    $matchResponse = Invoke-RestMethod -Uri "$API_URL/v1/properties/match" -Method POST -Headers $headers -Body $matchBody -ContentType "application/json"
    Write-Host "✓ Матчинг выполнен" -ForegroundColor Green
    Write-Host "  Найдено совпадений: $($matchResponse.matches.Length)" -ForegroundColor Gray

    if ($matchResponse.matches.Length -gt 0) {
        Write-Host "  Топ-3 результата:" -ForegroundColor Gray
        $matchResponse.matches | Select-Object -First 3 | ForEach-Object {
            Write-Host "    - $($_.property.title) (score: $([math]::Round($_.similarity, 3)))" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "✗ Ошибка матчинга" -ForegroundColor Red
    try {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $errorBody = $reader.ReadToEnd()
        Write-Host "  Ответ: $errorBody" -ForegroundColor Red
    } catch {
        Write-Host "  $_" -ForegroundColor Red
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "ИТОГИ ТЕСТИРОВАНИЯ" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Lead ID для проверки в БД: $leadId" -ForegroundColor Yellow
Write-Host "`nSQL для проверки эмбеддинга в БД:" -ForegroundColor Yellow
Write-Host "SELECT lead_id, title, embedding IS NOT NULL as has_embedding FROM leads WHERE lead_id = '$leadId';" -ForegroundColor Gray

Write-Host "`nДля проверки всех лидов:" -ForegroundColor Yellow
Write-Host "SELECT lead_id, title, embedding IS NOT NULL as has_embedding, created_at FROM leads ORDER BY created_at DESC LIMIT 10;" -ForegroundColor Gray

