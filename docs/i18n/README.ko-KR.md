# toss-openapi-cli

[English](../../README.md) | 한국어 | [简体中文](README.zh-CN.md) | [日本語](README.ja-JP.md)

## 설치

최신 릴리즈를 설치합니다.

```sh
npm install -g toss-openapi-cli
```

설치 후 binary를 확인합니다.

```sh
tosscli version
tosscli -v
tosscli doctor
```

`toss-openapi-cli`는 공개 Toss Open API만 사용하는 비공식 CLI 및 에이전트
툴킷입니다. 첫 대상은 `invest` 네임스페이스로 노출되는 토스증권 Open API입니다.

기본 명령어는 `tosscli`입니다. 출력이 구조화되어 있고 예측 가능하도록 설계해
자동화 에이전트, 스크립트, 터미널 사용자가 안정적으로 사용할 수 있습니다.

> [!IMPORTANT]
> 이 프로젝트는 Toss 또는 토스증권의 공식 제품이 아닙니다. 공개 문서화된
> Open API 표면을 기반으로 만들었으며, Toss 또는 토스증권이 보증, 지원,
> 유지보수한다는 의미로 해석하면 안 됩니다.

> [!NOTE]
> 이 프로젝트는 브라우저 세션을 재사용하거나, 토스증권 웹 로그인을 자동화하거나,
> 문서화되지 않은 웹 내부 API를 호출하지 않습니다. 브라우저 또는 문서화되지 않은
> 웹앱 동작 대신 문서화된 API 계약을 사용합니다.

## 상태

이 프로젝트는 초기 MVP 개발 단계입니다. 현재 구현은 `invest`
네임스페이스에 집중하며, `specs/invest/openapi.json`에 포함된 OpenAPI
스펙의 모든 operation을 CLI 명령으로 매핑합니다.

CLI는 자동화에서 예측 가능하고 터미널에서도 편하게 사용할 수 있도록 설계합니다.

- stdout JSON 우선 출력
- 안정적인 machine-readable 에러 응답
- 안정적인 exit code
- 가능한 범위에서 공식 API 개념을 따른 명령 구조
- `--dry-run`을 지원하는 주문 가능 워크플로
- 브라우저 세션 또는 문서화되지 않은 웹 API 클라이언트로 fallback하지 않음

예정된 public preview, release, agent usage 작업은 [Roadmap](../ROADMAP.md)을
참고하세요.

## 왜 사용하나요

`tosscli`는 문서화된 Toss Invest API 호출을 위한 로컬 CLI 경계를 제공합니다.
credential, token cache, account header, request construction, output formatting을
하나의 예측 가능한 도구 안에서 다룹니다.

이 프로젝트가 적합한 경우:

- 공개 OpenAPI만 사용하는 Toss Invest 통합 경로가 필요할 때
- 에이전트가 읽기 쉬운 JSON 출력과 구조화된 JSON 에러가 필요할 때
- credential, token cache, account header, order request construction을 CLI
  경계에서 다루고 싶을 때
- 실제 주문 요청을 보내기 전에 dry-run 주문 preview가 필요할 때

이 프로젝트는 브라우저 세션 재사용이나 문서화되지 않은 내부 API가 필요한 토스증권
웹앱 기능을 다루지 않습니다.

## 소스에서 빌드

```sh
go build -o bin/tosscli ./cmd/tosscli
./bin/tosscli version
```

개발 중에는 직접 실행할 수도 있습니다.

```sh
go run ./cmd/tosscli version
```

현재 이 저장소는 Go `1.26.4`를 사용합니다.

## 대체 설치

macOS와 Linux에서는 GitHub Releases 기반 standalone installer도 사용할 수 있습니다.

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | sh
```

특정 버전을 설치하려면 다음처럼 실행합니다.

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | TOSSCLI_VERSION=v0.1.1 sh
```

standalone installer는 기본적으로 `~/.local/bin`에 설치합니다. 이 경로가 `PATH`에
없으면 출력된 PATH 안내를 적용한 뒤 `tosscli`를 실행하세요.

## 인증

invest API는 OAuth2 client credentials 방식을 사용합니다. login을 실행한 뒤
credential을 대화형으로 입력할 수 있습니다.

```sh
tosscli invest auth login
```

OpenAPI 스펙의 credential 이름을 shell 환경변수로 제공할 수도 있습니다.

```sh
export TOSS_INVEST_CLIENT_ID="..."
export TOSS_INVEST_CLIENT_SECRET="..."
```

그러면 prompt 없이 토큰을 발급하고 캐시합니다.

```sh
tosscli invest auth login
tosscli invest auth status
```

가능한 경우 credential과 캐시된 token은 OS keyring에 저장됩니다. `tosscli`는
process 환경변수를 읽으며, `.env` 파일을 자동으로 로드하지 않습니다.

credential을 직접 넘길 수도 있습니다.

