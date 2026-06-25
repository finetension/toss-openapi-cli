package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
)

func TestVersionOutputsJSON(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTest("version")
	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
		BuiltBy string `json:"builtBy"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Version == "" || got.Commit == "" || got.Date == "" || got.BuiltBy == "" {
		t.Fatalf("version output has empty fields: %+v", got)
	}
}

func TestUnknownCommandOutputsStructuredUsageError(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTest("nope")
	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error.code = %q, want %q", got.Error.Code, apperr.CodeUsage)
	}
	if got.Error.Message == "" {
		t.Fatal("error.message is empty")
	}
}

func TestInvestAuthStatusOutputsJSON(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			values := map[string]string{
				"TOSS_INVEST_CLIENT_ID":     "client-id",
				"TOSS_INVEST_CLIENT_SECRET": "client-secret",
			}
			value, ok := values[key]
			return value, ok
		},
	}, "invest", "auth", "status")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Credentials struct {
			Configured bool   `json:"configured"`
			Source     string `json:"source"`
		} `json:"credentials"`
		Token struct {
			Configured bool   `json:"configured"`
			Valid      bool   `json:"valid"`
			Source     string `json:"source"`
		} `json:"token"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !got.Credentials.Configured || got.Credentials.Source != "env" {
		t.Fatalf("credentials = %+v", got.Credentials)
	}
	if got.Token.Configured || got.Token.Valid || got.Token.Source != "missing" {
		t.Fatalf("token = %+v", got.Token)
	}
}

