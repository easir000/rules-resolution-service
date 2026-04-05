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
	"github.com/pearsonspecter/rules-resolution/internal/database"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/handler"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
	"github.com/pearsonspecter/rules-resolution/internal/service"
)

func main() {
	cfg := config.Load()
	dbCfg := config.LoadDatabaseConfig()

	pool, err := config.NewPool(dbCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if cfg.RunMigrations {
		if err := database.RunMigrations(pool); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
		log.Println("✓ Migrations applied")
	}

	if cfg.SeedData {
		if err := database.SeedData(pool); err != nil {
			log.Fatalf("failed to seed  %v", err)
		}
		log.Println("✓ Seed data loaded")
	}

	overrideRepo := repository.NewOverrideRepository(pool)
	defaultRepo := repository.NewDefaultRepository(pool)

	defaults, err := defaultRepo.LoadAll(context.Background())
	if err != nil {
		log.Fatalf("failed to load defaults: %v", err)
	}

	resolutionSvc := service.NewResolutionService(defaults, overrideRepo)
	explainSvc := service.NewExplainService(defaults, overrideRepo)
	conflictSvc := service.NewConflictService(overrideRepo)

	resolveHandler := handler.NewResolveHandler(resolutionSvc, explainSvc)
	overrideHandler := handler.NewOverrideHandler(overrideRepo, conflictSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/api", func(r chi.Router) {
		resolveHandler.RegisterRoutes(r)
		overrideHandler.RegisterRoutes(r)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})
	})

	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":8080"
	}
	log.Printf("🚀 Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}

func resolveHandler(defaults map[string]map[domain.TraitKey]json.RawMessage, overrides []domain.Override) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var ctx domain.CaseContext
        if err := json.NewDecoder(r.Body).Decode(&ctx); err != nil {
            http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
            return
        }
        response := resolve(ctx, defaults, overrides)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}

