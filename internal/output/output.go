package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
)

type ErrorBody struct {
	Error   ErrorObject     `json:"error"`
	Headers map[string]*int `json:"headers,omitempty"`
}

type ErrorObject struct {
	RequestID string `json:"requestId,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Reason    string `json:"reason,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func WriteError(w io.Writer, err error) int {
	app := apperr.FromError(err)
	if app == nil {
		return apperr.ExitSuccess
	}

	reason := app.Code
	body := ErrorBody{
		Error: ErrorObject{
			RequestID: app.RequestID,
			Code:      app.Code,
			Message:   app.Message,
			Reason:    reason,
			Hint:      app.Hint,
		},
		Headers: app.Headers,
	}
	if writeErr := WriteJSON(w, body); writeErr != nil {
		_, _ = fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s","reason":"%s"}}`+"\n", apperr.CodeUnexpected, "Failed to encode error output", apperr.CodeUnexpected)
		return apperr.ExitUnexpected
	}
	return app.ExitCode
}