func TestInvestAuthLoginWithFlagsStoresCredentialsAndOutputsStatus(t *testing.T) {
	store := auth.NewMemorySecretStore()
	issuer := &fakeTokenIssuer{
		response: invest.OAuth2TokenResponse{
			AccessToken: "token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: store,
		TokenIssuer: issuer,
	}, "invest", "auth", "login", "--client-id", "client-id", "--client-secret", "client-secret")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if issuer.input.ClientID != "client-id" || issuer.input.ClientSecret != "client-secret" {
		t.Fatalf("issuer input = %+v", issuer.input)
	}

	var got struct {
		Credentials struct {
			Configured bool   `json:"configured"`
			Source     string `json:"source"`
		} `json:"credentials"`
		Token struct {
			Configured bool   `json:"configured"`
			Valid      bool   `json:"valid"`
			Source     string `json:"source"`
			ExpiresAt  string `json:"expiresAt"`
		} `json:"token"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !got.Credentials.Configured || got.Credentials.Source != "keyring" {
		t.Fatalf("credentials = %+v", got.Credentials)
	}
	if !got.Token.Configured || !got.Token.Valid || got.Token.Source != "keyring" || got.Token.ExpiresAt == "" {
		t.Fatalf("token = %+v", got.Token)
	}

	encodedCredentials, err := store.Get(auth.KeyringService, auth.InvestCredentials)
	if err != nil {
		t.Fatalf("stored credentials err = %v", err)
	}
	credentials, err := auth.DecodeCredentials(encodedCredentials)
	if err != nil {
		t.Fatalf("DecodeCredentials err = %v", err)
	}
	if credentials.ClientID != "client-id" || credentials.ClientSecret != "client-secret" {
		t.Fatalf("stored credentials = %+v", credentials)
	}
}

func TestInvestAuthLogoutClearsStoredAuth(t *testing.T) {
	store := auth.NewMemorySecretStore()
	if err := auth.StoreCredentials(store, auth.Credentials{ClientID: "client-id", ClientSecret: "client-secret"}); err != nil {
		t.Fatalf("StoreCredentials err = %v", err)
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: store,
	}, "invest", "auth", "logout")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Credentials struct {
			Configured bool   `json:"configured"`
			Source     string `json:"source"`
		} `json:"credentials"`
		Token struct {
			Configured bool   `json:"configured"`
			Source     string `json:"source"`
		} `json:"token"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Credentials.Configured || got.Credentials.Source != "missing" {
		t.Fatalf("credentials = %+v", got.Credentials)
	}
	if got.Token.Configured || got.Token.Source != "missing" {
		t.Fatalf("token = %+v", got.Token)
	}
}

func TestInvestAuthTokenRefreshesAndOutputsStatus(t *testing.T) {
	store := auth.NewMemorySecretStore()
	if err := auth.StoreCredentials(store, auth.Credentials{ClientID: "client-id", ClientSecret: "client-secret"}); err != nil {
		t.Fatalf("StoreCredentials err = %v", err)
	}
	issuer := &fakeTokenIssuer{
		response: invest.OAuth2TokenResponse{
			AccessToken: "token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: store,
		TokenIssuer: issuer,
	}, "invest", "auth", "token")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Configured bool   `json:"configured"`
		Valid      bool   `json:"valid"`
		Source     string `json:"source"`
		ExpiresAt  string `json:"expiresAt"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !got.Configured || !got.Valid || got.Source != "keyring" || got.ExpiresAt == "" {
		t.Fatalf("token status = %+v", got)
	}
}

func TestInvestAuthTokenMissingCredentials(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		TokenIssuer: &fakeTokenIssuer{},
	}, "invest", "auth", "token")

	if exitCode != apperr.ExitAuthConfig {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitAuthConfig, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeAuthConfig {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestAccountListOutputsAccounts(t *testing.T) {
	accountAPI := &fakeAccountAPI{
		response: invest.AccountsResponse{
			Result: []invest.Account{
				{
					AccountNo:   "12345678",
					AccountSeq:  1,
					AccountType: "BROKERAGE",
				},
			},
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		AccountAPI: accountAPI,
	}, "invest", "account", "list")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if accountAPI.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", accountAPI.accessToken)
	}

	var got invest.AccountsResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if len(got.Result) != 1 {
		t.Fatalf("len(result) = %d", len(got.Result))
	}
	if got.Result[0].AccountNo != "12345678" || got.Result[0].AccountSeq != 1 || got.Result[0].AccountType != "BROKERAGE" {
		t.Fatalf("account = %+v", got.Result[0])
	}
}

func TestInvestAccountListMissingCredentials(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		TokenIssuer: &fakeTokenIssuer{},
		AccountAPI:  &fakeAccountAPI{},
	}, "invest", "account", "list")

	if exitCode != apperr.ExitAuthConfig {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitAuthConfig, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeAuthConfig {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestMarketDataPricesOutputsPrices(t *testing.T) {
	marketData := &fakeMarketDataAPI{
		response: invest.PricesResponse{
			Result: []invest.Price{
				{
					Symbol:    "AAPL",
					Timestamp: "2026-03-25T22:30:00.456+09:00",
					LastPrice: "185.70",
					Currency:  "USD",
				},
			},
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		MarketData: marketData,
	}, "invest", "market-data", "prices", "--symbols", "AAPL,MSFT")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketData.symbols != "AAPL,MSFT" {
		t.Fatalf("symbols = %q", marketData.symbols)
	}

	var got invest.PricesResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if len(got.Result) != 1 {
		t.Fatalf("len(result) = %d", len(got.Result))
	}
	if got.Result[0].Symbol != "AAPL" || got.Result[0].LastPrice != "185.70" || got.Result[0].Currency != "USD" {
		t.Fatalf("price = %+v", got.Result[0])
	}
}

func TestInvestMarketDataPricesRequiresSymbols(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		MarketData: &fakeMarketDataAPI{},
	}, "invest", "market-data", "prices")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestOrderInfoBuyingPowerOutputsBuyingPower(t *testing.T) {
	orderInfo := &fakeOrderInfoAPI{
		buyingPowerResponse: invest.BuyingPowerResponse{
			Result: invest.BuyingPower{
				Currency:        "USD",
				CashBuyingPower: "3500.5",
			},
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderInfo: orderInfo,
	}, "invest", "order-info", "buying-power", "--account-seq", "1", "--currency", "USD")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderInfo.accessToken)
	}
	if orderInfo.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderInfo.accountSeq)
	}
	if orderInfo.currency != "USD" {
		t.Fatalf("currency = %q", orderInfo.currency)
	}

	var got invest.BuyingPowerResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Result.Currency != "USD" || got.Result.CashBuyingPower != "3500.5" {
		t.Fatalf("buying power = %+v", got.Result)
	}
}

func TestInvestOrderInfoSellableQuantityOutputsSellableQuantity(t *testing.T) {
	orderInfo := &fakeOrderInfoAPI{
		sellableQuantityResponse: invest.SellableQuantityResponse{
			Result: invest.SellableQuantity{
				SellableQuantity: "5.5",
			},
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderInfo: orderInfo,
	}, "invest", "order-info", "sellable-quantity", "--account-seq", "1", "--symbol", "AAPL")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderInfo.accessToken)
	}
	if orderInfo.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderInfo.accountSeq)
	}
	if orderInfo.symbol != "AAPL" {
		t.Fatalf("symbol = %q", orderInfo.symbol)
	}

	var got invest.SellableQuantityResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Result.SellableQuantity != "5.5" {
		t.Fatalf("sellable quantity = %+v", got.Result)
	}
}

func TestInvestOrderInfoBuyingPowerRequiresAccountSeq(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderInfo:   &fakeOrderInfoAPI{},
	}, "invest", "order-info", "buying-power", "--currency", "USD")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestOrderHistoryListOutputsOrders(t *testing.T) {
	orderHistory := &fakeOrderHistoryAPI{
		ordersResponse: invest.OrdersResponse{
			Result: json.RawMessage(`{"orders":[{"orderId":"order-1","symbol":"AAPL"}],"nextCursor":null,"hasNext":false}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderHistory: orderHistory,
	}, "invest", "order-history", "list", "--account-seq", "1", "--status", "CLOSED", "--symbol", "AAPL", "--from", "2026-03-01", "--to", "2026-03-31", "--cursor", "cursor-1", "--limit", "20")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderHistory.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderHistory.accessToken)
	}
	if orderHistory.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderHistory.accountSeq)
	}
	wantParams := invest.OrderListParams{
		Status: "CLOSED",
		Symbol: "AAPL",
		From:   "2026-03-01",
		To:     "2026-03-31",
		Cursor: "cursor-1",
		Limit:  20,
	}
	if !reflect.DeepEqual(orderHistory.params, wantParams) {
		t.Fatalf("params = %+v, want %+v", orderHistory.params, wantParams)
	}

	var got invest.OrdersResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderHistory.ordersResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestOrderHistoryGetOutputsOrder(t *testing.T) {
	orderHistory := &fakeOrderHistoryAPI{
		orderResponse: invest.OrderResponse{
			Result: json.RawMessage(`{"orderId":"order-1","symbol":"AAPL"}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderHistory: orderHistory,
	}, "invest", "order-history", "get", "order-1", "--account-seq", "1")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderHistory.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderHistory.accessToken)
	}
	if orderHistory.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderHistory.accountSeq)
	}
	if orderHistory.orderID != "order-1" {
		t.Fatalf("orderID = %q", orderHistory.orderID)
	}

	var got invest.OrderResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderHistory.orderResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestOrderHistoryListRequiresStatus(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore:  auth.NewMemorySecretStore(),
		OrderHistory: &fakeOrderHistoryAPI{},
	}, "invest", "order-history", "list", "--account-seq", "1")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestOrderCreateCreatesOrder(t *testing.T) {
	orderAPI := &fakeOrderAPI{
		createResponse: invest.OrderMutationResponse{
			Result: json.RawMessage(`{"orderId":"order-1","clientOrderId":"client-order-1"}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderAPI: orderAPI,
	}, "invest", "order", "create", "--account-seq", "1", "--client-order-id", "client-order-1", "--symbol", "aapl", "--side", "buy", "--order-type", "limit", "--time-in-force", "day", "--quantity", "1", "--price", "185.5")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderAPI.accessToken)
	}
	if orderAPI.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderAPI.accountSeq)
	}
	if orderAPI.createInput.ClientOrderID != "client-order-1" || orderAPI.createInput.Symbol != "aapl" || orderAPI.createInput.Side != "BUY" || orderAPI.createInput.OrderType != "LIMIT" || orderAPI.createInput.TimeInForce != "DAY" || orderAPI.createInput.Quantity != "1" || orderAPI.createInput.Price != "185.5" {
		t.Fatalf("createInput = %+v", orderAPI.createInput)
	}

	var got invest.OrderMutationResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderAPI.createResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestOrderCreateDryRunDoesNotRequireAuth(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderAPI:    &fakeOrderAPI{},
	}, "invest", "order", "create", "--dry-run", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "MARKET", "--order-amount", "100.5")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Method     string                    `json:"method"`
		Path       string                    `json:"path"`
		AccountSeq int64                     `json:"accountSeq"`
		Body       invest.OrderCreateRequest `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Method != "POST" || got.Path != "/api/v1/orders" || got.AccountSeq != 1 {
		t.Fatalf("dry-run = %+v", got)
	}
	if got.Body.ClientOrderID == "" {
		t.Fatal("generated clientOrderId is empty")
	}
	if got.Body.Symbol != "AAPL" || got.Body.Side != "BUY" || got.Body.OrderType != "MARKET" || got.Body.OrderAmount != "100.5" {
		t.Fatalf("body = %+v", got.Body)
	}
}

func TestInvestOrderCreateRequiresQuantityOrAmount(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderAPI:    &fakeOrderAPI{},
	}, "invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "MARKET")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestOrderCancelCancelsOrder(t *testing.T) {
	orderAPI := &fakeOrderAPI{
		cancelResponse: invest.OrderMutationResponse{
			Result: json.RawMessage(`{"orderId":"cancel-order-1"}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderAPI: orderAPI,
	}, "invest", "order", "cancel", "order-1", "--account-seq", "1", "--yes")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", orderAPI.accessToken)
	}
	if orderAPI.accountSeq != 1 {
		t.Fatalf("accountSeq = %d", orderAPI.accountSeq)
	}
	if orderAPI.orderID != "order-1" {
		t.Fatalf("orderID = %q", orderAPI.orderID)
	}

	var got invest.OrderMutationResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderAPI.cancelResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestOrderCancelDryRunDoesNotRequireAuth(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderAPI:    &fakeOrderAPI{},
	}, "invest", "order", "cancel", "order-1", "--account-seq", "1", "--dry-run")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Method     string         `json:"method"`
		Path       string         `json:"path"`
		AccountSeq int64          `json:"accountSeq"`
		Body       map[string]any `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Method != "POST" || got.Path != "/api/v1/orders/order-1/cancel" || got.AccountSeq != 1 || len(got.Body) != 0 {
		t.Fatalf("dry-run = %+v", got)
	}
}

func TestInvestAssetHoldingsOutputsHoldings(t *testing.T) {
	assetAPI := &fakeAssetAPI{
		response: invest.HoldingsResponse{
			Result: json.RawMessage(`{"items":[{"symbol":"AAPL","quantity":"10"}]}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		AssetAPI: assetAPI,
	}, "invest", "asset", "holdings", "--account-seq", "1", "--symbol", "AAPL")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if assetAPI.accessToken != "env-token" || assetAPI.accountSeq != 1 || assetAPI.symbol != "AAPL" {
		t.Fatalf("assetAPI = %+v", assetAPI)
	}

	var got invest.HoldingsResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, assetAPI.response.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestAssetHoldingsRequiresAccountSeq(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		AssetAPI:    &fakeAssetAPI{},
	}, "invest", "asset", "holdings")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage {
		t.Fatalf("error code = %q", got.Error.Code)
	}
}

