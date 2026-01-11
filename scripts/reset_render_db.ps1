# Скрипт для сброса и заполнения БД на Render
# Требуется установленный psql (PostgreSQL client)
#
# Установка psql на Windows:
# 1. Скачайте PostgreSQL с https://www.postgresql.org/download/windows/
# 2. При установке выберите только "Command Line Tools"
# 3. Добавьте путь к psql в PATH (обычно C:\Program Files\PostgreSQL\16\bin)

$DB_HOST = "dpg-d5ht8vi4d50c739akh2g-a.oregon-postgres.render.com"
$DB_NAME = "lead_exchange_bk"
$DB_USER = "lead_exchange_bk_user"
$DB_PASSWORD = "8m2gtTRBW0iAr7nY2Aadzz0VcZBEVKYM"
$DB_PORT = "5432"

$CONNECTION_STRING = "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require"

$SQL_FILE = "$PSScriptRoot\reset_and_seed_db.sql"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "RESET AND SEED DATABASE ON RENDER" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

Write-Host "`nDatabase: $DB_NAME" -ForegroundColor Yellow
Write-Host "Host: $DB_HOST" -ForegroundColor Yellow
Write-Host "SQL File: $SQL_FILE" -ForegroundColor Yellow

# Проверяем наличие файла SQL
if (-not (Test-Path $SQL_FILE)) {
    Write-Host "`nERROR: SQL file not found: $SQL_FILE" -ForegroundColor Red
    exit 1
}

Write-Host "`n[WARNING] This will DELETE ALL DATA and recreate tables!" -ForegroundColor Red
$confirm = Read-Host "Are you sure you want to continue? (yes/no)"

if ($confirm -ne "yes") {
    Write-Host "Aborted." -ForegroundColor Yellow
    exit 0
}

# Проверяем наличие psql
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlPath) {
    Write-Host "`nERROR: psql not found. Please install PostgreSQL client tools." -ForegroundColor Red
    Write-Host "Download from: https://www.postgresql.org/download/windows/" -ForegroundColor Yellow
    Write-Host "`nAlternatively, you can run the SQL manually:" -ForegroundColor Yellow
    Write-Host "1. Go to Render Dashboard -> Database -> Connect" -ForegroundColor Gray
    Write-Host "2. Use External Connection with psql or any SQL client" -ForegroundColor Gray
    Write-Host "3. Copy and paste contents of: $SQL_FILE" -ForegroundColor Gray
    exit 1
}

Write-Host "`nExecuting SQL..." -ForegroundColor Yellow

# Устанавливаем переменную окружения для пароля
$env:PGPASSWORD = $DB_PASSWORD

try {
    # Выполняем SQL файл
    $result = & psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $SQL_FILE 2>&1

    Write-Host "`nOutput:" -ForegroundColor Green
    Write-Host $result

    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "DATABASE RESET COMPLETE!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Cyan

    Write-Host "`nTest users created:" -ForegroundColor Yellow
    Write-Host "  - user@m.c / password (USER)" -ForegroundColor Gray
    Write-Host "  - agent@m.c / password (AGENT)" -ForegroundColor Gray
    Write-Host "  - admin@m.c / password (ADMIN)" -ForegroundColor Gray

    Write-Host "`nNext steps:" -ForegroundColor Yellow
    Write-Host "  1. Wait for Render to redeploy (if auto-deploy is on)" -ForegroundColor Gray
    Write-Host "  2. Test login at https://lead-exchange-bk.onrender.com/v1/auth/login" -ForegroundColor Gray
    Write-Host "  3. Reindex leads and properties to generate embeddings" -ForegroundColor Gray

} catch {
    Write-Host "`nERROR: Failed to execute SQL" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
} finally {
    # Очищаем переменную окружения
    Remove-Item Env:\PGPASSWORD -ErrorAction SilentlyContinue
}

