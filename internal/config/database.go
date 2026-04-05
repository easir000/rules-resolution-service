
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
		URL:            getEnv("DATABASE_URL", "postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable"),
		MaxConnections: getIntEnv("DB_MAX_CONNECTIONS", 25),
		MaxIdleTime:    getDurationEnv("DB_MAX_IDLE_TIME", 5*time.Minute),
		MaxLifetime:    getDurationEnv("DB_MAX_LIFETIME", 30*time.Minute),
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

// Use helpers from config.go - just placeholders here to avoid "undefined" errors
func getEnv(key, fallback string) string { return fallback }
func getIntEnv(key string, fallback int) int { return fallback }
func getDurationEnv(key string, fallback time.Duration) time.Duration { return fallback }
