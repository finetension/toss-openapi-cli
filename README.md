# toss-openapi-cli

English | [한국어](docs/i18n/README.ko-KR.md) | [简体中文](docs/i18n/README.zh-CN.md) | [日本語](docs/i18n/README.ja-JP.md)

## Install

Install the CLI:

```sh
npm install -g toss-openapi-cli
```

Install the companion agent skill:

```sh
npx skills add finetension/tosscli-skills
```

Then verify the installed CLI:

```sh
tosscli version
tosscli -v
tosscli doctor
```

`toss-openapi-cli` is an unofficial, public-OpenAPI-only CLI and agent toolkit
for Toss APIs, starting with the Toss Securities Open API exposed through the
`invest` namespace.

The primary command is `tosscli`. It keeps output structured and predictable so
it can be used reliably by automation agents, scripts, and terminal users.

> [!IMPORTANT]
> This is not an official Toss or Toss Securities product. It is built against
> public, documented Open API surfaces and should not be read as endorsed,
> supported, or maintained by Toss or Toss Securities.

> [!NOTE]
> This project does not reuse browser sessions, automate Toss Securities web
> login, or call undocumented web-internal APIs. It uses documented API
> contracts instead of browser or undocumented web-app behavior.

## Status

This project is in early MVP development. The current implementation focuses on
the `invest` namespace and maps every operation in the bundled OpenAPI spec at
`specs/invest/openapi.json`.

The CLI is designed to stay predictable for automation and comfortable for
terminal use:

- JSON-first output on stdout.
- Stable, machine-readable error responses.
- Stable exit codes.
- Commands shaped from official API concepts where practical.
- Order-capable workflows with dry-run support.
- No fallback to browser-session or undocumented web API clients.

See [Roadmap](docs/ROADMAP.md) for planned public preview and agent usage work.

## Why Use This

`tosscli` provides a local CLI boundary for documented Toss Invest API calls. It
keeps credentials, token caching, account headers, request construction, and
output formatting in one predictable tool.

Use this project when you want:

- A public OpenAPI-only Toss Invest integration path.
- Agent-readable JSON output and structured JSON errors.
- A CLI boundary for credentials, token caching, account headers, and order
  request construction.
- Dry-run order previews before any live order request is sent.

This project does not cover Toss Securities web-app features that require
browser session reuse or undocumented internal APIs.

## Build From Source

```sh
go build -o bin/tosscli ./cmd/tosscli
./bin/tosscli version
```

During development, you can also run commands directly:

```sh
go run ./cmd/tosscli version
```

This repository currently uses Go `1.26.4`.

## Alternative Install

macOS and Linux can also install from GitHub Releases with the standalone
installer:

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | sh
```

To install a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | TOSSCLI_VERSION=v0.1.1 sh
```

The standalone installer defaults to `~/.local/bin`. If that directory is not
on `PATH`, follow the printed PATH guidance before running `tosscli`.

## Authentication

The invest API uses OAuth2 client credentials. Run login and enter the
credentials interactively:

```sh
tosscli invest auth login
```

You can also provide the same credential names used by the OpenAPI spec as
shell environment variables:

```sh
export TOSS_INVEST_CLIENT_ID="..."
export TOSS_INVEST_CLIENT_SECRET="..."
```

Then issue and cache a token without being prompted:

```sh
tosscli invest auth login
tosscli invest auth status
```

Credentials and cached tokens are stored in the OS keyring when possible.
`tosscli` reads process environment variables; it does not automatically load
`.env` files.

You can also pass credentials directly:

```sh
tosscli invest auth login \
  --client-id "$TOSS_INVEST_CLIENT_ID" \
  --client-secret "$TOSS_INVEST_CLIENT_SECRET"
```

For non-interactive environments, an access token can be supplied directly:

```sh
export TOSS_INVEST_ACCESS_TOKEN="..."
```

If login or API calls fail with `access_denied` and `IP address not allowed`,
check the allowed IP settings for your Toss Open API credentials. The CLI cannot
override server-side IP restrictions.

To check the public IP address visible to external services:

```sh
tosscli doctor --show-ip
```

To add an allowed IP:

1. Open <https://www.tossinvest.com>.
2. Log in.
3. Click the settings button in the bottom-right corner.
4. Open the Open API tab.
5. Click the Add IP button.
6. Enter the IP address and click Add.

## Quick Start

Check local readiness:

```sh
tosscli doctor
```

