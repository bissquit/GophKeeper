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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Error("config error", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	stg, err := db.Open(ctx, cfg.DSN)
	if err != nil {
		logger.Error("storage init failed", "err", err)
		os.Exit(1)
	}
	defer stg.Close()
	logger.Info("storage ready")

	srv := server.NewServer(cfg, stg, logger)

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
