package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgresql://lead_exchange_bk_user:8m2gtTRBW0iAr7nY2Aadzz0VcZBEVKYM@dpg-d5ht8vi4d50c739akh2g-a.oregon-postgres.render.com/lead_exchange_bk?sslmode=require"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Connected to database...")

	// Создаём таблицу goose_db_version
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id SERIAL PRIMARY KEY,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Warning creating goose table: %v", err)
	}

	// Очищаем
	conn.Exec(ctx, "DELETE FROM goose_db_version")

	// Вставляем все версии как применённые
	versions := []int64{
		0,
		20251110162249,
		20251110170546,
		20251112114136,
		20251112114545,
		20251112193703,
		20251112201017,
		20251113000245,
		20251214194937,
		20251214210945,
		20251214222218,
		20251216120000,
		20251217120000,
		20251217120001,
		20260111000000,
	}

	for _, v := range versions {
		_, err = conn.Exec(ctx, "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)", v)
		if err != nil {
			log.Printf("Warning inserting version %d: %v", v, err)
		} else {
			fmt.Printf("  Version %d: OK\n", v)
		}
	}

	fmt.Println("\nGoose versions inserted successfully!")

	var count int
	conn.QueryRow(ctx, "SELECT count(*) FROM goose_db_version").Scan(&count)
	fmt.Printf("Total versions: %d\n", count)
}

