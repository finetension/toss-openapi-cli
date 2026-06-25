package cli

import (
	"encoding/json"
	"testing"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
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
