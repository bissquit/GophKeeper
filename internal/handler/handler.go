// Package handler provides HTTP request handlers for the GophKeeper server.
package handler

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/service"
)

// Handlers carries the dependencies shared by every HTTP handler method.
type Handlers struct {
	auth    *service.Auth
	secrets *service.Secrets
	logger  *slog.Logger
}

// NewHandlers builds a Handlers value with the given service dependencies.
func NewHandlers(auth *service.Auth, secrets *service.Secrets, logger *slog.Logger) *Handlers {
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
