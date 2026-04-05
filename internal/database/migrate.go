package database

import (
    "context"
    "embed"
    "fmt"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

func RunMigrations(dbURL string) error {
    pool, err := pgxpool.New(context.Background(), dbURL)
    if err != nil {
        return fmt.Errorf("connect: %w", err)
    }
    defer pool.Close()

    // Read and execute migration files in order
    files := []string{
        "migrations/001_create_schema.sql",
        "migrations/002_seed_data.sql",
    }

    for _, f := range files {
        content, err := migrations.ReadFile(f)
        if err != nil {
            return fmt.Errorf("read %s: %w", f, err)
        }
        if _, err := pool.Exec(context.Background(), string(content)); err != nil {
            return fmt.Errorf("exec %s: %w", f, err)
        }
    }
    return nil
}

func SeedData(pool *pgxpool.Pool) error {
    // Placeholder - real implementation would load JSON files
    return nil
}
