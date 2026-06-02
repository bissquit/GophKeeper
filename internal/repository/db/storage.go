// Package db is the PostgreSQL implementation of repository.Repository.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStorage implements repository.Repository against PostgreSQL
type PGStorage struct {
	pool *pgxpool.Pool
}

// NewDBStorage returns PGStorage object
func NewDBStorage(p *pgxpool.Pool) *PGStorage {
	return &PGStorage{pool: p}
}

// Ping verifies the connection pool can reach Postgres
func (s *PGStorage) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	return s.pool.Ping(ctx)
}

// CreateUser inserts a new user and returns its UUID
func (s *PGStorage) CreateUser(ctx context.Context, login, passwordHash string) (userID string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
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

	return "", err
}

// GetUserByLogin returns the user record matching login, or ErrUserNotFound
func (s *PGStorage) GetUserByLogin(ctx context.Context, login string) (repository.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
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

	return repository.User{}, err
}

// CreateSecret inserts the first version (version=1) of a new logical secret
func (s *PGStorage) CreateSecret(ctx context.Context, userID string, in repository.NewSecret) (repository.Secret, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	out := repository.Secret{
		UserID:  userID,
		Type:    in.Type,
		Name:    in.Name,
		Data:    in.Data,
		Meta:    in.Meta,
		Version: 1,
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO secrets (id, user_id, type, name, data, meta, version)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 1)
		 RETURNING secret_item_id, id, updated_at`,
		userID, string(in.Type), in.Name, in.Data, in.Meta,
	).Scan(&out.SecretItemID, &out.ID, &out.UpdatedAt)
	if err != nil {
		return repository.Secret{}, err
	}
	return out, nil
}

// appendVersionMaxAttempts caps the retry loop in AppendSecretVersion
const appendVersionMaxAttempts = 5

// AppendSecretVersion inserts a new version row for an existing logical secret.
//
// handle rare case when user tries to update secret at the same time. Return
// UniqueViolation (because UNIQUE (id, version)) when it happens and then next try.
// In worst case we'll receive 'contention' error - not a tragedy
func (s *PGStorage) AppendSecretVersion(ctx context.Context, userID, id string, data []byte, meta string) (repository.Secret, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for attempt := 0; attempt < appendVersionMaxAttempts; attempt++ {
		out, err := s.appendVersionOnce(ctx, userID, id, data, meta)
		if err == nil {
			return out, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			continue
		}
		return repository.Secret{}, err
	}
	return repository.Secret{}, fmt.Errorf("append version: contention after %d attempts", appendVersionMaxAttempts)
}

func (s *PGStorage) appendVersionOnce(ctx context.Context, userID, id string, data []byte, meta string) (repository.Secret, error) {
	out := repository.Secret{ID: id, UserID: userID, Data: data, Meta: meta}
	var t string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO secrets (id, user_id, type, name, data, meta, version)
		 SELECT id, user_id, type, name, $3, $4, MAX(version)+1
		 FROM secrets WHERE id = $1 AND user_id = $2
		 GROUP BY id, user_id, type, name
		 RETURNING secret_item_id, type, name, version, updated_at`,
		id, userID, data, meta,
	).Scan(&out.SecretItemID, &t, &out.Name, &out.Version, &out.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Secret{}, repository.ErrSecretNotFound
		}
		return repository.Secret{}, err
	}
	out.Type = repository.SecretType(t)
	return out, nil
}

// ListSecrets returns every row owned by userID — all versions of all secrets
func (s *PGStorage) ListSecrets(ctx context.Context, userID string) ([]repository.Secret, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT secret_item_id, id, type, name, data, meta, version, updated_at
		 FROM secrets WHERE user_id = $1
		 ORDER BY id, version`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []repository.Secret
	for rows.Next() {
		var sec repository.Secret
		var t string
		if err := rows.Scan(&sec.SecretItemID, &sec.ID, &t, &sec.Name, &sec.Data, &sec.Meta, &sec.Version, &sec.UpdatedAt); err != nil {
			return nil, err
		}
		sec.Type = repository.SecretType(t)
		sec.UserID = userID
		out = append(out, sec)
	}
	return out, rows.Err()
}

// DeleteSecret removes every version of the logical secret id owned by userID
func (s *PGStorage) DeleteSecret(ctx context.Context, userID, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		"DELETE FROM secrets WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrSecretNotFound
	}
	return nil
}
