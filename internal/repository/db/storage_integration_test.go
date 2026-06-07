//go:build integration

// Integration tests for PGStorage. They spin up a real Postgres in a
// throwaway Docker container via testcontainers-go. Excluded from the default
// test run by the build tag; enable with: go test -tags=integration ./...
package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testStorage *PGStorage

func TestMain(m *testing.M) {
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("gophkeeper"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pg.Terminate(ctx)
		log.Fatalf("dsn: %v", err)
	}

	s, err := Open(ctx, dsn)
	if err != nil {
		_ = pg.Terminate(ctx)
		log.Fatalf("open: %v", err)
	}
	testStorage = s

	code := m.Run()

	s.Close()
	_ = pg.Terminate(ctx)
	os.Exit(code)
}

func uniqLogin(t *testing.T) string {
	return fmt.Sprintf("u_%s", t.Name())
}

func TestPing(t *testing.T) {
	if err := testStorage.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestCreateUser_And_GetUserByLogin(t *testing.T) {
	ctx := context.Background()
	login := uniqLogin(t)

	uid, err := testStorage.CreateUser(ctx, login, "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if uid == "" {
		t.Fatal("empty uid")
	}

	got, err := testStorage.GetUserByLogin(ctx, login)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != uid || got.Login != login || got.PasswordHash != "hash" {
		t.Fatalf("unexpected user: %+v", got)
	}

	// duplicate
	_, err = testStorage.CreateUser(ctx, login, "hash")
	if !errors.Is(err, repository.ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestGetUserByLogin_NotFound(t *testing.T) {
	_, err := testStorage.GetUserByLogin(context.Background(), uniqLogin(t))
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSecretLifecycle(t *testing.T) {
	ctx := context.Background()
	uid, err := testStorage.CreateUser(ctx, uniqLogin(t), "h")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// create
	sec, err := testStorage.CreateSecret(ctx, uid, repository.NewSecret{
		Type: repository.SecretTypeText, Name: "n", Data: []byte("v1"), Meta: "m",
	})
	if err != nil || sec.Version != 1 || sec.ID == "" {
		t.Fatalf("create secret: %+v err=%v", sec, err)
	}

	// append version
	v2, err := testStorage.AppendSecretVersion(ctx, uid, sec.ID, []byte("v2"), "m2")
	if err != nil || v2.Version != 2 {
		t.Fatalf("append: %+v err=%v", v2, err)
	}

	// list — both versions for our user
	items, err := testStorage.ListSecrets(ctx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(items))
	}

	// delete all versions, second delete is a not-found
	if err := testStorage.DeleteSecret(ctx, uid, sec.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := testStorage.DeleteSecret(ctx, uid, sec.ID); !errors.Is(err, repository.ErrSecretNotFound) {
		t.Fatalf("second delete: expected ErrSecretNotFound, got %v", err)
	}
}
