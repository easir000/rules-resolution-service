package repository

import (
	"context"
	"time"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

type OverrideRepository interface {
	FindMatching(ctx context.Context, stepKey string, traitKey domain.TraitKey,
		caseCtx domain.CaseContext, asOfDate time.Time) ([]domain.Override, error)
	List(ctx context.Context, filters OverrideFilters) ([]domain.Override, error)
	GetByID(ctx context.Context, id string) (*domain.Override, error)
	Create(ctx context.Context, o *domain.Override) error
	Update(ctx context.Context, o *domain.Override) error
	UpdateStatus(ctx context.Context, id, status, updatedBy string) error
	Delete(ctx context.Context, id string) error
	FindConflicts(ctx context.Context) ([]domain.Conflict, error)
	GetHistory(ctx context.Context, overrideID string) ([]domain.AuditEntry, error)
}

type DefaultRepository interface {
	LoadAll(ctx context.Context) (map[string]map[domain.TraitKey][]byte, error)
	Get(ctx context.Context, stepKey string, traitKey domain.TraitKey) ([]byte, error)
}

type OverrideFilters struct {
	StepKey, TraitKey, State, Client, Investor, CaseType, Status string
	Limit, Offset int
}