package handler

import "net/http"

type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

func (h *AuditHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Placeholder implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"history":[]}`))
}