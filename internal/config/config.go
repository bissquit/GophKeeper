// Package config loads server runtime configuration from environment variables
// and command-line flags
package config

import (
	"errors"
	"flag"
	"os"
)

// ErrMissingJWTSecret signals that JWT_SECRET was not provided
var ErrMissingJWTSecret = errors.New("JWT_SECRET is required")

// Config holds the parameters required to run the GophKeeper server
type Config struct {
	ServerAddr string
	DSN        string
	JWTSecret  string
}

// GetDefaultConfig returns a Config with safe development defaults
func GetDefaultConfig() *Config {
	return &Config{
		ServerAddr: ":8080",
		DSN:        "",
		JWTSecret:  "", // required by default
	}
}

// GetConfig builds a Config by layering env vars and flags over the defaults
func GetConfig() (*Config, error) {
	cfg := GetDefaultConfig()

	if v, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.ServerAddr = v
	}
	if v, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DSN = v
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		cfg.JWTSecret = v
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address in host:port format")
	flag.StringVar(&cfg.DSN, "d", cfg.DSN, "PostgreSQL DSN")
	flag.Parse()

	if cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	return cfg, nil
}
