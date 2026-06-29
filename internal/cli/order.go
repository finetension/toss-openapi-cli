package cli

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestOrderCommand(deps Dependencies) *cobra.Command {
	cmd := newGroupCommand("order", "Manage Toss Invest orders.")
	cmd.AddCommand(newInvestOrderCreateCommand(deps))
	cmd.AddCommand(newInvestOrderModifyCommand(deps))
	cmd.AddCommand(newInvestOrderCancelCommand(deps))
	return cmd
}

func newInvestOrderCreateCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var clientOrderID string
	var symbol string
	var side string
	var orderType string
	var timeInForce string
	var quantity string
	var price string
	var orderAmount string
	var confirmHighValueOrder bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an order.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("order create does not accept arguments")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			if strings.TrimSpace(symbol) == "" {
				return apperr.Usage("--symbol is required")
			}
			if strings.TrimSpace(side) == "" {
				return apperr.Usage("--side is required")
			}
			if strings.TrimSpace(orderType) == "" {
				return apperr.Usage("--order-type is required")
			}
			if _, err := allowedValue("--side", side, "BUY", "SELL"); err != nil {
				return err
			}
			if _, err := allowedValue("--order-type", orderType, "LIMIT", "MARKET"); err != nil {
				return err
			}
			if strings.TrimSpace(timeInForce) != "" {
				if _, err := allowedValue("--time-in-force", timeInForce, "DAY", "CLS"); err != nil {
					return err
				}
			}
			hasQuantity := strings.TrimSpace(quantity) != ""
			hasOrderAmount := strings.TrimSpace(orderAmount) != ""
			if hasQuantity == hasOrderAmount {
				return apperr.Usage("exactly one of --quantity or --order-amount is required")
			}
			if strings.EqualFold(strings.TrimSpace(orderType), "LIMIT") && strings.TrimSpace(price) == "" {
				return apperr.Usage("--price is required for LIMIT orders")
			}
			if strings.EqualFold(strings.TrimSpace(orderType), "MARKET") && strings.TrimSpace(price) != "" {
				return apperr.Usage("--price is not allowed for MARKET orders")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := buildOrderCreateRequest(clientOrderID, symbol, side, orderType, timeInForce, quantity, price, orderAmount, confirmHighValueOrder)
			if err != nil {
				return err
			}
			if dryRun {
				return writeDryRun(cmd, "POST", "/api/v1/orders", accountSeq, input)
			}

			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderAPI := deps.OrderAPI
			if orderAPI == nil {
				orderAPI = invest.NewClient("", nil)
			}
			order, err := orderAPI.CreateOrder(context.Background(), accessToken, accountSeq, input)
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
	cmd.Flags().StringVar(&clientOrderID, "client-order-id", "", "Client order idempotency key.")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	cmd.Flags().StringVar(&side, "side", "", "Order side: BUY or SELL.")
	cmd.Flags().StringVar(&orderType, "order-type", "", "Order type: LIMIT or MARKET.")
	cmd.Flags().StringVar(&timeInForce, "time-in-force", "", "Time in force.")
	cmd.Flags().StringVar(&quantity, "quantity", "", "Order quantity.")
	cmd.Flags().StringVar(&price, "price", "", "Order price.")
	cmd.Flags().StringVar(&orderAmount, "order-amount", "", "Order amount.")
	cmd.Flags().BoolVar(&confirmHighValueOrder, "confirm-high-value-order", false, "Confirm high-value order.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Build the request without sending it.")
	applyHelp(cmd, "createOrder")
	return cmd
}

func newInvestOrderModifyCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var orderType string
	var quantity string
	var price string
	var confirmHighValueOrder bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "modify <orderId>",
		Short: "Modify an order.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return apperr.Usage("order modify requires exactly one orderId")
			}
			if strings.TrimSpace(args[0]) == "" {
				return apperr.Usage("orderId is required")
			}
			if !cmd.Flags().Changed("account-seq") {
				return apperr.Usage("--account-seq is required")
			}
			if strings.TrimSpace(orderType) == "" {
				return apperr.Usage("--order-type is required")
			}
			if _, err := allowedValue("--order-type", orderType, "LIMIT", "MARKET"); err != nil {
				return err
			}
			if strings.EqualFold(strings.TrimSpace(orderType), "LIMIT") && strings.TrimSpace(price) == "" {
				return apperr.Usage("--price is required for LIMIT orders")
			}
			if strings.EqualFold(strings.TrimSpace(orderType), "MARKET") && strings.TrimSpace(price) != "" {
				return apperr.Usage("--price is not allowed for MARKET orders")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			orderID := strings.TrimSpace(args[0])
			input := invest.OrderModifyRequest{
				OrderType:             strings.ToUpper(strings.TrimSpace(orderType)),
				Quantity:              strings.TrimSpace(quantity),
				Price:                 strings.TrimSpace(price),
				ConfirmHighValueOrder: confirmHighValueOrder,
			}
			if dryRun {
				return writeDryRun(cmd, "POST", "/api/v1/orders/"+orderID+"/modify", accountSeq, input)
			}

			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderAPI := deps.OrderAPI
			if orderAPI == nil {
				orderAPI = invest.NewClient("", nil)
			}
			order, err := orderAPI.ModifyOrder(context.Background(), accessToken, accountSeq, orderID, input)
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
	cmd.Flags().StringVar(&orderType, "order-type", "", "Order type: LIMIT or MARKET.")
	cmd.Flags().StringVar(&quantity, "quantity", "", "Order quantity.")
	cmd.Flags().StringVar(&price, "price", "", "Order price.")
	cmd.Flags().BoolVar(&confirmHighValueOrder, "confirm-high-value-order", false, "Confirm high-value order.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Build the request without sending it.")
	applyHelp(cmd, "modifyOrder")
	return cmd
}

func newInvestOrderCancelCommand(deps Dependencies) *cobra.Command {
	var accountSeq int64
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cancel <orderId>",
		Short: "Cancel an order.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return apperr.Usage("order cancel requires exactly one orderId")
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
			orderID := strings.TrimSpace(args[0])
			if dryRun {
				return writeDryRun(cmd, "POST", "/api/v1/orders/"+orderID+"/cancel", accountSeq, map[string]any{})
			}

			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			orderAPI := deps.OrderAPI
			if orderAPI == nil {
				orderAPI = invest.NewClient("", nil)
			}
			order, err := orderAPI.CancelOrder(context.Background(), accessToken, accountSeq, orderID)
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Build the request without sending it.")
	applyHelp(cmd, "cancelOrder")
	return cmd
}

func buildOrderCreateRequest(clientOrderID string, symbol string, side string, orderType string, timeInForce string, quantity string, price string, orderAmount string, confirmHighValueOrder bool) (invest.OrderCreateRequest, error) {
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		generated, err := newClientOrderID()
		if err != nil {
			return invest.OrderCreateRequest{}, err
		}
		clientOrderID = generated
	}

	return invest.OrderCreateRequest{
		ClientOrderID:         clientOrderID,
		Symbol:                strings.TrimSpace(symbol),
		Side:                  strings.ToUpper(strings.TrimSpace(side)),
		OrderType:             strings.ToUpper(strings.TrimSpace(orderType)),
		TimeInForce:           strings.ToUpper(strings.TrimSpace(timeInForce)),
		Quantity:              strings.TrimSpace(quantity),
		Price:                 strings.TrimSpace(price),
		OrderAmount:           strings.TrimSpace(orderAmount),
		ConfirmHighValueOrder: confirmHighValueOrder,
	}, nil
}

func newClientOrderID() (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", apperr.Wrap(apperr.CodeUnexpected, "Failed to generate client order ID", apperr.ExitUnexpected, err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
