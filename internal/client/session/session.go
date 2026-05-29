// Package session persists the current GophKeeper client session to a JSON
// file in the user's config directory.
package session

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotLoggedIn is returned by Load when no session file exists
var ErrNotLoggedIn = errors.New("not logged in")

// Session is the on-disk client session
type Session struct {
	Server string `json:"server"`
	Login  string `json:"login"`
	Token  string `json:"token"`
}

// Path returns the on-disk session file path
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gophkeeper", "session.json"), nil
}

// Load reads and decodes the session file, or returns ErrNotLoggedIn
func Load() (*Session, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotLoggedIn
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the session to disk with mode 0600
func Save(s *Session) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0600)
}

// Clear removes the session file
func Clear() error {
	p, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
