// Package config loads server runtime configuration from environment variables
// and command-line flags
package config

import (
	"flag"
	"os"
)

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
		JWTSecret:  "dev-secret-change-me-please-32+chars",
	}
}

// GetConfig builds a Config by layering env vars and flags over the defaults
func GetConfig() *Config {
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

	return cfg
}
