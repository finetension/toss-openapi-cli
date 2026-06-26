#!/usr/bin/env sh
set -eu

REPO="${TOSSCLI_REPO:-finetension/toss-openapi-cli}"
BINARY="${TOSSCLI_BINARY:-tosscli}"
VERSION="${TOSSCLI_VERSION:-latest}"
INSTALL_DIR="${TOSSCLI_INSTALL_DIR:-}"

log() {
  printf '%s\n' "$*" >&2
}

fail() {
  log "error: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

detect_os() {
  os="$(uname -s)"
  case "$os" in
    Darwin) printf 'darwin' ;;
    Linux) printf 'linux' ;;
    *) fail "unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64 | amd64) printf 'amd64' ;;
    arm64 | aarch64) printf 'arm64' ;;
    *) fail "unsupported architecture: $arch" ;;
  esac
}

download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$output"
  else
    fail "required command not found: curl or wget"
  fi
}

sha256_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    fail "required command not found: sha256sum or shasum"
  fi
}

resolve_install_dir() {
  if [ -n "$INSTALL_DIR" ]; then
    printf '%s' "$INSTALL_DIR"
    return
  fi
  if [ -d "$HOME/.local/bin" ]; then
    printf '%s' "$HOME/.local/bin"
    return
  fi
  if [ -w "/usr/local/bin" ]; then
    printf '%s' "/usr/local/bin"
    return
  fi
  printf '%s' "$HOME/.local/bin"
}

need_cmd uname
need_cmd awk
need_cmd grep
need_cmd tar
need_cmd mktemp
need_cmd chmod
need_cmd mkdir
need_cmd cp

os="$(detect_os)"
arch="$(detect_arch)"
archive="${BINARY}_${os}_${arch}.tar.gz"

if [ "$VERSION" = "latest" ]; then
  base_url="https://github.com/${REPO}/releases/latest/download"
else
  case "$VERSION" in
    v*) tag="$VERSION" ;;
    *) tag="v$VERSION" ;;
  esac
  base_url="https://github.com/${REPO}/releases/download/${tag}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

archive_path="$tmp_dir/$archive"
checksums_path="$tmp_dir/checksums.txt"

log "Downloading $archive from $REPO..."
download "$base_url/$archive" "$archive_path"
download "$base_url/checksums.txt" "$checksums_path"

expected="$(awk -v archive="$archive" '$2 == archive {print $1}' "$checksums_path")"
[ -n "$expected" ] || fail "checksum entry not found for $archive"

actual="$(sha256_file "$archive_path")"
[ "$expected" = "$actual" ] || fail "checksum mismatch for $archive"

tar -xzf "$archive_path" -C "$tmp_dir"
[ -f "$tmp_dir/$BINARY" ] || fail "archive did not contain $BINARY"

install_dir="$(resolve_install_dir)"
mkdir -p "$install_dir"

target="$install_dir/$BINARY"
if [ -w "$install_dir" ]; then
  cp "$tmp_dir/$BINARY" "$target"
else
  need_cmd sudo
  log "Installing to $target with sudo..."
  sudo cp "$tmp_dir/$BINARY" "$target"
fi
chmod +x "$target"

log "Installed $BINARY to $target"

if ! command -v "$BINARY" >/dev/null 2>&1; then
  log "$BINARY is installed, but $install_dir is not on PATH."
  log "Add it to PATH or run: $target version"
fi
"$target" version
