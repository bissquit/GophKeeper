package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/go-chi/chi/v5"
)

func withUser(r *http.Request, userID string) *http.Request {
	return r.WithContext(jwt.ContextWithUserID(r.Context(), userID))
}

func withChiID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func newErrSecretsHandlers() *Handlers {
	return &Handlers{secrets: &fakeSecrets{fail: true}, logger: discardLogger()}
}

func TestCreateSecret_OK(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"type":"text","name":"note","data":"aGVsbG8=","meta":"m"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp secretDTO
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID == "" || resp.Version != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateSecret_InvalidType(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"type":"bogus","name":"x","data":""}`)
	r := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSecret_BadContentType(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "text/plain")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSecret_BadJSON(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSecret_BadBase64(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"type":"text","name":"n","data":"!!!"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateSecret_ServiceFailure(t *testing.T) {
	h := newErrSecretsHandlers()

	body := strings.NewReader(`{"type":"text","name":"n","data":""}`)
	r := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.CreateSecret(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListSecrets_ReturnsCreated(t *testing.T) {
	h, _, fs := newTestHandlers()
	_, _ = fs.Create(context.Background(), "u1", repository.NewSecret{
		Type: repository.SecretTypeText, Name: "n", Data: []byte("x"),
	})

	r := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.ListSecrets(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Items []secretDTO `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
}

func TestListSecrets_ServiceFailure(t *testing.T) {
	h := newErrSecretsHandlers()

	r := httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	r = withUser(r, "u1")
	w := httptest.NewRecorder()
	h.ListSecrets(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestUpdateSecret_AppendsVersion(t *testing.T) {
	h, _, fs := newTestHandlers()
	sec, _ := fs.Create(context.Background(), "u1", repository.NewSecret{
		Type: repository.SecretTypeText, Name: "n", Data: []byte("v1"),
	})

	body := strings.NewReader(`{"data":"djI=","meta":"second"}`) // "v2"
	r := httptest.NewRequest(http.MethodPut, "/api/secrets/"+sec.ID, body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	r = withChiID(r, sec.ID)
	w := httptest.NewRecorder()
	h.UpdateSecret(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp secretDTO
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Version != 2 {
		t.Fatalf("expected version 2, got %d", resp.Version)
	}
}

func TestUpdateSecret_NotFound(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"data":"","meta":""}`)
	r := httptest.NewRequest(http.MethodPut, "/api/secrets/ghost", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	r = withChiID(r, "ghost")
	w := httptest.NewRecorder()
	h.UpdateSecret(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateSecret_BadContentType(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPut, "/api/secrets/x", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "text/plain")
	r = withUser(r, "u1")
	r = withChiID(r, "x")
	w := httptest.NewRecorder()
	h.UpdateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateSecret_BadJSON(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodPut, "/api/secrets/x", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	r = withChiID(r, "x")
	w := httptest.NewRecorder()
	h.UpdateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateSecret_BadBase64(t *testing.T) {
	h, _, _ := newTestHandlers()

	body := strings.NewReader(`{"data":"!!!","meta":""}`)
	r := httptest.NewRequest(http.MethodPut, "/api/secrets/x", body)
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, "u1")
	r = withChiID(r, "x")
	w := httptest.NewRecorder()
	h.UpdateSecret(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteSecret_OK(t *testing.T) {
	h, _, fs := newTestHandlers()
	sec, _ := fs.Create(context.Background(), "u1", repository.NewSecret{
		Type: repository.SecretTypeText, Name: "n", Data: []byte("v1"),
	})

	r := httptest.NewRequest(http.MethodDelete, "/api/secrets/"+sec.ID, nil)
	r = withUser(r, "u1")
	r = withChiID(r, sec.ID)
	w := httptest.NewRecorder()
	h.DeleteSecret(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	h, _, _ := newTestHandlers()

	r := httptest.NewRequest(http.MethodDelete, "/api/secrets/ghost", nil)
	r = withUser(r, "u1")
	r = withChiID(r, "ghost")
	w := httptest.NewRecorder()
	h.DeleteSecret(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteSecret_ServiceFailure(t *testing.T) {
	h := newErrSecretsHandlers()

	r := httptest.NewRequest(http.MethodDelete, "/api/secrets/x", nil)
	r = withUser(r, "u1")
	r = withChiID(r, "x")
	w := httptest.NewRecorder()
	h.DeleteSecret(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
