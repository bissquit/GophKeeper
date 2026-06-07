package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeServer routes by method+path to handlers supplied by the test
type route struct {
	method, path string
}

func newServer(t *testing.T, handlers map[route]http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h, ok := handlers[route{req.Method, req.URL.Path}]
		if !ok {
			http.Error(w, "no route: "+req.Method+" "+req.URL.Path, http.StatusMethodNotAllowed)
			return
		}
		h(w, req)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRegister_OK(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodPost, "/api/user/register"}: func(w http.ResponseWriter, r *http.Request) {
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type=%q", ct)
			}
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["login"] != "alice" || in["password"] != "pw" {
				t.Errorf("bad body: %+v", in)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok-1"})
		},
	})
	tok, err := New(srv.URL, "").Register(context.Background(), "alice", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tok != "tok-1" {
		t.Fatalf("token=%q", tok)
	}
}

func TestRegister_Conflict(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodPost, "/api/user/register"}: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "taken", http.StatusConflict)
		},
	})
	_, err := New(srv.URL, "").Register(context.Background(), "a", "p")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodPost, "/api/user/login"}: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "no", http.StatusUnauthorized)
		},
	})
	_, err := New(srv.URL, "").Login(context.Background(), "a", "p")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPing_OK(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodGet, "/ping"}: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})
	if err := New(srv.URL, "").Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestCreate_SendsBearerAndBytes(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodPost, "/api/secrets"}: func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
				t.Errorf("Authorization=%q", got)
			}
			var in Secret
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if in.Type != "text" || in.Name != "note" || !bytes.Equal(in.Data, []byte("ct")) {
				t.Errorf("bad in: %+v", in)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Secret{ID: "id-1", Type: in.Type, Name: in.Name, Version: 1})
		},
	})
	sec, err := New(srv.URL, "tok-1").Create(context.Background(), "text", "note", []byte("ct"), "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sec.ID != "id-1" || sec.Version != 1 {
		t.Fatalf("bad response: %+v", sec)
	}
}

func TestList_OK(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodGet, "/api/secrets"}: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []Secret{{ID: "a", Name: "n1"}, {ID: "b", Name: "n2"}},
			})
		},
	})
	items, err := New(srv.URL, "tok").List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 || items[0].Name != "n1" {
		t.Fatalf("bad items: %+v", items)
	}
}

func TestDelete_OK_And_NotFound(t *testing.T) {
	var deleted string
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodDelete, "/api/secrets/a"}: func(w http.ResponseWriter, r *http.Request) {
			deleted = "a"
			w.WriteHeader(http.StatusNoContent)
		},
		{http.MethodDelete, "/api/secrets/missing"}: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "no", http.StatusNotFound)
		},
	})
	c := New(srv.URL, "tok")
	if err := c.Delete(context.Background(), "a"); err != nil {
		t.Fatalf("delete a: %v", err)
	}
	if deleted != "a" {
		t.Fatalf("server did not see delete")
	}
	if err := c.Delete(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUnexpectedStatus(t *testing.T) {
	srv := newServer(t, map[route]http.HandlerFunc{
		{http.MethodGet, "/api/secrets"}: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		},
	})
	_, err := New(srv.URL, "tok").List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
