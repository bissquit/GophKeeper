// Package password provides helpers for bcrypt password hashing.
package password

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash returns a bcrypt hash of password using the default cost.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckHash reports whether password matches hash.
func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
