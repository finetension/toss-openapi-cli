package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestOrderHistoryCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order-history",
		Short: "Read Toss Invest order history.",
	}
	cmd.AddCommand(newInvestOrderHistoryListCommand(deps))
	cmd.AddCommand(newInvestOrderHistoryGetCommand(deps))
	return cmd
}

func newInvestOrderHistoryListCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var status string
	var symbol string
	var from string
	var to string
	var cursor string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List orders.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("order-history list does not accept arguments")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			if strings.TrimSpace(status) == "" {
				return apperr.Usage("--status is required")
			}
			if _, err := allowedValue("--status", status, "OPEN", "CLOSED"); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderHistory := deps.OrderHistory
			if orderHistory == nil {
				orderHistory = invest.NewClient("", nil)
			}
			params := invest.OrderListParams{
				Status: strings.ToUpper(strings.TrimSpace(status)),
				Symbol: strings.TrimSpace(symbol),
				From:   strings.TrimSpace(from),
				To:     strings.TrimSpace(to),
				Cursor: strings.TrimSpace(cursor),
			}
			if cmd.Flags().Changed("limit") {
				params.Limit = limit
			}
			orders, err := orderHistory.GetOrders(context.Background(), accessToken, accountSeq, params)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), orders); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	cmd.Flags().StringVar(&status, "status", "", "Order lifecycle status: OPEN or CLOSED.")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	cmd.Flags().StringVar(&from, "from", "", "Start date, inclusive, in YYYY-MM-DD.")
	cmd.Flags().StringVar(&to, "to", "", "End date, inclusive, in YYYY-MM-DD.")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor.")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size for CLOSED orders.")
	applyHelp(cmd, "getOrders")
	return cmd
}

func newInvestOrderHistoryGetCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64

	cmd := &cobra.Command{
		Use:   "get <orderId>",
		Short: "Get one order.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return apperr.Usage("order-history get requires exactly one orderId")
			}
			if strings.TrimSpace(args[0]) == "" {
				return apperr.Usage("orderId is required")
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
			orderHistory := deps.OrderHistory
			if orderHistory == nil {
				orderHistory = invest.NewClient("", nil)
			}
			order, err := orderHistory.GetOrder(context.Background(), accessToken, accountSeq, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), order); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	applyHelp(cmd, "getOrder")
	return cmd
}
