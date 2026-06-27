package cli

import (
	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

type dryRunRequest struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	AccountSeq int64  `json:"accountSeq"`
	Body       any    `json:"body"`
}

func writeDryRun(cmd *cobra.Command, method string, path string, accountSeq int64, body any) error {
	if err := output.WriteJSON(cmd.OutOrStdout(), dryRunRequest{
		Method:     method,
		Path:       path,
		AccountSeq: accountSeq,
		Body:       body,
	}); err != nil {
		return apperr.Unexpected(err)
	}
	return nil
}
