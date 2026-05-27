// Command gophkeeper-server runs the GophKeeper HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bissquit/gophkeeper/internal/config"
	"github.com/bissquit/gophkeeper/internal/repository/db"
	"github.com/bissquit/gophkeeper/internal/server"
	"github.com/bissquit/gophkeeper/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.GetConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := initDatabase(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	stg := db.NewDBStorage(pool)
	srv := server.NewServer(cfg, stg, pool, logger)

	httpSrv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      srv.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	logger.Info("server starting", "addr", cfg.ServerAddr)

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}

func initDatabase(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, errors.New("DATABASE_URI (or -d) is required")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(dbCtx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := migrations.InitializeDB(cfg.DSN); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("database connected")
	return pool, nil
}
