// Package api is a thin HTTP client
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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

func New(base, token string) *Client {
	return &Client{
		base:  base,
		http:  newHTTPClient(),
		token: token,
	}
}

func newHTTPClient() *http.Client {
	c := &http.Client{Timeout: 10 * time.Second}
	caPath := os.Getenv("GOPHKEEPER_CA")
	if caPath == "" {
		return c
	}
	pem, err := os.ReadFile(caPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot read GOPHKEEPER_CA:", err)
		return c
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		fmt.Fprintln(os.Stderr, "warning: GOPHKEEPER_CA contains no valid certificates")
		return c
	}
	c.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}
	return c
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
	in := map[string]string{"login": login, "password": plainPassword}
	var out authResponse
	if err := c.doJSON(ctx, http.MethodPost, path, in, &out); err != nil {
		return "", err
	}
	return out.Token, nil
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

// Secret is the server-side representation of a stored secret version.
// Data is automatically base64-encoded on the wire by encoding/json
type Secret struct {
	SecretItemID string    `json:"secret_item_id,omitempty"`
	ID           string    `json:"id,omitempty"`
	Type         string    `json:"type,omitempty"`
	Name         string    `json:"name,omitempty"`
	Data         []byte    `json:"data,omitempty"`
	Meta         string    `json:"meta,omitempty"`
	Version      int64     `json:"version,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// Create stores a new secret on the server and returns its first version
func (c *Client) Create(ctx context.Context, secretType, name string, data []byte, meta string) (Secret, error) {
	in := Secret{Type: secretType, Name: name, Data: data, Meta: meta}
	var out Secret
	if err := c.doJSON(ctx, http.MethodPost, "/api/secrets", in, &out); err != nil {
		return Secret{}, err
	}
	return out, nil
}

// List returns every version of every secret owned by the caller
func (c *Client) List(ctx context.Context) ([]Secret, error) {
	var out struct {
		Items []Secret `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/secrets", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// Delete removes every version of the logical secret with the given id
func (c *Client) Delete(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/secrets/"+id, nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		if out == nil || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(out)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrConflict
	default:
		return unexpectedStatus(resp)
	}
}
