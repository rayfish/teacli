#!/usr/bin/env sh
# teacli installer. Downloads the right release binary for this machine and
# installs it. All settings are overridable via environment variables:
#
#   TEACLI_RELEASE_HOST  where releases live   (default: https://github.com)
#   TEACLI_RELEASE_REPO  owner/repo            (default: rayfish/teacli)
#   TEACLI_VERSION       tag or "latest"       (default: latest)
#   TEACLI_INSTALL_DIR   install directory     (default: /usr/local/bin)
#
# The release host is not the Gitea or Forgejo server teacli talks to; it is
# only where the binaries are downloaded from. Point it at your own Gitea or
# Forgejo instance to install from a mirror.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rayfish/teacli/main/install.sh | sh
#   TEACLI_VERSION=v1.0.0 ./install.sh
set -eu

RELEASE_HOST="${TEACLI_RELEASE_HOST:-https://github.com}"
RELEASE_REPO="${TEACLI_RELEASE_REPO:-rayfish/teacli}"
TEACLI_VERSION="${TEACLI_VERSION:-latest}"
INSTALL_DIR="${TEACLI_INSTALL_DIR:-/usr/local/bin}"
BIN_NAME="teacli"

err() { echo "install: $*" >&2; exit 1; }

os=$(uname -s)
case "$os" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) err "unsupported OS '$os'. This installer supports Linux and macOS. On Windows, download teacli-windows-amd64.exe from the releases page." ;;
esac

arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) err "unsupported architecture '$arch' (need amd64 or arm64)" ;;
esac

asset="teacli-${os}-${arch}"

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1"; }
	fetch_to() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO- "$1"; }
	fetch_to() { wget -qO "$2" "$1"; }
else
	err "need curl or wget on PATH"
fi

# Resolve "latest" to a concrete tag (no jq required). GitHub serves release
# metadata from api.github.com; Gitea and Forgejo serve it from /api/v1 on the
# same host. Asset URLs have the same shape on all three.
if [ "$TEACLI_VERSION" = "latest" ]; then
	case "$RELEASE_HOST" in
		https://github.com | https://github.com/)
			api="https://api.github.com/repos/${RELEASE_REPO}/releases/latest" ;;
		*)
			api="${RELEASE_HOST%/}/api/v1/repos/${RELEASE_REPO}/releases/latest" ;;
	esac
	tag=$(fetch "$api" | tr ',' '\n' | grep '"tag_name"' | head -n1 | sed 's/.*"tag_name" *: *"\([^"]*\)".*/\1/')
	[ -n "$tag" ] || err "could not resolve the latest release tag from $api. Set TEACLI_VERSION to a tag, e.g. TEACLI_VERSION=v1.0.0."
else
	tag="$TEACLI_VERSION"
fi

base="${RELEASE_HOST%/}/${RELEASE_REPO}/releases/download/${tag}"
url="${base}/${asset}"

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

echo "Downloading ${asset} (${tag})"
fetch_to "$url" "$tmp" || err "download failed: $url"

# Best-effort checksum verification against the release's checksums.txt.
sumtool=""
if command -v sha256sum >/dev/null 2>&1; then
	sumtool="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	sumtool="shasum -a 256"
fi
if [ -n "$sumtool" ]; then
	sums=$(fetch "${base}/checksums.txt" 2>/dev/null || true)
	if [ -n "$sums" ]; then
		want=$(echo "$sums" | awk -v a="$asset" '$2 == a {print $1}' | head -n1)
		if [ -n "$want" ]; then
			got=$($sumtool "$tmp" | awk '{print $1}')
			[ "$want" = "$got" ] || err "checksum mismatch for ${asset} (want ${want}, got ${got})"
			echo "Checksum verified"
		fi
	fi
fi

chmod +x "$tmp"

target="${INSTALL_DIR}/${BIN_NAME}"
if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
	mv "$tmp" "$target"
elif command -v sudo >/dev/null 2>&1; then
	echo "Installing to ${target} (requires sudo)"
	sudo mkdir -p "$INSTALL_DIR"
	sudo mv "$tmp" "$target"
else
	err "cannot write to ${INSTALL_DIR} and sudo is unavailable. Set TEACLI_INSTALL_DIR to a writable directory."
fi
trap - EXIT

echo "Installed ${BIN_NAME} to ${target}"
"$target" --version 2>/dev/null || true
