package handler

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/go-chi/chi/v5"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
	"github.com/pearsonspecter/rules-resolution/internal/service"
)

type OverrideHandler struct {
	repo        repository.OverrideRepository
	conflictSvc *service.ConflictService
}

func NewOverrideHandler(repo repository.OverrideRepository, conflictSvc *service.ConflictService) *OverrideHandler {
	return &OverrideHandler{repo: repo, conflictSvc: conflictSvc}
}

func (h *OverrideHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := repository.OverrideFilters{
		StepKey: r.URL.Query().Get("stepKey"),
		TraitKey: r.URL.Query().Get("traitKey"),
		State: r.URL.Query().Get("state"),
		Client: r.URL.Query().Get("client"),
		Investor: r.URL.Query().Get("investor"),
		CaseType: r.URL.Query().Get("caseType"),
		Status: r.URL.Query().Get("status"),
	}
	overrides, err := h.repo.List(r.Context(), filters)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]domain.Override{"overrides": overrides})
}

func (h *OverrideHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func (h *OverrideHandler) Create(w http.ResponseWriter, r *http.Request) {
	var o domain.Override
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	// Compute specificity from selector
	o.Specificity = o.Selector.Specificity()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	if o.CreatedBy == "" { o.CreatedBy = "api" }
	if o.Status == "" { o.Status = "draft" }
	
	if err := h.repo.Create(r.Context(), &o); err != nil {
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func (h *OverrideHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var o domain.Override
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	o.ID = id
	o.Specificity = o.Selector.Specificity()
	o.UpdatedAt = time.Now()
	o.UpdatedBy = stringPtr("api")
	
	if err := h.repo.Update(r.Context(), &o); err != nil {
		http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func (h *OverrideHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct{ Status string `json:"status"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.Status != "draft" && req.Status != "active" && req.Status != "archived" {
		http.Error(w, `{"error":"invalid status"}`, http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateStatus(r.Context(), id, req.Status, "api"); err != nil {
		http.Error(w, `{"error":"status update failed"}`, http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status":"updated"}`))
}

func (h *OverrideHandler) GetConflicts(w http.ResponseWriter, r *http.Request) {
	conflicts, err := h.repo.FindConflicts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]domain.Conflict{"conflicts": conflicts})
}

func (h *OverrideHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	history, err := h.repo.GetHistory(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]domain.AuditEntry{"history": history})
}

func (h *OverrideHandler) RegisterRoutes(r chi.Router) {
	r.Route("/overrides", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Post("/", h.Create)
		r.Put("/{id}", h.Update)
		r.Patch("/{id}/status", h.UpdateStatus)
		r.Get("/conflicts", h.GetConflicts)
		r.Get("/{id}/history", h.GetHistory)
	})
}

func stringPtr(s string) *string { return &s }