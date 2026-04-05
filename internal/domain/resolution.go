package domain

import (
    "encoding/json"
    "fmt"
    "sort"
    "strings"
    "time"
)

// Resolve applies the specificity cascade algorithm
func Resolve(ctx CaseContext, defaults map[string]map[TraitKey]json.RawMessage,
    overrides []Override) ResolutionResponse {

    asOfDate := time.Now()
    if ctx.AsOfDate != nil && *ctx.AsOfDate != "" {
        if parsed, err := time.Parse("2006-01-02", *ctx.AsOfDate); err == nil {
            asOfDate = parsed
        }
    }

    stepKeys := make([]string, 0, len(defaults))
    for stepKey := range defaults {
        stepKeys = append(stepKeys, stepKey)
    }
    sort.Strings(stepKeys)

    response := ResolutionResponse{
        Context:    ctx,
        ResolvedAt: time.Now().UTC(),
        Steps:      make(map[string]ResolvedStep, len(stepKeys)),
    }

    for _, stepKey := range stepKeys {
        stepDefaults := defaults[stepKey]
        response.Steps[stepKey] = make(ResolvedStep)

        for traitKey, defaultValue := range stepDefaults {
            candidates := filterMatchingOverrides(overrides, stepKey, traitKey, ctx, asOfDate)

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
                response.Steps[stepKey][traitKey] = ResolvedTrait{
                    Value:       winner.Value,
                    Source:      "override",
                    OverrideID:  &winner.ID,
                    Explanation: buildExplanation(winner),
                }
            } else {
                response.Steps[stepKey][traitKey] = ResolvedTrait{
                    Value:       defaultValue,
                    Source:      "default",
                    Explanation: "No matching override — using default value",
                }
            }
        }
    }

    return response
}

func filterMatchingOverrides(overrides []Override, stepKey string, traitKey TraitKey,
    ctx CaseContext, asOfDate time.Time) []Override {

    var matches []Override
    for _, o := range overrides {
        if o.StepKey != stepKey || o.TraitKey != traitKey {
            continue
        }
        if !o.IsEffectiveAt(asOfDate) {
            continue
        }
        if o.Selector.Matches(ctx) {
            matches = append(matches, o)
        }
    }
    return matches
}

func buildExplanation(o Override) string {
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
