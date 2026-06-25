package auth

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

const (
	KeyringService    = "toss-openapi-cli"
	InvestCredentials = "invest.credentials"
	InvestToken       = "invest.token"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretStore interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Delete(service, account string) error
}

type KeyringSecretStore struct{}

func (KeyringSecretStore) Get(service, account string) (string, error) {
	value, err := keyring.Get(service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return value, nil
}

func (KeyringSecretStore) Set(service, account, value string) error {
	return keyring.Set(service, account, value)
}

func (KeyringSecretStore) Delete(service, account string) error {
	err := keyring.Delete(service, account)
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}
	return err
}

type MemorySecretStore struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{secrets: make(map[string]string)}
}

func (s *MemorySecretStore) Get(service, account string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.secrets[secretKey(service, account)]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (s *MemorySecretStore) Set(service, account, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets[secretKey(service, account)] = value
	return nil
}

func (s *MemorySecretStore) Delete(service, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := secretKey(service, account)
	if _, ok := s.secrets[key]; !ok {
		return ErrSecretNotFound
	}
	delete(s.secrets, key)
	return nil
}

func secretKey(service, account string) string {
	return service + "\x00" + account
}
