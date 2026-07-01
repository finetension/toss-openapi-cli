# Roadmap

This roadmap describes planned product work for `toss-openapi-cli`.

The project is an unofficial, public-OpenAPI-only CLI and agent toolkit for
Toss APIs, starting with the Toss Securities Open API exposed through the
`invest` namespace.

## Current Status

The initial `invest` command surface maps every operation in the bundled
OpenAPI spec at `specs/invest/openapi.json`.

The CLI is available through:

- npm: `npm install -g toss-openapi-cli`
- GitHub Releases with checksummed artifacts
- macOS/Linux standalone installer: `install.sh`

A companion agent skill is maintained separately at:

```txt
https://github.com/finetension/tosscli-skills
```

The skill is intentionally thin. `tosscli` remains the source of truth for
command behavior, validation, authentication, JSON output, dry-run behavior,
live mutations, and structured errors.

## Milestone 0: Order-Capable MVP

Goal: provide a practical Toss Invest CLI that can read account/market state
and perform order workflows through the public OpenAPI.

Status: mostly complete.

- [x] Go CLI binary: `tosscli`
- [x] `invest` namespace
- [x] All bundled Toss Invest OpenAPI operations mapped to CLI commands
- [x] OAuth2 client credentials auth flow
- [x] Environment variable credential overrides
- [x] Keyring-backed credential/token storage
- [x] JSON-first command output
- [x] Structured JSON errors
- [x] Stable exit code categories
- [x] Account, asset, market data, market info, and stock info commands
- [x] Order create, modify, cancel, history, and order-info commands
- [x] Order `--dry-run` support
- [x] `tosscli doctor`
- [ ] Stable example outputs for common read and order dry-run commands
- [ ] Manual smoke test checklist for real local environments

## Milestone 1: Public Preview Distribution

Goal: make the CLI easy to install, verify, and release.

Status: in progress.

- [x] GitHub Release artifacts for macOS, Linux, and Windows
- [x] SHA256 checksums for release artifacts
- [x] npm primary install path
- [x] macOS/Linux `install.sh` alternative installer
- [x] Version metadata in `tosscli version`
- [x] npm-style local version script: `scripts/version.sh`
- [x] README install and verification flow
- [x] Agent install prompt flow in README
- [ ] Troubleshooting guide for auth, keyring, PATH, and token expiry issues
- [ ] Public preview release notes
- [ ] Windows runtime validation before claiming full Windows support

## Milestone 2: Agent Skill UX

Goal: let users hand a repository link to an agent and get a clear install and
usage flow.

Status: in progress.

- [x] Separate companion skill repository: `finetension/tosscli-skills`
- [x] Canonical skill folder: `skills/toss-invest/`
- [x] Thin skill that delegates execution to `tosscli`
- [x] Skill README install flow for agents
- [x] CLI README companion skill prompt flow for agents
- [x] Skill guidance to inspect `tosscli help --all --json` once per workflow
- [x] Per-command `--help` guidance before first command use
- [ ] Verify `npx skills add finetension/tosscli-skills -g -a universal` in a
      fresh agent session
- [ ] Document confirmed install behavior for Codex, Claude Code, Cursor, and
      other supported agents
- [ ] Decide whether the skill needs host-specific metadata beyond the current
      minimal structure

## Milestone 3: CLI Reliability

Goal: make command behavior easier to inspect, test, and automate.

Planned:

- [ ] More request construction tests for account, market, and order commands
- [ ] API error preservation tests
- [ ] Auth and token expiry diagnostics in `tosscli doctor`
- [ ] Stable dry-run response shape for order commands
- [ ] Clearer validation messages for missing flags and invalid order input
- [ ] Examples for script and CI usage

## Milestone 4: Help and OAS Maintenance

Goal: keep CLI help useful for humans and agents while staying aligned with the
OpenAPI source.

Status: in progress.

- [x] Command help enriched with OAS-backed operation details
- [x] `tosscli help --all`
- [x] `tosscli help --all --json`
- [x] Help registry that separates OAS-derived details from CLI-only details
- [ ] Review remaining command help against `specs/invest/openapi.json`
- [ ] Define a maintainable path for future OAS-to-help mapping
- [ ] Decide whether OAS mapping should remain manual, become generated, or
      use a hybrid model
- [ ] Add spec check/diff commands if they become useful for maintenance

## Milestone 5: Order Workflow Improvements

Goal: make live order workflows clearer before requests are sent.

Planned:

- [ ] Stronger order preview summaries
- [ ] Clearer handling for market orders, sell orders, and high-value orders
- [ ] Optional audit log for order mutation commands
- [ ] More examples for create, modify, cancel, and order-history workflows
- [ ] Revisit whether any interactive confirmation behavior belongs in the CLI

## Milestone 6: Optional Expansion

These items are useful but are not required for the first public preview.

- [ ] Additional output formats such as table, CSV, YAML, or NDJSON
- [ ] OpenAPI spec check and diff commands
- [ ] Generated client experiments
- [ ] Homebrew tap
- [ ] Windows `install.ps1`
- [ ] Support for additional public Toss API products when they have a clear
      CLI use case

## Non-Goals

These are not planned product goals:

- Browser login automation.
- Toss Securities web session reuse.
- Undocumented web API scraping.
- A general-purpose trading platform.
- A profit-seeking automated trading bot.

## Feedback Areas

Early feedback is most useful on:

- Whether install flows are easy for both humans and agents.
- Whether commands and flags are easy to understand.
- Whether JSON output is easy to consume from scripts and agents.
- Whether order dry-run output is clear before live execution.
- What `tosscli doctor` should report for real user environments.
- Whether the companion skill helps agents use `tosscli` without duplicating
  CLI behavior.
