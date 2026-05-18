// Package db is the PostgreSQL implementation of repository.Repository.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStorage implements repository.Repository against PostgreSQL via pgxpool.
type PGStorage struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewDBStorage wires a pgxpool and slog logger into a PGStorage value.
func NewDBStorage(p *pgxpool.Pool, l *slog.Logger) *PGStorage {
	return &PGStorage{pool: p, logger: l}
}

// CreateUser inserts a new user and returns its UUID. It maps PostgreSQL's
// unique-violation error to repository.ErrUserAlreadyExists.
func (s *PGStorage) CreateUser(login, passwordHash string) (userID string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = s.pool.QueryRow(ctx,
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id",
		login, passwordHash,
	).Scan(&userID)

	if err == nil {
		return userID, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return "", fmt.Errorf("%w: %s", repository.ErrUserAlreadyExists, login)
	}

	s.logger.Error("create user error", "err", err)
	return "", err
}

// GetUserByLogin returns the user record matching login, or ErrUserNotFound.
func (s *PGStorage) GetUserByLogin(login string) (repository.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u repository.User
	err := s.pool.QueryRow(ctx,
		"SELECT id, login, password_hash FROM users WHERE login = $1",
		login,
	).Scan(&u.ID, &u.Login, &u.PasswordHash)

	if err == nil {
		return u, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return repository.User{}, repository.ErrUserNotFound
	}

	s.logger.Error("get user by login error", "err", err, "login", login)
	return repository.User{}, err
}
