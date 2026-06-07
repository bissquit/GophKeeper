package logging

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger_CallsNextAndCapturesStatus(t *testing.T) {
	mw := Logger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hi", nil)
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestLogger_DefaultsStatusToOK(t *testing.T) {
	mw := Logger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// next handler writes a body without calling WriteHeader explicitly
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("x"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
