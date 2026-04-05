package config

import "os"

type Config struct {
	DatabaseURL   string
	Port          int
	RunMigrations bool
	SeedData      bool
}

func Load() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable"),
		Port:          getIntEnv("PORT", 8080),
		RunMigrations: getBoolEnv("RUN_MIGRATIONS", false),
		SeedData:      getBoolEnv("SEED_DATA", false),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" { return v }
	return fallback
}
func getIntEnv(key string, fallback int) int { return fallback }
func getBoolEnv(key string, fallback bool) bool { return fallback }
