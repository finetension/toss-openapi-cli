package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestAssetCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset",
		Short: "Read Toss Invest assets.",
	}
	cmd.AddCommand(newInvestAssetHoldingsCommand(deps))
	return cmd
}

func newInvestAssetHoldingsCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var symbol string

	cmd := &cobra.Command{
		Use:   "holdings",
		Short: "Get account holdings.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("asset holdings does not accept arguments")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			assetAPI := deps.AssetAPI
			if assetAPI == nil {
				assetAPI = invest.NewClient("", nil)
			}
			holdings, err := assetAPI.GetHoldings(context.Background(), accessToken, accountSeq, strings.TrimSpace(symbol))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), holdings); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	applyHelp(cmd, "getHoldings")
	return cmd
}
