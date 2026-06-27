package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestStockInfoCommand(deps Dependencies) *cobra.Command {
	cmd := newGroupCommand("stock-info", "Read Toss Invest stock information.")
	cmd.AddCommand(newInvestStockInfoStocksCommand(deps))
	cmd.AddCommand(newInvestStockInfoWarningsCommand(deps))
	return cmd
}

func newInvestStockInfoStocksCommand(deps Dependencies) *cobra.Command {
	var symbols string

	cmd := &cobra.Command{
		Use:   "stocks",
		Short: "Get stock information.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("stock-info stocks does not accept arguments")
			}
			if strings.TrimSpace(symbols) == "" {
				return apperr.Usage("--symbols is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			stockInfo := deps.StockInfo
			if stockInfo == nil {
				stockInfo = invest.NewClient("", nil)
			}
			stocks, err := stockInfo.GetStocks(context.Background(), accessToken, strings.TrimSpace(symbols))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), stocks); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbols, "symbols", "", "Comma-separated Toss Invest symbols.")
	applyHelp(cmd, "getStocks")
	return cmd
}

func newInvestStockInfoWarningsCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "warnings <symbol>",
		Short: "Get stock warnings.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return apperr.Usage("stock-info warnings requires exactly one symbol")
			}
			if strings.TrimSpace(args[0]) == "" {
				return apperr.Usage("symbol is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			stockInfo := deps.StockInfo
			if stockInfo == nil {
				stockInfo = invest.NewClient("", nil)
			}
			warnings, err := stockInfo.GetStockWarnings(context.Background(), accessToken, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), warnings); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "getStockWarnings")
	return cmd
}
