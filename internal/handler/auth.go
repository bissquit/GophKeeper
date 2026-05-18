package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
	"github.com/bissquit/gophkeeper/internal/password"
	"github.com/bissquit/gophkeeper/internal/repository"
)

type userRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register creates a new user from a JSON {login, password} body and returns
// a freshly issued JWT in both the response body and Authorization header
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
	if req.Login == "" || req.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		h.logger.Error("hash password", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	userID, err := h.storage.CreateUser(req.Login, hash)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			http.Error(w, "login taken", http.StatusConflict)
			return
		}
		h.logger.Error("create user", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.issueToken(w, userID, req.Login)
}

// Login authenticates a user against the stored bcrypt hash and returns a JWT
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
	if req.Login == "" || req.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	u, err := h.storage.GetUserByLogin(req.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		h.logger.Error("get user", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !password.CheckHash(req.Password, u.PasswordHash) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	h.issueToken(w, u.ID, u.Login)
}

func (h *Handlers) issueToken(w http.ResponseWriter, userID, login string) {
	token, err := jwt.GenerateToken(userID, login, h.jwtSecret)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		h.logger.Error("encode token", "err", err)
	}
}
