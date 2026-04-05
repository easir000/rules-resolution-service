package service

import (
	"fmt"
	"time"

	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

// ConflictService detects conflicting override pairs
type ConflictService struct{}

func NewConflictService() *ConflictService {
	return &ConflictService{}
}

// FindConflicts identifies override pairs that conflict
func (s *ConflictService) FindConflicts(overrides []domain.Override) []domain.Conflict {
	var conflicts []domain.Conflict

	// Group overrides by step/trait for efficient comparison
	grouped := make(map[string][]domain.Override)
	for _, o := range overrides {
		if o.Status == "archived" {
			continue // Archived overrides don't conflict
		}
		key := fmt.Sprintf("%s:%s", o.StepKey, o.TraitKey)
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

// detectConflict returns a Conflict if two overrides conflict, nil otherwise
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
	reason := s.buildConflictReason(a, b)
	return &domain.Conflict{
		OverrideA: a.ID,
		OverrideB: b.ID,
		StepKey:   a.StepKey,
		TraitKey:  string(a.TraitKey),
		Reason:    reason,
	}
}

// dateRangesOverlap checks if two date ranges overlap
func (s *ConflictService) dateRangesOverlap(start1, end1, start2, end2 time.Time) bool {
	// Range 1: [start1, end1) or [start1, ∞) if end1 is nil
	// Range 2: [start2, end2) or [start2, ∞) if end2 is nil
	
	// No overlap if one ends before the other starts
	if end1 != nil && !end1.After(start2) {
		return false
	}
	if end2 != nil && !end2.After(start1) {
		return false
	}
	return true
}

// selectorsCompatible returns true if two selectors could match the same context
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

// buildConflictReason generates a human-readable conflict explanation
func (s *ConflictService) buildConflictReason(a, b *domain.Override) string {
	return fmt.Sprintf("Same step/trait, same specificity (%d), overlapping effective dates, compatible selectors {%s}",
		a.Specificity, formatSelector(&a.Selector))
}

// formatSelector helper (same as in resolution.go)
func formatSelector(s *domain.Selector) string {
	parts := make([]string, 0, 4)
	if s.State != nil {
		parts = append(parts, *s.State)
	}
	if s.Client != nil {
		parts = append(parts, *s.Client)
	}
	if s.Investor != nil {
		parts = append(parts, *s.Investor)
	}
	if s.CaseType != nil {
		parts = append(parts, *s.CaseType)
	}
	if len(parts) == 0 {
		return "default"
	}
	return join(parts, "+")
}

func join(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for _, s := range elems[1:] {
		result += sep + s
	}
	return result
}