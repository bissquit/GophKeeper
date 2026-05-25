package service

import (
	"fmt"
	"time"

	"github.com/bissquit/gophkeeper/internal/repository"
)

// fakeRepo is an in-memory repository.Repository used by the service tests
type fakeRepo struct {
	users   map[string]repository.User
	secrets map[string][]repository.Secret
	next    int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:   make(map[string]repository.User),
		secrets: make(map[string][]repository.Secret),
	}
}

func (r *fakeRepo) nextID(prefix string) string {
	r.next++
	return fmt.Sprintf("%s-%d", prefix, r.next)
}

func (r *fakeRepo) CreateUser(login, passwordHash string) (string, error) {
	if _, ok := r.users[login]; ok {
		return "", repository.ErrUserAlreadyExists
	}
	id := r.nextID("user")
	r.users[login] = repository.User{ID: id, Login: login, PasswordHash: passwordHash}
	return id, nil
}

func (r *fakeRepo) GetUserByLogin(login string) (repository.User, error) {
	u, ok := r.users[login]
	if !ok {
		return repository.User{}, repository.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeRepo) CreateSecret(userID string, in repository.NewSecret) (repository.Secret, error) {
	sec := repository.Secret{
		SecretItemID: r.nextID("item"),
		ID:           r.nextID("sec"),
		UserID:       userID,
		Type:         in.Type,
		Name:         in.Name,
		Data:         in.Data,
		Meta:         in.Meta,
		Version:      1,
		UpdatedAt:    time.Now(),
	}
	r.secrets[userID] = append(r.secrets[userID], sec)
	return sec, nil
}

func (r *fakeRepo) AppendSecretVersion(userID, id string, data []byte, meta string) (repository.Secret, error) {
	var latest *repository.Secret
	for i := range r.secrets[userID] {
		s := &r.secrets[userID][i]
		if s.ID == id && (latest == nil || s.Version > latest.Version) {
			latest = s
		}
	}
	if latest == nil {
		return repository.Secret{}, repository.ErrSecretNotFound
	}
	sec := repository.Secret{
		SecretItemID: r.nextID("item"),
		ID:           id,
		UserID:       userID,
		Type:         latest.Type,
		Name:         latest.Name,
		Data:         data,
		Meta:         meta,
		Version:      latest.Version + 1,
		UpdatedAt:    time.Now(),
	}
	r.secrets[userID] = append(r.secrets[userID], sec)
	return sec, nil
}

func (r *fakeRepo) ListSecrets(userID string) ([]repository.Secret, error) {
	return r.secrets[userID], nil
}

func (r *fakeRepo) DeleteSecret(userID, id string) error {
	out := r.secrets[userID][:0]
	found := false
	for _, s := range r.secrets[userID] {
		if s.ID == id {
			found = true
			continue
		}
		out = append(out, s)
	}
	if !found {
		return repository.ErrSecretNotFound
	}
	r.secrets[userID] = out
	return nil
}
