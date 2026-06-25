package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

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
	SecretStore auth.SecretStore
	EnvLookup   auth.EnvLookup
	TokenIssuer auth.TokenIssuer
	AccountAPI  AccountAPI
	MarketData  MarketDataAPI
	OrderInfo   OrderInfoAPI
}

type AccountAPI interface {
	GetAccounts(ctx context.Context, accessToken string) (invest.AccountsResponse, error)
}

type MarketDataAPI interface {
	GetPrices(ctx context.Context, symbols string) (invest.PricesResponse, error)
}

type OrderInfoAPI interface {
	GetBuyingPower(ctx context.Context, accessToken string, accountSeq int64, currency string) (invest.BuyingPowerResponse, error)
	GetSellableQuantity(ctx context.Context, accessToken string, accountSeq int64, symbol string) (invest.SellableQuantityResponse, error)
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
	cmd := &cobra.Command{
		Use:           "tosscli",
		Short:         "Unofficial CLI built on public Toss Open APIs.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

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

func newInvestCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invest",
		Short: "Toss Invest Open API commands.",
	}
	cmd.AddCommand(newInvestAccountCommand(deps))
	cmd.AddCommand(newInvestAuthCommand(deps))
	cmd.AddCommand(newInvestMarketDataCommand(deps))
	cmd.AddCommand(newInvestOrderInfoCommand(deps))
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
	return &cobra.Command{
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
}

func newInvestMarketDataCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market-data",
		Short: "Read Toss Invest market data.",
	}
	cmd.AddCommand(newInvestMarketDataPricesCommand(deps))
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
			marketData := deps.MarketData
			if marketData == nil {
				marketData = invest.NewClient("", nil)
			}
			prices, err := marketData.GetPrices(context.Background(), strings.TrimSpace(symbols))
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
	return cmd
}

func newInvestOrderInfoCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order-info",
		Short: "Read Toss Invest order information.",
	}
	cmd.AddCommand(newInvestOrderInfoBuyingPowerCommand(deps))
	cmd.AddCommand(newInvestOrderInfoSellableQuantityCommand(deps))
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
			credentials, err := readLoginCredentials(cmd, clientID, clientSecret)
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
	return cmd
}

func newInvestAuthLogoutCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
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
}

func newInvestAuthTokenCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
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
}

func newInvestAuthStatusCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
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
}

func readLoginCredentials(cmd *cobra.Command, clientID string, clientSecret string) (auth.Credentials, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	reader := bufio.NewReader(cmd.InOrStdin())

	if clientID == "" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Client ID: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client ID", apperr.ExitUsage, err)
		}
		clientID = strings.TrimSpace(line)
	}
	if clientSecret == "" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Client Secret: ")
		line, err := reader.ReadString('\n')
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
