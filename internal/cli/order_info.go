package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestOrderInfoCommand(deps Dependencies) *cobra.Command {
	cmd := newGroupCommand("order-info", "Read Toss Invest order information.")
	cmd.AddCommand(newInvestOrderInfoBuyingPowerCommand(deps))
	cmd.AddCommand(newInvestOrderInfoCommissionsCommand(deps))
	cmd.AddCommand(newInvestOrderInfoSellableQuantityCommand(deps))
	return cmd
}

func newInvestOrderInfoCommissionsCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64

	cmd := &cobra.Command{
		Use:   "commissions",
		Short: "Get account commission rates.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("order-info commissions does not accept arguments")
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
			orderInfo := deps.OrderInfo
			if orderInfo == nil {
				orderInfo = invest.NewClient("", nil)
			}
			commissions, err := orderInfo.GetCommissions(context.Background(), accessToken, accountSeq)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), commissions); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	applyHelp(cmd, "getCommissions")
	return cmd
}

func newInvestOrderInfoBuyingPowerCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var currency string

	cmd := &cobra.Command{
		Use:   "buying-power",
		Short: "Get cash buying power.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("order-info buying-power does not accept arguments")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			if strings.TrimSpace(currency) == "" {
				return apperr.Usage("--currency is required")
			}
			if _, err := allowedValue("--currency", currency, "KRW", "USD"); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderInfo := deps.OrderInfo
			if orderInfo == nil {
				orderInfo = invest.NewClient("", nil)
			}
			buyingPower, err := orderInfo.GetBuyingPower(context.Background(), accessToken, accountSeq, strings.ToUpper(strings.TrimSpace(currency)))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), buyingPower); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	cmd.Flags().StringVar(&currency, "currency", "", "Currency code.")
	applyHelp(cmd, "getBuyingPower")
	return cmd
}

func newInvestOrderInfoSellableQuantityCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var symbol string

	cmd := &cobra.Command{
		Use:   "sellable-quantity",
		Short: "Get sellable quantity for a symbol.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("order-info sellable-quantity does not accept arguments")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			if strings.TrimSpace(symbol) == "" {
				return apperr.Usage("--symbol is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderInfo := deps.OrderInfo
			if orderInfo == nil {
				orderInfo = invest.NewClient("", nil)
			}
			sellableQuantity, err := orderInfo.GetSellableQuantity(context.Background(), accessToken, accountSeq, strings.TrimSpace(symbol))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), sellableQuantity); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&accountSeq, "account-seq", 0, "Toss Invest account sequence.")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	applyHelp(cmd, "getSellableQuantity")
	return cmd
}
