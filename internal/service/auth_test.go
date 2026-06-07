package service

import (
	"context"
	"errors"
	"testing"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
)

var jwtSecret = []byte("test-secret-must-be-long-enough!!")

func TestAuth_RegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	a := NewAuth(repo, jwtSecret)

	token, err := a.Register(ctx, "alice", "p4ssw0rd")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	claims, err := jwt.ParseToken(token, jwtSecret)
	if err != nil || claims.Login != "alice" {
		t.Fatalf("token does not parse to alice: %v", err)
	}

	if _, err := a.Login(ctx, "alice", "p4ssw0rd"); err != nil {
		t.Fatalf("Login with correct password: %v", err)
	}
}

func TestAuth_RegisterDuplicate(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	a := NewAuth(repo, jwtSecret)
	_, _ = a.Register(ctx, "alice", "p4ssw0rd")

	_, err := a.Register(ctx, "alice", "p4ssw0rd")
	if !errors.Is(err, ErrLoginTaken) {
		t.Fatalf("expected ErrLoginTaken, got %v", err)
	}
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	a := NewAuth(repo, jwtSecret)
	_, _ = a.Register(ctx, "alice", "p4ssw0rd")

	_, err := a.Login(ctx, "alice", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuth_LoginUnknownUser(t *testing.T) {
	a := NewAuth(newFakeRepo(), jwtSecret)
	_, err := a.Login(context.Background(), "ghost", "x")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuth_EmptyInput(t *testing.T) {
	ctx := context.Background()
	a := NewAuth(newFakeRepo(), jwtSecret)
	if _, err := a.Register(ctx, "", "x"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Register empty login: %v", err)
	}
	if _, err := a.Login(ctx, "x", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Login empty password: %v", err)
	}
}
