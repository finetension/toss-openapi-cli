package cli

import (
	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/finetension/toss-openapi-cli/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("version does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := output.WriteJSON(cmd.OutOrStdout(), version.Get()); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
}
