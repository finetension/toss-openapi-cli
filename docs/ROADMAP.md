# Roadmap

This roadmap describes planned product work for `toss-openapi-cli`.

The project is an unofficial CLI for public Toss Open APIs, starting with the
Toss Securities Open API exposed through the `invest` namespace.

## Current Status

The initial `invest` command surface maps the bundled OpenAPI spec under
`specs/invest/openapi.json`.

The project is in early MVP development. The first public release is available
through GitHub Releases, with macOS/Linux installation through `install.sh`.

## Milestone 0: MVP Readiness

Goal: make the current `invest` command surface reliable enough for early
testing.

Status: in progress.

- [x] Go CLI binary: `tosscli`
- [x] `invest` namespace
- [x] OAuth2 client credentials auth flow
- [x] Environment variable credential overrides
- [x] Keyring-backed credential/token storage
- [x] JSON-first command output
- [x] Structured JSON errors
- [x] Stable exit code categories
- [x] Account and asset commands
- [x] Market data, market info, and stock info commands
- [x] Order create, modify, cancel, history, and order-info commands
- [x] Order `--dry-run` support
- [x] `tosscli doctor`
- [ ] Manual smoke test checklist
- [ ] Stable example outputs for common read and order dry-run commands

## Milestone 1: Public Preview

Goal: make the CLI easy to install, verify, and try in real local environments.

Status: in progress.

- [x] GitHub Release artifacts for macOS, Linux, and Windows
- [x] SHA256 checksums for release artifacts
- [x] `install.sh` for macOS/Linux
- [x] Version metadata in `tosscli version`
- [x] README install and verification flow
- [ ] Troubleshooting guide for auth, keyring, and token expiry issues
- [ ] Public preview release notes

## Milestone 2: CLI Reliability

Goal: make command behavior easier to inspect, test, and automate.

Planned:

- [ ] More request construction tests for account, market, and order commands
- [ ] API error preservation tests
- [ ] Auth and token expiry diagnostics in `tosscli doctor`
- [ ] Stable dry-run response shape for order commands
- [ ] Clearer validation messages for missing flags and invalid order input
- [ ] Examples for script and CI usage

## Milestone 3: Order Workflow Improvements

Goal: make live order workflows clearer before requests are sent.

Planned:

- [ ] Stronger order preview summaries
- [ ] Confirmation policy for selected high-risk order cases
- [ ] Clearer handling for market orders, sell orders, and high-value orders
- [ ] Optional audit log for order mutation commands
- [ ] More examples for create, modify, cancel, and order-history workflows

## Milestone 4: Agent Usage

Goal: make `tosscli` easier to use from automation agents while keeping the CLI
behavior stable for regular terminal users.

Planned:

- [ ] Agent usage examples in README
- [ ] Skill or instruction package for common agent workflows
- [ ] Install check, auth check, read-only query, and order dry-run workflows
- [ ] Example prompts that use JSON output and dry-run order previews

## Milestone 5: Optional Expansion

These items are useful but are not required for the first public preview.

- [ ] Additional output formats such as table, CSV, YAML, or NDJSON
- [ ] OpenAPI spec check and diff commands
- [ ] Generated client experiments
- [ ] Homebrew tap
- [ ] Windows `install.ps1`
- [ ] Support for additional public Toss API products when they have a clear CLI
      use case

## Non-Goals

These are not planned product goals:

- Browser login automation.
- Toss Securities web session reuse.
- Undocumented web API scraping.
- A general-purpose trading platform.
- A profit-seeking automated trading bot.

## Feedback Areas

Early feedback is most useful on:

- Whether commands and flags are easy to understand.
- Whether JSON output is easy to consume from scripts.
- Whether order dry-run output is clear before live execution.
- What `tosscli doctor` should report for real user environments.
- Whether the current install path is easy enough for the first public preview.
