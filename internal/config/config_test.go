package config

import (
	"errors"
	"flag"
	"os"
	"testing"
)

// resetFlags isolates GetConfig from the test binary's own flags
// (which include -test.* by default and would otherwise break flag.Parse)
func resetFlags(t *testing.T) {
	t.Helper()
	oldArgs := os.Args
	oldFlag := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlag
	})
	os.Args = []string{"gophkeeper-test"}
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
}

func TestGetConfig_MissingJWTSecret(t *testing.T) {
	resetFlags(t)
	t.Setenv("JWT_SECRET", "")
	if _, err := GetConfig(); !errors.Is(err, ErrMissingJWTSecret) {
		t.Fatalf("expected ErrMissingJWTSecret, got %v", err)
	}
}

func TestGetConfig_EnvApplied(t *testing.T) {
	resetFlags(t)
	t.Setenv("RUN_ADDRESS", ":9999")
	t.Setenv("DATABASE_URI", "postgres://x")
	t.Setenv("JWT_SECRET", "supersecret")

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.ServerAddr != ":9999" || cfg.DSN != "postgres://x" || cfg.JWTSecret != "supersecret" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	d := GetDefaultConfig()
	if d.ServerAddr != ":8080" || d.JWTSecret != "" {
		t.Fatalf("unexpected defaults: %+v", d)
	}
}
