// Package service contains transport-agnostic business logic
package service

import (
	"context"
	"errors"

	"github.com/bissquit/gophkeeper/internal/auth/jwt"
	"github.com/bissquit/gophkeeper/internal/password"
	"github.com/bissquit/gophkeeper/internal/repository"
)

var (
	// ErrLoginTaken is returned by Auth.Register when the login is already used.
	ErrLoginTaken = errors.New("login already taken")
	// ErrInvalidCredentials is returned by Auth.Login when login or password is wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidInput is returned when required input fields are missing/empty.
	ErrInvalidInput = errors.New("invalid input")
)

// Auth implements registration, password verification, and JWT issuance
type Auth struct {
	repo      repository.Repository
	jwtSecret []byte
}

// NewAuth wires the dependencies for the authentication service
func NewAuth(repo repository.Repository, jwtSecret []byte) *Auth {
	return &Auth{repo: repo, jwtSecret: jwtSecret}
}

// Register creates a new user with a bcrypt-hashed password and returns JWT
func (a *Auth) Register(ctx context.Context, login, plainPassword string) (token string, err error) {
	if login == "" || plainPassword == "" {
		return "", ErrInvalidInput
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return "", err
	}

	userID, err := a.repo.CreateUser(ctx, login, hash)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return "", ErrLoginTaken
		}
		return "", err
	}

	return jwt.GenerateToken(userID, login, a.jwtSecret)
}

// Login verifies the credentials and returns JWT
func (a *Auth) Login(ctx context.Context, login, plainPassword string) (token string, err error) {
	if login == "" || plainPassword == "" {
		return "", ErrInvalidInput
	}

	u, err := a.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if !password.CheckHash(plainPassword, u.PasswordHash) {
		return "", ErrInvalidCredentials
	}

	return jwt.GenerateToken(u.ID, u.Login, a.jwtSecret)
}
