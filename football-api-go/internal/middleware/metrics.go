package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// httpRequestDuration espelha o histograma exposto pela API v1
// (prometheus-fastapi-instrumentator): http_request_duration_seconds com labels
// method/handler/status_code — assim os mesmos painéis e alertas do Grafana
// cobrem v1 e v2 sem mudar as queries.
var httpRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duração das requisições HTTP em segundos.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "handler", "status_code"},
)

// Metrics instrumenta cada request com o histograma http_request_duration_seconds.
// Exclui /metrics e /api/v2/health para não poluir as métricas com scrapes e
// healthchecks (paridade com o excluded_handlers do instrumentator Python).
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/api/v2/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ww := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		handler := chi.RouteContext(r.Context()).RoutePattern()
		if handler == "" {
			handler = "unmatched"
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		httpRequestDuration.
			WithLabelValues(r.Method, handler, strconv.Itoa(status)).
			Observe(time.Since(start).Seconds())
	})
}
