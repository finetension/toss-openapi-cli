#!/usr/bin/env sh
set -eu

VERSION="${1:?usage: scripts/publish-npm-packages.sh <version>}"
OUT_DIR="${NPM_OUT_DIR:-dist/npm}"

NPM_DEP_MODE=version NPM_OUT_DIR="$OUT_DIR" scripts/build-npm-packages.sh "$VERSION"

npm publish "$OUT_DIR/tosscli-darwin-arm64" --access public
npm publish "$OUT_DIR/tosscli-darwin-x64" --access public
npm publish "$OUT_DIR/tosscli-linux-arm64" --access public
npm publish "$OUT_DIR/tosscli-linux-x64" --access public
npm publish "$OUT_DIR/tosscli-win32-x64" --access public
npm publish "$OUT_DIR/toss-openapi-cli" --access public
