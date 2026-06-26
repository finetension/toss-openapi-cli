package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/finetension/toss-openapi-cli/internal/version"
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type Dependencies struct {
	SecretStore  auth.SecretStore
	EnvLookup    auth.EnvLookup
	TokenIssuer  auth.TokenIssuer
	AccountAPI   AccountAPI
	MarketData   MarketDataAPI
	MarketInfo   MarketInfoAPI
	AssetAPI     AssetAPI
	StockInfo    StockInfoAPI
	OrderInfo    OrderInfoAPI
	OrderHistory OrderHistoryAPI
	OrderAPI     OrderAPI
}

type AccountAPI interface {
	GetAccounts(ctx context.Context, accessToken string) (invest.AccountsResponse, error)
}

type MarketDataAPI interface {
	GetPrices(ctx context.Context, accessToken string, symbols string) (invest.PricesResponse, error)
	GetOrderbook(ctx context.Context, accessToken string, symbol string) (invest.OrderbookResponse, error)
	GetTrades(ctx context.Context, accessToken string, symbol string, count int) (invest.TradesResponse, error)
	GetPriceLimit(ctx context.Context, accessToken string, symbol string) (invest.PriceLimitResponse, error)
	GetCandles(ctx context.Context, accessToken string, params invest.CandleParams) (invest.CandlesResponse, error)
}

type AssetAPI interface {
	GetHoldings(ctx context.Context, accessToken string, accountSeq int64, symbol string) (invest.HoldingsResponse, error)
}

type StockInfoAPI interface {
	GetStocks(ctx context.Context, accessToken string, symbols string) (invest.StocksResponse, error)
	GetStockWarnings(ctx context.Context, accessToken string, symbol string) (invest.StockWarningsResponse, error)
}

type MarketInfoAPI interface {
	GetExchangeRate(ctx context.Context, accessToken string, params invest.ExchangeRateParams) (invest.ExchangeRateResponse, error)
	GetMarketCalendar(ctx context.Context, accessToken string, market string, date string) (invest.MarketCalendarResponse, error)
}

type OrderInfoAPI interface {
	GetBuyingPower(ctx context.Context, accessToken string, accountSeq int64, currency string) (invest.BuyingPowerResponse, error)
	GetSellableQuantity(ctx context.Context, accessToken string, accountSeq int64, symbol string) (invest.SellableQuantityResponse, error)
	GetCommissions(ctx context.Context, accessToken string, accountSeq int64) (invest.CommissionsResponse, error)
}

type OrderHistoryAPI interface {
	GetOrders(ctx context.Context, accessToken string, accountSeq int64, params invest.OrderListParams) (invest.OrdersResponse, error)
	GetOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (invest.OrderResponse, error)
}

type OrderAPI interface {
	CreateOrder(ctx context.Context, accessToken string, accountSeq int64, input invest.OrderCreateRequest) (invest.OrderMutationResponse, error)
	ModifyOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string, input invest.OrderModifyRequest) (invest.OrderMutationResponse, error)
	CancelOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (invest.OrderMutationResponse, error)
}

func Execute() int {
	streams := IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	cmd := NewRootCommand(streams, Dependencies{})
	if err := cmd.Execute(); err != nil {
		return output.WriteError(cmd.OutOrStdout(), normalizeCobraError(err))
	}
	return apperr.ExitSuccess
}

func NewRootCommand(streams IOStreams, deps Dependencies) *cobra.Command {
	var showVersion bool
	cmd := &cobra.Command{
		Use:           "tosscli",
		Short:         "Unofficial CLI built on public Toss Open APIs.",
		Long:          "Unofficial CLI built on public Toss Open APIs.\n\nSuccessful command output is JSON on stdout. Errors are structured JSON.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage(fmt.Sprintf("unknown command %q", args[0]))
			}
			if showVersion {
				if err := output.WriteJSON(cmd.OutOrStdout(), version.Get()); err != nil {
					return apperr.Unexpected(err)
				}
				return nil
			}
			return cmd.Help()
		},
	}
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information.")

	if streams.Out != nil {
		cmd.SetOut(streams.Out)
	}
	if streams.ErrOut != nil {
		cmd.SetErr(streams.ErrOut)
	}
	if streams.In != nil {
		cmd.SetIn(streams.In)
	}

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newDoctorCommand(deps))
	cmd.AddCommand(newInvestCommand(deps))
	return cmd
}

func ExecuteForTest(args ...string) (stdout string, stderr string, exitCode int) {
	return ExecuteForTestWithDeps(Dependencies{}, args...)
}

