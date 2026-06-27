package cli

import (
	"context"
	"runtime"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/finetension/toss-openapi-cli/internal/version"
	"github.com/spf13/cobra"
)

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	Hint         string `json:"hint,omitempty"`
	Source       string `json:"source,omitempty"`
	Version      string `json:"version,omitempty"`
	Commit       string `json:"commit,omitempty"`
	Date         string `json:"date,omitempty"`
	BuiltBy      string `json:"builtBy,omitempty"`
	OS           string `json:"os,omitempty"`
	Arch         string `json:"arch,omitempty"`
	Valid        *bool  `json:"valid,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	AccountCount *int   `json:"accountCount,omitempty"`
	PublicIP     string `json:"publicIp,omitempty"`
}

func newDoctorCommand(deps Dependencies) *cobra.Command {
	var showIP bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local Toss Invest CLI readiness.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("doctor does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			report := runDoctor(context.Background(), deps, showIP)
			if err := output.WriteJSON(cmd.OutOrStdout(), report); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showIP, "show-ip", false, "Query and show the public IP address visible to external services.")
	applyHelp(cmd, "cli:doctor")
	return cmd
}

func runDoctor(ctx context.Context, deps Dependencies, showIP bool) doctorReport {
	service := auth.NewService(deps.SecretStore, deps.EnvLookup)
	status := service.Status()

	checks := []doctorCheck{
		doctorVersionCheck(),
	}
	if showIP {
		checks = append(checks, doctorPublicIPCheck(ctx, deps))
	}
	checks = append(checks, doctorCredentialsCheck(status.Credentials, status.Token))

	issuer := deps.TokenIssuer
	if issuer == nil {
		issuer = invest.NewClient("", nil)
	}
	accessToken, tokenStatus, tokenErr := doctorAccessToken(ctx, service, issuer)
	checks = append(checks, doctorTokenCheck(tokenStatus, tokenErr))

	accountCheck := doctorCheck{
		Name:   "account-list",
		Status: "skipped",
	}
	if tokenErr == nil {
		accountAPI := deps.AccountAPI
		if accountAPI == nil {
			accountAPI = invest.NewClient("", nil)
		}
		accounts, err := accountAPI.GetAccounts(ctx, accessToken)
		if err != nil {
			accountCheck.Status = "fail"
			accountCheck.Message = doctorErrorMessage(err)
			accountCheck.Hint = doctorErrorHint(err)
		} else {
			accountCount := len(accounts.Result)
			accountCheck.Status = "ok"
			accountCheck.AccountCount = &accountCount
		}
	} else {
		accountCheck.Message = "token check failed"
	}
	checks = append(checks, accountCheck)

	reportStatus := "ok"
	for _, check := range checks {
		if check.Status == "fail" {
			reportStatus = "fail"
			break
		}
	}
	return doctorReport{Status: reportStatus, Checks: checks}
}

func doctorVersionCheck() doctorCheck {
	info := version.Get()
	return doctorCheck{
		Name:    "version",
		Status:  "ok",
		Version: info.Version,
		Commit:  info.Commit,
		Date:    info.Date,
		BuiltBy: info.BuiltBy,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

func doctorPublicIPCheck(ctx context.Context, deps Dependencies) doctorCheck {
	check := doctorCheck{
		Name:   "public-ip",
		Status: "fail",
	}
	resolver := deps.PublicIP
	if resolver == nil {
		resolver = defaultPublicIPResolver{}
	}
	publicIP, err := resolver.PublicIP(ctx)
	if err != nil {
		check.Message = doctorErrorMessage(err)
		return check
	}
	check.Status = "ok"
	check.PublicIP = publicIP
	return check
}

func doctorCredentialsCheck(status auth.CredentialStatus, tokenStatus auth.TokenStatus) doctorCheck {
	check := doctorCheck{
		Name:   "credentials",
		Status: "fail",
		Source: status.Source,
	}
	if status.Configured {
		check.Status = "ok"
		return check
	}
	if tokenStatus.Configured && tokenStatus.Valid && tokenStatus.Source == "env" {
		check.Status = "skipped"
		check.Message = "Access token is provided directly"
		return check
	}
	check.Message = "Toss Invest credentials are not configured"
	return check
}

func doctorAccessToken(ctx context.Context, service *auth.Service, issuer auth.TokenIssuer) (string, auth.TokenStatus, error) {
	accessToken, err := service.AccessToken(ctx, issuer)
	if err != nil {
		return "", doctorFailedTokenStatus(service), err
	}
	tokenStatus, statusErr := service.Token(ctx, issuer)
	if statusErr != nil {
		return accessToken, doctorFailedTokenStatus(service), statusErr
	}
	return accessToken, tokenStatus, nil
}

func doctorFailedTokenStatus(service *auth.Service) auth.TokenStatus {
	status := service.Status()
	if status.Credentials.Configured {
		return auth.TokenStatus{Source: status.Credentials.Source}
	}
	if status.Token.Source != "" {
		return status.Token
	}
	return auth.TokenStatus{Source: "missing"}
}

func doctorTokenCheck(status auth.TokenStatus, err error) doctorCheck {
	check := doctorCheck{
		Name:      "token",
		Status:    "fail",
		Source:    status.Source,
		Valid:     &status.Valid,
		ExpiresAt: status.ExpiresAt,
	}
	if err != nil {
		check.Message = doctorErrorMessage(err)
		check.Hint = doctorErrorHint(err)
		return check
	}
	if status.Configured && status.Valid {
		check.Status = "ok"
	}
	return check
}

func doctorErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	normalized := normalizeCobraError(err)
	app := apperr.FromError(normalized)
	if app != nil {
		return app.Message
	}
	return err.Error()
}

func doctorErrorHint(err error) string {
	if isIPAllowlistError(err) {
		return ipAllowlistHint
	}
	return ""
}
