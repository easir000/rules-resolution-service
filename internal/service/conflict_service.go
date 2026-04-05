$content = @"
package service

import (
	"context"
	"time"

	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
)

type ConflictService struct {
	repo repository.OverrideRepository
}

func NewConflictService(repo repository.OverrideRepository) *ConflictService {
	return &ConflictService{repo: repo}
}

func (s *ConflictService) FindConflicts(ctx context.Context) ([]domain.Conflict, error) {
	return s.repo.FindConflicts(ctx)
}

func (s *ConflictService) WouldConflict(ctx context.Context, proposed domain.Override) ([]domain.Conflict, error) {
	all, err := s.repo.List(ctx, repository.OverrideFilters{
		StepKey: proposed.StepKey,
		TraitKey: string(proposed.TraitKey),
	})
	if err != nil {
		return nil, err
	}

	var conflicts []domain.Conflict
	for _, existing := range all {
		if existing.ID == proposed.ID { continue }
		if existing.Specificity != proposed.Specificity { continue }
		if existing.Status == "archived" { continue }

		if !dateRangesOverlap(existing.EffectiveDate, existing.ExpiresDate,
			proposed.EffectiveDate, proposed.ExpiresDate) {
			continue
		}
		if !selectorsCompatible(&existing.Selector, &proposed.Selector) {
			continue
		}
		conflicts = append(conflicts, domain.Conflict{
			OverrideA: existing.ID,
			OverrideB: proposed.ID,
			StepKey: proposed.StepKey,
			TraitKey: string(proposed.TraitKey),
			Reason: "Proposed override conflicts with existing rule",
		})
	}
	return conflicts, nil
}

func dateRangesOverlap(start1, end1, start2, end2 time.Time) bool {
	if !end1.IsZero() && !end1.After(start2) { return false }
	if !end2.IsZero() && !end2.After(start1) { return false }
	return true
}

func selectorsCompatible(a, b *domain.Selector) bool {
	if a.State != nil && b.State != nil && *a.State != *b.State { return false }
	if a.Client != nil && b.Client != nil && *a.Client != *b.Client { return false }
	if a.Investor != nil && b.Investor != nil && *a.Investor != *b.Investor { return false }
	if a.CaseType != nil && b.CaseType != nil && *a.CaseType != *b.CaseType { return false }
	return true
}
"@
Set-Content -Path "internal/service/conflict_service.go" -Value $content -Encoding utf8