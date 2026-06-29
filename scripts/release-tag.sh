#!/usr/bin/env sh
set -eu

BUMP="${1:-}"
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$ROOT_DIR"

case "$BUMP" in
  patch | minor | major) ;;
  *)
    echo "usage: scripts/release-tag.sh <patch|minor|major>" >&2
    exit 2
    ;;
esac

if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree is not clean. Commit or stash changes before release." >&2
  exit 1
fi

latest_tag="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | sed -n '1p')"
if [ -z "$latest_tag" ]; then
  echo "No release tag found. Expected a tag like v0.1.0." >&2
  exit 1
fi

latest="${latest_tag#v}"
IFS=. read -r major minor patch <<EOF
$latest
EOF

case "$BUMP" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
esac

version="$major.$minor.$patch"
tag="v$version"

echo "Latest tag: $latest_tag"
echo "Next tag:   $tag"

scripts/pre-release-check.sh "$version"

if [ "${DRY_RUN:-0}" = "1" ]; then
  echo "Dry run enabled. Tag was not created."
  exit 0
fi

git tag -a "$tag" -m "Release $tag"

echo "Created tag $tag."
echo "Push it to start the release workflow:"
echo "  git push origin $tag"
