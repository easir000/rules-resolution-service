package handler

import (
    "encoding/json"
    "net/http"
    "sort"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/pearsonspecter/rules-resolution/internal/domain"
    "github.com/pearsonspecter/rules-resolution/internal/service"
)

type OverrideHandler struct {
    overrides   []domain.Override
    conflictSvc *service.ConflictService
}

func NewOverrideHandler(overrides []domain.Override, conflictSvc *service.ConflictService) *OverrideHandler {
    return &OverrideHandler{overrides: overrides, conflictSvc: conflictSvc}
}

func (h *OverrideHandler) List(w http.ResponseWriter, r *http.Request) {
    stepKey := r.URL.Query().Get("stepKey")
    traitKey := r.URL.Query().Get("traitKey")
    state := r.URL.Query().Get("state")
    client := r.URL.Query().Get("client")
    investor := r.URL.Query().Get("investor")
    caseType := r.URL.Query().Get("caseType")
    status := r.URL.Query().Get("status")

    filtered := h.overrides
    if stepKey != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { return o.StepKey == stepKey })
    }
    if traitKey != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { return string(o.TraitKey) == traitKey })
    }
    if state != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { 
            return o.Selector.State == nil || *o.Selector.State == state 
        })
    }
    if client != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { 
            return o.Selector.Client == nil || *o.Selector.Client == client 
        })
    }
    if investor != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { 
            return o.Selector.Investor == nil || *o.Selector.Investor == investor 
        })
    }
    if caseType != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { 
            return o.Selector.CaseType == nil || *o.Selector.CaseType == caseType 
        })
    }
    if status != "" {
        filtered = filterOverrides(filtered, func(o domain.Override) bool { return o.Status == status })
    }

    sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID > filtered[j].ID })

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string][]domain.Override{"overrides": filtered})
}

func (h *OverrideHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    for _, o := range h.overrides {
        if o.ID == id {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(o)
            return
        }
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

func (h *OverrideHandler) Create(w http.ResponseWriter, r *http.Request) {
    var o domain.Override
    if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
        return
    }
    o.Specificity = o.Selector.Specificity()
    o.CreatedAt = time.Now()
    o.UpdatedAt = time.Now()
    if o.CreatedBy == "" { o.CreatedBy = "api" }
    if o.Status == "" { o.Status = "draft" }
    
    h.overrides = append(h.overrides, o)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(o)
}

func (h *OverrideHandler) Update(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var o domain.Override
    if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
        return
    }
    for i := range h.overrides {
        if h.overrides[i].ID == id {
            o.ID = id
            o.Specificity = o.Selector.Specificity()
            o.UpdatedAt = time.Now()
            h.overrides[i] = o
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(o)
            return
        }
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

func (h *OverrideHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var req struct{ Status string json:"status" }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
        return
    }
    if req.Status != "draft" && req.Status != "active" && req.Status != "archived" {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "invalid status"})
        return
    }
    for i := range h.overrides {
        if h.overrides[i].ID == id {
            h.overrides[i].Status = req.Status
            h.overrides[i].UpdatedAt = time.Now()
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
            return
        }
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

func (h *OverrideHandler) GetConflicts(w http.ResponseWriter, r *http.Request) {
    conflicts := h.conflictSvc.FindConflicts(h.overrides)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string][]domain.Conflict{"conflicts": conflicts})
}

func (h *OverrideHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string][]string{"history": {}})
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

func filterOverrides(overrides []domain.Override, fn func(domain.Override) bool) []domain.Override {
    var result []domain.Override
    for _, o := range overrides {
        if fn(o) {
            result = append(result, o)
        }
    }
    return result
}
