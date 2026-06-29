#!/usr/bin/env sh
set -eu

VERSION_INPUT="${1:-}"

if [ -z "$VERSION_INPUT" ]; then
  echo "usage: scripts/pre-release-check.sh <version-or-tag>" >&2
  echo "example: scripts/pre-release-check.sh 0.1.7" >&2
  exit 2
fi

VERSION="${VERSION_INPUT#v}"
TAG="v$VERSION"
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$ROOT_DIR"

case "$VERSION" in
  *[!0-9.]* | .* | *..* | *.)
    echo "Invalid version: $VERSION_INPUT" >&2
    exit 2
    ;;
esac

if ! printf '%s\n' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Version must be semver without prerelease metadata: $VERSION_INPUT" >&2
  exit 2
fi

if [ "${REQUIRE_CLEAN_WORKTREE:-1}" = "1" ]; then
  if [ -n "$(git status --porcelain)" ]; then
    echo "Working tree is not clean. Commit or stash changes before release." >&2
    exit 1
  fi
fi

if [ "${ALLOW_EXISTING_TAG:-0}" != "1" ] && git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
  echo "Tag already exists: $TAG" >&2
  exit 1
fi

echo "Running Go tests..."
go test ./...

echo "Building local npm packages..."
NPM_DEP_MODE=file scripts/build-npm-packages.sh "$VERSION"

echo "Checking npm package contents..."
pack_json="$(mktemp)"
trap 'rm -f "$pack_json"; if [ -n "${install_dir:-}" ]; then rm -rf "$install_dir"; fi' EXIT INT TERM

(cd dist/npm/toss-openapi-cli && npm pack --dry-run --json > "$pack_json")
node - "$pack_json" <<'NODE'
const fs = require("node:fs");

const pack = JSON.parse(fs.readFileSync(process.argv[2], "utf8"))[0];
const files = new Set(pack.files.map((file) => file.path));
const required = ["README.md", "bin/tosscli.js", "package.json"];
const missing = required.filter((file) => !files.has(file));

if (missing.length > 0) {
  console.error(`npm package is missing required files: ${missing.join(", ")}`);
  process.exit(1);
}
NODE

platform="$(node -p 'process.platform + "-" + process.arch')"
case "$platform" in
  darwin-arm64) platform_pkg="tosscli-darwin-arm64" ;;
  darwin-x64) platform_pkg="tosscli-darwin-x64" ;;
  linux-arm64) platform_pkg="tosscli-linux-arm64" ;;
  linux-x64) platform_pkg="tosscli-linux-x64" ;;
  win32-x64) platform_pkg="tosscli-win32-x64" ;;
  *)
    echo "Skipping local install check for unsupported platform: $platform"
    platform_pkg=""
    ;;
esac

if [ -n "$platform_pkg" ]; then
  echo "Checking local npm install for $platform_pkg..."
  install_dir="$(mktemp -d)"
  npm install -g --prefix "$install_dir" "./dist/npm/$platform_pkg" "./dist/npm/toss-openapi-cli" >/dev/null
  "$install_dir/bin/tosscli" version >/dev/null
fi

echo "Pre-release check passed for $TAG."
