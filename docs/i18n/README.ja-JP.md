# toss-openapi-cli

[English](../../README.md) | [한국어](README.ko-KR.md) | [简体中文](README.zh-CN.md) | 日本語

## インストール

最新 release をインストールします。

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | sh
```

インストール後に binary を確認します。

```sh
tosscli version
tosscli doctor
```

特定の version をインストールする場合:

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | TOSSCLI_VERSION=v0.1.1 sh
```

`toss-openapi-cli` は、公開されている Toss Open API だけを使う非公式の
CLI および Agent ツールキットです。最初の対象は、`invest` namespace として
公開されている Toss Securities Open API です。

主なコマンドは `tosscli` です。出力を構造化し、予測しやすく保つことで、
自動化 Agent、script、terminal user が安定して利用できるようにしています。

> [!IMPORTANT]
> このプロジェクトは Toss または Toss Securities の公式製品ではありません。
> 公開され、文書化された Open API の表面に基づいて構築されており、Toss
> または Toss Securities による承認、サポート、保守を意味するものではありません。

> [!NOTE]
> このプロジェクトはブラウザセッションを再利用せず、Toss Securities の Web
> ログインを自動化せず、文書化されていない Web 内部 API を呼び出しません。
> ブラウザや文書化されていない Web app の挙動ではなく、文書化された API 契約を
> 使用します。

## ステータス

このプロジェクトは初期 MVP 開発段階です。現在の実装は `invest`
namespace に集中しており、`specs/invest/openapi.json` に含まれる
OpenAPI spec のすべての operation を CLI コマンドに対応付けています。

CLI は自動化で予測しやすく、terminal でも使いやすいことを目指しています。

- stdout への JSON-first 出力
- 安定した machine-readable なエラー応答
- 安定した exit code
- 可能な範囲で公式 API の概念に沿ったコマンド構造
- `--dry-run` をサポートする注文可能な workflow
- ブラウザセッションや文書化されていない Web API client への fallback なし

予定している public preview、release、agent usage 作業については
[Roadmap](../ROADMAP.md) を参照してください。

## なぜ使うのか

`tosscli` は、文書化された Toss Invest API 呼び出しのためのローカル CLI
境界を提供します。credential、token cache、account header、request
construction、output formatting を 1 つの予測しやすいツールで扱います。

このプロジェクトが適しているケース:

- 公開 OpenAPI だけを使う Toss Invest 統合経路が必要な場合
- Agent が読みやすい JSON 出力と構造化 JSON エラーが必要な場合
- credential、token cache、account header、order request construction を
  CLI 境界で扱いたい場合
- 実際の注文リクエスト送信前に dry-run 注文 preview が必要な場合

このプロジェクトは、ブラウザセッション再利用や文書化されていない内部 API が
必要な Toss Securities Web app 機能は扱いません。

## ソースからのビルド

```sh
go build -o bin/tosscli ./cmd/tosscli
./bin/tosscli version
```

開発中は直接実行することもできます。

```sh
go run ./cmd/tosscli version
```

このリポジトリは現在 Go `1.26.4` を使用しています。

## 認証

invest API は OAuth2 client credentials を使用します。login を実行して
credential を対話的に入力できます。

```sh
tosscli invest auth login
```

OpenAPI spec で使われている credential 名を shell 環境変数として指定することも
できます。

```sh
export TOSS_INVEST_CLIENT_ID="..."
export TOSS_INVEST_CLIENT_SECRET="..."
```

その場合は prompt なしで token を発行して cache します。

```sh
tosscli invest auth login
tosscli invest auth status
```

可能な場合、credential と cached token は OS keyring に保存されます。`tosscli`
は process 環境変数を読み、`.env` file を自動では読み込みません。

credential を直接渡すこともできます。

```sh
tosscli invest auth login \
  --client-id "$TOSS_INVEST_CLIENT_ID" \
  --client-secret "$TOSS_INVEST_CLIENT_SECRET"
```

非対話環境では access token を直接指定できます。

```sh
export TOSS_INVEST_ACCESS_TOKEN="..."
```

