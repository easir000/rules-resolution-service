package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Dimension represents the four configuration axes
type Dimension string

const (
	DimState     Dimension = "state"
	DimClient    Dimension = "client"
	DimInvestor  Dimension = "investor"
	DimCaseType  Dimension = "caseType"
)

// Step represents a canonical workflow step
type Step struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TraitKey represents configurable attributes of a step
type TraitKey string

const (
	TraitSLAHours          TraitKey = "slaHours"
	TraitRequiredDocuments TraitKey = "requiredDocuments"
	TraitFeeAmount         TraitKey = "feeAmount"
	TraitFeeAuthRequired   TraitKey = "feeAuthRequired"
	TraitAssignedRole      TraitKey = "assignedRole"
	TraitTemplateID        TraitKey = "templateId"
)

// Selector defines which dimensions an override pins (empty = wildcard)
type Selector struct {
    State    *string `json:"state,omitempty"`
    Client   *string `json:"client,omitempty"`
    Investor *string `json:"investor,omitempty"`
    CaseType *string `json:"caseType,omitempty"`
}

// Specificity returns the count of pinned dimensions (0-4)
func (s Selector) Specificity() int {
	score := 0
	if s.State != nil {
		score++
	}
	if s.Client != nil {
		score++
	}
	if s.Investor != nil {
		score++
	}
	if s.CaseType != nil {
		score++
	}
	return score
}

// Matches returns true if this selector matches the given context
func (s Selector) Matches(ctx CaseContext) bool {
	if s.State != nil && *s.State != ctx.State {
		return false
	}
	if s.Client != nil && *s.Client != ctx.Client {
		return false
	}
	if s.Investor != nil && *s.Investor != ctx.Investor {
		return false
	}
	if s.CaseType != nil && *s.CaseType != ctx.CaseType {
		return false
	}
	return true
}

// CaseContext represents the input dimensions for resolution
type CaseContext struct {
	State    string  `json:"state"`
	Client   string  `json:"client"`
	Investor string  `json:"investor"`
	CaseType string  `json:"caseType"`
	AsOfDate *string `json:"asOfDate,omitempty"` // Optional date for temporal resolution
}

// Override represents a rule that can override default values
type Override struct {
    ID            string          `json:"id"`
    StepKey       string          `json:"stepKey"`
    TraitKey      TraitKey        `json:"traitKey"`  // TraitKey is type alias for string
    Selector      Selector        `json:"selector"`
    Value         json.RawMessage `json:"value"`     // Can be any JSON value
    Specificity   int             `json:"specificity"`
    EffectiveDate time.Time       `json:"effectiveDate"`
    ExpiresDate   *time.Time      `json:"expiresDate,omitempty"`
    Status        string          `json:"status"`
    Description   string          `json:"description,omitempty"`
    CreatedBy     string          `json:"createdBy"`
    CreatedAt     time.Time       `json:"createdAt"`
    UpdatedBy     *string         `json:"updatedBy,omitempty"`
    UpdatedAt     time.Time       `json:"updatedAt"`
}

// IsEffectiveAt returns true if the override is active at the given date
func (o Override) IsEffectiveAt(date time.Time) bool {
	if o.Status != "active" {
		return false
	}
	if o.EffectiveDate.After(date) {
		return false
	}
	if o.ExpiresDate != nil && !o.ExpiresDate.After(date) {
		return false
	}
	return true
}

// DefaultValue represents the baseline configuration for a step/trait
type DefaultValue struct {
	ID        uuid.UUID       `json:"-"`
	StepKey   string          `json:"stepKey"`
	TraitKey  TraitKey        `json:"traitKey"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// ResolvedTrait represents the final resolved value for a trait
type ResolvedTrait struct {
	Value      json.RawMessage `json:"value"`
	Source     string          `json:"source"` // "default" or "override"
	OverrideID *string         `json:"overrideId,omitempty"`
	Explanation string         `json:"explanation,omitempty"`
}

// ResolvedStep represents all resolved traits for a single step
type ResolvedStep map[TraitKey]ResolvedTrait

// ResolutionResponse is the API response for /api/resolve
type ResolutionResponse struct {
	Context    CaseContext              `json:"context"`
	ResolvedAt time.Time                `json:"resolvedAt"`
	Steps      map[string]ResolvedStep  `json:"steps"`
}

// ExplainCandidate represents a candidate override in an explain trace
type ExplainCandidate struct {
	OverrideID    string          `json:"overrideId"`
	Selector      Selector        `json:"selector"`
	Specificity   int             `json:"specificity"`
	EffectiveDate time.Time       `json:"effectiveDate"`
	Value         json.RawMessage `json:"value"`
	Outcome       string          `json:"outcome"` // SELECTED, SHADOWED, FILTERED
}

// ExplainResponse is the API response for /api/resolve/explain
type ExplainResponse struct {
	Step          string           `json:"step"`
	Trait         TraitKey         `json:"trait"`
	ResolvedValue json.RawMessage  `json:"resolvedValue"`
	ResolvedFrom  *ExplainSource   `json:"resolvedFrom,omitempty"`
	Candidates    []ExplainCandidate `json:"candidates"`
}

// ExplainSource details why a particular override was selected
type ExplainSource struct {
	OverrideID    string    `json:"overrideId"`
	Selector      Selector  `json:"selector"`
	Specificity   int       `json:"specificity"`
	EffectiveDate time.Time `json:"effectiveDate"`
}

// Conflict represents two overrides that conflict
type Conflict struct {
	OverrideA string `json:"overrideA"`
	OverrideB string `json:"overrideB"`
	StepKey   string `json:"stepKey"`
	TraitKey  string `json:"traitKey"`
	Reason    string `json:"reason"`
}

// ConflictResponse is the API response for /api/overrides/conflicts
type ConflictResponse struct {
	Conflicts []Conflict `json:"conflicts"`
}

// AuditEntry represents a change in the override history
type AuditEntry struct {
	ID         uuid.UUID    `json:"id"`
	OverrideID string       `json:"overrideId"`
	ChangedAt  time.Time    `json:"changedAt"`
	ChangedBy  string       `json:"changedBy"`
	ChangeType string       `json:"changeType"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	Summary    string       `json:"summary"`
}
