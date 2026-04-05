package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

// ExplainService provides detailed resolution traces for debugging
type ExplainService struct {
	defaults map[string]map[domain.TraitKey]json.RawMessage
}

func NewExplainService(defaults map[string]map[domain.TraitKey]json.RawMessage) *ExplainService {
	return &ExplainService{defaults: defaults}
}

// Explain returns a detailed trace for a single step/trait resolution
func (s *ExplainService) Explain(stepKey string, traitKey domain.TraitKey,
	ctx domain.CaseContext, overrides []domain.Override) domain.ExplainResponse {

	asOfDate := time.Now()
	if ctx.AsOfDate != nil && *ctx.AsOfDate != "" {
		if parsed, err := time.Parse("2006-01-02", *ctx.AsOfDate); err == nil {
			asOfDate = parsed
		}
	}

	// Get default value
	defaultValue := s.defaults[stepKey][traitKey]

	// Find ALL candidates (including filtered ones for full trace)
	allCandidates := s.findAllCandidates(overrides, stepKey, traitKey, ctx, asOfDate)

	// Sort for selection
	sort.Slice(allCandidates, func(i, j int) bool {
		if allCandidates[i].Specificity != allCandidates[j].Specificity {
			return allCandidates[i].Specificity > allCandidates[j].Specificity
		}
		if !allCandidates[i].EffectiveDate.Equal(allCandidates[j].EffectiveDate) {
			return allCandidates[i].EffectiveDate.After(allCandidates[j].EffectiveDate)
		}
		return allCandidates[i].ID < allCandidates[j].ID
	})

	response := domain.ExplainResponse{
		Step:       stepKey,
		Trait:      traitKey,
		Candidates: make([]domain.ExplainCandidate, 0, len(allCandidates)),
	}

	if len(allCandidates) > 0 {
		winner := allCandidates[0]
		response.ResolvedValue = winner.Value
		response.ResolvedFrom = &domain.ExplainSource{
			OverrideID:    winner.ID,
			Selector:      winner.Selector,
			Specificity:   winner.Specificity,
			EffectiveDate: winner.EffectiveDate,
		}

		// Build candidate traces with outcomes
		for i, c := range allCandidates {
			outcome := "SHADOWED"
			if i == 0 {
				outcome = fmt.Sprintf("SELECTED — highest specificity (%d)", c.Specificity)
			} else if c.Specificity < allCandidates[0].Specificity {
				outcome = fmt.Sprintf("SHADOWED — lower specificity (%d < %d)", 
					c.Specificity, allCandidates[0].Specificity)
			} else if c.EffectiveDate.Before(allCandidates[0].EffectiveDate) {
				outcome = fmt.Sprintf("SHADOWED — earlier effective date (%s < %s)",
					c.EffectiveDate.Format("2006-01-02"), 
					allCandidates[0].EffectiveDate.Format("2006-01-02"))
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

	return response
}

// findAllCandidates returns all overrides that could potentially match
// (used for explain traces, even if filtered by date/status)
func (s *ExplainService) findAllCandidates(overrides []domain.Override,
	stepKey string, traitKey domain.TraitKey, ctx domain.CaseContext, 
	asOfDate time.Time) []domain.Override {

	var candidates []domain.Override
	for _, o := range overrides {
		if o.StepKey != stepKey || o.TraitKey != traitKey {
			continue
		}
		if !o.Selector.Matches(ctx) {
			continue
		}
		// Include all for trace, but mark if filtered
		candidates = append(candidates, o)
	}
	return candidates
}