package invest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
