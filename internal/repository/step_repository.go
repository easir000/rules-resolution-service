package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
)

type StepRepository struct {
	db *pgxpool.Pool
}

func NewStepRepository(db *pgxpool.Pool) *StepRepository {
	return &StepRepository{db: db}
}

func (r *StepRepository) LoadDefaults(ctx context.Context) (map[string]map[domain.TraitKey]json.RawMessage, error) {
	// Return hardcoded defaults matching defaults.json
	return map[string]map[domain.TraitKey]json.RawMessage{
		"title-search": {
			"slaHours":          json.RawMessage("720"),
			"requiredDocuments": json.RawMessage(`["title_commitment","tax_certificate"]`),
			"feeAmount":         json.RawMessage("35000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"processor"`),
			"templateId":        json.RawMessage(`"title-review-standard-v1"`),
		},
		"file-complaint": {
			"slaHours":          json.RawMessage("480"),
			"requiredDocuments": json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet"]`),
			"feeAmount":         json.RawMessage("65000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"attorney"`),
			"templateId":        json.RawMessage(`"complaint-standard-v1"`),
		},
		"serve-borrower": {
			"slaHours":          json.RawMessage("2880"),
			"requiredDocuments": json.RawMessage(`["affidavit_of_service","return_of_service"]`),
			"feeAmount":         json.RawMessage("25000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"processor"`),
			"templateId":        json.RawMessage(`"service-standard-v1"`),
		},
		"obtain-judgment": {
			"slaHours":          json.RawMessage("4320"),
			"requiredDocuments": json.RawMessage(`["motion_for_judgment","affidavit_of_indebtedness","proposed_judgment"]`),
			"feeAmount":         json.RawMessage("45000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"attorney"`),
			"templateId":        json.RawMessage(`"judgment-standard-v1"`),
		},
		"schedule-sale": {
			"slaHours":          json.RawMessage("1440"),
			"requiredDocuments": json.RawMessage(`["notice_of_sale","publication_proof"]`),
			"feeAmount":         json.RawMessage("30000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"processor"`),
			"templateId":        json.RawMessage(`"sale-notice-standard-v1"`),
		},
		"conduct-sale": {
			"slaHours":          json.RawMessage("720"),
			"requiredDocuments": json.RawMessage(`["certificate_of_sale","sale_report"]`),
			"feeAmount":         json.RawMessage("50000"),
			"feeAuthRequired":   json.RawMessage("false"),
			"assignedRole":      json.RawMessage(`"attorney"`),
			"templateId":        json.RawMessage(`"sale-report-standard-v1"`),
		},
	}, nil
}