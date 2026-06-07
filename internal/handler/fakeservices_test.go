package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bissquit/gophkeeper/internal/repository"
	"github.com/bissquit/gophkeeper/internal/service"
)

// errTaDa is a generic non-sentinel error used to exercise the default 500 branch
var errTaDa = errors.New("ta-da")

// fakeAuth is an in-memory authService for handler tests
type fakeAuth struct {
	users map[string]string // login -> password
	fail  bool              // when true, all methods return errTaDa
}

func newFakeAuth() *fakeAuth {
	return &fakeAuth{users: map[string]string{}}
}

func (a *fakeAuth) Register(_ context.Context, login, plainPassword string) (string, error) {
	if a.fail {
		return "", errTaDa
	}
	if login == "" || plainPassword == "" {
		return "", service.ErrInvalidInput
	}
	if _, ok := a.users[login]; ok {
		return "", service.ErrLoginTaken
	}
	a.users[login] = plainPassword
	return "fake-token-" + login, nil
}

func (a *fakeAuth) Login(_ context.Context, login, plainPassword string) (string, error) {
	if a.fail {
		return "", errTaDa
	}
	if login == "" || plainPassword == "" {
		return "", service.ErrInvalidInput
	}
	p, ok := a.users[login]
	if !ok || p != plainPassword {
		return "", service.ErrInvalidCredentials
	}
	return "fake-token-" + login, nil
}

// fakeSecrets is an in-memory secretsService for handler tests
type fakeSecrets struct {
	next  int
	items []repository.Secret
	fail  bool
}

func newFakeSecrets() *fakeSecrets { return &fakeSecrets{} }

func (s *fakeSecrets) Create(_ context.Context, userID string, in repository.NewSecret) (repository.Secret, error) {
	if s.fail {
		return repository.Secret{}, errTaDa
	}
	if userID == "" || in.Name == "" {
		return repository.Secret{}, service.ErrInvalidInput
	}
	if !service.IsValidSecretType(in.Type) {
		return repository.Secret{}, service.ErrInvalidSecretType
	}
	s.next++
	sec := repository.Secret{
		SecretItemID: fmt.Sprintf("item-%d", s.next),
		ID:           fmt.Sprintf("sec-%d", s.next),
		UserID:       userID,
		Type:         in.Type,
		Name:         in.Name,
		Data:         in.Data,
		Meta:         in.Meta,
		Version:      1,
		UpdatedAt:    time.Now(),
	}
	s.items = append(s.items, sec)
	return sec, nil
}

func (s *fakeSecrets) Update(_ context.Context, userID, id string, data []byte, meta string) (repository.Secret, error) {
	if s.fail {
		return repository.Secret{}, errTaDa
	}
	if userID == "" || id == "" {
		return repository.Secret{}, service.ErrInvalidInput
	}
	var latest *repository.Secret
	for i := range s.items {
		it := &s.items[i]
		if it.ID == id && it.UserID == userID && (latest == nil || it.Version > latest.Version) {
			latest = it
		}
	}
	if latest == nil {
		return repository.Secret{}, repository.ErrSecretNotFound
	}
	s.next++
	sec := repository.Secret{
		SecretItemID: fmt.Sprintf("item-%d", s.next),
		ID:           id,
		UserID:       userID,
		Type:         latest.Type,
		Name:         latest.Name,
		Data:         data,
		Meta:         meta,
		Version:      latest.Version + 1,
		UpdatedAt:    time.Now(),
	}
	s.items = append(s.items, sec)
	return sec, nil
}

func (s *fakeSecrets) List(_ context.Context, userID string) ([]repository.Secret, error) {
	if s.fail {
		return nil, errTaDa
	}
	if userID == "" {
		return nil, service.ErrInvalidInput
	}
	var out []repository.Secret
	for _, it := range s.items {
		if it.UserID == userID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (s *fakeSecrets) Delete(_ context.Context, userID, id string) error {
	if s.fail {
		return errTaDa
	}
	if userID == "" || id == "" {
		return service.ErrInvalidInput
	}
	out := s.items[:0]
	found := false
	for _, it := range s.items {
		if it.ID == id && it.UserID == userID {
			found = true
			continue
		}
		out = append(out, it)
	}
	if !found {
		return repository.ErrSecretNotFound
	}
	s.items = out
	return nil
}
