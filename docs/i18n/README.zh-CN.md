# toss-openapi-cli

[English](../../README.md) | [한국어](README.ko-KR.md) | 简体中文 | [日本語](README.ja-JP.md)

`toss-openapi-cli` 是一个只使用公开 Toss Open API 的非官方 CLI 和 Agent
工具包。当前首先支持通过 `invest` 命名空间暴露的 Toss Securities Open API。

主命令是 `tosscli`。它保持输出结构化且可预测，因此自动化 Agent、脚本和终端
用户都可以可靠地使用它。

> [!IMPORTANT]
> 本项目不是 Toss 或 Toss Securities 的官方产品。它只基于公开、已文档化的
> Open API 表面构建，不应被理解为由 Toss 或 Toss Securities 背书、支持或维护。

> [!NOTE]
> 本项目不会复用浏览器会话、自动化 Toss Securities 网页登录，也不会调用未文档化的
> Web 内部 API。它使用文档化的 API 契约，而不是浏览器或未文档化的 Web app 行为。

## 状态

本项目处于早期 MVP 开发阶段。当前实现聚焦 `invest` 命名空间，并将
`specs/invest/openapi.json` 中的 OpenAPI spec 的所有 operation 映射为
CLI 命令。

CLI 的设计目标是在自动化场景中保持可预测，同时也适合终端使用：

- stdout 优先输出 JSON
- 稳定、机器可读的错误响应
- 稳定的 exit code
- 尽量沿用官方 API 概念设计命令结构
- 支持 `--dry-run` 的下单工作流
- 不 fallback 到浏览器会话或未文档化的 Web API 客户端

计划中的 public preview、release 和 agent usage 工作请参见
[Roadmap](../ROADMAP.md)。

## 为什么使用

`tosscli` 为文档化的 Toss Invest API 调用提供本地 CLI 边界。它把
credential、token cache、account header、request construction 和 output
formatting 放在一个可预测的工具中处理。

适合使用本项目的情况：

- 需要只使用公开 OpenAPI 的 Toss Invest 集成路径
- 需要 Agent 易于读取的 JSON 输出和结构化 JSON 错误
- 希望在 CLI 边界处理 credential、token cache、account header 和订单请求构造
- 在发送真实订单请求前需要 dry-run 订单预览

本项目不覆盖需要浏览器会话复用或未文档化内部 API 的 Toss Securities Web app
功能。

## 从源码安装

公开 release artifacts 尚未发布。首次 release 之后，可以使用以下命令安装：

```sh
curl -fsSL https://raw.githubusercontent.com/finetension/toss-openapi-cli/main/install.sh | sh
```

在此之前，请在本地构建：

```sh
go build -o bin/tosscli ./cmd/tosscli
./bin/tosscli version
```

开发时也可以直接运行：

```sh
go run ./cmd/tosscli version
```

当前仓库使用 Go `1.26.4`。

## 认证

invest API 使用 OAuth2 client credentials。`tosscli` 支持 OpenAPI spec 中
使用的 credential 名称：

```sh
export TOSS_INVEST_CLIENT_ID="..."
export TOSS_INVEST_CLIENT_SECRET="..."
```

然后签发并缓存 token：

```sh
tosscli invest auth login
tosscli invest auth status
```

在可能的情况下，credential 和缓存 token 会保存到操作系统 keyring。

也可以直接传入 credential：

```sh
tosscli invest auth login \
  --client-id "$TOSS_INVEST_CLIENT_ID" \
  --client-secret "$TOSS_INVEST_CLIENT_SECRET"
```

在非交互环境中，可以直接提供 access token：

```sh
export TOSS_INVEST_ACCESS_TOKEN="..."
```

## 快速开始

检查本地准备状态：

```sh
tosscli doctor
```

`doctor` 只检查四项：CLI version、credential 是否可用、token 是否可用、
以及 read-only account list 访问。它不会测试订单执行。

检查认证并列出账户：

