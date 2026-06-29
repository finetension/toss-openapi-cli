package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type commandHelp struct {
	Short       string
	Description string
	OperationID string
	RateLimit   string
	Rules       []string
	Examples    []string
	Flags       map[string]string
}

func applyHelp(cmd *cobra.Command, key string) {
	help, ok := helpCatalog[key]
	if !ok {
		return
	}
	if help.Short != "" {
		cmd.Short = help.Short
	}
	cmd.Long = renderLongHelp(help)
	cmd.Example = renderExamples(help.Examples)
	for name, usage := range help.Flags {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			flag.Usage = usage
		}
	}
}

func renderExamples(examples []string) string {
	if len(examples) == 0 {
		return ""
	}
	lines := make([]string, 0, len(examples))
	for _, example := range examples {
		lines = append(lines, "  "+example)
	}
	return strings.Join(lines, "\n")
}

func renderLongHelp(help commandHelp) string {
	parts := make([]string, 0, 4)
	if help.Description != "" {
		parts = append(parts, strings.TrimSpace(help.Description))
	}
	if help.OperationID != "" || help.RateLimit != "" {
		var lines []string
		if help.OperationID != "" {
			lines = append(lines, fmt.Sprintf("OpenAPI operation: %s.", help.OperationID))
		}
		if help.RateLimit != "" {
			lines = append(lines, fmt.Sprintf("Rate limit group: %s.", help.RateLimit))
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	if len(help.Rules) > 0 {
		lines := []string{"Details:"}
		for _, rule := range help.Rules {
			lines = append(lines, "  - "+rule)
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

var helpCatalog = map[string]commandHelp{
	"cli:doctor": {
		Short:       "Check local Toss Invest CLI readiness.",
		Description: "Checks CLI version, credential availability, token availability, and read-only account list access.",
		Rules: []string{
			"Does not test order execution.",
			"Does not query third-party public IP services unless --show-ip is provided.",
			"Does not print credential values, access tokens, account numbers, or account sequence values.",
		},
		Examples: []string{
			"tosscli doctor",
			"tosscli doctor --show-ip",
		},
		Flags: map[string]string{
			"show-ip": "Query and show the public IP address visible to external services. Useful when Toss Open API returns IP address not allowed.",
		},
	},
	"cli:auth-login": {
		Short:       "Configure Toss Invest credentials.",
		Description: "Stores Toss Invest OAuth2 client credentials in the OS keyring and verifies them by issuing an access token.",
		Rules: []string{
			"Credentials can be passed with flags or entered interactively.",
			"Environment variables TOSS_INVEST_CLIENT_ID and TOSS_INVEST_CLIENT_SECRET override stored credentials.",
		},
		Examples: []string{
			"tosscli invest auth login",
			"tosscli invest auth login --client-id \"$TOSS_INVEST_CLIENT_ID\" --client-secret \"$TOSS_INVEST_CLIENT_SECRET\"",
		},
		Flags: map[string]string{
			"client-id":     "Toss Invest OAuth2 client ID. Optional when TOSS_INVEST_CLIENT_ID is set or interactive input is used.",
			"client-secret": "Toss Invest OAuth2 client secret. Optional when TOSS_INVEST_CLIENT_SECRET is set or interactive input is used.",
		},
	},
	"cli:auth-status": {
		Short:       "Show Toss Invest authentication status.",
		Description: "Shows whether credentials and cached token material are configured without printing secret values.",
		Examples:    []string{"tosscli invest auth status"},
	},
	"cli:auth-token": {
		Short:       "Issue or refresh a Toss Invest access token.",
		Description: "Ensures an access token is available and prints token metadata only. The raw token value is not printed.",
		Examples:    []string{"tosscli invest auth token"},
	},
	"cli:auth-logout": {
		Short:       "Clear stored Toss Invest credentials and token.",
		Description: "Deletes Toss Invest credentials and cached token data from the OS keyring.",
		Examples:    []string{"tosscli invest auth logout"},
	},
	"getAccounts": {
		Short:       "List Toss Invest accounts.",
		Description: "Lists active user accounts. The returned accountSeq is used as the account header value for holdings, order, buying-power, and other account-scoped APIs.",
		OperationID: "getAccounts",
		RateLimit:   "ACCOUNT",
		Examples:    []string{"tosscli invest account list"},
	},
	"getHoldings": {
		Short:       "Get account holdings.",
		Description: "Reads holdings for a Toss Invest account. Supports Korean and US stocks; overseas options and bonds are excluded.",
		OperationID: "getHoldings",
		RateLimit:   "ASSET",
		Examples: []string{
			"tosscli invest asset holdings --account-seq 123456789",
			"tosscli invest asset holdings --account-seq 123456789 --symbol AAPL",
		},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
			"symbol":      "Stock symbol filter. Optional. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
		},
	},
	"getPrices": {
		Short:       "Get current prices for one or more symbols.",
		Description: "Reads current price data for up to 200 comma-separated symbols.",
		OperationID: "getPrices",
		RateLimit:   "MARKET_DATA",
		Examples: []string{
			"tosscli invest market-data prices --symbols AAPL",
			"tosscli invest market-data prices --symbols 005930,000660",
		},
		Flags: map[string]string{
			"symbols": "Stock symbols. Required. Comma-separated, up to 200. Examples: 005930,000660 or AAPL,MSFT. Pattern: letters, digits, '.', ',', '-'.",
		},
	},
	"getOrderbook": {
		Short:       "Get orderbook for a symbol.",
		Description: "Reads bid and ask prices and volumes for a symbol.",
		OperationID: "getOrderbook",
		RateLimit:   "MARKET_DATA",
		Examples:    []string{"tosscli invest market-data orderbook --symbol AAPL"},
		Flags: map[string]string{
			"symbol": "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
		},
	},
	"getTrades": {
		Short:       "Get recent trades for a symbol.",
		Description: "Reads same-day recent trades for a symbol.",
		OperationID: "getTrades",
		RateLimit:   "MARKET_DATA",
		Examples:    []string{"tosscli invest market-data trades --symbol AAPL --count 20"},
		Flags: map[string]string{
			"symbol": "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
			"count":  "Trade count. Optional. Range: 1-50. Default: 50.",
		},
	},
	"getPriceLimit": {
		Short:       "Get daily price limits for a symbol.",
		Description: "Reads upper and lower daily price limits for a symbol. Markets without price limits may return null values.",
		OperationID: "getPriceLimit",
		RateLimit:   "MARKET_DATA",
		Examples:    []string{"tosscli invest market-data price-limits --symbol 005930"},
		Flags: map[string]string{
			"symbol": "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
		},
	},
	"getCandles": {
		Short:       "Get candles for a symbol.",
		Description: "Reads OHLCV candle data for a symbol. Returns up to 200 candles.",
		OperationID: "getCandles",
		RateLimit:   "MARKET_DATA_CHART",
		Rules: []string{
			"Paginated responses include nextBefore. Pass that value to --before to request the next page.",
		},
		Examples: []string{
			"tosscli invest market-data candles --symbol AAPL --interval 1d",
			"tosscli invest market-data candles --symbol 005930 --interval 1m --count 100",
		},
		Flags: map[string]string{
			"symbol":   "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
			"interval": "Candle interval. Required. Allowed: 1m, 1d.",
			"count":    "Candle count. Optional. Range: 1-200. Default: 100.",
			"before":   "Exclusive upper bound timestamp. Optional. Format: ISO 8601 date-time. Response nextBefore can be passed here for pagination.",
			"adjusted": "Request adjusted prices. Optional. Default: true in the OpenAPI spec.",
		},
	},
	"getStocks": {
		Short:       "Get stock information.",
		Description: "Reads basic stock reference information for up to 200 comma-separated symbols.",
		OperationID: "getStocks",
		RateLimit:   "STOCK",
		Examples:    []string{"tosscli invest stock-info stocks --symbols AAPL,MSFT"},
		Flags: map[string]string{
			"symbols": "Stock symbols. Required. Comma-separated, up to 200. Examples: 005930,AAPL. Pattern: letters, digits, '.', ',', '-'.",
		},
	},
	"getStockWarnings": {
		Short:       "Get stock warnings.",
		Description: "Reads active buy warnings and volatility interruption information for a symbol.",
		OperationID: "getStockWarnings",
		RateLimit:   "STOCK",
		Examples:    []string{"tosscli invest stock-info warnings AAPL"},
	},
	"getExchangeRate": {
		Short:       "Get an exchange rate.",
		Description: "Reads KRW/USD exchange-rate information. When date-time is omitted, the current effective rate is returned.",
		OperationID: "getExchangeRate",
		RateLimit:   "MARKET_INFO",
		Examples:    []string{"tosscli invest market-info exchange-rate --base-currency KRW --quote-currency USD"},
		Flags: map[string]string{
			"base-currency":  "Base currency. Required. Allowed: KRW, USD.",
			"quote-currency": "Quote currency. Required. Allowed: KRW, USD.",
			"date-time":      "Exchange-rate timestamp. Optional. Format: ISO 8601 date-time. Defaults to the current effective rate.",
		},
	},
	"getKrMarketCalendar": marketCalendarHelp("KR", "Korean", "YYYY-MM-DD, KST basis"),
	"getUsMarketCalendar": marketCalendarHelp("US", "US", "YYYY-MM-DD, US local date"),
	"getOrders": {
		Short:       "List orders.",
		Description: "Lists orders for an account using an order lifecycle group filter.",
		OperationID: "getOrders",
		RateLimit:   "ORDER_HISTORY",
		Rules: []string{
			"status=OPEN returns all open orders; limit and cursor are ignored.",
			"status=CLOSED uses limit, cursor, from, and to for pagination and date filtering.",
			"from and to are inclusive dates based on orderedAt in KST.",
		},
		Examples: []string{
			"tosscli invest order-history list --account-seq 123456789 --status OPEN",
			"tosscli invest order-history list --account-seq 123456789 --status CLOSED --limit 20",
		},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
			"status":      "Order lifecycle group. Required. Allowed: OPEN, CLOSED.",
			"symbol":      "Stock symbol filter. Optional. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
			"from":        "Start date, inclusive. Optional. Format: YYYY-MM-DD. Based on orderedAt in KST.",
			"to":          "End date, inclusive. Optional. Format: YYYY-MM-DD. Based on orderedAt in KST.",
			"cursor":      "Pagination cursor. Optional. Ignored for OPEN, used for CLOSED.",
			"limit":       "Page size. Optional. Range: 1-100. Default: 20. Ignored for OPEN, used for CLOSED.",
		},
	},
	"getOrder": {
		Short:       "Get one order.",
		Description: "Reads details for a single order in any lifecycle state.",
		OperationID: "getOrder",
		RateLimit:   "ORDER_HISTORY",
		Examples:    []string{"tosscli invest order-history get order-id --account-seq 123456789"},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
		},
	},
	"createOrder": {
		Short:       "Create an order.",
		Description: "Creates a buy or sell order.",
		OperationID: "createOrder",
		RateLimit:   "ORDER",
		Rules: []string{
			"--dry-run prints the request preview as JSON and does not call the Toss API.",
			"Without --dry-run, this command sends a live order request to the Toss API.",
			"Provide exactly one of --quantity or --order-amount.",
			"LIMIT orders require --price.",
			"MARKET orders must not include --price.",
			"--order-amount is for US MARKET buy orders.",
		},
		Examples: []string{
			"tosscli invest order create --dry-run --account-seq 123456789 --symbol AAPL --side BUY --order-type LIMIT --quantity 1 --price 100",
			"tosscli invest order create --dry-run --account-seq 123456789 --symbol AAPL --side BUY --order-type MARKET --order-amount 100.5",
		},
		Flags: orderFlagHelp(true),
	},
	"modifyOrder": {
		Short:       "Modify an order.",
		Description: "Modifies price or quantity for an existing order.",
		OperationID: "modifyOrder",
		RateLimit:   "ORDER",
		Rules: []string{
			"--dry-run prints the request preview as JSON and does not call the Toss API.",
			"Without --dry-run, this command sends a live order modification request to the Toss API.",
			"KR stock orders require --quantity and it must be a positive integer.",
			"US stock orders do not support quantity modification; price changes only.",
			"LIMIT modifications require --price.",
			"MARKET modifications must not include --price.",
		},
		Examples: []string{
			"tosscli invest order modify order-id --dry-run --account-seq 123456789 --order-type LIMIT --quantity 1 --price 101",
		},
		Flags: orderFlagHelp(false),
	},
	"cancelOrder": {
		Short:       "Cancel an order.",
		Description: "Cancels an existing order. Already-filled orders cannot be canceled.",
		OperationID: "cancelOrder",
		RateLimit:   "ORDER",
		Rules: []string{
			"--dry-run prints the request preview as JSON and does not call the Toss API.",
			"Without --dry-run, this command sends a live order cancellation request to the Toss API.",
		},
		Examples: []string{"tosscli invest order cancel order-id --dry-run --account-seq 123456789"},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
			"dry-run":     "Print the request preview as JSON without calling the Toss API.",
		},
	},
	"getBuyingPower": {
		Short:       "Get cash buying power.",
		Description: "Reads cash-based buying power for an account and currency.",
		OperationID: "getBuyingPower",
		RateLimit:   "ORDER_INFO",
		Examples:    []string{"tosscli invest order-info buying-power --account-seq 123456789 --currency USD"},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
			"currency":    "Currency code. Required. Allowed: KRW, USD.",
		},
	},
	"getSellableQuantity": {
		Short:       "Get sellable quantity for a symbol.",
		Description: "Reads sellable quantity for a symbol in an account.",
		OperationID: "getSellableQuantity",
		RateLimit:   "ORDER_INFO",
		Examples:    []string{"tosscli invest order-info sellable-quantity --account-seq 123456789 --symbol AAPL"},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
			"symbol":      "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'.",
		},
	},
	"getCommissions": {
		Short:       "Get account commission rates.",
		Description: "Reads market-specific commission rates for an account.",
		OperationID: "getCommissions",
		RateLimit:   "ORDER_INFO",
		Examples:    []string{"tosscli invest order-info commissions --account-seq 123456789"},
		Flags: map[string]string{
			"account-seq": "Account sequence. Required. Source: tosscli invest account list.",
		},
	},
}

