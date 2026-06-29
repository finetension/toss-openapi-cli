#!/usr/bin/env sh
set -eu

INPUT="${1:-}"
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

cd "$ROOT_DIR"

usage() {
  echo "usage: scripts/version.sh <patch|minor|major|version>" >&2
  echo "examples:" >&2
  echo "  scripts/version.sh patch" >&2
  echo "  scripts/version.sh 0.1.10" >&2
}

case "$INPUT" in
  patch | minor | major)
    bump="$INPUT"
    version=""
    ;;
  v[0-9]*.[0-9]*.[0-9]* | [0-9]*.[0-9]*.[0-9]*)
    bump=""
    version="${INPUT#v}"
    ;;
  *)
    usage
    exit 2
    ;;
esac

if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree is not clean. Commit or stash changes before release." >&2
  exit 1
fi

git fetch origin main >/dev/null

latest_tag="$(git ls-remote --tags --refs origin 'v[0-9]*.[0-9]*.[0-9]*' \
  | sed 's|.*refs/tags/||' \
  | sort -V \
  | tail -n 1)"

if [ -n "$bump" ]; then
  if [ -z "$latest_tag" ]; then
    echo "No release tag found. Expected a tag like v0.1.0." >&2
    exit 1
  fi

  latest="${latest_tag#v}"
  IFS=. read -r major minor patch <<EOF
$latest
EOF

  case "$bump" in
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
fi

case "$version" in
  *[!0-9.]* | .* | *..* | *.)
    echo "Invalid version: $INPUT" >&2
    exit 2
    ;;
esac

if ! printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Version must be semver without prerelease metadata: $INPUT" >&2
  exit 2
fi

tag="v$version"

head_sha="$(git rev-parse HEAD)"
origin_sha="$(git rev-parse origin/main)"
if [ "$head_sha" != "$origin_sha" ]; then
  echo "HEAD does not match origin/main. Push or pull before release." >&2
  echo "HEAD:        $head_sha" >&2
  echo "origin/main: $origin_sha" >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  echo "Local tag already exists: $tag" >&2
  exit 1
fi

if git ls-remote --exit-code --tags origin "$tag" >/dev/null 2>&1; then
  echo "Remote tag already exists: $tag" >&2
  exit 1
fi

if [ -n "$latest_tag" ]; then
  echo "Latest tag: $latest_tag"
else
  echo "Latest tag: none"
fi
echo "Next tag:   $tag"

scripts/pre-release-check.sh "$version"
goreleaser check

if [ "${DRY_RUN:-0}" = "1" ]; then
  echo "Dry run enabled. Tag was not created."
  exit 0
fi

git tag -a "$tag" -m "Release $tag"

echo "Created tag $tag."
echo "Push it to start the release workflow:"
echo "  git push origin $tag"
