package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/bissquit/gophkeeper/internal/service"
	"github.com/go-chi/chi/v5"
)

// secretDTO is the JSON used in every request and response body for secrets
type secretDTO struct {
	SecretItemID string    `json:"secret_item_id,omitempty"`
	ID           string    `json:"id,omitempty"`
	Type         string    `json:"type,omitempty"`
	Name         string    `json:"name,omitempty"`
	Data         string    `json:"data,omitempty"`
	Meta         string    `json:"meta,omitempty"`
	Version      int64     `json:"version,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

func toDTO(s repository.Secret) secretDTO {
	return secretDTO{
		SecretItemID: s.SecretItemID,
		ID:           s.ID,
		Type:         string(s.Type),
		Name:         s.Name,
		Data:         base64.StdEncoding.EncodeToString(s.Data),
		Meta:         s.Meta,
		Version:      s.Version,
		UpdatedAt:    s.UpdatedAt,
	}
}

func userIDFrom(r *http.Request) string {
	uid, _ := r.Context().Value(jwt.UserIDKey).(string)
	return uid
}

// CreateSecret stores the first version of a new logical secret for the caller
func (h *Handlers) CreateSecret(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !h.validateContentTypeJSON(w, r) {
		return
	}

	var in secretDTO
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	data, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		http.Error(w, "data must be base64", http.StatusBadRequest)
		return
	}

	sec, err := h.secrets.Create(r.Context(), userIDFrom(r), repository.NewSecret{
		Type: repository.SecretType(in.Type),
		Name: in.Name,
		Data: data,
		Meta: in.Meta,
	})
	if err != nil {
		h.writeSecretError(w, "create secret", err)
		return
	}

	writeJSON(w, http.StatusCreated, toDTO(sec))
}

// UpdateSecret appends a new version (data + meta) to an existing logical secret
func (h *Handlers) UpdateSecret(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if !h.validateContentTypeJSON(w, r) {
		return
	}

	id := chi.URLParam(r, "id")

	var in secretDTO
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	data, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		http.Error(w, "data must be base64", http.StatusBadRequest)
		return
	}

	sec, err := h.secrets.Update(r.Context(), userIDFrom(r), id, data, in.Meta)
	if err != nil {
		h.writeSecretError(w, "update secret", err)
		return
	}

	writeJSON(w, http.StatusOK, toDTO(sec))
}

// ListSecrets returns every row (every version of every secret) for the caller
func (h *Handlers) ListSecrets(w http.ResponseWriter, r *http.Request) {
	items, err := h.secrets.List(r.Context(), userIDFrom(r))
	if err != nil {
		h.writeSecretError(w, "list secrets", err)
		return
	}

	out := make([]secretDTO, 0, len(items))
	for _, s := range items {
		out = append(out, toDTO(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// DeleteSecret removes every version of the given logical secret
func (h *Handlers) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.secrets.Delete(r.Context(), userIDFrom(r), id); err != nil {
		h.writeSecretError(w, "delete secret", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) writeSecretError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		http.Error(w, "invalid input", http.StatusBadRequest)
	case errors.Is(err, service.ErrInvalidSecretType):
		http.Error(w, "invalid type", http.StatusBadRequest)
	case errors.Is(err, repository.ErrSecretNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	default:
		h.logger.Error(op, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
