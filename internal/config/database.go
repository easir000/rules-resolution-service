package config

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseConfig struct {
    URL            string
    MaxConnections int
    MaxIdleTime    time.Duration
    MaxLifetime    time.Duration
}

func LoadDatabaseConfig() DatabaseConfig {
    return DatabaseConfig{
        URL:            getDBEnv("DATABASE_URL", "postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable"),
        MaxConnections: getDBInt("DB_MAX_CONNECTIONS", 25),
        MaxIdleTime:    getDBDuration("DB_MAX_IDLE_TIME", 5*time.Minute),
        MaxLifetime:    getDBDuration("DB_MAX_LIFETIME", 30*time.Minute),
    }
}

func NewPool(cfg DatabaseConfig) (*pgxpool.Pool, error) {
    poolConfig, err := pgxpool.ParseConfig(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("parse database URL: %w", err)
    }
    poolConfig.MaxConns = int32(cfg.MaxConnections)
    poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
    poolConfig.MaxConnLifetime = cfg.MaxLifetime
    return pgxpool.NewWithConfig(context.Background(), poolConfig)
}

// Use different names to avoid conflict with config.go
func getDBEnv(key, fallback string) string { return fallback }
func getDBInt(key string, fallback int) int { return fallback }
func getDBDuration(key string, fallback time.Duration) time.Duration { return fallback }
