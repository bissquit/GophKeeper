package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// newTestHandlers builds Handlers backed by fresh in-memory fakes
func newTestHandlers() (*Handlers, *fakeAuth, *fakeSecrets) {
	fa := newFakeAuth()
	fs := newFakeSecrets()
	return NewHandlers(fa, fs, discardLogger()), fa, fs
}

func newErrAuthHandlers() *Handlers {
	return &Handlers{auth: &fakeAuth{fail: true}, logger: discardLogger()}
}

func TestRegister_OK(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"login":"alice","password":"p4ssw0rd"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == "" {
		t.Fatal("empty token in response")
	}
	if w.Header().Get("Authorization") == "" {
		t.Fatal("missing Authorization header")
	}
}

func TestRegister_DuplicateLogin(t *testing.T) {
	h, fa, _ := newTestHandlers()
	_, _ = fa.Register(context.Background(), "alice", "p")

	body := strings.NewReader(`{"login":"alice","password":"p"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestRegister_BadContentType(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(nil))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegister_BadJSON(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegister_EmptyLogin(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"login":"","password":"p"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegister_ServiceFailure(t *testing.T) {
	h := newErrAuthHandlers()

	body := strings.NewReader(`{"login":"a","password":"p"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/register", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLogin_OK(t *testing.T) {
	h, fa, _ := newTestHandlers()
	_, _ = fa.Register(context.Background(), "alice", "p4ssw0rd")

	body := strings.NewReader(`{"login":"alice","password":"p4ssw0rd"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h, fa, _ := newTestHandlers()
	_, _ = fa.Register(context.Background(), "alice", "p4ssw0rd")

	body := strings.NewReader(`{"login":"alice","password":"wrong"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_BadJSON(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"login":"a","password":""}`)
	r := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
