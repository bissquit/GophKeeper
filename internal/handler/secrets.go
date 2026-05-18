package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
)

func (h *Handlers) ListSecrets(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(jwt.UserIDKey).(string)
	h.logger.Info("list secrets", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id": userID,
		"items":   []any{},
	})
}
