// Package handler provides HTTP request handlers for the GophKeeper server.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/repository"
)

// authService is the subset of behavior handler needs from the auth service
type authService interface {
	Register(ctx context.Context, login, plainPassword string) (token string, err error)
	Login(ctx context.Context, login, plainPassword string) (token string, err error)
}

// secretsService is the subset of behavior handler needs from the secrets service
type secretsService interface {
	Create(ctx context.Context, userID string, in repository.NewSecret) (repository.Secret, error)
	Update(ctx context.Context, userID, id string, data []byte, meta string) (repository.Secret, error)
	List(ctx context.Context, userID string) ([]repository.Secret, error)
	Delete(ctx context.Context, userID, id string) error
}

// Handlers carries the dependencies shared by every HTTP handler method.
type Handlers struct {
	auth    authService
	secrets secretsService
	logger  *slog.Logger
}

// NewHandlers builds a Handlers value with the given service dependencies.
func NewHandlers(auth authService, secrets secretsService, logger *slog.Logger) *Handlers {
	return &Handlers{auth: auth, secrets: secrets, logger: logger}
}

func (h *Handlers) validateContentTypeJSON(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
