// Worker de transcodificação de vídeos (PRD 052): consome a fila match_videos
// e roda ffmpeg. Binário separado da API porque a imagem dela é scratch (sem
// ffmpeg); este roda em imagem alpine com ffmpeg instalado.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
	"github.com/thiagotn/football-manager/football-api-go/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" {
		slog.Error("R2 storage not configured — worker cannot run")
		os.Exit(1)
	}
	storage, err := services.NewStorageService(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBaseURL)
	if err != nil {
		slog.Error("storage init failed", "error", err)
		os.Exit(1)
	}

	// /healthz + /metrics (sem Service/Ingress — só probes e Prometheus scrape)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	srv := &http.Server{Addr: ":9090", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	w := worker.New(pool, storage, "/tmp/work")

	done := make(chan struct{})
	go func() {
		w.Run(ctx) // retorna só após terminar o job corrente
		close(done)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker...")
	cancel()
	select {
	case <-done:
	case <-time.After(170 * time.Second): // < terminationGracePeriodSeconds (180)
		slog.Warn("worker shutdown timed out")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	slog.Info("worker stopped")
}
