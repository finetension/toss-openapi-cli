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

func TestGetAccounts(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAccept string
	var gotAuthorization string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountsResponse{
			Result: []Account{
				{
					AccountNo:   "12345678",
					AccountSeq:  1,
					AccountType: "BROKERAGE",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetAccounts(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("GetAccounts err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/accounts" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/accounts")
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if len(got.Result) != 1 {
		t.Fatalf("len(result) = %d", len(got.Result))
	}
	if got.Result[0].AccountNo != "12345678" || got.Result[0].AccountSeq != 1 || got.Result[0].AccountType != "BROKERAGE" {
		t.Fatalf("account = %+v", got.Result[0])
	}
}

func TestGetPrices(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotSymbols string
	var gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotSymbols = r.URL.Query().Get("symbols")
		gotAccept = r.Header.Get("Accept")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PricesResponse{
			Result: []Price{
				{
					Symbol:    "AAPL",
					Timestamp: "2026-03-25T22:30:00.456+09:00",
					LastPrice: "185.70",
					Currency:  "USD",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetPrices(context.Background(), "AAPL,MSFT")
	if err != nil {
		t.Fatalf("GetPrices err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/prices" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/prices")
	}
	if gotSymbols != "AAPL,MSFT" {
		t.Fatalf("symbols = %q", gotSymbols)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if len(got.Result) != 1 {
		t.Fatalf("len(result) = %d", len(got.Result))
	}
	if got.Result[0].Symbol != "AAPL" || got.Result[0].LastPrice != "185.70" || got.Result[0].Currency != "USD" {
		t.Fatalf("price = %+v", got.Result[0])
	}
}
