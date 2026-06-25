package invest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueOAuth2Token(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotContentType string
	var gotAccept string
	var gotGrantType string
	var gotClientID string
	var gotClientSecret string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm err = %v", err)
		}
		gotGrantType = r.Form.Get("grant_type")
		gotClientID = r.Form.Get("client_id")
		gotClientSecret = r.Form.Get("client_secret")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OAuth2TokenResponse{
			AccessToken: "token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.IssueOAuth2Token(context.Background(), OAuth2TokenRequest{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	if err != nil {
		t.Fatalf("IssueOAuth2Token err = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/oauth2/token" {
		t.Fatalf("path = %q, want %q", gotPath, "/oauth2/token")
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotGrantType != "client_credentials" {
		t.Fatalf("grant_type = %q", gotGrantType)
	}
	if gotClientID != "client-id" {
		t.Fatalf("client_id = %q", gotClientID)
	}
	if gotClientSecret != "client-secret" {
		t.Fatalf("client_secret = %q", gotClientSecret)
	}
	if got.AccessToken != "token" || got.TokenType != "Bearer" || got.ExpiresIn != 86400 {
		t.Fatalf("response = %+v", got)
	}
}

func TestIssueOAuth2TokenOAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"Client authentication failed."}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.IssueOAuth2Token(context.Background(), OAuth2TokenRequest{
		ClientID:     "bad",
		ClientSecret: "bad",
	})
	if err == nil {
		t.Fatal("IssueOAuth2Token err = nil, want error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d", apiErr.StatusCode)
	}
	if apiErr.Code != "invalid_client" {
		t.Fatalf("Code = %q", apiErr.Code)
	}
	if apiErr.Message != "Client authentication failed." {
		t.Fatalf("Message = %q", apiErr.Message)
	}
}
