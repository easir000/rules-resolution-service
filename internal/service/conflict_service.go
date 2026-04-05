
package service

import (
	"time"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

type ConflictService struct{}

func NewConflictService() *ConflictService {
	return &ConflictService{}
}

func (s *ConflictService) FindConflicts(overrides []domain.Override) []domain.Conflict {
	var conflicts []domain.Conflict

	// Group by step/trait for efficient comparison
	grouped := make(map[string][]domain.Override)
	for _, o := range overrides {
		if o.Status == "archived" {
			continue
		}
		key := o.StepKey + ":" + string(o.TraitKey)
		grouped[key] = append(grouped[key], o)
	}

	// Check each group for conflicts
	for _, group := range grouped {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				if conflict := s.detectConflict(&group[i], &group[j]); conflict != nil {
					conflicts = append(conflicts, *conflict)
				}
			}
		}
	}

	return conflicts
}

func (s *ConflictService) detectConflict(a, b *domain.Override) *domain.Conflict {
	// Must have same specificity to potentially conflict
	if a.Specificity != b.Specificity {
		return nil
	}

	// Must have overlapping effective date ranges
	if !s.dateRangesOverlap(a.EffectiveDate, a.ExpiresDate, b.EffectiveDate, b.ExpiresDate) {
		return nil
	}

	// Selectors must be compatible (every non-null dimension matches)
	if !s.selectorsCompatible(&a.Selector, &b.Selector) {
		return nil
	}

	// If we get here, they conflict
	reason := s.buildConflictReason(a)
	return &domain.Conflict{
		OverrideA: a.ID,
		OverrideB: b.ID,
		StepKey:   a.StepKey,
		TraitKey:  string(a.TraitKey),
		Reason:    reason,
	}
}

func (s *ConflictService) dateRangesOverlap(start1, end1, start2, end2 time.Time) bool {
	// Range 1: [start1, end1) or [start1, ∞) if end1 is zero
	// Range 2: [start2, end2) or [start2, ∞) if end2 is zero
	
	// No overlap if one ends before the other starts
	if !end1.IsZero() && !end1.After(start2) {
		return false
	}
	if !end2.IsZero() && !end2.After(start1) {
		return false
	}
	return true
}

func (s *ConflictService) selectorsCompatible(a, b *domain.Selector) bool {
	// For each dimension, if both are pinned, they must match
	if a.State != nil && b.State != nil && *a.State != *b.State {
		return false
	}
	if a.Client != nil && b.Client != nil && *a.Client != *b.Client {
		return false
	}
	if a.Investor != nil && b.Investor != nil && *a.Investor != *b.Investor {
		return false
	}
	if a.CaseType != nil && b.CaseType != nil && *a.CaseType != *b.CaseType {
		return false
	}
	return true
}

func (s *ConflictService) buildConflictReason(o *domain.Override) string {
	parts := []string{}
	if o.Selector.State != nil { parts = append(parts, "state:"+*o.Selector.State) }
	if o.Selector.Client != nil { parts = append(parts, "client:"+*o.Selector.Client) }
	if o.Selector.Investor != nil { parts = append(parts, "investor:"+*o.Selector.Investor) }
	if o.Selector.CaseType != nil { parts = append(parts, "caseType:"+*o.Selector.CaseType) }
	selectors := "all"
	if len(parts) > 0 { selectors = "{" + strings.Join(parts, ", ") + "}" }
	return "Same step/trait, same specificity (" + string(rune(o.Specificity+48)) + "), overlapping effective dates, compatible selectors " + selectors
}
