package auth

import (
	"testing"
	"time"
)

func TestCredentialsRoundTrip(t *testing.T) {
	encoded, err := EncodeCredentials(Credentials{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	if err != nil {
		t.Fatalf("EncodeCredentials err = %v", err)
	}

	got, err := DecodeCredentials(encoded)
	if err != nil {
		t.Fatalf("DecodeCredentials err = %v", err)
	}
	if got.ClientID != "client-id" || got.ClientSecret != "client-secret" {
		t.Fatalf("credentials = %+v", got)
	}
}

func TestCachedTokenValidUsesRefreshBuffer(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)

	valid := CachedToken{
		AccessToken: "token",
		ExpiresAt:   now.Add(TokenRefreshBuffer + time.Second),
	}
	if !valid.Valid(now) {
		t.Fatal("token should be valid outside refresh buffer")
	}

	needsRefresh := CachedToken{
		AccessToken: "token",
		ExpiresAt:   now.Add(TokenRefreshBuffer),
	}
	if needsRefresh.Valid(now) {
		t.Fatal("token should be invalid at refresh buffer boundary")
	}

	missingToken := CachedToken{
		ExpiresAt: now.Add(time.Hour),
	}
	if missingToken.Valid(now) {
		t.Fatal("token without access token should be invalid")
	}
}
