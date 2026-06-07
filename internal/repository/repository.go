// Package repository defines storage-agnostic domain types and the contract
package repository

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrUserAlreadyExists is returned when a registration uses a taken login
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrUserNotFound is returned when no user matches the given login
	ErrUserNotFound = errors.New("user not found")
	// ErrSecretNotFound is returned when no secret matches the given id for a user
	ErrSecretNotFound = errors.New("secret not found")
)

// User is the persisted user record
type User struct {
	ID           string
	Login        string
	PasswordHash string
}

// SecretType enumerates the kinds of payloads GophKeeper can store
type SecretType string

const (
	// SecretTypeCredentials marks a login/password pair
	SecretTypeCredentials SecretType = "credentials"
	// SecretTypeText marks an arbitrary text blob
	SecretTypeText SecretType = "text"
	// SecretTypeBinary marks an arbitrary binary blob
	SecretTypeBinary SecretType = "binary"
	// SecretTypeCard marks a bank card record
	SecretTypeCard SecretType = "card"
)

// Secret is one row of the append-only secrets table — a single version of a logical secret
type Secret struct {
	SecretItemID string
	ID           string
	UserID       string
	Type         SecretType
	Name         string
	Data         []byte
	Meta         string
	Version      int64
	UpdatedAt    time.Time
}

// NewSecret is the data required to create the first version of a secret
type NewSecret struct {
	Type SecretType
	Name string
	Data []byte
	Meta string
}

// Repository is the contract that any concrete storage backend must satisfy
type Repository interface {
	// Ping verifies the storage is reachable
	Ping(ctx context.Context) error

	CreateUser(ctx context.Context, login, passwordHash string) (userID string, err error)
	GetUserByLogin(ctx context.Context, login string) (user User, err error)

	CreateSecret(ctx context.Context, userID string, in NewSecret) (secret Secret, err error)
	AppendSecretVersion(ctx context.Context, userID, id string, data []byte, meta string) (secret Secret, err error)
	ListSecrets(ctx context.Context, userID string) ([]Secret, error)
	DeleteSecret(ctx context.Context, userID, id string) error
}
