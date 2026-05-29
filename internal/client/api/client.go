// Package api is a thin HTTP client
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrUnauthorized is returned for 401 responses
var ErrUnauthorized = errors.New("unauthorized")

// ErrConflict is returned for 409 responses (e.g., login already taken)
var ErrConflict = errors.New("conflict")

// ErrNotFound is returned for 404 responses
var ErrNotFound = errors.New("not found")

// Client is a server API client
type Client struct {
	base  string
	http  *http.Client
	token string
}

// New builds a Client targeting base URL with optional bearer token
func New(base, token string) *Client {
	return &Client{
		base:  base,
		http:  &http.Client{Timeout: 10 * time.Second},
		token: token,
	}
}

type authResponse struct {
	Token string `json:"token"`
}

// Register creates a new user and returns JWT
func (c *Client) Register(ctx context.Context, login, plainPassword string) (string, error) {
	return c.authCall(ctx, "/api/user/register", login, plainPassword)
}

// Login authenticates an existing user and returns JWT
func (c *Client) Login(ctx context.Context, login, plainPassword string) (string, error) {
	return c.authCall(ctx, "/api/user/login", login, plainPassword)
}

func (c *Client) authCall(ctx context.Context, path, login, plainPassword string) (string, error) {
	body, _ := json.Marshal(map[string]string{"login": login, "password": plainPassword})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var out authResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "", err
		}
		return out.Token, nil
	case http.StatusConflict:
		return "", ErrConflict
	case http.StatusUnauthorized:
		return "", ErrUnauthorized
	default:
		return "", unexpectedStatus(resp)
	}
}

// Ping checks server health (no auth)
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return unexpectedStatus(resp)
	}
	return nil
}

func unexpectedStatus(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bytes.TrimSpace(b))
}
