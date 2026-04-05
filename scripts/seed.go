$content = @"
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("✓ Connected to database")
	fmt.Println("✓ Seed complete (placeholder)")
}
"@
Set-Content -Path "scripts/seed.go" -Value $content -Encoding utf8