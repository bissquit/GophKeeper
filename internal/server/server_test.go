package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bissquit/gophkeeper/internal/config"
	"github.com/bissquit/gophkeeper/internal/repository"
)

// stubRepo is a Repository that returns zero values
type stubRepo struct{}

func (stubRepo) CreateUser(context.Context, string, string) (string, error) {
	return "uid", nil
}
func (stubRepo) GetUserByLogin(context.Context, string) (repository.User, error) {
	return repository.User{}, repository.ErrUserNotFound
}
func (stubRepo) CreateSecret(context.Context, string, repository.NewSecret) (repository.Secret, error) {
	return repository.Secret{}, nil
}
func (stubRepo) AppendSecretVersion(context.Context, string, string, []byte, string) (repository.Secret, error) {
	return repository.Secret{}, repository.ErrUserNotFound
}
func (stubRepo) ListSecrets(context.Context, string) ([]repository.Secret, error) {
	return nil, nil
}
func (stubRepo) DeleteSecret(context.Context, string, string) error {
	return nil
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(cfg, stubRepo{}, nil, logger).Handler()
}

func TestRoute_RegisterWired(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register",
		strings.NewReader(`{"login":"a","password":"p"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// stubRepo returns ("uid", nil), so auth.Register succeeds and we expect 200
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoute_SecretsRequireAuth(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}
