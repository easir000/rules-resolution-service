
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

type pgOverrideRepository struct {
	pool *pgxpool.Pool
}

func NewOverrideRepository(pool *pgxpool.Pool) OverrideRepository {
	return &pgOverrideRepository{pool: pool}
}

func (r *pgOverrideRepository) FindMatching(ctx context.Context, stepKey string,
	traitKey domain.TraitKey, caseCtx domain.CaseContext, asOfDate time.Time) ([]domain.Override, error) {

	query := `SELECT id, step_key, trait_key, state, client, investor, case_type,
		specificity, value, effective_date, expires_date, status, description,
		created_by, created_at, updated_by, updated_at
		FROM overrides WHERE step_key = $1 AND trait_key = $2 AND status = 'active'
		AND effective_date <= $3 AND (expires_date IS NULL OR expires_date > $3)
		AND (state IS NULL OR state = $4) AND (client IS NULL OR client = $5)
		AND (investor IS NULL OR investor = $6) AND (case_type IS NULL OR case_type = $7)
		ORDER BY specificity DESC, effective_date DESC, id ASC`

	rows, err := r.pool.Query(ctx, query, stepKey, traitKey, asOfDate,
		nullIfEmpty(caseCtx.State), nullIfEmpty(caseCtx.Client),
		nullIfEmpty(caseCtx.Investor), nullIfEmpty(caseCtx.CaseType))
	if err != nil {
		return nil, fmt.Errorf("query matching overrides: %w", err)
	}
	defer rows.Close()
	return scanOverrides(rows)
}

func (r *pgOverrideRepository) List(ctx context.Context, filters OverrideFilters) ([]domain.Override, error) {
	query := `SELECT id, step_key, trait_key, state, client, investor, case_type,
		specificity, value, effective_date, expires_date, status, description,
		created_by, created_at, updated_by, updated_at FROM overrides WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filters.StepKey != "" {
		query += fmt.Sprintf(" AND step_key = $%d", argIdx); args = append(args, filters.StepKey); argIdx++
	}
	if filters.TraitKey != "" {
		query += fmt.Sprintf(" AND trait_key = $%d", argIdx); args = append(args, filters.TraitKey); argIdx++
	}
	if filters.State != "" {
		query += fmt.Sprintf(" AND (state = $%d OR state IS NULL)", argIdx); args = append(args, filters.State); argIdx++
	}
	if filters.Client != "" {
		query += fmt.Sprintf(" AND (client = $%d OR client IS NULL)", argIdx); args = append(args, filters.Client); argIdx++
	}
	if filters.Investor != "" {
		query += fmt.Sprintf(" AND (investor = $%d OR investor IS NULL)", argIdx); args = append(args, filters.Investor); argIdx++
	}
	if filters.CaseType != "" {
		query += fmt.Sprintf(" AND (case_type = $%d OR case_type IS NULL)", argIdx); args = append(args, filters.CaseType); argIdx++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx); args = append(args, filters.Status); argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list overrides: %w", err)
	}
	defer rows.Close()
	return scanOverrides(rows)
}

func (r *pgOverrideRepository) GetByID(ctx context.Context, id string) (*domain.Override, error) {
	query := `SELECT id, step_key, trait_key, state, client, investor, case_type,
		specificity, value, effective_date, expires_date, status, description,
		created_by, created_at, updated_by, updated_at FROM overrides WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	o, err := scanOverrideRow(row)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgOverrideRepository) Create(ctx context.Context, o *domain.Override) error {
	query := `INSERT INTO overrides (id, step_key, trait_key, state, client, investor, case_type,
		value, effective_date, expires_date, status, description, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.pool.Exec(ctx, query, o.ID, o.StepKey, o.TraitKey,
		o.Selector.State, o.Selector.Client, o.Selector.Investor, o.Selector.CaseType,
		o.Value, o.EffectiveDate, o.ExpiresDate, o.Status, o.Description, o.CreatedBy)
	return err
}

func (r *pgOverrideRepository) Update(ctx context.Context, o *domain.Override) error {
	query := `UPDATE overrides SET step_key=$2, trait_key=$3, state=$4, client=$5,
		investor=$6, case_type=$7, value=$8, effective_date=$9, expires_date=$10,
		status=$11, description=$12, updated_by=$13, updated_at=NOW() WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, o.ID, o.StepKey, o.TraitKey,
		o.Selector.State, o.Selector.Client, o.Selector.Investor, o.Selector.CaseType,
		o.Value, o.EffectiveDate, o.ExpiresDate, o.Status, o.Description, o.UpdatedBy)
	return err
}

