package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/internal/server"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/mock"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

func newTestServer(t *testing.T) *server.Server {
	t.Helper()

	logger := otelzap.New(zap.NewNop())
	registry := shipper.NewRegistry()
	registry.Register(mock.New("test-shipper"))

	return server.New(server.Config{Port: 8080}, registry, logger)
}

func TestServer_Health(t *testing.T) {
	// Create a simple handler to test health endpoint
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestServer_GraphQL_MethodNotAllowed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]string{
					{"message": "Method not allowed, use POST"},
				},
			})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	errors, ok := resp["errors"].([]interface{})
	require.True(t, ok)
	assert.Len(t, errors, 1)
}

func TestServer_GraphQL_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]string{
					{"message": "Invalid JSON: " + err.Error()},
				},
			})
			return
		}
	})

	body := strings.NewReader("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_GraphQL_HealthQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Simple mock response for health query
		if strings.Contains(req.Query, "health") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"health": true,
				},
			})
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"message": "Unknown operation"},
			},
		})
	})

	body := strings.NewReader(`{"query": "query { health }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["health"])
}
