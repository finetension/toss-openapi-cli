package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestMarketDataCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-data",
		Short: "Read Toss Invest market data.",
	}
	cmd.AddCommand(newInvestMarketDataOrderbookCommand(deps))
	cmd.AddCommand(newInvestMarketDataCandlesCommand(deps))
	cmd.AddCommand(newInvestMarketDataPriceLimitsCommand(deps))
	cmd.AddCommand(newInvestMarketDataPricesCommand(deps))
	cmd.AddCommand(newInvestMarketDataTradesCommand(deps))
	return cmd
}

func newInvestMarketDataOrderbookCommand(deps Dependencies) *cobra.Command {
	var symbol string

	cmd := &cobra.Command{
		Use:   "orderbook",
		Short: "Get orderbook for a symbol.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-data orderbook does not accept arguments")
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
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			orderbook, err := marketData.GetOrderbook(context.Background(), accessToken, strings.TrimSpace(symbol))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), orderbook); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	applyHelp(cmd, "getOrderbook")
	return cmd
}

func newInvestMarketDataPricesCommand(deps Dependencies) *cobra.Command {
	var symbols string

	cmd := &cobra.Command{
		Use:   "prices",
		Short: "Get current prices for one or more symbols.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-data prices does not accept arguments")
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
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			prices, err := marketData.GetPrices(context.Background(), accessToken, strings.TrimSpace(symbols))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), prices); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbols, "symbols", "", "Comma-separated Toss Invest symbols.")
	applyHelp(cmd, "getPrices")
	return cmd
}

func newInvestMarketDataTradesCommand(deps Dependencies) *cobra.Command {
	var symbol string
	var count int

	cmd := &cobra.Command{
		Use:   "trades",
		Short: "Get recent trades for a symbol.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-data trades does not accept arguments")
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
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			trades, err := marketData.GetTrades(context.Background(), accessToken, strings.TrimSpace(symbol), count)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), trades); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	cmd.Flags().IntVar(&count, "count", 0, "Trade count.")
	applyHelp(cmd, "getTrades")
	return cmd
}

func newInvestMarketDataPriceLimitsCommand(deps Dependencies) *cobra.Command {
	var symbol string

	cmd := &cobra.Command{
		Use:   "price-limits",
		Short: "Get price limits for a symbol.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-data price-limits does not accept arguments")
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
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			limits, err := marketData.GetPriceLimit(context.Background(), accessToken, strings.TrimSpace(symbol))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), limits); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	applyHelp(cmd, "getPriceLimit")
	return cmd
}

func newInvestMarketDataCandlesCommand(deps Dependencies) *cobra.Command {
	var symbol string
	var interval string
	var count int
	var before string
	var adjusted bool

	cmd := &cobra.Command{
		Use:   "candles",
		Short: "Get candles for a symbol.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-data candles does not accept arguments")
			}
			if strings.TrimSpace(symbol) == "" {
				return apperr.Usage("--symbol is required")
			}
			if strings.TrimSpace(interval) == "" {
				return apperr.Usage("--interval is required")
			}
			if _, err := allowedValue("--interval", interval, "1m", "1d"); err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			params := invest.CandleParams{
				Symbol:   strings.TrimSpace(symbol),
				Interval: strings.ToLower(strings.TrimSpace(interval)),
				Count:    count,
				Before:   strings.TrimSpace(before),
			}
			if cmd.Flags().Changed("adjusted") {
				params.Adjusted = &adjusted
			}
			candles, err := marketData.GetCandles(context.Background(), accessToken, params)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), candles); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "Toss Invest symbol.")
	cmd.Flags().StringVar(&interval, "interval", "", "Candle interval.")
	cmd.Flags().IntVar(&count, "count", 0, "Candle count.")
	cmd.Flags().StringVar(&before, "before", "", "Exclusive upper bound timestamp.")
	cmd.Flags().BoolVar(&adjusted, "adjusted", false, "Request adjusted prices.")
	applyHelp(cmd, "getCandles")
	return cmd
}
