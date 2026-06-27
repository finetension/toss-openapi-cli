package cli

import (
	"context"
	"io"

	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
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
	PublicIP     PublicIPResolver
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

type PublicIPResolver interface {
	PublicIP(ctx context.Context) (string, error)
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
