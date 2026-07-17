// @title          rachao.app API v2
// @version        2.0
// @description    Go rewrite of the rachao.app backend. All endpoints live under `/api/v2`.
// @description    Authentication uses JWT in the `Authorization: Bearer <token>` header.
// @contact.name   rachao.app
// @contact.url    https://rachao.app
// @license.name   Proprietary
// @host           localhost:8080
// @BasePath       /api/v2
// @schemes        http https
// @securityDefinitions.apikey BearerAuth
// @in             header
// @name           Authorization
// @description    JWT access token. Use the value returned by `POST /auth/login`.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// tzdata embutido no binário: a imagem de produção é scratch (sem
	// /usr/share/zoneinfo) e o app depende de America/Sao_Paulo — crons do
	// scheduler e janela de votação (handlers/votes.go). Sem este embed,
	// time.LoadLocation falha e tudo cai silenciosamente para UTC.
	_ "time/tzdata"

	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/server"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
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

	// Background scheduler: hourly status sync + daily recurrence job.
	// Skipped in test env to avoid jobs firing during integration tests.
	var scheduler *services.Scheduler
	if cfg.AppEnv != "test" {
		services.InitJobMetrics()
		scheduler = services.NewScheduler(pool)
		if err := scheduler.Start(); err != nil {
			slog.Error("scheduler start failed", "error", err)
			os.Exit(1)
		}
		defer scheduler.Stop()
	}

	router := server.NewRouter(cfg, pool)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // 60s for SSE (chat endpoint)
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}