func ExecuteForTestWithDeps(deps Dependencies, args ...string) (stdout string, stderr string, exitCode int) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(IOStreams{Out: &out, ErrOut: &errOut}, deps)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		exitCode = output.WriteError(&out, normalizeCobraError(err))
		return out.String(), errOut.String(), exitCode
	}
	return out.String(), errOut.String(), apperr.ExitSuccess
}

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

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	Source       string `json:"source,omitempty"`
	Version      string `json:"version,omitempty"`
	Commit       string `json:"commit,omitempty"`
	Date         string `json:"date,omitempty"`
	BuiltBy      string `json:"builtBy,omitempty"`
	OS           string `json:"os,omitempty"`
	Arch         string `json:"arch,omitempty"`
	Valid        *bool  `json:"valid,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	AccountCount *int   `json:"accountCount,omitempty"`
}

func newDoctorCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local Toss Invest CLI readiness.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("doctor does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			report := runDoctor(context.Background(), deps)
			if err := output.WriteJSON(cmd.OutOrStdout(), report); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "cli:doctor")
	return cmd
}

func runDoctor(ctx context.Context, deps Dependencies) doctorReport {
	service := auth.NewService(deps.SecretStore, deps.EnvLookup)
	status := service.Status()

	checks := []doctorCheck{
		doctorVersionCheck(),
		doctorCredentialsCheck(status.Credentials, status.Token),
	}

	issuer := deps.TokenIssuer
	if issuer == nil {
		issuer = invest.NewClient("", nil)
	}
	accessToken, tokenStatus, tokenErr := doctorAccessToken(ctx, service, issuer)
	checks = append(checks, doctorTokenCheck(tokenStatus, tokenErr))

	accountCheck := doctorCheck{
		Name:   "account-list",
		Status: "skipped",
	}
	if tokenErr == nil {
		accountAPI := deps.AccountAPI
		if accountAPI == nil {
			accountAPI = invest.NewClient("", nil)
		}
		accounts, err := accountAPI.GetAccounts(ctx, accessToken)
		if err != nil {
			accountCheck.Status = "fail"
			accountCheck.Message = doctorErrorMessage(err)
		} else {
			accountCount := len(accounts.Result)
			accountCheck.Status = "ok"
			accountCheck.AccountCount = &accountCount
		}
	} else {
		accountCheck.Message = "token check failed"
	}
	checks = append(checks, accountCheck)

	reportStatus := "ok"
	for _, check := range checks {
		if check.Status == "fail" {
			reportStatus = "fail"
			break
		}
	}
	return doctorReport{Status: reportStatus, Checks: checks}
}

func doctorVersionCheck() doctorCheck {
	info := version.Get()
	return doctorCheck{
		Name:    "version",
		Status:  "ok",
		Version: info.Version,
		Commit:  info.Commit,
		Date:    info.Date,
		BuiltBy: info.BuiltBy,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

func doctorCredentialsCheck(status auth.CredentialStatus, tokenStatus auth.TokenStatus) doctorCheck {
	check := doctorCheck{
		Name:   "credentials",
		Status: "fail",
		Source: status.Source,
	}
	if status.Configured {
		check.Status = "ok"
		return check
	}
	if tokenStatus.Configured && tokenStatus.Valid && tokenStatus.Source == "env" {
		check.Status = "skipped"
		check.Message = "Access token is provided directly"
		return check
	}
	check.Message = "Toss Invest credentials are not configured"
	return check
}

func doctorAccessToken(ctx context.Context, service *auth.Service, issuer auth.TokenIssuer) (string, auth.TokenStatus, error) {
	accessToken, err := service.AccessToken(ctx, issuer)
	if err != nil {
		return "", auth.TokenStatus{Source: "missing"}, err
	}
	tokenStatus, statusErr := service.Token(ctx, issuer)
	if statusErr != nil {
		return accessToken, auth.TokenStatus{Source: "missing"}, statusErr
	}
	return accessToken, tokenStatus, nil
}

func doctorTokenCheck(status auth.TokenStatus, err error) doctorCheck {
	check := doctorCheck{
		Name:      "token",
		Status:    "fail",
		Source:    status.Source,
		Valid:     &status.Valid,
		ExpiresAt: status.ExpiresAt,
	}
	if err != nil {
		check.Message = doctorErrorMessage(err)
		return check
	}
	if status.Configured && status.Valid {
		check.Status = "ok"
	}
	return check
}

func doctorErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	normalized := normalizeCobraError(err)
	app := apperr.FromError(normalized)
	if app != nil {
		return app.Message
	}
	return err.Error()
}

func newInvestCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invest",
		Short: "Toss Invest Open API commands.",
	}
	cmd.AddCommand(newInvestAccountCommand(deps))
	cmd.AddCommand(newInvestAssetCommand(deps))
	cmd.AddCommand(newInvestAuthCommand(deps))
	cmd.AddCommand(newInvestMarketDataCommand(deps))
	cmd.AddCommand(newInvestMarketInfoCommand(deps))
	cmd.AddCommand(newInvestOrderInfoCommand(deps))
	cmd.AddCommand(newInvestOrderHistoryCommand(deps))
	cmd.AddCommand(newInvestOrderCommand(deps))
	cmd.AddCommand(newInvestStockInfoCommand(deps))
	return cmd
}

func newInvestAccountCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage Toss Invest accounts.",
	}
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
				Interval: strings.TrimSpace(interval),
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

func newInvestMarketInfoCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-info",
		Short: "Read Toss Invest market information.",
	}
	cmd.AddCommand(newInvestMarketInfoExchangeRateCommand(deps))
	cmd.AddCommand(newInvestMarketInfoCalendarCommand(deps))
	return cmd
}

func newInvestMarketInfoExchangeRateCommand(deps Dependencies) *cobra.Command {
	var baseCurrency string
	var quoteCurrency string
	var dateTime string

	cmd := &cobra.Command{
		Use:   "exchange-rate",
		Short: "Get an exchange rate.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-info exchange-rate does not accept arguments")
			}
			if strings.TrimSpace(baseCurrency) == "" {
				return apperr.Usage("--base-currency is required")
			}
			if strings.TrimSpace(quoteCurrency) == "" {
				return apperr.Usage("--quote-currency is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			marketInfo := deps.MarketInfo
			if marketInfo == nil {
				marketInfo = invest.NewClient("", nil)
			}
			rate, err := marketInfo.GetExchangeRate(context.Background(), accessToken, invest.ExchangeRateParams{
				BaseCurrency:  strings.ToUpper(strings.TrimSpace(baseCurrency)),
				QuoteCurrency: strings.ToUpper(strings.TrimSpace(quoteCurrency)),
				DateTime:      strings.TrimSpace(dateTime),
			})
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), rate); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&baseCurrency, "base-currency", "", "Base currency.")
	cmd.Flags().StringVar(&quoteCurrency, "quote-currency", "", "Quote currency.")
	cmd.Flags().StringVar(&dateTime, "date-time", "", "Exchange-rate timestamp.")
	applyHelp(cmd, "getExchangeRate")
	return cmd
}

func newInvestMarketInfoCalendarCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Get market calendars.",
	}
	cmd.AddCommand(newInvestMarketInfoCalendarMarketCommand(deps, "kr", "KR"))
	cmd.AddCommand(newInvestMarketInfoCalendarMarketCommand(deps, "us", "US"))
	return cmd
}

func newInvestMarketInfoCalendarMarketCommand(deps Dependencies, use string, market string) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   use,
		Short: "Get " + market + " market calendar.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("market-info calendar " + use + " does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, err := accessTokenForInvest(context.Background(), deps)
			if err != nil {
				return err
			}
			marketInfo := deps.MarketInfo
			if marketInfo == nil {
				marketInfo = invest.NewClient("", nil)
			}
			calendar, err := marketInfo.GetMarketCalendar(context.Background(), accessToken, market, strings.TrimSpace(date))
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), calendar); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Calendar date in YYYY-MM-DD.")
	if market == "KR" {
		applyHelp(cmd, "getKrMarketCalendar")
	} else {
		applyHelp(cmd, "getUsMarketCalendar")
	}
	return cmd
}

func newInvestStockInfoCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stock-info",
		Short: "Read Toss Invest stock information.",
	}
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

func newInvestOrderInfoCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order-info",
		Short: "Read Toss Invest order information.",
	}
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
			buyingPower, err := orderInfo.GetBuyingPower(context.Background(), accessToken, accountSeq, strings.TrimSpace(currency))
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

func accessTokenForInvest(ctx context.Context, deps Dependencies) (string, error) {
	service := auth.NewService(deps.SecretStore, deps.EnvLookup)
	issuer := deps.TokenIssuer
	if issuer == nil {
		issuer = invest.NewClient("", nil)
	}
	return service.AccessToken(ctx, issuer)
}

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
				Status: strings.TrimSpace(status),
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

func newInvestOrderCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Manage Toss Invest orders.",
	}
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
			hasQuantity := strings.TrimSpace(quantity) != ""
			hasOrderAmount := strings.TrimSpace(orderAmount) != ""
			if hasQuantity == hasOrderAmount {
				return apperr.Usage("exactly one of --quantity or --order-amount is required")
			}
			if strings.EqualFold(strings.TrimSpace(orderType), "LIMIT") && strings.TrimSpace(price) == "" {
				return apperr.Usage("--price is required for LIMIT orders")
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
			if strings.EqualFold(strings.TrimSpace(orderType), "LIMIT") && strings.TrimSpace(price) == "" {
				return apperr.Usage("--price is required for LIMIT orders")
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

func newClientOrderID() (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", apperr.Wrap(apperr.CodeUnexpected, "Failed to generate client order ID", apperr.ExitUnexpected, err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func newInvestAuthCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Toss Invest authentication.",
	}
	cmd.AddCommand(newInvestAuthLoginCommand(deps))
	cmd.AddCommand(newInvestAuthLogoutCommand(deps))
	cmd.AddCommand(newInvestAuthStatusCommand(deps))
	cmd.AddCommand(newInvestAuthTokenCommand(deps))
	return cmd
}

func newInvestAuthLoginCommand(deps Dependencies) *cobra.Command {
	var clientID string
	var clientSecret string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Configure Toss Invest credentials.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth login does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			credentials, err := readLoginCredentials(cmd, deps.EnvLookup, clientID, clientSecret)
			if err != nil {
				return err
			}
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			issuer := deps.TokenIssuer
			if issuer == nil {
				issuer = invest.NewClient("", nil)
			}
			status, err := service.Login(context.Background(), issuer, credentials)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&clientID, "client-id", "", "Toss Invest client ID.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Toss Invest client secret.")
	applyHelp(cmd, "cli:auth-login")
	return cmd
}

func newInvestAuthLogoutCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear stored Toss Invest credentials and token.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth logout does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			status, err := service.Logout()
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "cli:auth-logout")
	return cmd
}

func newInvestAuthTokenCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Issue or refresh a Toss Invest access token.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth token does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			issuer := deps.TokenIssuer
			if issuer == nil {
				issuer = invest.NewClient("", nil)
			}
			status, err := service.Token(context.Background(), issuer)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "cli:auth-token")
	return cmd
}

func newInvestAuthStatusCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show Toss Invest authentication status.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth status does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			if err := output.WriteJSON(cmd.OutOrStdout(), service.Status()); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "cli:auth-status")
	return cmd
}

func readLoginCredentials(cmd *cobra.Command, env auth.EnvLookup, clientID string, clientSecret string) (auth.Credentials, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if env == nil {
		env = auth.DefaultEnvLookup
	}
	if clientID == "" {
		if value, ok := env("TOSS_INVEST_CLIENT_ID"); ok {
			clientID = strings.TrimSpace(value)
		}
	}
	if clientSecret == "" {
		if value, ok := env("TOSS_INVEST_CLIENT_SECRET"); ok {
			clientSecret = strings.TrimSpace(value)
		}
	}

	reader := bufio.NewReader(cmd.InOrStdin())

	if clientID == "" {
		line, err := readPromptLine(cmd.ErrOrStderr(), reader, "Client ID: ")
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client ID", apperr.ExitUsage, err)
		}
		clientID = strings.TrimSpace(line)
	}
	if clientSecret == "" {
		line, err := readSecretPrompt(cmd, reader)
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client secret", apperr.ExitUsage, err)
		}
		clientSecret = strings.TrimSpace(line)
	}
	if clientID == "" {
		return auth.Credentials{}, apperr.Usage("client ID is required")
	}
	if clientSecret == "" {
		return auth.Credentials{}, apperr.Usage("client secret is required")
	}
	return auth.Credentials{ClientID: clientID, ClientSecret: clientSecret}, nil
}

func readPromptLine(stderr io.Writer, reader *bufio.Reader, prompt string) (string, error) {
	_, _ = fmt.Fprint(stderr, prompt)
	return reader.ReadString('\n')
}

func readSecretPrompt(cmd *cobra.Command, reader *bufio.Reader) (string, error) {
	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprint(stderr, "Client Secret: ")
	if file, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		secret, err := term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(stderr)
		return string(secret), err
	}
	return reader.ReadString('\n')
}

func normalizeCobraError(err error) error {
	if err == nil {
		return nil
	}
	var app *apperr.AppError
	if errors.As(err, &app) {
		return app
	}
	if errors.Is(err, auth.ErrCredentialsMissing) {
		return apperr.New(apperr.CodeAuthConfig, "Missing Toss Invest credentials", apperr.ExitAuthConfig)
	}
	var apiErr *invest.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		if code == "" {
			code = apperr.CodeAPI
		}
		message := apiErr.Message
		if message == "" {
			message = "Toss Invest API request failed"
		}
		return apperr.New(code, message, apperr.ExitAPI)
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "invalid argument") {
		return apperr.Usage(msg)
	}
	return apperr.Unexpected(err)
}
