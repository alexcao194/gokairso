package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexcao194/kairso/internal/api/rest"
	"github.com/alexcao194/kairso/internal/config"
	"github.com/alexcao194/kairso/pkg/logger"
	_ "github.com/alexcao194/kairso/pkg/pagination"

	"github.com/jackc/pgx/v5/pgxpool"
)

// @title Kairso API
// @version 1.0
// @description API for Kairso
// @host 192.168.1.4:8041
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	logWriter, err := logger.InitDefault()
	if err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}
	defer logWriter.Close()

	slog.Info("Starting Golish API Server")
	cfg := config.Load()
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DB.DatabaseURL())
	if err != nil {
		slog.Error("Failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		slog.Error("Failed to ping db", "error", err)
		os.Exit(1)
	}
	server := rest.NewServer(cfg, db)
	go func() {
		if err := server.Run(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("Server exited gracefully")
}