func TestInvestOrderInfoCommissionsOutputsCommissions(t *testing.T) {
	orderInfo := &fakeOrderInfoAPI{
		commissionsResponse: invest.CommissionsResponse{
			Result: json.RawMessage(`[{"marketCountry":"US","commissionRate":"0.1"}]`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderInfo: orderInfo,
	}, "invest", "order-info", "commissions", "--account-seq", "1")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderInfo.accessToken != "env-token" || orderInfo.accountSeq != 1 {
		t.Fatalf("orderInfo = %+v", orderInfo)
	}

	var got invest.CommissionsResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderInfo.commissionsResponse.Result) {
		t.Fatalf("result = %s", got.Result)
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

type fakeAccountAPI struct {
	accessToken string
	response    invest.AccountsResponse
	err         error
}

func (f *fakeAccountAPI) GetAccounts(ctx context.Context, accessToken string) (invest.AccountsResponse, error) {
	f.accessToken = accessToken
	return f.response, f.err
}

type fakeMarketDataAPI struct {
	symbols  string
	response invest.PricesResponse
	err      error
}

func (f *fakeMarketDataAPI) GetPrices(ctx context.Context, symbols string) (invest.PricesResponse, error) {
	f.symbols = symbols
	return f.response, f.err
}

type fakeAssetAPI struct {
	accessToken string
	accountSeq  int64
	symbol      string
	response    invest.HoldingsResponse
	err         error
}

func (f *fakeAssetAPI) GetHoldings(ctx context.Context, accessToken string, accountSeq int64, symbol string) (invest.HoldingsResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.symbol = symbol
	return f.response, f.err
}

type fakeOrderInfoAPI struct {
	accessToken              string
	accountSeq               int64
	currency                 string
	symbol                   string
	buyingPowerResponse      invest.BuyingPowerResponse
	sellableQuantityResponse invest.SellableQuantityResponse
	commissionsResponse      invest.CommissionsResponse
	err                      error
}

func (f *fakeOrderInfoAPI) GetBuyingPower(ctx context.Context, accessToken string, accountSeq int64, currency string) (invest.BuyingPowerResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.currency = currency
	return f.buyingPowerResponse, f.err
}

func (f *fakeOrderInfoAPI) GetSellableQuantity(ctx context.Context, accessToken string, accountSeq int64, symbol string) (invest.SellableQuantityResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.symbol = symbol
	return f.sellableQuantityResponse, f.err
}

func (f *fakeOrderInfoAPI) GetCommissions(ctx context.Context, accessToken string, accountSeq int64) (invest.CommissionsResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	return f.commissionsResponse, f.err
}

type fakeOrderHistoryAPI struct {
	accessToken    string
	accountSeq     int64
	params         invest.OrderListParams
	orderID        string
	ordersResponse invest.OrdersResponse
	orderResponse  invest.OrderResponse
	err            error
}

func (f *fakeOrderHistoryAPI) GetOrders(ctx context.Context, accessToken string, accountSeq int64, params invest.OrderListParams) (invest.OrdersResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.params = params
	return f.ordersResponse, f.err
}

func (f *fakeOrderHistoryAPI) GetOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (invest.OrderResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.orderID = orderID
	return f.orderResponse, f.err
}

type fakeOrderAPI struct {
	accessToken    string
	accountSeq     int64
	createInput    invest.OrderCreateRequest
	orderID        string
	createResponse invest.OrderMutationResponse
	cancelResponse invest.OrderMutationResponse
	err            error
}

func (f *fakeOrderAPI) CreateOrder(ctx context.Context, accessToken string, accountSeq int64, input invest.OrderCreateRequest) (invest.OrderMutationResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.createInput = input
	return f.createResponse, f.err
}

func (f *fakeOrderAPI) CancelOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (invest.OrderMutationResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.orderID = orderID
	return f.cancelResponse, f.err
}

func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		t.Fatalf("json.Compact err = %v", err)
	}
	return out.String()
}
