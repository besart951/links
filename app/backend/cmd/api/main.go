package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/links/backend/internal/config"
	"github.com/links/backend/internal/db"
	"github.com/links/backend/internal/server"
)

func main() {
	// Load central .env from repo root (two levels up from app/backend)
	if err := godotenv.Load("../../.env"); err != nil {
		slog.Warn("no .env file found, falling back to environment variables")
	}

	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	if err := db.Migrate(ctx, cfg.DatabaseURL); err != nil {
		slog.Error("database migration failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	srv := server.New(cfg, pool)

	slog.Info("backend starting", "addr", fmt.Sprintf(":%s", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, srv); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
