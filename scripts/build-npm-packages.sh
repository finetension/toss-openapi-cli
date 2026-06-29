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
cp -R "$ROOT_DIR/npm/." "$ROOT_DIR/$OUT_DIR/"

update_package_json() {
  package_dir="$1"

  node - "$package_dir/package.json" "$VERSION" "$DEP_MODE" <<'NODE'
const fs = require("node:fs");

const [path, version, depMode] = process.argv.slice(2);
const pkg = JSON.parse(fs.readFileSync(path, "utf8"));

pkg.version = version;

if (pkg.optionalDependencies) {
  for (const dep of Object.keys(pkg.optionalDependencies)) {
    const packageName = dep.replace("@finetension/", "");
    pkg.optionalDependencies[dep] = depMode === "file" ? `file:../${packageName}` : version;
  }
}

fs.writeFileSync(path, `${JSON.stringify(pkg, null, 2)}\n`);
NODE
}

write_platform_package() {
  package="$1"
  goos="$2"
  goarch="$3"
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

  update_package_json "$package_dir"
}

write_platform_package "tosscli-darwin-arm64" "darwin" "arm64"
write_platform_package "tosscli-darwin-x64" "darwin" "amd64"
write_platform_package "tosscli-linux-arm64" "linux" "arm64"
write_platform_package "tosscli-linux-x64" "linux" "amd64"
write_platform_package "tosscli-win32-x64" "windows" "amd64"

root_dir="$ROOT_DIR/$OUT_DIR/toss-openapi-cli"
cp "$ROOT_DIR/README.md" "$root_dir/README.md"
chmod +x "$root_dir/bin/tosscli.js"
update_package_json "$root_dir"
