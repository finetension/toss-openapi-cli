package apperr

import "fmt"

const (
	ExitSuccess    = 0
	ExitUnexpected = 1
	ExitUsage      = 2
	ExitAuthConfig = 3
	ExitAPI        = 4
	ExitSpec       = 5
)

const (
	CodeUnexpected = "UNEXPECTED_ERROR"
	CodeUsage      = "USAGE_ERROR"
	CodeAuthConfig = "AUTH_CONFIG_ERROR"
	CodeAPI        = "API_ERROR"
	CodeSpec       = "SPEC_ERROR"
)

type AppError struct {
	Code     string
	Message  string
	ExitCode int
	Cause    error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(code, message string, exitCode int) *AppError {
	return &AppError{Code: code, Message: message, ExitCode: exitCode}
}

func Wrap(code, message string, exitCode int, cause error) *AppError {
	return &AppError{Code: code, Message: message, ExitCode: exitCode, Cause: cause}
}

func Usage(message string) *AppError {
	return New(CodeUsage, message, ExitUsage)
}

func Unexpected(err error) *AppError {
	if err == nil {
		return New(CodeUnexpected, "Unexpected error", ExitUnexpected)
	}
	return Wrap(CodeUnexpected, "Unexpected error", ExitUnexpected, err)
}

func FromError(err error) *AppError {
	if err == nil {
		return nil
	}
	if app, ok := err.(*AppError); ok {
		return app
	}
	return Unexpected(err)
}
