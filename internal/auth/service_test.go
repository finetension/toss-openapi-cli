package auth

import (
	"context"
	"testing"
	"time"

	"github.com/finetension/toss-openapi-cli/internal/invest"
)

func TestStatusUsesEnvCredentialsAndToken(t *testing.T) {
	store := NewMemorySecretStore()
	service := NewService(store, func(key string) (string, bool) {
		values := map[string]string{
			"TOSS_INVEST_CLIENT_ID":     "client-id",
			"TOSS_INVEST_CLIENT_SECRET": "client-secret",
			"TOSS_INVEST_ACCESS_TOKEN":  "access-token",
		}
		value, ok := values[key]
		return value, ok
	})

	got := service.Status()
	if !got.Credentials.Configured || got.Credentials.Source != "env" {
		t.Fatalf("credentials status = %+v", got.Credentials)
	}
	if !got.Token.Configured || !got.Token.Valid || got.Token.Source != "env" {
		t.Fatalf("token status = %+v", got.Token)
	}
}

func TestStatusUsesKeyringTokenValidity(t *testing.T) {
	store := NewMemorySecretStore()
	if err := StoreCredentials(store, Credentials{ClientID: "client-id", ClientSecret: "client-secret"}); err != nil {
		t.Fatalf("StoreCredentials err = %v", err)
	}
	expiresAt := time.Now().Add(TokenRefreshBuffer + time.Hour).UTC()
	if err := StoreToken(store, CachedToken{AccessToken: "token", ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("StoreToken err = %v", err)
	}

	service := NewService(store, func(key string) (string, bool) { return "", false })

	got := service.Status()
	if !got.Credentials.Configured || got.Credentials.Source != "keyring" {
		t.Fatalf("credentials status = %+v", got.Credentials)
	}
	if !got.Token.Configured || !got.Token.Valid || got.Token.Source != "keyring" {
		t.Fatalf("token status = %+v", got.Token)
	}
	if got.Token.ExpiresAt == "" {
		t.Fatal("token expiresAt is empty")
	}
}

func TestLoginStoresCredentialsAndToken(t *testing.T) {
	store := NewMemorySecretStore()
	service := NewService(store, func(key string) (string, bool) { return "", false })
	issuer := &fakeTokenIssuer{
		response: invest.OAuth2TokenResponse{
			AccessToken: "issued-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}

	got, err := service.Login(context.Background(), issuer, Credentials{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	if err != nil {
		t.Fatalf("Login err = %v", err)
	}

	if issuer.input.ClientID != "client-id" || issuer.input.ClientSecret != "client-secret" {
		t.Fatalf("issuer input = %+v", issuer.input)
	}
	if !got.Credentials.Configured || got.Credentials.Source != "keyring" {
		t.Fatalf("credentials status = %+v", got.Credentials)
	}
	if !got.Token.Configured || !got.Token.Valid || got.Token.Source != "keyring" {
		t.Fatalf("token status = %+v", got.Token)
	}

	encodedCredentials, err := store.Get(KeyringService, InvestCredentials)
	if err != nil {
		t.Fatalf("stored credentials err = %v", err)
	}
	credentials, err := DecodeCredentials(encodedCredentials)
	if err != nil {
		t.Fatalf("DecodeCredentials err = %v", err)
	}
	if credentials.ClientID != "client-id" || credentials.ClientSecret != "client-secret" {
		t.Fatalf("stored credentials = %+v", credentials)
	}

	encodedToken, err := store.Get(KeyringService, InvestToken)
	if err != nil {
		t.Fatalf("stored token err = %v", err)
	}
	token, err := DecodeCachedToken(encodedToken)
	if err != nil {
		t.Fatalf("DecodeCachedToken err = %v", err)
	}
	if token.AccessToken != "issued-token" {
		t.Fatalf("stored token = %+v", token)
	}
}

type fakeTokenIssuer struct {
	input    invest.OAuth2TokenRequest
	response invest.OAuth2TokenResponse
	err      error
}

func (f *fakeTokenIssuer) IssueOAuth2Token(ctx context.Context, input invest.OAuth2TokenRequest) (invest.OAuth2TokenResponse, error) {
	f.input = input
	return f.response, f.err
}
