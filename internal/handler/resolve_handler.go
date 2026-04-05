package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/service"
)

// ResolveHandler handles /api/resolve endpoints
type ResolveHandler struct {
	resolutionSvc *service.ResolutionService
	explainSvc    *service.ExplainService
}

func NewResolveHandler(resolutionSvc *service.ResolutionService,
	explainSvc *service.ExplainService) *ResolveHandler {
	return &ResolveHandler{
		resolutionSvc: resolutionSvc,
		explainSvc:    explainSvc,
	}
}

// Resolve handles POST /api/resolve
func (h *ResolveHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var ctx domain.CaseContext
	if err := json.NewDecoder(r.Body).Decode(&ctx); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if ctx.State == "" || ctx.Client == "" || ctx.Investor == "" || ctx.CaseType == "" {
		http.Error(w, `{"error": "state, client, investor, and caseType are required"}`, http.StatusBadRequest)
		return
	}

	response := h.resolutionSvc.Resolve(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Explain handles POST /api/resolve/explain
func (h *ResolveHandler) Explain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		domain.CaseContext
		StepKey  string          `json:"stepKey"`
		TraitKey domain.TraitKey `json:"traitKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.StepKey == "" || req.TraitKey == "" {
		http.Error(w, `{"error": "stepKey and traitKey are required for explain"}`, http.StatusBadRequest)
		return
	}

	response := h.explainSvc.Explain(req.StepKey, req.TraitKey, req.CaseContext)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes mounts the handler on a chi router
func (h *ResolveHandler) RegisterRoutes(r chi.Router) {
	r.Post("/resolve", h.Resolve)
	r.Post("/resolve/explain", h.Explain)
}