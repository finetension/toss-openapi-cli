package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/finetension/toss-openapi-cli/internal/invest"
)

type EnvLookup func(key string) (string, bool)

var ErrCredentialsMissing = errors.New("credentials missing")

func DefaultEnvLookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

type Service struct {
	store SecretStore
	env   EnvLookup
	now   func() time.Time
}

type TokenIssuer interface {
	IssueOAuth2Token(ctx context.Context, input invest.OAuth2TokenRequest) (invest.OAuth2TokenResponse, error)
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

func (s *Service) Login(ctx context.Context, issuer TokenIssuer, credentials Credentials) (Status, error) {
	token, err := issuer.IssueOAuth2Token(ctx, invest.OAuth2TokenRequest{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
	})
	if err != nil {
		return Status{}, err
	}

	if err := StoreCredentials(s.store, credentials); err != nil {
		return Status{}, err
	}
	if err := StoreToken(s.store, CachedToken{
		AccessToken: token.AccessToken,
		ExpiresAt:   s.now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC(),
	}); err != nil {
		return Status{}, err
	}
	return s.Status(), nil
}

func (s *Service) Token(ctx context.Context, issuer TokenIssuer) (TokenStatus, error) {
	_, status, err := s.accessToken(ctx, issuer)
	return status, err
}

func (s *Service) AccessToken(ctx context.Context, issuer TokenIssuer) (string, error) {
	token, _, err := s.accessToken(ctx, issuer)
	return token, err
}

func (s *Service) accessToken(ctx context.Context, issuer TokenIssuer) (string, TokenStatus, error) {
	if _, ok := s.env("TOSS_INVEST_ACCESS_TOKEN"); ok {
		token, _ := s.env("TOSS_INVEST_ACCESS_TOKEN")
		return token, TokenStatus{Configured: true, Valid: true, Source: "env"}, nil
	}

	if encoded, err := s.store.Get(KeyringService, InvestToken); err == nil {
		token, decodeErr := DecodeCachedToken(encoded)
		if decodeErr == nil && token.Valid(s.now()) {
			return token.AccessToken, TokenStatus{
				Configured: true,
				Valid:      true,
				Source:     "keyring",
				ExpiresAt:  token.ExpiresAt.Format(time.RFC3339),
			}, nil
		}
	}

	credentials, err := s.credentials()
	if err != nil {
		return "", TokenStatus{}, err
	}
	token, err := issuer.IssueOAuth2Token(ctx, invest.OAuth2TokenRequest{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
	})
	if err != nil {
		return "", TokenStatus{}, err
	}
	cached := CachedToken{
		AccessToken: token.AccessToken,
		ExpiresAt:   s.now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC(),
	}
	if err := StoreToken(s.store, cached); err != nil {
		return "", TokenStatus{}, err
	}
	return cached.AccessToken, TokenStatus{
		Configured: true,
		Valid:      true,
		Source:     "keyring",
		ExpiresAt:  cached.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) Logout() (Status, error) {
	if err := s.store.Delete(KeyringService, InvestCredentials); err != nil && !errors.Is(err, ErrSecretNotFound) {
		return Status{}, err
	}
	if err := s.store.Delete(KeyringService, InvestToken); err != nil && !errors.Is(err, ErrSecretNotFound) {
		return Status{}, err
	}
	return s.Status(), nil
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
	_, hasID := s.envCredentialValue("TOSS_INVEST_API_KEY", "TOSS_INVEST_CLIENT_ID")
	_, hasSecret := s.envCredentialValue("TOSS_INVEST_SECRET_KEY", "TOSS_INVEST_CLIENT_SECRET")
	return hasID && hasSecret
}

func (s *Service) credentials() (Credentials, error) {
	if clientID, hasID := s.envCredentialValue("TOSS_INVEST_API_KEY", "TOSS_INVEST_CLIENT_ID"); hasID {
		if clientSecret, hasSecret := s.envCredentialValue("TOSS_INVEST_SECRET_KEY", "TOSS_INVEST_CLIENT_SECRET"); hasSecret {
			return Credentials{ClientID: clientID, ClientSecret: clientSecret}, nil
		}
	}

	encoded, err := s.store.Get(KeyringService, InvestCredentials)
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return Credentials{}, ErrCredentialsMissing
		}
		return Credentials{}, err
	}
	credentials, err := DecodeCredentials(encoded)
	if err != nil {
		return Credentials{}, err
	}
	if credentials.ClientID == "" || credentials.ClientSecret == "" {
		return Credentials{}, ErrCredentialsMissing
	}
	return credentials, nil
}

func (s *Service) envCredentialValue(primary string, aliases ...string) (string, bool) {
	if value, ok := s.env(primary); ok {
		return value, true
	}
	for _, alias := range aliases {
		if value, ok := s.env(alias); ok {
			return value, true
		}
	}
	return "", false
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
