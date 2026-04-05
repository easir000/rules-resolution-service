package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

// OverrideRepository handles persistence for override records
type OverrideRepository struct {
	db *pgx.Conn
}

func NewOverrideRepository(db *pgx.Conn) *OverrideRepository {
	return &OverrideRepository{db: db}
}

// FindMatching returns overrides that match the given criteria for resolution
func (r *OverrideRepository) FindMatching(ctx context.Context, stepKey string, 
	traitKey domain.TraitKey, caseCtx domain.CaseContext, asOfDate time.Time) ([]domain.Override, error) {

	query := `
		SELECT 
			id, step_key, trait_key, state, client, investor, case_type,
			specificity, value, effective_date, expires_date, status,
			description, created_by, created_at, updated_by, updated_at
		FROM overrides
		WHERE step_key = $1 
			AND trait_key = $2
			AND status = 'active'
			AND effective_date <= $3
			AND (expires_date IS NULL OR expires_date > $3)
			AND (state IS NULL OR state = $4)
			AND (client IS NULL OR client = $5)
			AND (investor IS NULL OR investor = $6)
			AND (case_type IS NULL OR case_type = $7)
		ORDER BY specificity DESC, effective_date DESC, id ASC
	`

	rows, err := r.db.Query(ctx, query, stepKey, traitKey, asOfDate,
		nullIfEmpty(caseCtx.State), nullIfEmpty(caseCtx.Client),
		nullIfEmpty(caseCtx.Investor), nullIfEmpty(caseCtx.CaseType))
	if err != nil {
		return nil, fmt.Errorf("query overrides: %w", err)
	}
	defer rows.Close()

	var overrides []domain.Override
	for rows.Next() {
		o, err := scanOverride(rows)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

// GetAll returns all overrides with optional filtering
func (r *OverrideRepository) GetAll(ctx context.Context, filters map[string]string) ([]domain.Override, error) {
	query := `
		SELECT 
			id, step_key, trait_key, state, client, investor, case_type,
			specificity, value, effective_date, expires_date, status,
			description, created_by, created_at, updated_by, updated_at
		FROM overrides
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	// Apply filters
	for field, value := range filters {
		if value != "" {
			query += fmt.Sprintf(" AND %s = $%d", field, argIdx)
			args = append(args, value)
			argIdx++
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query overrides: %w", err)
	}
	defer rows.Close()

	var overrides []domain.Override
	for rows.Next() {
		o, err := scanOverride(rows)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

// GetByID retrieves a single override by ID
func (r *OverrideRepository) GetByID(ctx context.Context, id string) (*domain.Override, error) {
	query := `
		SELECT 
			id, step_key, trait_key, state, client, investor, case_type,
			specificity, value, effective_date, expires_date, status,
			description, created_by, created_at, updated_by, updated_at
		FROM overrides WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	return scanOverride(row)
}

// Create inserts a new override
func (r *OverrideRepository) Create(ctx context.Context, o *domain.Override) error {
	query := `
		INSERT INTO overrides (
			id, step_key, trait_key, state, client, investor, case_type,
			value, effective_date, expires_date, status, description, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		o.ID, o.StepKey, o.TraitKey,
		o.Selector.State, o.Selector.Client, o.Selector.Investor, o.Selector.CaseType,
		o.Value, o.EffectiveDate, o.ExpiresDate, o.Status, o.Description, o.CreatedBy)
	return err
}

// Update modifies an existing override
func (r *OverrideRepository) Update(ctx context.Context, o *domain.Override) error {
	query := `
		UPDATE overrides SET
			step_key = $2, trait_key = $3, state = $4, client = $5,
			investor = $6, case_type = $7, value = $8, effective_date = $9,
			expires_date = $10, status = $11, description = $12,
			updated_by = $13, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		o.ID, o.StepKey, o.TraitKey,
		o.Selector.State, o.Selector.Client, o.Selector.Investor, o.Selector.CaseType,
		o.Value, o.EffectiveDate, o.ExpiresDate, o.Status, o.Description, o.UpdatedBy)
	return err
}

// UpdateStatus changes only the status field
func (r *OverrideRepository) UpdateStatus(ctx context.Context, id, status, updatedBy string) error {
	query := `UPDATE overrides SET status = $1, updated_by = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, updatedBy, id)
	return err
}

// GetHistory returns the audit trail for an override
func (r *OverrideRepository) GetHistory(ctx context.Context, overrideID string) ([]domain.AuditEntry, error) {
	query := `
		SELECT id, override_id, changed_at, changed_by, change_type,
		       before_state, after_state, summary
		FROM override_history
		WHERE override_id = $1
		ORDER BY changed_at DESC
	`
	rows, err := r.db.Query(ctx, query, overrideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		var entry domain.AuditEntry
		var before, after []byte
		err := rows.Scan(&entry.ID, &entry.OverrideID, &entry.ChangedAt, 
			&entry.ChangedBy, &entry.ChangeType, &before, &after, &entry.Summary)
		if err != nil {
			return nil, err
		}
		if before != nil {
			entry.Before = before
		}
		if after != nil {
			entry.After = after
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// Helper functions
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanOverride(row interface{ Scan(...interface{}) error }) (domain.Override, error) {
	var o domain.Override
	var state, client, investor, caseType sql.NullString
	var expiresDate sql.NullTime
	var updatedBy sql.NullString

	err := row.Scan(
		&o.ID, &o.StepKey, &o.TraitKey,
		&state, &client, &investor, &caseType,
		&o.Specificity, &o.Value, &o.EffectiveDate, &expiresDate,
		&o.Status, &o.Description, &o.CreatedBy, &o.CreatedAt,
		&updatedBy, &o.UpdatedAt,
	)
	if err != nil {
		return o, err
	}

	// Convert nullable fields
	if state.Valid {
		o.Selector.State = &state.String
	}
	if client.Valid {
		o.Selector.Client = &client.String
	}
	if investor.Valid {
		o.Selector.Investor = &investor.String
	}
	if caseType.Valid {
		o.Selector.CaseType = &caseType.String
	}
	if expiresDate.Valid {
		o.ExpiresDate = &expiresDate.Time
	}
	if updatedBy.Valid {
		o.UpdatedBy = &updatedBy.String
	}

	return o, nil
}