package handler

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
)

type OverrideHandler struct{}

func NewOverrideHandler() *OverrideHandler { return &OverrideHandler{} }

func (h *OverrideHandler) List(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode([]string{})
}
func (h *OverrideHandler) Get(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"id": chi.URLParam(r, "id")})
}
func (h *OverrideHandler) Create(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
func (h *OverrideHandler) Update(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
func (h *OverrideHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "changed"})
}
func (h *OverrideHandler) GetConflicts(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string][]string{"conflicts": {}})
}
func (h *OverrideHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
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