```sh
tosscli invest auth login \
  --client-id "$TOSS_INVEST_CLIENT_ID" \
  --client-secret "$TOSS_INVEST_CLIENT_SECRET"
```

비대화형 환경에서는 access token을 직접 공급할 수 있습니다.

```sh
export TOSS_INVEST_ACCESS_TOKEN="..."
```

## 빠른 시작

로컬 준비 상태를 확인합니다.

```sh
tosscli doctor
```

`doctor`는 CLI version, credential 존재 여부, token 사용 가능 여부,
read-only account list 접근만 확인합니다. 주문 실행은 검사하지 않습니다.

인증 상태를 확인하고 계좌를 조회합니다.

```sh
tosscli invest auth status
tosscli invest account list
```

시장 데이터를 조회합니다.

```sh
tosscli invest market-data prices --symbols AAPL
tosscli invest market-data orderbook --symbol AAPL
tosscli invest stock-info stocks --symbols AAPL
```

계좌 상태를 조회합니다.

```sh
ACCOUNT_SEQ="123456789"

tosscli invest asset holdings --account-seq "$ACCOUNT_SEQ"
tosscli invest order-info buying-power --account-seq "$ACCOUNT_SEQ" --currency USD
tosscli invest order-history list --account-seq "$ACCOUNT_SEQ"
```

## 주문

주문 명령은 실제 주문 생성, 정정, 취소를 수행할 수 있습니다. 먼저
`--dry-run`으로 요청 내용을 확인하면 Toss Invest로 요청을 보내지 않습니다.

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

실제로 주문을 보낼 의도가 있을 때만 `--dry-run`을 제거합니다.

```sh
tosscli invest order create \
  --account-seq "$ACCOUNT_SEQ" \
  --symbol AAPL \
  --side BUY \
  --order-type LIMIT \
  --quantity 1 \
  --price 100
```

주문 정정 또는 취소:

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

## 출력 계약

성공한 명령은 stdout에 formatted JSON을 출력합니다. 가능한 경우 모든 응답을
커스텀 envelope으로 감싸지 않고 Toss Invest API 응답 형태를 유지합니다.

에러도 JSON입니다.

```json
{
  "error": {
    "code": "AUTH_CONFIG_ERROR",
    "message": "Toss Invest credentials are not configured",
    "reason": "AUTH_CONFIG_ERROR"
  }
}
```

Exit code:

| Code | 의미 |
| ---: | --- |
| `0` | 성공 |
| `1` | 예상하지 못한 CLI 에러 |
| `2` | 사용법 에러 |
| `3` | 인증 또는 설정 에러 |
| `4` | Toss Invest API 에러 |
| `5` | 스펙 또는 생성 표면 에러 |

## 명령어

진단:

```sh
tosscli doctor
```

인증:

```sh
tosscli invest auth login
tosscli invest auth status
tosscli invest auth token
tosscli invest auth logout
```

계좌와 자산:

```sh
tosscli invest account list
tosscli invest asset holdings
```

시장 데이터:

```sh
tosscli invest market-data prices
tosscli invest market-data orderbook
tosscli invest market-data trades
tosscli invest market-data price-limits
tosscli invest market-data candles
```

시장 및 종목 정보:

```sh
tosscli invest stock-info stocks
tosscli invest stock-info warnings <symbol>
tosscli invest market-info exchange-rate
tosscli invest market-info calendar kr
tosscli invest market-info calendar us
```

주문 정보와 주문 내역:

```sh
tosscli invest order-info buying-power
tosscli invest order-info sellable-quantity
tosscli invest order-info commissions
tosscli invest order-history list
tosscli invest order-history get <orderId>
```

주문 변경:

```sh
tosscli invest order create
tosscli invest order modify <orderId>
tosscli invest order cancel <orderId>
```

필수 flag는 각 명령의 `--help`에서 확인합니다.

```sh
tosscli invest order create --help
```

## 개발

테스트 실행:

```sh
go test ./...
```

CLI 빌드:

```sh
go build -o bin/tosscli ./cmd/tosscli
```

릴리즈 설정 확인:

```sh
goreleaser check
```

로컬 snapshot artifact 생성:

```sh
goreleaser release --snapshot --clean
```

## 릴리즈

공개 릴리즈는 GitHub Releases와 GoReleaser를 기준으로 배포합니다.

- macOS, Linux, Windows용 cross-platform archive
- SHA256 checksum
- macOS/Linux용 `install.sh`
- Windows는 zip을 먼저 제공하고, `install.ps1`은 Windows 수동 검증 후 제공

Homebrew, Scoop, Winget, npm wrapper, signing, notarization은 GitHub Releases
경로가 안정화된 이후로 미룹니다.

## 면책

이 프로젝트는 비공식이며 있는 그대로 제공됩니다. Trading API는 실제 자금을
움직일 수 있고 금전적 손실을 만들 수 있습니다. mutation 요청을 보내기 전에
명령, credential, 계좌 번호, symbol, 수량, 가격, 주문 상태를 검토하세요.
