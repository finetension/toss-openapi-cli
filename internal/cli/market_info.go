package cli

import (
	"context"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
)

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
			if _, err := allowedValue("--base-currency", baseCurrency, "KRW", "USD"); err != nil {
				return err
			}
			if _, err := allowedValue("--quote-currency", quoteCurrency, "KRW", "USD"); err != nil {
				return err
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