## クイックスタート

ローカルの準備状態を確認します。

```sh
tosscli doctor
```

`doctor` が確認するのは CLI version、credential availability、token
availability、read-only account list access の 4 つだけです。注文実行は
テストしません。

認証を確認し、口座を一覧します。

```sh
tosscli invest auth status
tosscli invest account list
```

市場データを取得します。

```sh
tosscli invest market-data prices --symbols AAPL
tosscli invest market-data orderbook --symbol AAPL
tosscli invest stock-info stocks --symbols AAPL
```

口座状態を取得します。

```sh
ACCOUNT_SEQ="123456789"

tosscli invest asset holdings --account-seq "$ACCOUNT_SEQ"
tosscli invest order-info buying-power --account-seq "$ACCOUNT_SEQ" --currency USD
tosscli invest order-history list --account-seq "$ACCOUNT_SEQ"
```

## 注文

注文コマンドは実際の注文作成、変更、取消を実行できます。まず `--dry-run`
でリクエスト内容を確認してください。この場合、Toss Invest には送信されません。

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

実際に注文を送信する意図がある場合だけ `--dry-run` を外します。

```sh
tosscli invest order create \
  --account-seq "$ACCOUNT_SEQ" \
  --symbol AAPL \
  --side BUY \
  --order-type LIMIT \
  --quantity 1 \
  --price 100
```

注文の変更または取消:

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

## 出力契約

成功したコマンドは stdout に formatted JSON を出力します。可能な場合、
すべての応答を custom envelope で包まず、Toss Invest API の response
shape を維持します。

エラーも JSON です。

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

| Code | 意味 |
| ---: | --- |
| `0` | 成功 |
| `1` | 予期しない CLI エラー |
| `2` | 使用方法エラー |
| `3` | 認証または設定エラー |
| `4` | Toss Invest API エラー |
| `5` | Spec または生成 surface エラー |

## コマンド

診断:

```sh
tosscli doctor
```

認証:

```sh
tosscli invest auth login
tosscli invest auth status
tosscli invest auth token
tosscli invest auth logout
```

口座と資産:

```sh
tosscli invest account list
tosscli invest asset holdings
```

市場データ:

```sh
tosscli invest market-data prices
tosscli invest market-data orderbook
tosscli invest market-data trades
tosscli invest market-data price-limits
tosscli invest market-data candles
```

市場および銘柄情報:

```sh
tosscli invest stock-info stocks
tosscli invest stock-info warnings <symbol>
tosscli invest market-info exchange-rate
tosscli invest market-info calendar kr
tosscli invest market-info calendar us
```

注文情報と履歴:

```sh
tosscli invest order-info buying-power
tosscli invest order-info sellable-quantity
tosscli invest order-info commissions
tosscli invest order-history list
tosscli invest order-history get <orderId>
```

注文変更:

```sh
tosscli invest order create
tosscli invest order modify <orderId>
tosscli invest order cancel <orderId>
```

必須 flag は各コマンドの `--help` で確認できます。

```sh
tosscli invest order create --help
```

## 開発

テストを実行:

```sh
go test ./...
```

CLI をビルド:

```sh
go build -o bin/tosscli ./cmd/tosscli
```

リリース設定を確認:

```sh
goreleaser check
```

ローカル snapshot artifact を作成:

```sh
goreleaser release --snapshot --clean
```

## リリース

公開 release は GitHub Releases と GoReleaser で配布します。

- macOS、Linux、Windows 向け cross-platform archive
- SHA256 checksum
- macOS/Linux 向け `install.sh`
- Windows はまず zip を提供し、`install.ps1` は Windows で手動検証した後に提供

Homebrew、Scoop、Winget、npm wrapper、signing、notarization は GitHub
Releases 経路が安定した後に検討します。

## 免責事項

このプロジェクトは非公式であり、現状のまま提供されます。Trading API は実際の資金を
動かし、金銭的損失を発生させる可能性があります。mutation request を送信する前に、
コマンド、credential、口座番号、symbol、数量、価格、注文状態を確認してください。
