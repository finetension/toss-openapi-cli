package invest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		gotPath = r.URL.EscapedPath()
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

func TestGetBuyingPower(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotCurrency string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCurrency = r.URL.Query().Get("currency")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BuyingPowerResponse{
			Result: BuyingPower{
				Currency:        "USD",
				CashBuyingPower: "3500.5",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetBuyingPower(context.Background(), "access-token", 1, "USD")
	if err != nil {
		t.Fatalf("GetBuyingPower err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/buying-power" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/buying-power")
	}
	if gotCurrency != "USD" {
		t.Fatalf("currency = %q", gotCurrency)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if got.Result.Currency != "USD" || got.Result.CashBuyingPower != "3500.5" {
		t.Fatalf("buying power = %+v", got.Result)
	}
}

func TestGetSellableQuantity(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotSymbol string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotSymbol = r.URL.Query().Get("symbol")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SellableQuantityResponse{
			Result: SellableQuantity{
				SellableQuantity: "5.5",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetSellableQuantity(context.Background(), "access-token", 1, "AAPL")
	if err != nil {
		t.Fatalf("GetSellableQuantity err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/sellable-quantity" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/sellable-quantity")
	}
	if gotSymbol != "AAPL" {
		t.Fatalf("symbol = %q", gotSymbol)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if got.Result.SellableQuantity != "5.5" {
		t.Fatalf("sellable quantity = %+v", got.Result)
	}
}

func TestGetOrders(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotQuery string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"orders":[{"orderId":"order-1","symbol":"AAPL"}],"nextCursor":null,"hasNext":false}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetOrders(context.Background(), "access-token", 1, OrderListParams{
		Status: "CLOSED",
		Symbol: "AAPL",
		From:   "2026-03-01",
		To:     "2026-03-31",
		Cursor: "cursor-1",
		Limit:  20,
	})
	if err != nil {
		t.Fatalf("GetOrders err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/orders" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/orders")
	}
	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("ParseQuery err = %v", err)
	}
	if values.Get("status") != "CLOSED" || values.Get("symbol") != "AAPL" || values.Get("from") != "2026-03-01" || values.Get("to") != "2026-03-31" || values.Get("cursor") != "cursor-1" || values.Get("limit") != "20" {
		t.Fatalf("query = %q", gotQuery)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if string(got.Result) != `{"orders":[{"orderId":"order-1","symbol":"AAPL"}],"nextCursor":null,"hasNext":false}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestGetOrder(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"orderId":"order/id","symbol":"AAPL"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetOrder(context.Background(), "access-token", 1, "order/id")
	if err != nil {
		t.Fatalf("GetOrder err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/orders/order%2Fid" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if string(got.Result) != `{"orderId":"order/id","symbol":"AAPL"}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestCreateOrder(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotContentType string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string
	var gotBody OrderCreateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("Decode body err = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"orderId":"order-1","clientOrderId":"client-order-1"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.CreateOrder(context.Background(), "access-token", 1, OrderCreateRequest{
		ClientOrderID: "client-order-1",
		Symbol:        "AAPL",
		Side:          "BUY",
		OrderType:     "LIMIT",
		TimeInForce:   "DAY",
		Quantity:      "1",
		Price:         "185.5",
	})
	if err != nil {
		t.Fatalf("CreateOrder err = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/v1/orders" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/orders")
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if gotBody.ClientOrderID != "client-order-1" || gotBody.Symbol != "AAPL" || gotBody.Side != "BUY" || gotBody.OrderType != "LIMIT" || gotBody.Quantity != "1" || gotBody.Price != "185.5" {
		t.Fatalf("body = %+v", gotBody)
	}
	if string(got.Result) != `{"orderId":"order-1","clientOrderId":"client-order-1"}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestModifyOrder(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotContentType string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string
	var gotBody OrderModifyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("Decode body err = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"orderId":"modified-order-1"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.ModifyOrder(context.Background(), "access-token", 1, "order/id", OrderModifyRequest{
		OrderType: "LIMIT",
		Quantity:  "15",
		Price:     "185.5",
	})
	if err != nil {
		t.Fatalf("ModifyOrder err = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/v1/orders/order%2Fid/modify" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if gotBody.OrderType != "LIMIT" || gotBody.Quantity != "15" || gotBody.Price != "185.5" {
		t.Fatalf("body = %+v", gotBody)
	}
	if string(got.Result) != `{"orderId":"modified-order-1"}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestCancelOrder(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotContentType string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("Decode body err = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"orderId":"cancel-order-1"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.CancelOrder(context.Background(), "access-token", 1, "order/id")
	if err != nil {
		t.Fatalf("CancelOrder err = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/v1/orders/order%2Fid/cancel" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if len(gotBody) != 0 {
		t.Fatalf("body = %+v", gotBody)
	}
	if string(got.Result) != `{"orderId":"cancel-order-1"}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestGetHoldings(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotSymbol string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotSymbol = r.URL.Query().Get("symbol")
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"items":[{"symbol":"AAPL","quantity":"10"}]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetHoldings(context.Background(), "access-token", 1, "AAPL")
	if err != nil {
		t.Fatalf("GetHoldings err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/holdings" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/holdings")
	}
	if gotSymbol != "AAPL" {
		t.Fatalf("symbol = %q", gotSymbol)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if string(got.Result) != `{"items":[{"symbol":"AAPL","quantity":"10"}]}` {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestGetCommissions(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAccept string
	var gotAuthorization string
	var gotAccount string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotAuthorization = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("X-Tossinvest-Account")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"marketCountry":"US","commissionRate":"0.1"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	got, err := client.GetCommissions(context.Background(), "access-token", 1)
	if err != nil {
		t.Fatalf("GetCommissions err = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/api/v1/commissions" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/commissions")
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotAccount != "1" {
		t.Fatalf("X-Tossinvest-Account = %q", gotAccount)
	}
	if string(got.Result) != `[{"marketCountry":"US","commissionRate":"0.1"}]` {
		t.Fatalf("result = %s", got.Result)
	}
}
