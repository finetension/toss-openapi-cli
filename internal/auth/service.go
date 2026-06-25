package auth

import (
	"os"
	"time"
)

type EnvLookup func(key string) (string, bool)

func DefaultEnvLookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

type Service struct {
	store SecretStore
	env   EnvLookup
	now   func() time.Time
}

func NewService(store SecretStore, env EnvLookup) *Service {
	if store == nil {
		store = KeyringSecretStore{}
	}
	if env == nil {
		env = DefaultEnvLookup
	}
	return &Service{
		store: store,
		env:   env,
		now:   time.Now,
	}
}

type Status struct {
	Credentials CredentialStatus `json:"credentials"`
	Token       TokenStatus      `json:"token"`
}

type CredentialStatus struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source"`
}

type TokenStatus struct {
	Configured bool   `json:"configured"`
	Valid      bool   `json:"valid"`
	Source     string `json:"source"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

func (s *Service) Status() Status {
	credentialStatus := CredentialStatus{Source: "missing"}
	if s.hasEnvCredentials() {
		credentialStatus = CredentialStatus{Configured: true, Source: "env"}
	} else if _, err := s.store.Get(KeyringService, InvestCredentials); err == nil {
		credentialStatus = CredentialStatus{Configured: true, Source: "keyring"}
	}

	tokenStatus := TokenStatus{Source: "missing"}
	if _, ok := s.env("TOSS_INVEST_ACCESS_TOKEN"); ok {
		tokenStatus = TokenStatus{Configured: true, Valid: true, Source: "env"}
	} else if encoded, err := s.store.Get(KeyringService, InvestToken); err == nil {
		if token, decodeErr := DecodeCachedToken(encoded); decodeErr == nil {
			tokenStatus = TokenStatus{
				Configured: true,
				Valid:      token.Valid(s.now()),
				Source:     "keyring",
				ExpiresAt:  token.ExpiresAt.Format(time.RFC3339),
			}
		} else {
			tokenStatus = TokenStatus{Configured: true, Valid: false, Source: "keyring"}
		}
	}

	return Status{Credentials: credentialStatus, Token: tokenStatus}
}

func (s *Service) hasEnvCredentials() bool {
	_, hasID := s.env("TOSS_INVEST_CLIENT_ID")
	_, hasSecret := s.env("TOSS_INVEST_CLIENT_SECRET")
	return hasID && hasSecret
}

func StoreCredentials(store SecretStore, credentials Credentials) error {
	encoded, err := EncodeCredentials(credentials)
	if err != nil {
		return err
	}
	return store.Set(KeyringService, InvestCredentials, encoded)
}

func StoreToken(store SecretStore, token CachedToken) error {
	encoded, err := EncodeCachedToken(token)
	if err != nil {
		return err
	}
	return store.Set(KeyringService, InvestToken, encoded)
}
