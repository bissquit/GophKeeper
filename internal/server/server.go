// Package server wires routes, middlewares, and handlers into an HTTP server
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
	"github.com/bissquit/gophkeeper/internal/service"
	"github.com/go-chi/chi/v5"
)

// Server bundles the chi router with the dependencies it needs at request time
type Server struct {
	router  *chi.Mux
	storage repository.Repository
	logger  *slog.Logger
}

// NewServer constructs a Server and registers all routes
func NewServer(cfg *config.Config, storage repository.Repository, logger *slog.Logger) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		storage: storage,
		logger:  logger,
	}
	s.setupRoutes(cfg)
	return s
}

func (s *Server) setupRoutes(cfg *config.Config) {
	secret := []byte(cfg.JWTSecret)

	authSvc := service.NewAuth(s.storage, secret)
	secretsSvc := service.NewSecrets(s.storage)
	h := handler.NewHandlers(authSvc, secretsSvc, s.logger)

	s.router.Use(logging.Logger(s.logger))

	s.router.Post("/api/user/register", h.Register)
	s.router.Post("/api/user/login", h.Login)
	s.router.Get("/ping", s.Ping)

	s.router.Group(func(r chi.Router) {
		r.Use(jwt.JWT(secret))
		r.Get("/api/secrets", h.ListSecrets)
		r.Post("/api/secrets", h.CreateSecret)
		r.Put("/api/secrets/{id}", h.UpdateSecret)
		r.Delete("/api/secrets/{id}", h.DeleteSecret)
	})
}

// Ping is a liveness/readiness probe that also verifies storage reachability
func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := s.storage.Ping(ctx); err != nil {
		s.logger.Error("storage ping failed", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Handler exposes the configured chi router
func (s *Server) Handler() http.Handler {
	return s.router
}
