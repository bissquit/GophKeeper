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

// ErrMissingTLS signals that TLS_CERT_FILE or TLS_KEY_FILE was not provided
var ErrMissingTLS = errors.New("TLS_CERT_FILE and TLS_KEY_FILE are required")

// Config holds the parameters required to run the GophKeeper server
type Config struct {
	ServerAddr  string
	DSN         string
	JWTSecret   string
	TLSCertFile string
	TLSKeyFile  string
}

// GetDefaultConfig returns a Config with safe development defaults
func GetDefaultConfig() *Config {
	return &Config{
		ServerAddr: ":8080",
		DSN:        "",
		JWTSecret:  "", // required
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
	if v, ok := os.LookupEnv("TLS_CERT_FILE"); ok {
		cfg.TLSCertFile = v
	}
	if v, ok := os.LookupEnv("TLS_KEY_FILE"); ok {
		cfg.TLSKeyFile = v
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "server address in host:port format")
	flag.StringVar(&cfg.DSN, "d", cfg.DSN, "PostgreSQL DSN")
	flag.Parse()

	if cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		return nil, ErrMissingTLS
	}
	return cfg, nil
}
