// Временный скрипт для сброса БД на Render
// Запуск: go run scripts/reset_db.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	// External connection string для Render
	// Формат: postgresql://user:password@host/database?sslmode=require
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Используем значение по умолчанию для Render
		connStr = "postgresql://lead_exchange_bk_user:8m2gtTRBW0iAr7nY2Aadzz0VcZBEVKYM@dpg-d5ht8vi4d50c739akh2g-a.oregon-postgres.render.com/lead_exchange_bk?sslmode=require"
	}

	fmt.Println("Connecting to database...")
	fmt.Printf("Host: %s\n", extractHost(connStr))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Connected successfully!")

	// SQL команды для выполнения
	commands := []string{
		// ЧАСТЬ 1: ПОЛНАЯ ОЧИСТКА БД
		"DROP TABLE IF EXISTS deals CASCADE",
		"DROP TABLE IF EXISTS properties CASCADE",
		"DROP TABLE IF EXISTS leads CASCADE",
		"DROP TABLE IF EXISTS users CASCADE",
		"DROP TABLE IF EXISTS goose_db_version CASCADE",
		"DROP EXTENSION IF EXISTS vector CASCADE",

		// ЧАСТЬ 2: СОЗДАНИЕ РАСШИРЕНИЙ
		"CREATE EXTENSION IF NOT EXISTS vector",

		// ЧАСТЬ 3: СОЗДАНИЕ ТАБЛИЦ
		`CREATE TABLE IF NOT EXISTS users (
			user_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email      TEXT UNIQUE NOT NULL,
			password_hash TEXT        NOT NULL,
			first_name TEXT        NOT NULL,
			last_name  TEXT        NOT NULL,
			phone      TEXT UNIQUE,
			agency_name TEXT,
			avatar_url TEXT,
			role       TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS leads (
			lead_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title           TEXT        NOT NULL,
			description     TEXT,
			requirement     JSONB       NOT NULL,
			contact_name    TEXT        NOT NULL,
			contact_phone   TEXT        NOT NULL,
			contact_email   TEXT,
			status          TEXT        NOT NULL,
			owner_user_id   UUID        NOT NULL,
			created_user_id UUID        NOT NULL,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS deals (
			deal_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			lead_id        UUID        NOT NULL,
			property_id    UUID,
			seller_user_id UUID        NOT NULL,
			buyer_user_id  UUID        NOT NULL,
			status         TEXT        NOT NULL,
			amount         BIGINT,
			commission     BIGINT,
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS properties (
			property_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title           TEXT        NOT NULL,
			description     TEXT,
			address         TEXT        NOT NULL,
			property_type   TEXT        NOT NULL,
			area            DOUBLE PRECISION,
			price           BIGINT,
			rooms           INT,
			status          TEXT        NOT NULL,
			owner_user_id   UUID        NOT NULL,
			created_user_id UUID        NOT NULL,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Дополнительные колонки
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE'",
		"ALTER TABLE leads ADD COLUMN IF NOT EXISTS city TEXT",
		"ALTER TABLE leads ADD COLUMN IF NOT EXISTS property_type TEXT DEFAULT 'PROPERTY_TYPE_UNSPECIFIED'",
		"ALTER TABLE properties ADD COLUMN IF NOT EXISTS city TEXT",

		// Эмбеддинги с размерностью 1024
		"ALTER TABLE leads ADD COLUMN IF NOT EXISTS embedding vector(1024)",
		"ALTER TABLE properties ADD COLUMN IF NOT EXISTS embedding vector(1024)",

		// Полнотекстовый поиск
		"ALTER TABLE properties ADD COLUMN IF NOT EXISTS search_vector tsvector",
		"ALTER TABLE leads ADD COLUMN IF NOT EXISTS search_vector tsvector",
	}

	fmt.Println("\nExecuting schema commands...")
	for i, cmd := range commands {
		_, err := conn.Exec(ctx, cmd)
		if err != nil {
			log.Printf("Warning on command %d: %v", i+1, err)
		} else {
			fmt.Printf("  [%d/%d] OK\n", i+1, len(commands))
		}
	}

	// ЧАСТЬ 4: ТЕСТОВЫЕ ДАННЫЕ
	fmt.Println("\nInserting test users...")
	_, err = conn.Exec(ctx, `
		INSERT INTO users (user_id, email, password_hash, first_name, last_name, phone, agency_name, avatar_url, role)
		VALUES
			('8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', 'user@m.c', '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', 'Поль', 'Зователёв', '+79991112233', 'Best Realty', 'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png', 'USER'),
			('aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'user2@m.c', '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', 'ПольДва', 'ЗователёвДва', '+79991112244', 'Worst Realty', 'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png', 'USER'),
			('f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'admin@m.c', '$2a$10$NvlZBQmOscWN4lm9IwEQUu4Mz.27V5408.u6FA0XaRSXFiifgtndi', 'Админ', 'Нистраторов', '+79992223344', 'Lead Exchange HQ', 'https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png', 'ADMIN')
		ON CONFLICT (user_id) DO NOTHING
	`)
	if err != nil {
		log.Printf("Warning inserting users: %v", err)
	} else {
		fmt.Println("  Users inserted OK")
	}

	fmt.Println("Inserting test leads...")
	_, err = conn.Exec(ctx, `
		INSERT INTO leads (lead_id, title, description, requirement, contact_name, contact_phone, contact_email, status, owner_user_id, created_user_id, city)
		VALUES
			('a8b55f9d-32c2-4e1f-97c7-341f49b7c012', '3-комнатная квартира в центре', 'Просторная квартира рядом с метро и парком', '{"roomNumber": 3, "preferredPrice": "8000000", "district": "Центральный"}', 'Иван Петров', '+79991112233', 'ivan.petrov@example.com', 'PUBLISHED', '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', 'Санкт-Петербург'),
			('b5d7a10e-418d-42a3-bb32-87e90d4a7a24', 'Дом у моря', 'Двухэтажный дом с видом на залив', '{"rooms": 5, "preferredPrice": "25000000", "region": "Приморский"}', 'Ольга Сидорова', '+79995557788', 'olga.sid@example.com', 'PUBLISHED', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'Санкт-Петербург'),
			('c7d9e1ff-8a9e-4a4e-9b5c-b47c3fddf311', 'Квартира для инвестиций', 'Новая квартира в развивающемся районе', '{"roomNumber": 1, "preferredPrice": "4200000", "yield": "7%"}', 'Дмитрий Котов', '+79993334455', 'd.kotov@example.com', 'PURCHASED', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', 'Москва'),
			('e1b88dcf-1225-4d0d-827f-4ea8fdf99664', '2-комнатная квартира у метро', 'Ищу 2-комнатную квартиру в Санкт-Петербурге, рядом с метро, бюджет до 12 млн', '{"roomNumber": 2, "preferredPrice": "12000000", "metro": "близко"}', 'Мария Белова', '+79998889900', 'm.belova@example.com', 'PUBLISHED', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'Санкт-Петербург')
		ON CONFLICT (lead_id) DO NOTHING
	`)
	if err != nil {
		log.Printf("Warning inserting leads: %v", err)
	} else {
		fmt.Println("  Leads inserted OK")
	}

	fmt.Println("Inserting test properties...")
	_, err = conn.Exec(ctx, `
		INSERT INTO properties (property_id, title, description, address, property_type, area, price, rooms, status, owner_user_id, created_user_id, city)
		VALUES
			('d1a2b3c4-1234-5678-9abc-def012345678', '2-комнатная квартира на Невском', 'Светлая квартира с видом на Невский проспект, евроремонт, рядом метро', 'Санкт-Петербург, Невский проспект, 100', 'APARTMENT', 65.5, 11500000, 2, 'PUBLISHED', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'Санкт-Петербург'),
			('d2b3c4d5-2345-6789-abcd-ef0123456789', '3-комнатная квартира в центре', 'Просторная квартира в историческом центре, высокие потолки, парковка', 'Санкт-Петербург, ул. Рубинштейна, 25', 'APARTMENT', 95.0, 18000000, 3, 'PUBLISHED', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'Санкт-Петербург'),
			('d3c4d5e6-3456-789a-bcde-f01234567890', '2-комнатная квартира у метро Московская', 'Уютная квартира, 5 минут до метро, тихий двор, балкон', 'Санкт-Петербург, Московский проспект, 180', 'APARTMENT', 54.0, 9800000, 2, 'PUBLISHED', '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', '8c6f9c70-9312-4f17-94b0-2a2b9230f5d1', 'Санкт-Петербург'),
			('d4d5e6f7-4567-89ab-cdef-012345678901', 'Студия в новостройке', 'Современная студия с отделкой, закрытый двор, подземный паркинг', 'Санкт-Петербург, ул. Оптиков, 52', 'APARTMENT', 32.0, 5500000, 1, 'PUBLISHED', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'aea6842b-c540-4aa8-aa1f-90b1b46aba12', 'Санкт-Петербург'),
			('d5e6f7a8-5678-9abc-def0-123456789012', '1-комнатная квартира на Петроградке', 'Квартира с ремонтом, рядом парк, школа, детский сад', 'Санкт-Петербург, ул. Большая Пушкарская, 15', 'APARTMENT', 42.0, 8200000, 1, 'PUBLISHED', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'f4e8f58b-94f4-4e0f-bd85-1b06b8a3f242', 'Санкт-Петербург')
		ON CONFLICT (property_id) DO NOTHING
	`)
	if err != nil {
		log.Printf("Warning inserting properties: %v", err)
	} else {
		fmt.Println("  Properties inserted OK")
	}

	// ЧАСТЬ 5: ПРОВЕРКА
	fmt.Println("\n=== VERIFICATION ===")

	var userCount, leadCount, propCount, dealCount int
	conn.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&userCount)
	conn.QueryRow(ctx, "SELECT count(*) FROM leads").Scan(&leadCount)
	conn.QueryRow(ctx, "SELECT count(*) FROM properties").Scan(&propCount)
	conn.QueryRow(ctx, "SELECT count(*) FROM deals").Scan(&dealCount)

	fmt.Printf("Users:      %d\n", userCount)
	fmt.Printf("Leads:      %d\n", leadCount)
	fmt.Printf("Properties: %d\n", propCount)
	fmt.Printf("Deals:      %d\n", dealCount)

	// Проверяем размерность эмбеддингов
	var udtName string
	err = conn.QueryRow(ctx, `
		SELECT udt_name FROM information_schema.columns
		WHERE table_name = 'leads' AND column_name = 'embedding'
	`).Scan(&udtName)
	if err == nil {
		fmt.Printf("\nEmbedding column type: %s\n", udtName)
	}

	fmt.Println("\n=== DATABASE RESET COMPLETE ===")
	fmt.Println("Test users: user@m.c, agent@m.c, admin@m.c (password: password)")
}

func extractHost(connStr string) string {
	parts := strings.Split(connStr, "@")
	if len(parts) > 1 {
		hostPart := strings.Split(parts[1], "/")[0]
		return hostPart
	}
	return "unknown"
}