func (r *pgOverrideRepository) UpdateStatus(ctx context.Context, id, status, updatedBy string) error {
	_, err := r.pool.Exec(ctx, `UPDATE overrides SET status=$1, updated_by=$2, updated_at=NOW() WHERE id=$3`,
		status, updatedBy, id)
	return err
}

func (r *pgOverrideRepository) Delete(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, "archived", "system")
}

func (r *pgOverrideRepository) FindConflicts(ctx context.Context) ([]domain.Conflict, error) {
	query := `WITH pairs AS (
		SELECT a.id as id_a, b.id as id_b, a.step_key, a.trait_key, a.specificity,
			a.effective_date as eff_a, a.expires_date as exp_a,
			b.effective_date as eff_b, b.expires_date as exp_b
		FROM overrides a JOIN overrides b ON a.step_key=b.step_key AND a.trait_key=b.trait_key
			AND a.specificity=b.specificity AND a.id < b.id
		WHERE a.status IN ('active','draft') AND b.status IN ('active','draft')
			AND (a.expires_date IS NULL OR a.expires_date > b.effective_date)
			AND (b.expires_date IS NULL OR b.expires_date > a.effective_date)
			AND (a.state IS NULL OR b.state IS NULL OR a.state=b.state)
			AND (a.client IS NULL OR b.client IS NULL OR a.client=b.client)
			AND (a.investor IS NULL OR b.investor IS NULL OR a.investor=b.investor)
			AND (a.case_type IS NULL OR b.case_type IS NULL OR a.case_type=b.case_type)
	)
	SELECT id_a, id_b, step_key, trait_key,
		format('Same step/trait, specificity %d, overlapping dates', specificity) as reason
		FROM pairs ORDER BY step_key, trait_key, id_a`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find conflicts: %w", err)
	}
	defer rows.Close()

	var conflicts []domain.Conflict
	for rows.Next() {
		var c domain.Conflict
		if err := rows.Scan(&c.OverrideA, &c.OverrideB, &c.StepKey, &c.TraitKey, &c.Reason); err != nil {
			return nil, err
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, rows.Err()
}

func (r *pgOverrideRepository) GetHistory(ctx context.Context, overrideID string) ([]domain.AuditEntry, error) {
	query := `SELECT id, override_id, changed_at, changed_by, change_type,
		before_state, after_state, summary FROM override_history
		WHERE override_id = $1 ORDER BY changed_at DESC`
	rows, err := r.pool.Query(ctx, query, overrideID)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		var before, after []byte
		if err := rows.Scan(&e.ID, &e.OverrideID, &e.ChangedAt, &e.ChangedBy,
			&e.ChangeType, &before, &after, &e.Summary); err != nil {
			return nil, err
		}
		if before != nil {
			e.Before = before
		}
		if after != nil {
			e.After = after
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Helpers
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanOverrides(rows pgx.Rows) ([]domain.Override, error) {
	var overrides []domain.Override
	for rows.Next() {
		o, err := scanOverrideRow(rows)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

func scanOverrideRow(row interface{ Scan(...interface{}) error }) (domain.Override, error) {
	var o domain.Override
	var state, client, investor, caseType sql.NullString
	var expiresDate, updatedAt sql.NullTime
	var updatedBy sql.NullString

	err := row.Scan(&o.ID, &o.StepKey, &o.TraitKey, &state, &client, &investor, &caseType,
		&o.Specificity, &o.Value, &o.EffectiveDate, &expiresDate, &o.Status, &o.Description,
		&o.CreatedBy, &o.CreatedAt, &updatedBy, &updatedAt)
	if err != nil {
		return o, err
	}

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

