package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
)

func executeForTestWithInput(input string, deps Dependencies, args ...string) (stdout string, stderr string, exitCode int) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(IOStreams{In: strings.NewReader(input), Out: &out, ErrOut: &errOut}, deps)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		exitCode = output.WriteError(&out, normalizeCobraError(err))
		return out.String(), errOut.String(), exitCode
	}
	return out.String(), errOut.String(), apperr.ExitSuccess
}

func mapEnvLookup(values map[string]string) auth.EnvLookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

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

func TestRootVersionFlagOutputsJSON(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-v"}} {
		stdout, stderr, exitCode := ExecuteForTest(args...)
		if exitCode != apperr.ExitSuccess {
			t.Fatalf("args %v exitCode = %d, want %d; stdout=%s stderr=%s", args, exitCode, apperr.ExitSuccess, stdout, stderr)
		}
		if stderr != "" {
			t.Fatalf("args %v stderr = %q, want empty", args, stderr)
		}

		var got struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
			Date    string `json:"date"`
			BuiltBy string `json:"builtBy"`
		}
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("args %v stdout is not valid JSON: %v\n%s", args, err, stdout)
		}
		if got.Version == "" || got.Commit == "" || got.Date == "" || got.BuiltBy == "" {
			t.Fatalf("args %v version output has empty fields: %+v", args, got)
		}
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

func TestUnknownInvestSubcommandsOutputStructuredUsageError(t *testing.T) {
	tests := [][]string{
		{"invest", "nope"},
		{"invest", "account", "nope"},
		{"invest", "asset", "nope"},
		{"invest", "auth", "nope"},
		{"invest", "market-data", "nope"},
		{"invest", "market-info", "nope"},
		{"invest", "market-info", "calendar", "today"},
		{"invest", "order", "nope"},
		{"invest", "order-history", "nope"},
		{"invest", "order-info", "nope"},
		{"invest", "stock-info", "nope"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, exitCode := ExecuteForTest(args...)
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
				t.Fatalf("error.code = %q, want %q; stdout=%s", got.Error.Code, apperr.CodeUsage, stdout)
			}
			if !strings.Contains(got.Error.Message, "unknown command") {
				t.Fatalf("error.message = %q, want unknown command", got.Error.Message)
			}
		})
	}
}

func TestDoctorOutputsReadinessChecks(t *testing.T) {
	accountAPI := &fakeAccountAPI{
		response: invest.AccountsResponse{
			Result: []invest.Account{
				{AccountSeq: 1},
				{AccountSeq: 2},
			},
		},
	}
	issuer := &fakeTokenIssuer{
		response: invest.OAuth2TokenResponse{
			AccessToken: "issued-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}

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
		TokenIssuer: issuer,
		AccountAPI:  accountAPI,
	}, "doctor")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if accountAPI.accessToken != "issued-token" {
		t.Fatalf("account accessToken = %q", accountAPI.accessToken)
	}
	if bytes.Contains([]byte(stdout), []byte("client-secret")) || bytes.Contains([]byte(stdout), []byte("issued-token")) {
		t.Fatalf("doctor output leaked secret material: %s", stdout)
	}

	var got struct {
		Status string            `json:"status"`
		Checks []testDoctorCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Status != "ok" {
		t.Fatalf("status = %q, want ok; stdout=%s", got.Status, stdout)
	}
	if len(got.Checks) != 4 {
		t.Fatalf("len(checks) = %d, want 4; checks=%+v", len(got.Checks), got.Checks)
	}
	checks := doctorChecksByName(got.Checks)
	if checks["version"].Status != "ok" {
		t.Fatalf("version check = %+v", checks["version"])
	}
	if checks["credentials"].Status != "ok" || checks["credentials"].Source != "env" {
		t.Fatalf("credentials check = %+v", checks["credentials"])
	}
	if checks["token"].Status != "ok" || checks["token"].Source != "keyring" {
		t.Fatalf("token check = %+v", checks["token"])
	}
	if checks["account-list"].Status != "ok" || checks["account-list"].AccountCount == nil || *checks["account-list"].AccountCount != 2 {
		t.Fatalf("account-list check = %+v", checks["account-list"])
	}
}

func TestDoctorSupportsDirectAccessTokenWithoutCredentials(t *testing.T) {
	accountAPI := &fakeAccountAPI{
		response: invest.AccountsResponse{Result: []invest.Account{{AccountSeq: 1}}},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup:   testEnvAccessToken,
		AccountAPI:  accountAPI,
	}, "doctor")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if accountAPI.accessToken != "env-token" {
		t.Fatalf("account accessToken = %q", accountAPI.accessToken)
	}

	var got struct {
		Status string            `json:"status"`
		Checks []testDoctorCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Status != "ok" {
		t.Fatalf("status = %q, want ok; stdout=%s", got.Status, stdout)
	}
	checks := doctorChecksByName(got.Checks)
	if checks["credentials"].Status != "skipped" || checks["credentials"].Source != "missing" {
		t.Fatalf("credentials check = %+v", checks["credentials"])
	}
	if checks["token"].Status != "ok" || checks["token"].Source != "env" {
		t.Fatalf("token check = %+v", checks["token"])
	}
}

func TestDoctorShowIPIncludesPublicIPCheck(t *testing.T) {
	accountAPI := &fakeAccountAPI{
		response: invest.AccountsResponse{Result: []invest.Account{{AccountSeq: 1}}},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup:   testEnvAccessToken,
		PublicIP:    fakePublicIPResolver{publicIP: "203.0.113.10"},
		AccountAPI:  accountAPI,
	}, "doctor", "--show-ip")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Status string            `json:"status"`
		Checks []testDoctorCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	checks := doctorChecksByName(got.Checks)
	if checks["public-ip"].Status != "ok" || checks["public-ip"].PublicIP != "203.0.113.10" {
		t.Fatalf("public-ip check = %+v", checks["public-ip"])
	}
}

