// Package handler provides HTTP request handlers for the GophKeeper server.
package handler

import (
	"log/slog"
	"mime"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/repository"
)

// Handlers carries the dependencies shared by every HTTP handler method
type Handlers struct {
	storage   repository.Repository
	logger    *slog.Logger
	jwtSecret []byte
}

// NewHandlers builds a Handlers value with the given dependencies
func NewHandlers(storage repository.Repository, logger *slog.Logger, jwtSecret []byte) *Handlers {
	return &Handlers{storage: storage, logger: logger, jwtSecret: jwtSecret}
}

func (h *Handlers) validateContentTypeJSON(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return false
	}
	return true
}
