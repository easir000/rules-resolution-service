package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

type pgDefaultRepository struct {
	pool *pgxpool.Pool
}

func NewDefaultRepository(pool *pgxpool.Pool) DefaultRepository {
	return &pgDefaultRepository{pool: pool}
}

// LoadAll returns all defaults as a nested map: stepKey -> traitKey -> JSON value
func (r *pgDefaultRepository) LoadAll(ctx context.Context) (map[string]map[domain.TraitKey][]byte, error) {
	query := `SELECT step_key, trait_key, value FROM defaults`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[domain.TraitKey][]byte)
	for rows.Next() {
		var stepKey string
		var traitKey domain.TraitKey
		var value []byte
		if err := rows.Scan(&stepKey, &traitKey, &value); err != nil {
			return nil, err
		}
		if result[stepKey] == nil {
			result[stepKey] = make(map[domain.TraitKey][]byte)
		}
		result[stepKey][traitKey] = value
	}
	return result, rows.Err()
}

// Get returns a single default value
func (r *pgDefaultRepository) Get(ctx context.Context, stepKey string, traitKey domain.TraitKey) ([]byte, error) {
	query := `SELECT value FROM defaults WHERE step_key = $1 AND trait_key = $2`
	var value []byte
	err := r.pool.QueryRow(ctx, query, stepKey, traitKey).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("get default: %w", err)
	}
	return value, nil
}