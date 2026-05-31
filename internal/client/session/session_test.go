package session

import (
	"errors"
	"os"
	"runtime"
	"testing"
)

// isolate redirects os.UserConfigDir() to a fresh tmp dir for the test
func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	isolate(t)
	want := &Session{Server: "http://srv", Login: "alice", Token: "tok"}
	if err := Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if *got != *want {
		t.Fatalf("round-trip mismatch: got %+v", got)
	}

	p, _ := Path()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := st.Mode().Perm(); mode != 0600 {
		t.Fatalf("expected mode 0600, got %o", mode)
	}
}

func TestLoad_NotLoggedIn(t *testing.T) {
	isolate(t)
	if _, err := Load(); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("expected ErrNotLoggedIn, got %v", err)
	}
}

func TestClear(t *testing.T) {
	isolate(t)
	if err := Clear(); err != nil {
		t.Fatalf("Clear on empty must be a no-op, got %v", err)
	}

	if err := Save(&Session{Login: "alice"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := Load(); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("after Clear, Load must return ErrNotLoggedIn, got %v", err)
	}
}
