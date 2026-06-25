package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
)

type ErrorBody struct {
	Error ErrorObject `json:"error"`
}

type ErrorObject struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
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
			Code:    app.Code,
			Message: app.Message,
			Reason:  reason,
		},
	}
	if writeErr := WriteJSON(w, body); writeErr != nil {
		_, _ = fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s","reason":"%s"}}`+"\n", apperr.CodeUnexpected, "Failed to encode error output", apperr.CodeUnexpected)
		return apperr.ExitUnexpected
	}
	return app.ExitCode
}
