package database

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql seed_data.sql
var migrations embed.FS

// RunMigrations applies schema.sql to the database
func RunMigrations(pool *pgxpool.Pool) error {
	schema, err := migrations.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema.sql: %w", err)
	}
	if _, err := pool.Exec(context.Background(), string(schema)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	return nil
}

// SeedData loads initial data from seed_data.sql
func SeedData(pool *pgxpool.Pool) error {
	seed, err := migrations.ReadFile("seed_data.sql")
	if err != nil {
		return fmt.Errorf("read seed_data.sql: %w", err)
	}
	if _, err := pool.Exec(context.Background(), string(seed)); err != nil {
		return fmt.Errorf("execute seed  %w", err)
	}
	return nil
}

// SeedFromJSON loads data from JSON files (alternative to SQL seed)
func SeedFromJSON(pool *pgxpool.Pool, defaultsPath, overridesPath string) error {
	// Load defaults.json
	defaultsData, err := os.ReadFile(defaultsPath)
	if err != nil {
		return fmt.Errorf("read defaults.json: %w", err)
	}
	var defaults map[string]map[string]json.RawMessage
	if err := json.Unmarshal(defaultsData, &defaults); err != nil {
		return fmt.Errorf("parse defaults.json: %w", err)
	}

	// Insert defaults
	for stepKey, traits := range defaults {
		for traitKey, value := range traits {
			query := `
				INSERT INTO defaults (step_key, trait_key, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (step_key, trait_key) DO UPDATE SET value = EXCLUDED.value
			`
			if _, err := pool.Exec(context.Background(), query, stepKey, traitKey, value); err != nil {
				return fmt.Errorf("insert default %s.%s: %w", stepKey, traitKey, err)
			}
		}
	}

	// Load overrides.json
	overridesData, err := os.ReadFile(overridesPath)
	if err != nil {
		return fmt.Errorf("read overrides.json: %w", err)
	}
	var overrides []struct {
		ID            string             `json:"id"`
		StepKey       string             `json:"stepKey"`
		TraitKey      string             `json:"traitKey"`
		Selector      map[string]*string `json:"selector"`
		Value         json.RawMessage    `json:"value"`
		EffectiveDate string             `json:"effectiveDate"`
		ExpiresDate   *string            `json:"expiresDate"`
		Status        string             `json:"status"`
		Description   string             `json:"description"`
		CreatedBy     string             `json:"createdBy"`
	}
	if err := json.Unmarshal(overridesData, &overrides); err != nil {
		return fmt.Errorf("parse overrides.json: %w", err)
	}

	// Insert overrides
	for _, o := range overrides {
		effectiveDate, _ := time.Parse("2006-01-02", o.EffectiveDate)
		var expiresDate *time.Time
		if o.ExpiresDate != nil {
			t, _ := time.Parse("2006-01-02", *o.ExpiresDate)
			expiresDate = &t
		}

		query := `
			INSERT INTO overrides (
				id, step_key, trait_key, state, client, investor, case_type,
				value, effective_date, expires_date, status, description, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE SET
				value = EXCLUDED.value,
				effective_date = EXCLUDED.effective_date,
				expires_date = EXCLUDED.expires_date,
				status = EXCLUDED.status,
				description = EXCLUDED.description
		`
		sel := o.Selector
		if _, err := pool.Exec(context.Background(), query,
			o.ID, o.StepKey, o.TraitKey,
			sel["state"], sel["client"], sel["investor"], sel["caseType"],
			o.Value, effectiveDate, expiresDate, o.Status, o.Description, o.CreatedBy); err != nil {
			return fmt.Errorf("insert override %s: %w", o.ID, err)
		}
	}
	return nil
}