package cli

import (
	"context"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestAccountCommand(deps Dependencies) *cobra.Command {
	cmd := newGroupCommand("account", "Manage Toss Invest accounts.")
	cmd.AddCommand(newInvestAccountListCommand(deps))
	return cmd
}

func newInvestAccountListCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Toss Invest accounts.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("account list does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}

			accountAPI := deps.AccountAPI
			if accountAPI == nil {
				accountAPI = invest.NewClient("", nil)
			}
			accounts, err := accountAPI.GetAccounts(context.Background(), accessToken)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), accounts); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "getAccounts")
	return cmd
}
