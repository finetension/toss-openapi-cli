package invest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DefaultBaseURL = "https://openapi.tossinvest.com"

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	httpClient Doer
}

func NewClient(baseURL string, httpClient Doer) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

type OAuth2TokenRequest struct {
	ClientID     string
	ClientSecret string
}

type OAuth2TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type AccountsResponse struct {
	Result []Account `json:"result"`
}

type Account struct {
	AccountNo   string `json:"accountNo"`
	AccountSeq  int64  `json:"accountSeq"`
	AccountType string `json:"accountType"`
}

type PricesResponse struct {
	Result []Price `json:"result"`
}

type Price struct {
	Symbol    string `json:"symbol"`
	Timestamp string `json:"timestamp"`
	LastPrice string `json:"lastPrice"`
	Currency  string `json:"currency"`
}

type BuyingPowerResponse struct {
	Result BuyingPower `json:"result"`
}

type BuyingPower struct {
	Currency        string `json:"currency"`
	CashBuyingPower string `json:"cashBuyingPower"`
}

type SellableQuantityResponse struct {
	Result SellableQuantity `json:"result"`
}

type SellableQuantity struct {
	SellableQuantity string `json:"sellableQuantity"`
}

type HoldingsResponse struct {
	Result json.RawMessage `json:"result"`
}

type CommissionsResponse struct {
	Result json.RawMessage `json:"result"`
}

type OrderListParams struct {
	Status string
	Symbol string
	From   string
	To     string
	Cursor string
	Limit  int
}

type OrdersResponse struct {
	Result json.RawMessage `json:"result"`
}

type OrderResponse struct {
	Result json.RawMessage `json:"result"`
}

type OrderCreateRequest struct {
	ClientOrderID         string `json:"clientOrderId,omitempty"`
	Symbol                string `json:"symbol"`
	Side                  string `json:"side"`
	OrderType             string `json:"orderType"`
	TimeInForce           string `json:"timeInForce,omitempty"`
	Quantity              string `json:"quantity,omitempty"`
	Price                 string `json:"price,omitempty"`
	OrderAmount           string `json:"orderAmount,omitempty"`
	ConfirmHighValueOrder bool   `json:"confirmHighValueOrder,omitempty"`
}

type OrderMutationResponse struct {
	Result json.RawMessage `json:"result"`
}

type APIError struct {
	StatusCode      int
	Code            string
	Message         string
	Reason          string
	RequestID       string
	RawBody         string
	WWWAuthenticate string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	if e.Code != "" {
		return e.Code
	}
	return fmt.Sprintf("http status %d", e.StatusCode)
}

