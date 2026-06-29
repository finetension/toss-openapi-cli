#!/usr/bin/env sh
set -eu

VERSION_INPUT="${1:-}"

if [ -z "$VERSION_INPUT" ]; then
  echo "usage: scripts/release-notes.sh <version-or-tag>" >&2
  echo "example: scripts/release-notes.sh 0.1.7" >&2
  exit 2
fi

VERSION="${VERSION_INPUT#v}"
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$ROOT_DIR"

awk -v version="$VERSION" '
  $0 ~ "^## \\[" version "\\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$" {
    found = 1
    in_section = 1
    next
  }

  in_section && /^## \[/ {
    exit
  }

  in_section {
    print
  }

  END {
    if (!found) {
      exit 1
    }
  }
' CHANGELOG.md
