# Скрипт переиндексации всех лидов и объектов после сброса БД
# Запуск: .\reindex_all.ps1

$API_URL = "https://lead-exchange-bk.onrender.com"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "REINDEX ALL LEADS AND PROPERTIES" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Авторизация
Write-Host "`n[1/3] Authenticating..." -ForegroundColor Yellow
$loginBody = '{"email":"admin@m.c","password":"password"}'
try {
    $loginResponse = Invoke-RestMethod -Uri "$API_URL/v1/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
    $token = $loginResponse.token
    Write-Host "  OK - Token received" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: Failed to login - $_" -ForegroundColor Red
    exit 1
}

$headers = @{ "Authorization" = "Bearer $token" }

# Переиндексация лидов
Write-Host "`n[2/3] Reindexing leads..." -ForegroundColor Yellow
$leadIds = @(
    "a8b55f9d-32c2-4e1f-97c7-341f49b7c012",
    "b5d7a10e-418d-42a3-bb32-87e90d4a7a24",
    "c7d9e1ff-8a9e-4a4e-9b5c-b47c3fddf311",
    "e1b88dcf-1225-4d0d-827f-4ea8fdf99664"
)

$successLeads = 0
$failedLeads = 0

foreach ($id in $leadIds) {
    Write-Host "  Reindexing lead $id..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$API_URL/v1/leads/$id/reindex" -Method POST -Headers $headers -ContentType "application/json" -TimeoutSec 60
        Write-Host " OK" -ForegroundColor Green
        $successLeads++
    } catch {
        Write-Host " FAILED" -ForegroundColor Red
        $failedLeads++
        # Получаем детали ошибки
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            Write-Host "    Error: $($reader.ReadToEnd())" -ForegroundColor Red
        } catch {}
    }
    # Небольшая пауза между запросами
    Start-Sleep -Milliseconds 500
}

Write-Host "  Leads: $successLeads OK, $failedLeads failed" -ForegroundColor $(if ($failedLeads -eq 0) { "Green" } else { "Yellow" })

# Переиндексация объектов
Write-Host "`n[3/3] Reindexing properties..." -ForegroundColor Yellow
$propertyIds = @(
    "d1a2b3c4-1234-5678-9abc-def012345678",
    "d2b3c4d5-2345-6789-abcd-ef0123456789",
    "d3c4d5e6-3456-789a-bcde-f01234567890",
    "d4d5e6f7-4567-89ab-cdef-012345678901",
    "d5e6f7a8-5678-9abc-def0-123456789012"
)

$successProps = 0
$failedProps = 0

foreach ($id in $propertyIds) {
    Write-Host "  Reindexing property $id..." -NoNewline
    try {
        $response = Invoke-RestMethod -Uri "$API_URL/v1/properties/$id/reindex" -Method POST -Headers $headers -ContentType "application/json" -TimeoutSec 60
        Write-Host " OK" -ForegroundColor Green
        $successProps++
    } catch {
        Write-Host " FAILED" -ForegroundColor Red
        $failedProps++
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            Write-Host "    Error: $($reader.ReadToEnd())" -ForegroundColor Red
        } catch {}
    }
    Start-Sleep -Milliseconds 500
}

Write-Host "  Properties: $successProps OK, $failedProps failed" -ForegroundColor $(if ($failedProps -eq 0) { "Green" } else { "Yellow" })

# Итоги
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "REINDEX COMPLETE" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Leads:      $successLeads/$($leadIds.Length) indexed" -ForegroundColor $(if ($failedLeads -eq 0) { "Green" } else { "Yellow" })
Write-Host "Properties: $successProps/$($propertyIds.Length) indexed" -ForegroundColor $(if ($failedProps -eq 0) { "Green" } else { "Yellow" })

if ($failedLeads -eq 0 -and $failedProps -eq 0) {
    Write-Host "`nAll items indexed successfully!" -ForegroundColor Green
    Write-Host "You can now test matching at the frontend." -ForegroundColor Gray
} else {
    Write-Host "`nSome items failed to index. Check the errors above." -ForegroundColor Yellow
    Write-Host "Make sure the database has correct embedding dimensions (1024)." -ForegroundColor Gray
}

