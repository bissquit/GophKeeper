package service

import (
	"errors"

	"github.com/bissquit/gophkeeper/internal/repository"
)

var ErrInvalidSecretType = errors.New("invalid secret type")

// Secrets implements CRUD over the versioned secret store
type Secrets struct {
	repo repository.Repository
}

// NewSecrets returns a Secrets service backed by repo
func NewSecrets(repo repository.Repository) *Secrets {
	return &Secrets{repo: repo}
}

// Create stores the first version of a new logical secret.
func (s *Secrets) Create(userID string, in repository.NewSecret) (repository.Secret, error) {
	if userID == "" || in.Name == "" {
		return repository.Secret{}, ErrInvalidInput
	}
	if !isValidSecretType(in.Type) {
		return repository.Secret{}, ErrInvalidSecretType
	}
	return s.repo.CreateSecret(userID, in)
}

// Update appends a new version (data + meta) to an existing logical secret
func (s *Secrets) Update(userID, id string, data []byte, meta string) (repository.Secret, error) {
	if userID == "" || id == "" {
		return repository.Secret{}, ErrInvalidInput
	}
	return s.repo.AppendSecretVersion(userID, id, data, meta)
}

// List returns every stored row for the user (every version of every secret)
func (s *Secrets) List(userID string) ([]repository.Secret, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.ListSecrets(userID)
}

// Delete removes every version of the logical secret id
func (s *Secrets) Delete(userID, id string) error {
	if userID == "" || id == "" {
		return ErrInvalidInput
	}
	return s.repo.DeleteSecret(userID, id)
}

func isValidSecretType(t repository.SecretType) bool {
	switch t {
	case repository.SecretTypeCredentials,
		repository.SecretTypeText,
		repository.SecretTypeBinary,
		repository.SecretTypeCard:
		return true
	}
	return false
}
