// Package server wires routes, middlewares, and handlers into an HTTP server.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
	"github.com/bissquit/gophkeeper/internal/config"
	"github.com/bissquit/gophkeeper/internal/handler"
	"github.com/bissquit/gophkeeper/internal/logging"
	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server bundles the chi router with the dependencies it needs at request time.
type Server struct {
	config  *config.Config
	storage repository.Repository
	router  *chi.Mux
	db      *pgxpool.Pool
	logger  *slog.Logger
}

// NewServer constructs a Server and registers all routes.
func NewServer(cfg *config.Config, storage repository.Repository, db *pgxpool.Pool, logger *slog.Logger) *Server {
	s := &Server{
		config:  cfg,
		storage: storage,
		router:  chi.NewRouter(),
		db:      db,
		logger:  logger,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	secret := []byte(s.config.JWTSecret)
	h := handler.NewHandlers(s.storage, s.logger, secret)

	s.router.Use(logging.Logger(s.logger))

	s.router.Post("/api/user/register", h.Register)
	s.router.Post("/api/user/login", h.Login)
	s.router.Get("/ping", s.Ping)

	s.router.Group(func(r chi.Router) {
		r.Use(jwt.JWT(secret))
		r.Get("/api/secrets", h.ListSecrets)
	})
}

// Ping is a liveness/readiness probe that also verifies DB connectivity.
func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		s.logger.Error("db ping failed", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Handler exposes the configured chi router so the caller can wrap it in
// http.Server with custom timeouts.
func (s *Server) Handler() http.Handler {
	return s.router
}
