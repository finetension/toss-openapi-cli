#!/usr/bin/env sh
set -eu

VERSION="${1:-0.0.0-dev}"
DEP_MODE="${NPM_DEP_MODE:-version}"
OUT_DIR="${NPM_OUT_DIR:-dist/npm}"
COMMIT="$(git rev-parse HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

ldflags="-s -w"
ldflags="$ldflags -X github.com/finetension/toss-openapi-cli/internal/version.Version=$VERSION"
ldflags="$ldflags -X github.com/finetension/toss-openapi-cli/internal/version.Commit=$COMMIT"
ldflags="$ldflags -X github.com/finetension/toss-openapi-cli/internal/version.Date=$DATE"
ldflags="$ldflags -X github.com/finetension/toss-openapi-cli/internal/version.BuiltBy=npm"

rm -rf "$ROOT_DIR/$OUT_DIR"
mkdir -p "$ROOT_DIR/$OUT_DIR"

write_platform_package() {
  package="$1"
  goos="$2"
  goarch="$3"
  node_os="$4"
  node_cpu="$5"
  binary="tosscli"
  if [ "$goos" = "windows" ]; then
    binary="tosscli.exe"
  fi

  package_dir="$ROOT_DIR/$OUT_DIR/$package"
  mkdir -p "$package_dir/bin"

  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -ldflags "$ldflags" \
    -o "$package_dir/bin/$binary" \
    "$ROOT_DIR/cmd/tosscli"

  cat > "$package_dir/package.json" <<EOF
{
  "name": "@finetension/$package",
  "version": "$VERSION",
  "description": "Platform binary for toss-openapi-cli.",
  "license": "MIT",
  "os": ["$node_os"],
  "cpu": ["$node_cpu"],
  "files": ["bin"],
  "publishConfig": {
    "access": "public"
  }
}
EOF
}

write_platform_package "tosscli-darwin-arm64" "darwin" "arm64" "darwin" "arm64"
write_platform_package "tosscli-darwin-x64" "darwin" "amd64" "darwin" "x64"
write_platform_package "tosscli-linux-arm64" "linux" "arm64" "linux" "arm64"
write_platform_package "tosscli-linux-x64" "linux" "amd64" "linux" "x64"
write_platform_package "tosscli-win32-x64" "windows" "amd64" "win32" "x64"

root_dir="$ROOT_DIR/$OUT_DIR/toss-openapi-cli"
mkdir -p "$root_dir/bin"
cp "$ROOT_DIR/npm/toss-openapi-cli/tosscli.js" "$root_dir/bin/tosscli.js"
chmod +x "$root_dir/bin/tosscli.js"

dep_value() {
  package="$1"
  if [ "$DEP_MODE" = "file" ]; then
    printf '"file:../%s"' "$package"
  else
    printf '"%s"' "$VERSION"
  fi
}

cat > "$root_dir/package.json" <<EOF
{
  "name": "toss-openapi-cli",
  "version": "$VERSION",
  "description": "Unofficial CLI for public Toss Open APIs.",
  "license": "MIT",
  "bin": {
    "tosscli": "./bin/tosscli.js"
  },
  "files": ["bin"],
  "optionalDependencies": {
    "@finetension/tosscli-darwin-arm64": $(dep_value "tosscli-darwin-arm64"),
    "@finetension/tosscli-darwin-x64": $(dep_value "tosscli-darwin-x64"),
    "@finetension/tosscli-linux-arm64": $(dep_value "tosscli-linux-arm64"),
    "@finetension/tosscli-linux-x64": $(dep_value "tosscli-linux-x64"),
    "@finetension/tosscli-win32-x64": $(dep_value "tosscli-win32-x64")
  },
  "engines": {
    "node": ">=18"
  }
}
EOF