func TestDoctorReportsMissingCredentialsAndSkipsAccountList(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		TokenIssuer: &fakeTokenIssuer{},
		AccountAPI:  &fakeAccountAPI{},
	}, "doctor")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Status string            `json:"status"`
		Checks []testDoctorCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Status != "fail" {
		t.Fatalf("status = %q, want fail; stdout=%s", got.Status, stdout)
	}
	checks := doctorChecksByName(got.Checks)
	if checks["credentials"].Status != "fail" || checks["credentials"].Message == "" {
		t.Fatalf("credentials check = %+v", checks["credentials"])
	}
	if checks["token"].Status != "fail" || checks["token"].Message == "" {
		t.Fatalf("token check = %+v", checks["token"])
	}
	if checks["account-list"].Status != "skipped" || checks["account-list"].Message == "" {
		t.Fatalf("account-list check = %+v", checks["account-list"])
	}
}

func TestDoctorReportsTokenIssueFailureWithCredentialSource(t *testing.T) {
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
		TokenIssuer: &fakeTokenIssuer{err: &invest.APIError{
			Code:    "access_denied",
			Message: "IP address not allowed",
		}},
		AccountAPI: &fakeAccountAPI{},
	}, "doctor")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Status string            `json:"status"`
		Checks []testDoctorCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Status != "fail" {
		t.Fatalf("status = %q, want fail; stdout=%s", got.Status, stdout)
	}
	checks := doctorChecksByName(got.Checks)
	if checks["credentials"].Status != "ok" || checks["credentials"].Source != "env" {
		t.Fatalf("credentials check = %+v", checks["credentials"])
	}
	if checks["token"].Status != "fail" || checks["token"].Source != "env" || checks["token"].Message != "IP address not allowed" {
		t.Fatalf("token check = %+v", checks["token"])
	}
	if !strings.Contains(checks["token"].Hint, "tosscli doctor --show-ip") {
		t.Fatalf("token hint = %q", checks["token"].Hint)
	}
	if checks["account-list"].Status != "skipped" || checks["account-list"].Message == "" {
		t.Fatalf("account-list check = %+v", checks["account-list"])
	}
}

