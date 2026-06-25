package auth

import (
	"testing"
	"time"
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
