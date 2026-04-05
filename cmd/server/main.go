
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pearsonspecter/rules-resolution/internal/config"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/handler"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
	"github.com/pearsonspecter/rules-resolution/internal/service"
)

func main() {
	cfg := config.Load()

	// In-memory defaults (replace with DB in production)
	defaults := loadDefaults()
	overrides := loadTestOverrides()

	// Initialize services
	resolutionSvc := service.NewResolutionService(defaults)
	explainSvc := service.NewExplainService(defaults)
	conflictSvc := service.NewConflictService()

	// Initialize handlers
	resolveHandler := handler.NewResolveHandler(resolutionSvc, explainSvc)
	overrideHandler := handler.NewOverrideHandler(overrides, conflictSvc)
	overrideHandler := handler.NewOverrideHandler(overrides, conflictSvc)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// API routes
	r.Route("/api", func(r chi.Router) {
		resolveHandler.RegisterRoutes(r)
	overrideHandler.RegisterRoutes(r)
		overrideHandler.RegisterRoutes(r)
		
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{\"status\":\"ok\"}"))
		})
	})

	// Start server
	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":8080"
	}
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func loadDefaults() map[string]map[domain.TraitKey][]byte {
	return map[string]map[domain.TraitKey][]byte{
		"title-search": {
			"slaHours": []byte("720"),
			"requiredDocuments": []byte("[\"title_commitment\",\"tax_certificate\"]"),
			"feeAmount": []byte("35000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"processor\""),
			"templateId": []byte("\"title-review-standard-v1\""),
		},
		"file-complaint": {
			"slaHours": []byte("480"),
			"requiredDocuments": []byte("[\"complaint\",\"summons\",\"lis_pendens\",\"cover_sheet\"]"),
			"feeAmount": []byte("65000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"attorney\""),
			"templateId": []byte("\"complaint-standard-v1\""),
		},
		"serve-borrower": {
			"slaHours": []byte("2880"),
			"requiredDocuments": []byte("[\"affidavit_of_service\",\"return_of_service\"]"),
			"feeAmount": []byte("25000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"processor\""),
			"templateId": []byte("\"service-standard-v1\""),
		},
		"obtain-judgment": {
			"slaHours": []byte("4320"),
			"requiredDocuments": []byte("[\"motion_for_judgment\",\"affidavit_of_indebtedness\",\"proposed_judgment\"]"),
			"feeAmount": []byte("45000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"attorney\""),
			"templateId": []byte("\"judgment-standard-v1\""),
		},
		"schedule-sale": {
			"slaHours": []byte("1440"),
			"requiredDocuments": []byte("[\"notice_of_sale\",\"publication_proof\"]"),
			"feeAmount": []byte("30000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"processor\""),
			"templateId": []byte("\"sale-notice-standard-v1\""),
		},
		"conduct-sale": {
			"slaHours": []byte("720"),
			"requiredDocuments": []byte("[\"certificate_of_sale\",\"sale_report\"]"),
			"feeAmount": []byte("50000"),
			"feeAuthRequired": []byte("false"),
			"assignedRole": []byte("\"attorney\""),
			"templateId": []byte("\"sale-report-standard-v1\""),
		},
	}
}

func loadTestOverrides() []domain.Override {
	return []domain.Override{
		{ID: "ovr-001", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("360"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-002", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("[\"complaint\",\"summons\",\"lis_pendens\",\"cover_sheet\",\"verification_of_complaint\"]"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-003", StepKey: "serve-borrower", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("2160"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-014", StepKey: "conduct-sale", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("\"sale-report-fl-v2\""), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-020", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("240"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
		{ID: "ovr-025", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("\"complaint-fl-chase-v2\""), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
		{ID: "ovr-026", StepKey: "obtain-judgment", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("2880"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
		{ID: "ovr-053", StepKey: "file-complaint", TraitKey: "feeAmount", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("60000"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
		{ID: "ovr-030", StepKey: "title-search", TraitKey: "feeAuthRequired", Selector: domain.Selector{Client: ptr("Chase")}, Value: json.RawMessage("true"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-031", StepKey: "file-complaint", TraitKey: "feeAuthRequired", Selector: domain.Selector{Client: ptr("Chase")}, Value: json.RawMessage("true"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-034", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("168"), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
		{ID: "ovr-035", StepKey: "file-complaint", TraitKey: "feeAmount", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("55000"), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
		{ID: "ovr-036", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("[\"complaint\",\"summons\",\"lis_pendens\",\"cover_sheet\",\"verification_of_complaint\",\"hud_face_sheet\",\"fha_servicing_history\"]"), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
		{ID: "ovr-037", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("\"complaint-fl-chase-fha-v3\""), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
		{ID: "ovr-038", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{Investor: ptr("FHA")}, Value: json.RawMessage("[\"complaint\",\"summons\",\"lis_pendens\",\"cover_sheet\",\"hud_face_sheet\"]"), EffectiveDate: mustParseDate("2025-03-01"), Status: "active", Specificity: 1},
		{ID: "ovr-039", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{Investor: ptr("VA")}, Value: json.RawMessage("[\"complaint\",\"summons\",\"lis_pendens\",\"cover_sheet\",\"va_loan_summary\",\"va_appraisal\"]"), EffectiveDate: mustParseDate("2025-03-01"), Status: "active", Specificity: 1},
		{ID: "ovr-047", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FannieMae"), CaseType: ptr("FC-Judicial")}, Value: json.RawMessage("\"complaint-fl-chase-fnma-judicial-v3\""), EffectiveDate: mustParseDate("2025-11-01"), Status: "active", Specificity: 4},
		{ID: "ovr-048", StepKey: "title-search", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Nationstar")}, Value: json.RawMessage("480"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
		{ID: "ovr-005", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("TX")}, Value: json.RawMessage("336"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-042", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("TX"), CaseType: ptr("FC-NonJudicial")}, Value: json.RawMessage("240"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 2},
		{ID: "ovr-043", StepKey: "obtain-judgment", TraitKey: "slaHours", Selector: domain.Selector{CaseType: ptr("FC-NonJudicial")}, Value: json.RawMessage("0"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
		{ID: "ovr-055", StepKey: "title-search", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("OH")}, Value: json.RawMessage("504"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
	}
}

func ptr(s string) *string { return &s }
func mustParseDate(d string) time.Time { t, _ := time.Parse("2006-01-02", d); return t }

