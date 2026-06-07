package service

import (
	"context"
	"errors"
	"testing"

	"github.com/bissquit/gophkeeper/internal/repository"
)

func TestSecrets_CreateAndList(t *testing.T) {
	ctx := context.Background()
	s := NewSecrets(newFakeRepo())

	sec, err := s.Create(ctx, "u1", repository.NewSecret{
		Type: repository.SecretTypeText,
		Name: "note",
		Data: []byte("hello"),
		Meta: "m1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sec.Version != 1 || sec.ID == "" {
		t.Fatalf("unexpected secret: %+v", sec)
	}

	items, err := s.List(ctx, "u1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].ID != sec.ID {
		t.Fatalf("List mismatch: %+v", items)
	}
}

func TestSecrets_CreateInvalidType(t *testing.T) {
	s := NewSecrets(newFakeRepo())

	_, err := s.Create(context.Background(), "u1", repository.NewSecret{Type: "bogus", Name: "x"})
	if !errors.Is(err, ErrInvalidSecretType) {
		t.Fatalf("expected ErrInvalidSecretType, got %v", err)
	}
}

func TestSecrets_UpdateAppendsVersion(t *testing.T) {
	ctx := context.Background()
	s := NewSecrets(newFakeRepo())
	sec, _ := s.Create(ctx, "u1", repository.NewSecret{
		Type: repository.SecretTypeText,
		Name: "note",
		Data: []byte("v1"),
	})

	updated, err := s.Update(ctx, "u1", sec.ID, []byte("v2"), "second")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Version != 2 || string(updated.Data) != "v2" {
		t.Fatalf("unexpected updated: %+v", updated)
	}

	items, _ := s.List(ctx, "u1")
	if len(items) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(items))
	}
}

func TestSecrets_UpdateNotFound(t *testing.T) {
	s := NewSecrets(newFakeRepo())
	_, err := s.Update(context.Background(), "u1", "ghost-id", []byte("x"), "")
	if !errors.Is(err, repository.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSecrets_Delete(t *testing.T) {
	ctx := context.Background()
	s := NewSecrets(newFakeRepo())
	sec, _ := s.Create(ctx, "u1", repository.NewSecret{Type: repository.SecretTypeText, Name: "x"})

	if err := s.Delete(ctx, "u1", sec.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete(ctx, "u1", sec.ID); !errors.Is(err, repository.ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound on second delete, got %v", err)
	}
}

func TestSecrets_EmptyInput(t *testing.T) {
	ctx := context.Background()
	s := NewSecrets(newFakeRepo())
	if _, err := s.Create(ctx, "", repository.NewSecret{Type: repository.SecretTypeText, Name: "x"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create empty userID: %v", err)
	}
	if err := s.Delete(ctx, "u1", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Delete empty id: %v", err)
	}
}