func explainHandler(overrides []domain.Override) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            domain.CaseContext
            StepKey  string          `json:"stepKey"`
            TraitKey domain.TraitKey `json:"traitKey"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
            return
        }

        asOfDate := time.Now()
        if req.AsOfDate != nil && *req.AsOfDate != "" {
            if parsed, err := time.Parse("2006-01-02", *req.AsOfDate); err == nil {
                asOfDate = parsed
            }
        }

        // Find all candidates
        var candidates []domain.Override
        for _, o := range overrides {
            if o.StepKey == req.StepKey && o.TraitKey == req.TraitKey &&
                o.Selector.Matches(req.CaseContext) &&
                o.Status == "active" &&
                !o.EffectiveDate.After(asOfDate) {
                candidates = append(candidates, o)
            }
        }

        // Sort
        sort.Slice(candidates, func(i, j int) bool {
            if candidates[i].Specificity != candidates[j].Specificity {
                return candidates[i].Specificity > candidates[j].Specificity
            }
            if !candidates[i].EffectiveDate.Equal(candidates[j].EffectiveDate) {
                return candidates[i].EffectiveDate.After(candidates[j].EffectiveDate)
            }
            return candidates[i].ID < candidates[j].ID
        })

        // Build response with outcomes
        type candidateOutput struct {
            OverrideID    string          `json:"overrideId"`
            Selector      domain.Selector `json:"selector"`
            Specificity   int             `json:"specificity"`
            EffectiveDate time.Time       `json:"effectiveDate"`
            Value         json.RawMessage `json:"value"`
            Outcome       string          `json:"outcome"`
        }

        var outputs []candidateOutput
        for i, c := range candidates {
            outcome := "SHADOWED"
            if i == 0 {
                outcome = fmt.Sprintf("SELECTED — highest specificity (%d)", c.Specificity)
            } else if c.Specificity < candidates[0].Specificity {
                outcome = fmt.Sprintf("SHADOWED — lower specificity (%d < %d)", c.Specificity, candidates[0].Specificity)
            } else if c.EffectiveDate.Before(candidates[0].EffectiveDate) {
                outcome = fmt.Sprintf("SHADOWED — earlier effective date (%s < %s)",
                    c.EffectiveDate.Format("2006-01-02"), candidates[0].EffectiveDate.Format("2006-01-02"))
            }
            outputs = append(outputs, candidateOutput{
                OverrideID:    c.ID,
                Selector:      c.Selector,
                Specificity:   c.Specificity,
                EffectiveDate: c.EffectiveDate,
                Value:         c.Value,
                Outcome:       outcome,
            })
        }

        response := map[string]interface{}{
            "step":       req.StepKey,
            "trait":      req.TraitKey,
            "candidates": outputs,
        }
        if len(candidates) > 0 {
            winner := candidates[0]
            response["resolvedValue"] = winner.Value
            response["resolvedFrom"] = map[string]interface{}{
                "overrideId":    winner.ID,
                "selector":      winner.Selector,
                "specificity":   winner.Specificity,
                "effectiveDate": winner.EffectiveDate,
            }
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}

// [loadDefaults, loadTestOverrides, resolve, buildExplanation, ptr, mustParseDate functions from before...]
// (Include all the functions from the previous working main.go here)

func loadDefaults() map[string]map[domain.TraitKey]json.RawMessage {
    return map[string]map[domain.TraitKey]json.RawMessage{
        "title-search": {
            "slaHours": json.RawMessage("720"),
            "requiredDocuments": json.RawMessage(`["title_commitment","tax_certificate"]`),
            "feeAmount": json.RawMessage("35000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"processor"`),
            "templateId": json.RawMessage(`"title-review-standard-v1"`),
        },
        "file-complaint": {
            "slaHours": json.RawMessage("480"),
            "requiredDocuments": json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet"]`),
            "feeAmount": json.RawMessage("65000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"attorney"`),
            "templateId": json.RawMessage(`"complaint-standard-v1"`),
        },
        "serve-borrower": {
            "slaHours": json.RawMessage("2880"),
            "requiredDocuments": json.RawMessage(`["affidavit_of_service","return_of_service"]`),
            "feeAmount": json.RawMessage("25000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"processor"`),
            "templateId": json.RawMessage(`"service-standard-v1"`),
        },
        "obtain-judgment": {
            "slaHours": json.RawMessage("4320"),
            "requiredDocuments": json.RawMessage(`["motion_for_judgment","affidavit_of_indebtedness","proposed_judgment"]`),
            "feeAmount": json.RawMessage("45000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"attorney"`),
            "templateId": json.RawMessage(`"judgment-standard-v1"`),
        },
        "schedule-sale": {
            "slaHours": json.RawMessage("1440"),
            "requiredDocuments": json.RawMessage(`["notice_of_sale","publication_proof"]`),
            "feeAmount": json.RawMessage("30000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"processor"`),
            "templateId": json.RawMessage(`"sale-notice-standard-v1"`),
        },
        "conduct-sale": {
            "slaHours": json.RawMessage("720"),
            "requiredDocuments": json.RawMessage(`["certificate_of_sale","sale_report"]`),
            "feeAmount": json.RawMessage("50000"),
            "feeAuthRequired": json.RawMessage("false"),
            "assignedRole": json.RawMessage(`"attorney"`),
            "templateId": json.RawMessage(`"sale-report-standard-v1"`),
        },
    }
}

func loadTestOverrides() []domain.Override {
    return []domain.Override{
        {ID: "ovr-001", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("360"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-002", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet","verification_of_complaint"]`), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-003", StepKey: "serve-borrower", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage("2160"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-014", StepKey: "conduct-sale", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL")}, Value: json.RawMessage(`"sale-report-fl-v2"`), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-020", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("240"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
        {ID: "ovr-025", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage(`"complaint-fl-chase-v2"`), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
        {ID: "ovr-026", StepKey: "obtain-judgment", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("2880"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
        {ID: "ovr-053", StepKey: "file-complaint", TraitKey: "feeAmount", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase")}, Value: json.RawMessage("60000"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
        {ID: "ovr-030", StepKey: "title-search", TraitKey: "feeAuthRequired", Selector: domain.Selector{Client: ptr("Chase")}, Value: json.RawMessage("true"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-031", StepKey: "file-complaint", TraitKey: "feeAuthRequired", Selector: domain.Selector{Client: ptr("Chase")}, Value: json.RawMessage("true"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-034", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("168"), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
        {ID: "ovr-035", StepKey: "file-complaint", TraitKey: "feeAmount", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage("55000"), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
        {ID: "ovr-036", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet","verification_of_complaint","hud_face_sheet","fha_servicing_history"]`), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
        {ID: "ovr-037", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FHA")}, Value: json.RawMessage(`"complaint-fl-chase-fha-v3"`), EffectiveDate: mustParseDate("2025-09-01"), Status: "active", Specificity: 3},
        {ID: "ovr-038", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{Investor: ptr("FHA")}, Value: json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet","hud_face_sheet"]`), EffectiveDate: mustParseDate("2025-03-01"), Status: "active", Specificity: 1},
        {ID: "ovr-039", StepKey: "file-complaint", TraitKey: "requiredDocuments", Selector: domain.Selector{Investor: ptr("VA")}, Value: json.RawMessage(`["complaint","summons","lis_pendens","cover_sheet","va_loan_summary","va_appraisal"]`), EffectiveDate: mustParseDate("2025-03-01"), Status: "active", Specificity: 1},
        {ID: "ovr-047", StepKey: "file-complaint", TraitKey: "templateId", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Chase"), Investor: ptr("FannieMae"), CaseType: ptr("FC-Judicial")}, Value: json.RawMessage(`"complaint-fl-chase-fnma-judicial-v3"`), EffectiveDate: mustParseDate("2025-11-01"), Status: "active", Specificity: 4},
        {ID: "ovr-048", StepKey: "title-search", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("FL"), Client: ptr("Nationstar")}, Value: json.RawMessage("480"), EffectiveDate: mustParseDate("2025-06-01"), Status: "active", Specificity: 2},
        {ID: "ovr-005", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("TX")}, Value: json.RawMessage("336"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-042", StepKey: "file-complaint", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("TX"), CaseType: ptr("FC-NonJudicial")}, Value: json.RawMessage("240"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 2},
        {ID: "ovr-043", StepKey: "obtain-judgment", TraitKey: "slaHours", Selector: domain.Selector{CaseType: ptr("FC-NonJudicial")}, Value: json.RawMessage("0"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
        {ID: "ovr-055", StepKey: "title-search", TraitKey: "slaHours", Selector: domain.Selector{State: ptr("OH")}, Value: json.RawMessage("504"), EffectiveDate: mustParseDate("2025-01-01"), Status: "active", Specificity: 1},
    }
}

func resolve(ctx domain.CaseContext, defaults map[string]map[domain.TraitKey]json.RawMessage, overrides []domain.Override) domain.ResolutionResponse {
    asOfDate := time.Now()
    if ctx.AsOfDate != nil && *ctx.AsOfDate != "" {
        if parsed, err := time.Parse("2006-01-02", *ctx.AsOfDate); err == nil {
            asOfDate = parsed
        }
    }
    response := domain.ResolutionResponse{
        Context:    ctx,
        ResolvedAt: time.Now().UTC(),
        Steps:      make(map[string]domain.ResolvedStep),
    }
    for stepKey, traits := range defaults {
        response.Steps[stepKey] = make(domain.ResolvedStep)
        for traitKey, defaultValue := range traits {
            var candidates []domain.Override
            for _, o := range overrides {
                if o.StepKey == stepKey && o.TraitKey == traitKey &&
                    o.Status == "active" &&
                    !o.EffectiveDate.After(asOfDate) &&
                    o.Selector.Matches(ctx) {
                    candidates = append(candidates, o)
                }
            }
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
                    Value: winner.Value, Source: "override", OverrideID: &winner.ID,
                    Explanation: buildExplanation(winner),
                }
            } else {
                response.Steps[stepKey][traitKey] = domain.ResolvedTrait{
                    Value: defaultValue, Source: "default",
                    Explanation: "No matching override",
                }
            }
        }
    }
    return response
}

func buildExplanation(o domain.Override) string {
    parts := []string{}
    if o.Selector.State != nil { parts = append(parts, *o.Selector.State) }
    if o.Selector.Client != nil { parts = append(parts, *o.Selector.Client) }
    if o.Selector.Investor != nil { parts = append(parts, *o.Selector.Investor) }
    if o.Selector.CaseType != nil { parts = append(parts, *o.Selector.CaseType) }
    if len(parts) == 0 { return "Default value" }
    return fmt.Sprintf("%s override (specificity %d)", strings.Join(parts, "+"), o.Specificity)
}

func ptr(s string) *string { return &s }
func mustParseDate(d string) time.Time { t, _ := time.Parse("2006-01-02", d); return t }
