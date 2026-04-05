package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Context struct {
	State, Client, Investor, CaseType string `json:"-"`
}

func main() {
	http.HandleFunc("/api/resolve", func(w http.ResponseWriter, r *http.Request) {
		var ctx Context
		json.NewDecoder(r.Body).Decode(&ctx)
		
		response := map[string]interface{}{
			"context": ctx,
			"resolvedAt": "2026-04-05T12:00:00Z",
			"steps": map[string]map[string]map[string]interface{}{
				"file-complaint": {
					"slaHours": map[string]interface{}{
						"value": 240, "source": "override", "overrideId": "ovr-020",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	log.Println("🚀 Test server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}