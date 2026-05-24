package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/service"
)

type userRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register creates a new user from a JSON {login, password} body and returns JWT
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !h.validateContentTypeJSON(w, r) {
		return
	}

	var req userRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Register(req.Login, req.Password)
	if err != nil {
		h.writeAuthError(w, "register", err)
		return
	}
	writeToken(w, token)
}

// Login authenticates a user against the stored bcrypt hash and returns JWT
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !h.validateContentTypeJSON(w, r) {
		return
	}

	var req userRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Login(req.Login, req.Password)
	if err != nil {
		h.writeAuthError(w, "login", err)
		return
	}
	writeToken(w, token)
}

func (h *Handlers) writeAuthError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		http.Error(w, "login and password required", http.StatusBadRequest)
	case errors.Is(err, service.ErrLoginTaken):
		http.Error(w, "login taken", http.StatusConflict)
	case errors.Is(err, service.ErrInvalidCredentials):
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	default:
		h.logger.Error(op, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func writeToken(w http.ResponseWriter, token string) {
	w.Header().Set("Authorization", "Bearer "+token)
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
