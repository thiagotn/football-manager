package unit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

func TestMetricsMiddleware(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics)
	r.Get("/api/v2/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Handle("/metrics", promhttp.Handler())

	// Request instrumentada
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v2/ping", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	// /metrics deve expor o histograma com o status_code observado
	mrec := httptest.NewRecorder()
	r.ServeHTTP(mrec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, mrec.Code)
	body := mrec.Body.String()
	assert.Contains(t, body, "http_request_duration_seconds")
	assert.Contains(t, body, `status_code="200"`)
	assert.Contains(t, body, `handler="/api/v2/ping"`)
}