```sh
tosscli invest auth status
tosscli invest account list
```

读取市场数据：

```sh
tosscli invest market-data prices --symbols AAPL
tosscli invest market-data orderbook --symbol AAPL
tosscli invest stock-info stocks --symbols AAPL
```

读取账户状态：

```sh
ACCOUNT_SEQ="123456789"

tosscli invest asset holdings --account-seq "$ACCOUNT_SEQ"
tosscli invest order-info buying-power --account-seq "$ACCOUNT_SEQ" --currency USD
tosscli invest order-history list --account-seq "$ACCOUNT_SEQ"
```

## 订单

订单命令可以创建、修改或取消真实订单。请先使用 `--dry-run` 检查请求内容，
此时不会向 Toss Invest 发送请求。

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

只有在明确想发送订单时，才移除 `--dry-run`：

```sh
tosscli invest order create \
  --account-seq "$ACCOUNT_SEQ" \
  --symbol AAPL \
  --side BUY \
  --order-type LIMIT \
  --quantity 1 \
  --price 100
```

修改或取消订单：

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

## 输出契约

成功命令会向 stdout 输出格式化 JSON。在可行的情况下，CLI 会保留
Toss Invest API 的响应形状，而不是把所有响应包装进自定义 envelope。

错误也是 JSON：

```json
{
  "error": {
    "code": "AUTH_CONFIG_ERROR",
    "message": "Toss Invest credentials are not configured",
    "reason": "AUTH_CONFIG_ERROR"
  }
}
```

Exit code：

| Code | 含义 |
| ---: | --- |
| `0` | 成功 |
| `1` | 非预期 CLI 错误 |
| `2` | 用法错误 |
| `3` | 认证或配置错误 |
| `4` | Toss Invest API 错误 |
| `5` | Spec 或生成表面错误 |

## 命令

诊断：

```sh
tosscli doctor
```

认证：

```sh
tosscli invest auth login
tosscli invest auth status
tosscli invest auth token
tosscli invest auth logout
```

账户和资产：

```sh
tosscli invest account list
tosscli invest asset holdings
```

市场数据：

```sh
tosscli invest market-data prices
tosscli invest market-data orderbook
tosscli invest market-data trades
tosscli invest market-data price-limits
tosscli invest market-data candles
```

市场和股票信息：

```sh
tosscli invest stock-info stocks
tosscli invest stock-info warnings <symbol>
tosscli invest market-info exchange-rate
tosscli invest market-info calendar kr
tosscli invest market-info calendar us
```

订单信息和历史：

```sh
tosscli invest order-info buying-power
tosscli invest order-info sellable-quantity
tosscli invest order-info commissions
tosscli invest order-history list
tosscli invest order-history get <orderId>
```

订单变更：

```sh
tosscli invest order create
tosscli invest order modify <orderId>
tosscli invest order cancel <orderId>
```

使用任意命令的 `--help` 查看必需 flag：

```sh
tosscli invest order create --help
```

## 开发

运行测试：

```sh
go test ./...
```

构建 CLI：

```sh
go build -o bin/tosscli ./cmd/tosscli
```

检查发布配置：

```sh
goreleaser check
```

创建本地 snapshot artifacts：

```sh
goreleaser release --snapshot --clean
```

## 发布计划

首个公开发布路径计划基于 GitHub Releases 和 GoReleaser：

- 面向 macOS、Linux、Windows 的 cross-platform archives
- SHA256 checksums
- 面向 macOS/Linux 的 `install.sh`
- Windows 先提供 zip，`install.ps1` 在 Windows 手动验证后再提供

Homebrew、Scoop、Winget、npm wrapper、signing、notarization 会在 GitHub
Releases 路径稳定后再考虑。

## 免责声明

本项目是非官方项目，按现状提供。Trading API 可能移动真实资金并造成财务损失。
发送 mutation 请求前，请检查命令、credential、账户编号、symbol、数量、价格和订单状态。