func TestIPAllowlistAPIErrorIncludesHint(t *testing.T) {
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
		TokenIssuer: &fakeTokenIssuer{err: &invest.APIError{
			Code:    "access_denied",
			Message: "IP address not allowed",
		}},
	}, "invest", "auth", "token")

	if exitCode != apperr.ExitAPI {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitAPI, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Hint    string `json:"hint"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != "access_denied" || got.Error.Message != "IP address not allowed" {
		t.Fatalf("error = %+v", got.Error)
	}
	if !strings.Contains(got.Error.Hint, "tosscli doctor --show-ip") {
		t.Fatalf("hint = %q", got.Error.Hint)
	}
}

func TestOrderCreateHelpIncludesOASBackedRules(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTest("invest", "order", "create", "--help")
	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"OpenAPI operation: createOrder.",
		"Rate limit group: ORDER.",
		"--dry-run prints the request preview as JSON and does not call the Toss API.",
		"Without --dry-run, this command sends a live order request to the Toss API.",
		"Provide exactly one of --quantity or --order-amount.",
		"LIMIT orders require --price.",
		"--order-amount is for US MARKET amount-based orders.",
		"Allowed: DAY, CLS.",
		"US MARKET only.",
		"Repeated values return the previous order result for 10 minutes.",
		"--account-seq int",
		"Source: tosscli invest account list.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "--account-seq tosscli invest account list") {
		t.Fatalf("help used pflag backtick placeholder unexpectedly:\n%s", stdout)
	}
	if strings.Contains(stdout, "--yes") {
		t.Fatalf("help exposed unsupported --yes flag:\n%s", stdout)
	}
	if strings.Contains(stdout, "OPG") {
		t.Fatalf("help exposed time-in-force value not present in OAS:\n%s", stdout)
	}
}

func TestCandlesHelpIncludesOASBackedFlagMetadata(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTest("invest", "market-data", "candles", "--help")
	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"OpenAPI operation: getCandles.",
		"Rate limit group: MARKET_DATA_CHART.",
		"Candle interval. Required. Allowed: 1m, 1d.",
		"Candle count. Optional. Range: 1-200. Default: 100.",
		"Format: ISO 8601 date-time.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help missing %q\n%s", want, stdout)
		}
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

func TestInvestAuthLoginUsesEnvCredentials(t *testing.T) {
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
		EnvLookup: mapEnvLookup(map[string]string{
			"TOSS_INVEST_CLIENT_ID":     "env-client-id",
			"TOSS_INVEST_CLIENT_SECRET": "env-client-secret",
		}),
		TokenIssuer: issuer,
	}, "invest", "auth", "login")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if issuer.input.ClientID != "env-client-id" || issuer.input.ClientSecret != "env-client-secret" {
		t.Fatalf("issuer input = %+v", issuer.input)
	}
	if strings.Contains(stdout, "env-client-secret") || strings.Contains(stdout, `"AccessToken"`) || strings.Contains(stdout, "issued-token") {
		t.Fatalf("stdout leaked secret material: %s", stdout)
	}
}

func TestInvestAuthLoginPromptsForMissingCredentials(t *testing.T) {
	store := auth.NewMemorySecretStore()
	issuer := &fakeTokenIssuer{
		response: invest.OAuth2TokenResponse{
			AccessToken: "token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}

	stdout, stderr, exitCode := executeForTestWithInput("prompt-client-id\nprompt-client-secret\n", Dependencies{
		SecretStore: store,
		TokenIssuer: issuer,
	}, "invest", "auth", "login")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "Client ID: Client Secret: " {
		t.Fatalf("stderr = %q", stderr)
	}
	if issuer.input.ClientID != "prompt-client-id" || issuer.input.ClientSecret != "prompt-client-secret" {
		t.Fatalf("issuer input = %+v", issuer.input)
	}
	if strings.Contains(stdout, "prompt-client-secret") || strings.Contains(stdout, `"AccessToken"`) || strings.Contains(stdout, "issued-token") {
		t.Fatalf("stdout leaked secret material: %s", stdout)
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
		EnvLookup:  testEnvAccessToken,
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
	if marketData.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketData.accessToken)
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

func TestInvestMarketDataOrderbookOutputsOrderbook(t *testing.T) {
	marketData := &fakeMarketDataAPI{
		orderbookResponse: invest.OrderbookResponse{
			Result: json.RawMessage(`{"currency":"USD","asks":[],"bids":[]}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketData: marketData,
	}, "invest", "market-data", "orderbook", "--symbol", "AAPL")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketData.symbol != "AAPL" {
		t.Fatalf("symbol = %q", marketData.symbol)
	}
	if marketData.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketData.accessToken)
	}

	var got invest.OrderbookResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketData.orderbookResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestMarketDataTradesOutputsTrades(t *testing.T) {
	marketData := &fakeMarketDataAPI{
		tradesResponse: invest.TradesResponse{
			Result: json.RawMessage(`[{"price":"185.70","volume":"15"}]`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketData: marketData,
	}, "invest", "market-data", "trades", "--symbol", "AAPL", "--count", "10")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketData.symbol != "AAPL" || marketData.count != 10 {
		t.Fatalf("marketData = %+v", marketData)
	}
	if marketData.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketData.accessToken)
	}

	var got invest.TradesResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketData.tradesResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestStockInfoStocksOutputsStocks(t *testing.T) {
	stockInfo := &fakeStockInfoAPI{
		stocksResponse: invest.StocksResponse{
			Result: json.RawMessage(`[{"symbol":"AAPL","name":"Apple"}]`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup: testEnvAccessToken,
		StockInfo: stockInfo,
	}, "invest", "stock-info", "stocks", "--symbols", "AAPL,MSFT")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if stockInfo.symbols != "AAPL,MSFT" {
		t.Fatalf("symbols = %q", stockInfo.symbols)
	}
	if stockInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", stockInfo.accessToken)
	}

	var got invest.StocksResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, stockInfo.stocksResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestStockInfoWarningsOutputsWarnings(t *testing.T) {
	stockInfo := &fakeStockInfoAPI{
		warningsResponse: invest.StockWarningsResponse{
			Result: json.RawMessage(`[{"warningType":"OVERHEATED"}]`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup: testEnvAccessToken,
		StockInfo: stockInfo,
	}, "invest", "stock-info", "warnings", "AAPL")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if stockInfo.symbol != "AAPL" {
		t.Fatalf("symbol = %q", stockInfo.symbol)
	}
	if stockInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", stockInfo.accessToken)
	}

	var got invest.StockWarningsResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, stockInfo.warningsResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestMarketDataPriceLimitsOutputsLimits(t *testing.T) {
	marketData := &fakeMarketDataAPI{
		priceLimitResponse: invest.PriceLimitResponse{
			Result: json.RawMessage(`{"currency":"USD","upperLimitPrice":null,"lowerLimitPrice":null}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketData: marketData,
	}, "invest", "market-data", "price-limits", "--symbol", "AAPL")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketData.symbol != "AAPL" {
		t.Fatalf("symbol = %q", marketData.symbol)
	}
	if marketData.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketData.accessToken)
	}

	var got invest.PriceLimitResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketData.priceLimitResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestMarketDataCandlesOutputsCandles(t *testing.T) {
	adjusted := false
	marketData := &fakeMarketDataAPI{
		candlesResponse: invest.CandlesResponse{
			Result: json.RawMessage(`{"candles":[{"closePrice":"185.70"}],"nextBefore":null}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketData: marketData,
	}, "invest", "market-data", "candles", "--symbol", "AAPL", "--interval", "1d", "--count", "10", "--before", "2026-03-25T09:00:00+09:00", "--adjusted=false")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := invest.CandleParams{Symbol: "AAPL", Interval: "1d", Count: 10, Before: "2026-03-25T09:00:00+09:00", Adjusted: &adjusted}
	if !reflect.DeepEqual(marketData.candleParams, want) {
		t.Fatalf("candleParams = %+v", marketData.candleParams)
	}
	if marketData.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketData.accessToken)
	}

	var got invest.CandlesResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketData.candlesResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestMarketDataCandlesRejectsInvalidInterval(t *testing.T) {
	marketData := &fakeMarketDataAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketData: marketData,
	}, "invest", "market-data", "candles", "--symbol", "AAPL", "--interval", "5m")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketData.accessToken != "" {
		t.Fatalf("market data API should not be called; accessToken = %q", marketData.accessToken)
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

func TestInvestMarketInfoExchangeRateOutputsRate(t *testing.T) {
	marketInfo := &fakeMarketInfoAPI{
		exchangeRateResponse: invest.ExchangeRateResponse{
			Result: json.RawMessage(`{"baseCurrency":"USD","quoteCurrency":"KRW","rate":"1380.5"}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketInfo: marketInfo,
	}, "invest", "market-info", "exchange-rate", "--base-currency", "usd", "--quote-currency", "krw", "--date-time", "2026-03-25T09:30:00+09:00")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := invest.ExchangeRateParams{BaseCurrency: "USD", QuoteCurrency: "KRW", DateTime: "2026-03-25T09:30:00+09:00"}
	if !reflect.DeepEqual(marketInfo.exchangeRateParams, want) {
		t.Fatalf("exchangeRateParams = %+v", marketInfo.exchangeRateParams)
	}
	if marketInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketInfo.accessToken)
	}

	var got invest.ExchangeRateResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketInfo.exchangeRateResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestMarketInfoExchangeRateRejectsInvalidCurrency(t *testing.T) {
	marketInfo := &fakeMarketInfoAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketInfo: marketInfo,
	}, "invest", "market-info", "exchange-rate", "--base-currency", "EUR", "--quote-currency", "KRW")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketInfo.accessToken != "" {
		t.Fatalf("market info API should not be called; accessToken = %q", marketInfo.accessToken)
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

func TestInvestMarketInfoCalendarOutputsCalendar(t *testing.T) {
	marketInfo := &fakeMarketInfoAPI{
		calendarResponse: invest.MarketCalendarResponse{
			Result: json.RawMessage(`{"today":{"date":"2026-03-25"}}`),
		},
	}

	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		EnvLookup:  testEnvAccessToken,
		MarketInfo: marketInfo,
	}, "invest", "market-info", "calendar", "us", "--date", "2026-03-25")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if marketInfo.market != "US" || marketInfo.date != "2026-03-25" {
		t.Fatalf("marketInfo = %+v", marketInfo)
	}
	if marketInfo.accessToken != "env-token" {
		t.Fatalf("accessToken = %q", marketInfo.accessToken)
	}

	var got invest.MarketCalendarResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, marketInfo.calendarResponse.Result) {
		t.Fatalf("result = %s", got.Result)
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

func TestInvestOrderInfoBuyingPowerRejectsInvalidCurrency(t *testing.T) {
	orderInfo := &fakeOrderInfoAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup:   testEnvAccessToken,
		OrderInfo:   orderInfo,
	}, "invest", "order-info", "buying-power", "--account-seq", "1", "--currency", "EUR")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderInfo.accessToken != "" {
		t.Fatalf("order info API should not be called; accessToken = %q", orderInfo.accessToken)
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

func TestInvestOrderHistoryListRejectsInvalidStatus(t *testing.T) {
	orderHistory := &fakeOrderHistoryAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore:  auth.NewMemorySecretStore(),
		EnvLookup:    testEnvAccessToken,
		OrderHistory: orderHistory,
	}, "invest", "order-history", "list", "--account-seq", "1", "--status", "PENDING")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderHistory.accessToken != "" {
		t.Fatalf("order history API should not be called; accessToken = %q", orderHistory.accessToken)
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

func TestInvestOrderCreateRejectsPriceForMarket(t *testing.T) {
	orderAPI := &fakeOrderAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderAPI: orderAPI,
	}, "invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "MARKET", "--quantity", "1", "--price", "100")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "" {
		t.Fatalf("order API should not be called; accessToken = %q", orderAPI.accessToken)
	}
	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage || got.Error.Message != "--price is not allowed for MARKET orders" {
		t.Fatalf("error = %+v", got.Error)
	}
}

func TestInvestOrderCreateRejectsInvalidEnums(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "side",
			args: []string{"invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "HOLD", "--order-type", "LIMIT", "--quantity", "1", "--price", "100"},
		},
		{
			name: "order-type",
			args: []string{"invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "STOP", "--quantity", "1", "--price", "100"},
		},
		{
			name: "time-in-force",
			args: []string{"invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "LIMIT", "--time-in-force", "GTC", "--quantity", "1", "--price", "100"},
		},
		{
			name: "time-in-force OPG not in OAS",
			args: []string{"invest", "order", "create", "--account-seq", "1", "--symbol", "AAPL", "--side", "BUY", "--order-type", "LIMIT", "--time-in-force", "OPG", "--quantity", "1", "--price", "100"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orderAPI := &fakeOrderAPI{}
			stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
				SecretStore: auth.NewMemorySecretStore(),
				EnvLookup:   testEnvAccessToken,
				OrderAPI:    orderAPI,
			}, tc.args...)

			if exitCode != apperr.ExitUsage {
				t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if orderAPI.accessToken != "" {
				t.Fatalf("order API should not be called; accessToken = %q", orderAPI.accessToken)
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
		})
	}
}

func TestInvestOrderModifyModifiesOrder(t *testing.T) {
	orderAPI := &fakeOrderAPI{
		modifyResponse: invest.OrderMutationResponse{
			Result: json.RawMessage(`{"orderId":"modified-order-1"}`),
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
	}, "invest", "order", "modify", "order-1", "--account-seq", "1", "--order-type", "limit", "--quantity", "15", "--price", "185.5")

	if exitCode != apperr.ExitSuccess {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "env-token" || orderAPI.accountSeq != 1 || orderAPI.orderID != "order-1" {
		t.Fatalf("orderAPI = %+v", orderAPI)
	}
	if orderAPI.modifyInput.OrderType != "LIMIT" || orderAPI.modifyInput.Quantity != "15" || orderAPI.modifyInput.Price != "185.5" {
		t.Fatalf("modifyInput = %+v", orderAPI.modifyInput)
	}

	var got invest.OrderMutationResponse
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if compactJSON(t, got.Result) != compactJSON(t, orderAPI.modifyResponse.Result) {
		t.Fatalf("result = %s", got.Result)
	}
}

func TestInvestOrderModifyDryRunDoesNotRequireAuth(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderAPI:    &fakeOrderAPI{},
	}, "invest", "order", "modify", "order-1", "--dry-run", "--account-seq", "1", "--order-type", "LIMIT", "--price", "185.5")

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
		Body       invest.OrderModifyRequest `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Method != "POST" || got.Path != "/api/v1/orders/order-1/modify" || got.AccountSeq != 1 {
		t.Fatalf("dry-run = %+v", got)
	}
	if got.Body.OrderType != "LIMIT" || got.Body.Price != "185.5" {
		t.Fatalf("body = %+v", got.Body)
	}
}

func TestInvestOrderModifyRejectsPriceForMarket(t *testing.T) {
	orderAPI := &fakeOrderAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup: func(key string) (string, bool) {
			if key == "TOSS_INVEST_ACCESS_TOKEN" {
				return "env-token", true
			}
			return "", false
		},
		OrderAPI: orderAPI,
	}, "invest", "order", "modify", "order-1", "--account-seq", "1", "--order-type", "MARKET", "--quantity", "1", "--price", "100")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "" {
		t.Fatalf("order API should not be called; accessToken = %q", orderAPI.accessToken)
	}
	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if got.Error.Code != apperr.CodeUsage || got.Error.Message != "--price is not allowed for MARKET orders" {
		t.Fatalf("error = %+v", got.Error)
	}
}

func TestInvestOrderModifyRejectsInvalidOrderType(t *testing.T) {
	orderAPI := &fakeOrderAPI{}
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		EnvLookup:   testEnvAccessToken,
		OrderAPI:    orderAPI,
	}, "invest", "order", "modify", "order-1", "--account-seq", "1", "--order-type", "STOP", "--quantity", "1", "--price", "100")

	if exitCode != apperr.ExitUsage {
		t.Fatalf("exitCode = %d, want %d; stdout=%s stderr=%s", exitCode, apperr.ExitUsage, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if orderAPI.accessToken != "" {
		t.Fatalf("order API should not be called; accessToken = %q", orderAPI.accessToken)
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

func TestInvestOrderModifyRequiresPriceForLimit(t *testing.T) {
	stdout, stderr, exitCode := ExecuteForTestWithDeps(Dependencies{
		SecretStore: auth.NewMemorySecretStore(),
		OrderAPI:    &fakeOrderAPI{},
	}, "invest", "order", "modify", "order-1", "--account-seq", "1", "--order-type", "LIMIT")

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
	}, "invest", "order", "cancel", "order-1", "--account-seq", "1")

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

type fakePublicIPResolver struct {
	publicIP string
	err      error
}

func (f fakePublicIPResolver) PublicIP(ctx context.Context) (string, error) {
	return f.publicIP, f.err
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
	accessToken        string
	symbol             string
	symbols            string
	count              int
	candleParams       invest.CandleParams
	response           invest.PricesResponse
	orderbookResponse  invest.OrderbookResponse
	tradesResponse     invest.TradesResponse
	priceLimitResponse invest.PriceLimitResponse
	candlesResponse    invest.CandlesResponse
	err                error
}

func (f *fakeMarketDataAPI) GetPrices(ctx context.Context, accessToken string, symbols string) (invest.PricesResponse, error) {
	f.accessToken = accessToken
	f.symbols = symbols
	return f.response, f.err
}

func (f *fakeMarketDataAPI) GetOrderbook(ctx context.Context, accessToken string, symbol string) (invest.OrderbookResponse, error) {
	f.accessToken = accessToken
	f.symbol = symbol
	return f.orderbookResponse, f.err
}

func (f *fakeMarketDataAPI) GetTrades(ctx context.Context, accessToken string, symbol string, count int) (invest.TradesResponse, error) {
	f.accessToken = accessToken
	f.symbol = symbol
	f.count = count
	return f.tradesResponse, f.err
}

func (f *fakeMarketDataAPI) GetPriceLimit(ctx context.Context, accessToken string, symbol string) (invest.PriceLimitResponse, error) {
	f.accessToken = accessToken
	f.symbol = symbol
	return f.priceLimitResponse, f.err
}

func (f *fakeMarketDataAPI) GetCandles(ctx context.Context, accessToken string, params invest.CandleParams) (invest.CandlesResponse, error) {
	f.accessToken = accessToken
	f.candleParams = params
	return f.candlesResponse, f.err
}

type fakeMarketInfoAPI struct {
	accessToken          string
	market               string
	date                 string
	exchangeRateParams   invest.ExchangeRateParams
	exchangeRateResponse invest.ExchangeRateResponse
	calendarResponse     invest.MarketCalendarResponse
	err                  error
}

func (f *fakeMarketInfoAPI) GetExchangeRate(ctx context.Context, accessToken string, params invest.ExchangeRateParams) (invest.ExchangeRateResponse, error) {
	f.accessToken = accessToken
	f.exchangeRateParams = params
	return f.exchangeRateResponse, f.err
}

func (f *fakeMarketInfoAPI) GetMarketCalendar(ctx context.Context, accessToken string, market string, date string) (invest.MarketCalendarResponse, error) {
	f.accessToken = accessToken
	f.market = market
	f.date = date
	return f.calendarResponse, f.err
}

type fakeStockInfoAPI struct {
	accessToken      string
	symbol           string
	symbols          string
	stocksResponse   invest.StocksResponse
	warningsResponse invest.StockWarningsResponse
	err              error
}

func (f *fakeStockInfoAPI) GetStocks(ctx context.Context, accessToken string, symbols string) (invest.StocksResponse, error) {
	f.accessToken = accessToken
	f.symbols = symbols
	return f.stocksResponse, f.err
}

func (f *fakeStockInfoAPI) GetStockWarnings(ctx context.Context, accessToken string, symbol string) (invest.StockWarningsResponse, error) {
	f.accessToken = accessToken
	f.symbol = symbol
	return f.warningsResponse, f.err
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
	modifyInput    invest.OrderModifyRequest
	orderID        string
	createResponse invest.OrderMutationResponse
	modifyResponse invest.OrderMutationResponse
	cancelResponse invest.OrderMutationResponse
	err            error
}

func (f *fakeOrderAPI) CreateOrder(ctx context.Context, accessToken string, accountSeq int64, input invest.OrderCreateRequest) (invest.OrderMutationResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.createInput = input
	return f.createResponse, f.err
}

func (f *fakeOrderAPI) ModifyOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string, input invest.OrderModifyRequest) (invest.OrderMutationResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.orderID = orderID
	f.modifyInput = input
	return f.modifyResponse, f.err
}

func (f *fakeOrderAPI) CancelOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (invest.OrderMutationResponse, error) {
	f.accessToken = accessToken
	f.accountSeq = accountSeq
	f.orderID = orderID
	return f.cancelResponse, f.err
}

type testDoctorCheck struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	Hint         string `json:"hint"`
	Source       string `json:"source"`
	AccountCount *int   `json:"accountCount"`
	PublicIP     string `json:"publicIp"`
}

func doctorChecksByName(checks []testDoctorCheck) map[string]testDoctorCheck {
	byName := make(map[string]testDoctorCheck, len(checks))
	for _, check := range checks {
		byName[check.Name] = check
	}
	return byName
}

func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		t.Fatalf("json.Compact err = %v", err)
	}
	return out.String()
}

func testEnvAccessToken(key string) (string, bool) {
	if key == "TOSS_INVEST_ACCESS_TOKEN" {
		return "env-token", true
	}
	return "", false
}
