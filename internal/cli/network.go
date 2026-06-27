package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/finetension/toss-openapi-cli/internal/invest"
)

const ipAllowlistHint = "The current IP appears to be blocked by Toss Open API allowed IP settings. Run `tosscli doctor --show-ip`, then add that IP at tossinvest.com > Settings > Open API > Add IP."

type defaultPublicIPResolver struct{}

func (defaultPublicIPResolver) PublicIP(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://checkip.amazonaws.com", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "tosscli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("public IP lookup failed with HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("public IP lookup returned an invalid IP address")
	}
	return ip, nil
}

func isIPAllowlistError(err error) bool {
	var apiErr *invest.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.EqualFold(apiErr.Code, "access_denied") &&
		strings.Contains(strings.ToLower(apiErr.Message), "ip address not allowed")
}