func (c *Client) IssueOAuth2Token(ctx context.Context, input OAuth2TokenRequest) (OAuth2TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", input.ClientID)
	form.Set("client_secret", input.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return OAuth2TokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var out OAuth2TokenResponse
	if err := c.doJSON(req, &out); err != nil {
		return OAuth2TokenResponse{}, err
	}
	return out, nil
}

func (c *Client) GetAccounts(ctx context.Context, accessToken string) (AccountsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/accounts", nil)
	if err != nil {
		return AccountsResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var out AccountsResponse
	if err := c.doJSON(req, &out); err != nil {
		return AccountsResponse{}, err
	}
	return out, nil
}

func (c *Client) GetPrices(ctx context.Context, symbols string) (PricesResponse, error) {
	values := url.Values{}
	values.Set("symbols", symbols)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/prices?"+values.Encode(), nil)
	if err != nil {
		return PricesResponse{}, err
	}
	req.Header.Set("Accept", "application/json")

	var out PricesResponse
	if err := c.doJSON(req, &out); err != nil {
		return PricesResponse{}, err
	}
	return out, nil
}

func (c *Client) GetBuyingPower(ctx context.Context, accessToken string, accountSeq int64, currency string) (BuyingPowerResponse, error) {
	values := url.Values{}
	values.Set("currency", currency)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/buying-power?"+values.Encode(), nil)
	if err != nil {
		return BuyingPowerResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out BuyingPowerResponse
	if err := c.doJSON(req, &out); err != nil {
		return BuyingPowerResponse{}, err
	}
	return out, nil
}

func (c *Client) GetSellableQuantity(ctx context.Context, accessToken string, accountSeq int64, symbol string) (SellableQuantityResponse, error) {
	values := url.Values{}
	values.Set("symbol", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/sellable-quantity?"+values.Encode(), nil)
	if err != nil {
		return SellableQuantityResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out SellableQuantityResponse
	if err := c.doJSON(req, &out); err != nil {
		return SellableQuantityResponse{}, err
	}
	return out, nil
}

func (c *Client) GetHoldings(ctx context.Context, accessToken string, accountSeq int64, symbol string) (HoldingsResponse, error) {
	values := url.Values{}
	if symbol != "" {
		values.Set("symbol", symbol)
	}

	path := c.baseURL + "/api/v1/holdings"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return HoldingsResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out HoldingsResponse
	if err := c.doJSON(req, &out); err != nil {
		return HoldingsResponse{}, err
	}
	return out, nil
}

func (c *Client) GetCommissions(ctx context.Context, accessToken string, accountSeq int64) (CommissionsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/commissions", nil)
	if err != nil {
		return CommissionsResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out CommissionsResponse
	if err := c.doJSON(req, &out); err != nil {
		return CommissionsResponse{}, err
	}
	return out, nil
}

func (c *Client) GetOrders(ctx context.Context, accessToken string, accountSeq int64, params OrderListParams) (OrdersResponse, error) {
	values := url.Values{}
	values.Set("status", params.Status)
	if params.Symbol != "" {
		values.Set("symbol", params.Symbol)
	}
	if params.From != "" {
		values.Set("from", params.From)
	}
	if params.To != "" {
		values.Set("to", params.To)
	}
	if params.Cursor != "" {
		values.Set("cursor", params.Cursor)
	}
	if params.Limit > 0 {
		values.Set("limit", strconv.Itoa(params.Limit))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/orders?"+values.Encode(), nil)
	if err != nil {
		return OrdersResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out OrdersResponse
	if err := c.doJSON(req, &out); err != nil {
		return OrdersResponse{}, err
	}
	return out, nil
}

func (c *Client) GetOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (OrderResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/orders/"+url.PathEscape(orderID), nil)
	if err != nil {
		return OrderResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out OrderResponse
	if err := c.doJSON(req, &out); err != nil {
		return OrderResponse{}, err
	}
	return out, nil
}

func (c *Client) CreateOrder(ctx context.Context, accessToken string, accountSeq int64, input OrderCreateRequest) (OrderMutationResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return OrderMutationResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/orders", bytes.NewReader(body))
	if err != nil {
		return OrderMutationResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out OrderMutationResponse
	if err := c.doJSON(req, &out); err != nil {
		return OrderMutationResponse{}, err
	}
	return out, nil
}

func (c *Client) CancelOrder(ctx context.Context, accessToken string, accountSeq int64, orderID string) (OrderMutationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/orders/"+url.PathEscape(orderID)+"/cancel", strings.NewReader("{}"))
	if err != nil {
		return OrderMutationResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tossinvest-Account", strconv.FormatInt(accountSeq, 10))

	var out OrderMutationResponse
	if err := c.doJSON(req, &out); err != nil {
		return OrderMutationResponse{}, err
	}
	return out, nil
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return parseAPIError(resp, body)
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func parseAPIError(resp *http.Response, body []byte) error {
	apiErr := &APIError{
		StatusCode:      resp.StatusCode,
		RequestID:       resp.Header.Get("X-Request-Id"),
		RawBody:         string(body),
		WWWAuthenticate: resp.Header.Get("WWW-Authenticate"),
	}

	var oauthErr struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &oauthErr); err == nil && oauthErr.Error != "" {
		apiErr.Code = oauthErr.Error
		apiErr.Reason = oauthErr.Error
		apiErr.Message = oauthErr.ErrorDescription
		return apiErr
	}

	var envelope struct {
		Error struct {
			RequestID string          `json:"requestId"`
			Code      string          `json:"code"`
			Message   string          `json:"message"`
			Data      json.RawMessage `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Code != "" {
		apiErr.Code = envelope.Error.Code
		apiErr.Reason = envelope.Error.Code
		apiErr.Message = envelope.Error.Message
		if envelope.Error.RequestID != "" {
			apiErr.RequestID = envelope.Error.RequestID
		}
		return apiErr
	}

	apiErr.Code = "http-error"
	apiErr.Reason = "http-error"
	apiErr.Message = http.StatusText(resp.StatusCode)
	return apiErr
}