func marketCalendarHelp(market string, label string, dateFormat string) commandHelp {
	operationIDs := map[string]string{
		"KR": "getKrMarketCalendar",
		"US": "getUsMarketCalendar",
	}
	return commandHelp{
		Short:       "Get " + label + " market calendar.",
		Description: "Reads market operating days and session times for the previous, current, and next business day.",
		OperationID: operationIDs[market],
		RateLimit:   "MARKET_INFO",
		Examples:    []string{"tosscli invest market-info calendar " + strings.ToLower(market)},
		Flags: map[string]string{
			"date": "Reference date. Optional. Format: " + dateFormat + ".",
		},
	}
}

func orderFlagHelp(includeCreateOnly bool) map[string]string {
	flags := map[string]string{
		"account-seq":              "Account sequence. Required. Source: tosscli invest account list.",
		"order-type":               "Order type. Required. Allowed: LIMIT, MARKET.",
		"time-in-force":            "Time in force. Optional. Allowed: DAY, CLS, OPG. Example: CLS for LOC with LIMIT.",
		"quantity":                 "Order quantity as a decimal string.",
		"price":                    "Order price as a decimal string. Required for LIMIT, not allowed for MARKET.",
		"confirm-high-value-order": "Confirm high-value order. Optional. Required by the API for selected high-value orders.",
		"dry-run":                  "Print the request preview as JSON without calling the Toss API.",
	}
	if includeCreateOnly {
		flags["symbol"] = "Stock symbol. Required. Examples: 005930, AAPL. Pattern: letters, digits, '.', '-'."
		flags["side"] = "Order side. Required. Allowed: BUY, SELL."
		flags["quantity"] = "Order quantity as a decimal string. Use exactly one of --quantity or --order-amount."
		flags["client-order-id"] = "Client order idempotency key. Optional. Returned as clientOrderId when provided."
		flags["order-amount"] = "Order amount in USD as a decimal string. Use exactly one of --quantity or --order-amount. US MARKET buy only."
	} else {
		flags["quantity"] = "Modified order quantity as a decimal string. KR stock orders require a positive integer. US stock orders do not support quantity modification."
	}
	return flags
}
