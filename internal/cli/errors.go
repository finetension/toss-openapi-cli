package cli

import (
	"errors"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
)

func normalizeCobraError(err error) error {
	if err == nil {
		return nil
	}
	var app *apperr.AppError
	if errors.As(err, &app) {
		return app
	}
	if errors.Is(err, auth.ErrCredentialsMissing) {
		return apperr.New(apperr.CodeAuthConfig, "Missing Toss Invest credentials", apperr.ExitAuthConfig)
	}
	var apiErr *invest.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		if code == "" {
			code = apperr.CodeAPI
		}
		message := apiErr.Message
		if message == "" {
			message = "Toss Invest API request failed"
		}
		app := apperr.New(code, message, apperr.ExitAPI)
		app.RequestID = apiErr.RequestID
		if len(apiErr.Headers) > 0 {
			app.Headers = map[string]*int(apiErr.Headers)
		}
		if isIPAllowlistError(err) {
			app.Hint = ipAllowlistHint
		}
		return app
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "invalid argument") {
		return apperr.Usage(msg)
	}
	return apperr.Unexpected(err)
}
