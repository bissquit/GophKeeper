package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testSecret = []byte("test-secret-must-be-long-must-be-long!!")

func TestGenerateAndParse(t *testing.T) {
	token, err := GenerateToken("u1", "alice", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != "u1" || claims.Login != "alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseWithWrongSecret(t *testing.T) {
	token, _ := GenerateToken("u1", "alice", testSecret)
	if _, err := ParseToken(token, []byte("other-secret")); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := UserIDFromContext(r.Context())
		if uid != "u1" {
			t.Errorf("user_id not in context: %q", uid)
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := JWT(testSecret)(next)

	t.Run("valid token", func(t *testing.T) {
		token, _ := GenerateToken("u1", "alice", testSecret)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("malformed header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Token abc")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}
