package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
)

type ResolutionService struct {
	defaults       map[string]map[domain.TraitKey][]byte
	overrideRepo   repository.OverrideRepository
}

func NewResolutionService(
	defaults map[string]map[domain.TraitKey][]byte,
	overrideRepo repository.OverrideRepository,
) *ResolutionService {
	return &ResolutionService{
		defaults:     defaults,
		overrideRepo: overrideRepo,
	}
}

// Resolve applies the specificity cascade algorithm using PostgreSQL
func (s *ResolutionService) Resolve(ctx context.Context, caseCtx domain.CaseContext) (domain.ResolutionResponse, error) {
	asOfDate := time.Now()
	if caseCtx.AsOfDate != nil && *caseCtx.AsOfDate != "" {
		if parsed, err := time.Parse("2006-01-02", *caseCtx.AsOfDate); err == nil {
			asOfDate = parsed
		}
	}

	response := domain.ResolutionResponse{
		Context:    caseCtx,
		ResolvedAt: time.Now().UTC(),
		Steps:      make(map[string]domain.ResolvedStep),
	}

	// Resolve each step/trait independently
	for stepKey, traits := range s.defaults {
		response.Steps[stepKey] = make(domain.ResolvedStep)
		for traitKey, defaultValue := range traits {
			// Query PostgreSQL for matching overrides
			candidates, err := s.overrideRepo.FindMatching(ctx, stepKey, traitKey, caseCtx, asOfDate)
			if err != nil {
				return response, fmt.Errorf("find overrides for %s.%s: %w", stepKey, traitKey, err)
			}

			// Sort by specificity DESC, effectiveDate DESC, ID ASC (deterministic)
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].Specificity != candidates[j].Specificity {
					return candidates[i].Specificity > candidates[j].Specificity
				}
				if !candidates[i].EffectiveDate.Equal(candidates[j].EffectiveDate) {
					return candidates[i].EffectiveDate.After(candidates[j].EffectiveDate)
				}
				return candidates[i].ID < candidates[j].ID
			})

			if len(candidates) > 0 {
				winner := candidates[0]
				response.Steps[stepKey][traitKey] = domain.ResolvedTrait{
					Value:       winner.Value,
					Source:      "override",
					OverrideID:  &winner.ID,
					Explanation: buildExplanation(winner),
				}
			} else {
				response.Steps[stepKey][traitKey] = domain.ResolvedTrait{
					Value:       defaultValue,
					Source:      "default",
					Explanation: "No matching override — using default value",
				}
			}
		}
	}

	return response, nil
}

// Explain returns detailed resolution trace for a single step/trait
func (s *ResolutionService) Explain(ctx context.Context, stepKey string, traitKey domain.TraitKey,
	caseCtx domain.CaseContext) (domain.ExplainResponse, error) {

	asOfDate := time.Now()
	if caseCtx.AsOfDate != nil && *caseCtx.AsOfDate != "" {
		if parsed, err := time.Parse("2006-01-02", *caseCtx.AsOfDate); err == nil {
			asOfDate = parsed
		}
	}

	// Get default value
	defaultValue := s.defaults[stepKey][traitKey]

	// Find ALL candidates (including filtered ones for full trace)
	candidates, err := s.overrideRepo.FindMatching(ctx, stepKey, traitKey, caseCtx, asOfDate)
	if err != nil {
		return domain.ExplainResponse{}, err
	}

	// Sort for selection
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Specificity != candidates[j].Specificity {
			return candidates[i].Specificity > candidates[j].Specificity
		}
		if !candidates[i].EffectiveDate.Equal(candidates[j].EffectiveDate) {
			return candidates[i].EffectiveDate.After(candidates[j].EffectiveDate)
		}
		return candidates[i].ID < candidates[j].ID
	})

	response := domain.ExplainResponse{
		Step:       stepKey,
		Trait:      traitKey,
		Candidates: make([]domain.ExplainCandidate, 0, len(candidates)),
	}

	if len(candidates) > 0 {
		winner := candidates[0]
		response.ResolvedValue = winner.Value
		response.ResolvedFrom = &domain.ExplainSource{
			OverrideID:    winner.ID,
			Selector:      winner.Selector,
			Specificity:   winner.Specificity,
			EffectiveDate: winner.EffectiveDate,
		}

		// Build candidate traces with outcomes
		for i, c := range candidates {
			outcome := "SHADOWED"
			if i == 0 {
				outcome = fmt.Sprintf("SELECTED — highest specificity (%d)", c.Specificity)
			} else if c.Specificity < candidates[0].Specificity {
				outcome = fmt.Sprintf("SHADOWED — lower specificity (%d < %d)", 
					c.Specificity, candidates[0].Specificity)
			} else if c.EffectiveDate.Before(candidates[0].EffectiveDate) {
				outcome = fmt.Sprintf("SHADOWED — earlier effective date (%s < %s)",
					c.EffectiveDate.Format("2006-01-02"), 
					candidates[0].EffectiveDate.Format("2006-01-02"))
			}
			response.Candidates = append(response.Candidates, domain.ExplainCandidate{
				OverrideID:    c.ID,
				Selector:      c.Selector,
				Specificity:   c.Specificity,
				EffectiveDate: c.EffectiveDate,
				Value:         c.Value,
				Outcome:       outcome,
			})
		}
	} else {
		// No overrides matched — return default with explanation
		response.ResolvedValue = defaultValue
		response.Candidates = []domain.ExplainCandidate{{
			Outcome: "NO_MATCH — fell back to default",
		}}
	}

	return response, nil
}

func buildExplanation(o domain.Override) string {
	parts := []string{}
	if o.Selector.State != nil {
		parts = append(parts, *o.Selector.State)
	}
	if o.Selector.Client != nil {
		parts = append(parts, *o.Selector.Client)
	}
	if o.Selector.Investor != nil {
		parts = append(parts, *o.Selector.Investor)
	}
	if o.Selector.CaseType != nil {
		parts = append(parts, *o.Selector.CaseType)
	}
	if len(parts) == 0 {
		return "Default value"
	}
	return fmt.Sprintf("%s override (specificity %d)", strings.Join(parts, "+"), o.Specificity)
}