`doctor` checks only four things: CLI version, credential availability, token
availability, and read-only account list access. It does not test order
execution.

Check auth and list accounts:

```sh
tosscli invest auth status
tosscli invest account list
```

Read market data:

```sh
tosscli invest market-data prices --symbols AAPL
tosscli invest market-data orderbook --symbol AAPL
tosscli invest stock-info stocks --symbols AAPL
```

Read account state:

```sh
ACCOUNT_SEQ="123456789"

tosscli invest asset holdings --account-seq "$ACCOUNT_SEQ"
tosscli invest order-info buying-power --account-seq "$ACCOUNT_SEQ" --currency USD
tosscli invest order-history list --account-seq "$ACCOUNT_SEQ"
```

## Orders

Order commands can place, modify, or cancel real orders. Start with `--dry-run`
to inspect the request without sending it to Toss Invest.

```sh
tosscli invest order create \
  --dry-run \
  --account-seq "$ACCOUNT_SEQ" \
  --symbol AAPL \
  --side BUY \
  --order-type LIMIT \
  --quantity 1 \
  --price 100
```

Remove `--dry-run` only when you intend to send the order:

```sh
tosscli invest order create \
  --account-seq "$ACCOUNT_SEQ" \
  --symbol AAPL \
  --side BUY \
  --order-type LIMIT \
  --quantity 1 \
  --price 100
```

Modify or cancel an order:

```sh
tosscli invest order modify "$ORDER_ID" \
  --dry-run \
  --account-seq "$ACCOUNT_SEQ" \
  --quantity 1 \
  --price 101

tosscli invest order cancel "$ORDER_ID" \
  --dry-run \
  --account-seq "$ACCOUNT_SEQ"
```

## Output Contract

Successful commands print formatted JSON to stdout. Where practical, the CLI
preserves the Toss Invest API response shape instead of wrapping every response
in a custom envelope.

Errors are also JSON:

```json
{
  "error": {
    "code": "AUTH_CONFIG_ERROR",
    "message": "Toss Invest credentials are not configured",
    "reason": "AUTH_CONFIG_ERROR"
  }
}
```

Exit codes:

| Code | Meaning |
| ---: | --- |
| `0` | Success |
| `1` | Unexpected CLI error |
| `2` | Usage error |
| `3` | Authentication or configuration error |
| `4` | Toss Invest API error |
| `5` | Spec or generated-surface error |

## Commands

Diagnostics:

```sh
tosscli doctor
```

Authentication:

```sh
tosscli invest auth login
tosscli invest auth status
tosscli invest auth token
tosscli invest auth logout
```

Accounts and assets:

```sh
tosscli invest account list
tosscli invest asset holdings
```

Market data:

```sh
tosscli invest market-data prices
tosscli invest market-data orderbook
tosscli invest market-data trades
tosscli invest market-data price-limits
tosscli invest market-data candles
```

Market and stock information:

```sh
tosscli invest stock-info stocks
tosscli invest stock-info warnings <symbol>
tosscli invest market-info exchange-rate
tosscli invest market-info calendar kr
tosscli invest market-info calendar us
```

Order information and history:

```sh
tosscli invest order-info buying-power
tosscli invest order-info sellable-quantity
tosscli invest order-info commissions
tosscli invest order-history list
tosscli invest order-history get <orderId>
```

Order mutations:

```sh
tosscli invest order create
tosscli invest order modify <orderId>
tosscli invest order cancel <orderId>
```

Use `--help` on any command to see required flags:

```sh
tosscli invest order create --help
```

## Development

Run the test suite:

```sh
go test ./...
```

Build the CLI:

```sh
go build -o bin/tosscli ./cmd/tosscli
```

Check the release configuration:

```sh
goreleaser check
```

Create local snapshot artifacts:

```sh
goreleaser release --snapshot --clean
```

## Release

Public releases are published through GitHub Releases and GoReleaser:

- Cross-platform archives for macOS, Linux, and Windows.
- SHA256 checksums.
- `install.sh` for macOS/Linux.
- Windows zip first, with `install.ps1` after manual Windows validation.

Homebrew, Scoop, Winget, npm wrappers, signing, and notarization are intentionally
deferred until the GitHub Releases path is stable.

## Disclaimer

This project is unofficial and provided as-is. Trading APIs can move real money
and may create financial loss. Review commands, credentials, account numbers,
symbols, quantities, prices, and order status before sending mutation requests.
