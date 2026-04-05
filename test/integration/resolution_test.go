package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pearsonspecter/rules-resolution/internal/config"
	"github.com/pearsonspecter/rules-resolution/internal/database"
	"github.com/pearsonspecter/rules-resolution/internal/domain"
	"github.com/pearsonspecter/rules-resolution/internal/handler"
	"github.com/pearsonspecter/rules-resolution/internal/repository"
	"github.com/pearsonspecter/rules-resolution/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestScenario struct {
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Context             domain.CaseContext  `json:"context"`
	ExpectedResolutions []ExpectedResolution `json:"expectedResolutions"`
}

type ExpectedResolution struct {
	StepKey            string          `json:"stepKey"`
	TraitKey           string          `json:"traitKey"`
	ExpectedValue      json.RawMessage `json:"expectedValue"`
	ExpectedSource     string          `json:"expectedSource"`
	ExpectedOverrideID *string         `json:"expectedOverrideId"`
	Explanation        string          `json:"explanation"`
}

func TestResolutionScenarios(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Set INTEGRATION_TEST=true to run")
	}

	// Setup test database
	pool := setupTestDB(t)
	defer pool.Close()

	// Load test data
	require.NoError(t, database.SeedData(pool))

	// Setup services
	defaults := loadDefaults(t, pool)
	overrideRepo := repository.NewOverrideRepository(pool.Acquire())
	resolutionSvc := service.NewResolutionService(defaults, overrideRepo)

	// Setup handler
	r := chi.NewRouter()
	h := handler.NewResolveHandler(resolutionSvc, nil)
	r.Post("/api/resolve", h.Resolve)

	// Load and run test scenarios
	scenarios := loadScenarios(t)
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body, _ := json.Marshal(scenario.Context)
			req := httptest.NewRequest("POST", "/api/resolve", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var response domain.ResolutionResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

			// Validate each expected resolution
			for _, expected := range scenario.ExpectedResolutions {
				step := response.Steps[expected.StepKey]
				require.NotNil(t, step, "step %s not found", expected.StepKey)

				trait := step[domain.TraitKey(expected.TraitKey)]
				require.NotNil(t, trait, "trait %s not found in step %s", 
					expected.TraitKey, expected.StepKey)

				assert.JSONEq(t, string(expected.ExpectedValue), string(trait.Value),
					"wrong value for %s.%s in scenario %s", 
					expected.StepKey, expected.TraitKey, scenario.Name)

				assert.Equal(t, expected.ExpectedSource, trait.Source,
					"wrong source for %s.%s", expected.StepKey, expected.TraitKey)

				if expected.ExpectedOverrideID != nil {
					require.NotNil(t, trait.OverrideID)
					assert.Equal(t, *expected.ExpectedOverrideID, *trait.OverrideID,
						"wrong override ID for %s.%s", expected.StepKey, expected.TraitKey)
				}
			}
		})
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	cfg := config.Load()
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	require.NoError(t, err)

	// Run migrations
	require.NoError(t, database.RunMigrations(cfg.DatabaseURL))
	return pool
}

func loadDefaults(t *testing.T, pool *pgxpool.Pool) map[string]map[domain.TraitKey]json.RawMessage {
	repo := repository.NewStepRepository(pool)
	defaults, err := repo.LoadDefaults(context.Background())
	require.NoError(t, err)
	return defaults
}

func loadScenarios(t *testing.T) []TestScenario {
	data, err := os.ReadFile("../../sr_backend_assignment_data/test_scenarios.json")
	require.NoError(t, err)

	var scenarios []TestScenario
	require.NoError(t, json.Unmarshal(data, &scenarios))
	return scenarios
}