---
name: release
description: Use when preparing a toss-openapi-cli release, updating the changelog, creating release tags, or verifying release readiness.
---

# Release Skill

This project uses one root `CHANGELOG.md`. Do not create per-release fragment
files.

## Changelog

Before tagging a release, add a section:

```md
## [0.1.7] - YYYY-MM-DD
```

Use short, user-facing entries first. Put internal packaging, CI, or refactor
notes after user-visible changes.

Common headings:

- `Added`
- `Changed`
- `Fixed`
- `Removed`
- `Internal`

Keep `## [Unreleased]` at the top for future notes.

## Release Commands

Run one of these from a clean `main` branch:

```sh
pnpm release:patch
pnpm release:minor
pnpm release:major
```

The command calculates the next version from the latest `v*` tag, validates the
matching `CHANGELOG.md` section, runs tests, verifies npm package contents, and
creates an annotated local tag.

Push the printed tag command to start the GitHub Actions release workflow.

## Manual Check

To validate a specific version without tagging:

```sh
pnpm release:check 0.1.7
```
