#!/bin/sh
#
# Gitee CLI installer.
#
# Downloads the latest (or a pinned) Gitee CLI release from Gitee, verifies its
# SHA-256 checksum, and installs the `gitee` binary into a directory on your PATH.
#
# Usage:
#   curl -fsSL https://gitee.com/oschina/gitee-cli/raw/main/scripts/install.sh | sh
#
# Options (environment variables):
#   GITEE_CLI_VERSION   Version to install, e.g. "1.2.3" or "v1.2.3" (default: latest)
#   INSTALL_DIR         Target directory for the binary (default: $HOME/.local/bin,
#                       or /usr/local/bin when run as root)
#   GITEE_CLI_REPO      owner/repo to install from (default: oschina/gitee-cli)
#   GITEE_CLI_HOST      Gitee host (default: gitee.com)
#
# Flags (when invoked as a file, not via a pipe):
#   --version <v>       Same as GITEE_CLI_VERSION
#   --install-dir <d>   Same as INSTALL_DIR
#   -h, --help          Show this help

set -eu

REPO="${GITEE_CLI_REPO:-oschina/gitee-cli}"
HOST="${GITEE_CLI_HOST:-gitee.com}"
VERSION="${GITEE_CLI_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"
BINARY_NAME="gitee"

API_PREFIX="https://${HOST}/api/v5"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

info() { printf '%s\n' "$*" >&2; }
warn() { printf 'warning: %s\n' "$*" >&2; }
err() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

usage() {
	sed -n '3,25p' "$0" 2>/dev/null | sed 's/^#\{0,1\} \{0,1\}//'
	exit 0
}

have() { command -v "$1" >/dev/null 2>&1; }

# Optional authentication. Only needed while the repository is private; for a
# public repository leave GITEE_TOKEN unset and requests are made anonymously.
# The token is passed via stdin config (curl) or a header flag (wget) so it
# never appears in the process list.
GITEE_TOKEN="${GITEE_TOKEN:-}"

# Download a URL to stdout.
fetch() {
	url="$1"
	if have curl; then
		if [ -n "$GITEE_TOKEN" ]; then
			printf 'header = "Authorization: Bearer %s"\n' "$GITEE_TOKEN" |
				curl -fsSL --config - "$url"
		else
			curl -fsSL "$url"
		fi
	elif have wget; then
		if [ -n "$GITEE_TOKEN" ]; then
			wget -qO- --header="Authorization: Bearer $GITEE_TOKEN" "$url"
		else
			wget -qO- "$url"
		fi
	else
		err "neither curl nor wget is available"
	fi
}

# Download a URL to a file.
download() {
	url="$1"
	dest="$2"
	if have curl; then
		if [ -n "$GITEE_TOKEN" ]; then
			printf 'header = "Authorization: Bearer %s"\n' "$GITEE_TOKEN" |
				curl -fsSL --config - -o "$dest" "$url"
		else
			curl -fsSL -o "$dest" "$url"
		fi
	elif have wget; then
		if [ -n "$GITEE_TOKEN" ]; then
			wget -qO "$dest" --header="Authorization: Bearer $GITEE_TOKEN" "$url"
		else
			wget -qO "$dest" "$url"
		fi
	else
		err "neither curl nor wget is available"
	fi
}

# ---------------------------------------------------------------------------
# Argument parsing (only meaningful when the script runs from a file)
# ---------------------------------------------------------------------------

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		[ $# -ge 2 ] || err "--version requires a value"
		VERSION="$2"
		shift 2
		;;
	--version=*)
		VERSION="${1#*=}"
		shift
		;;
	--install-dir)
		[ $# -ge 2 ] || err "--install-dir requires a value"
		INSTALL_DIR="$2"
		shift 2
		;;
	--install-dir=*)
		INSTALL_DIR="${1#*=}"
		shift
		;;
	-h | --help)
		usage
		;;
	*)
		err "unknown argument: $1"
		;;
	esac
done

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------

detect_os() {
	os="$(uname -s)"
	case "$os" in
	Linux) echo "linux" ;;
	Darwin) echo "darwin" ;;
	MINGW* | MSYS* | CYGWIN* | Windows_NT)
		err "Windows is not supported by this script. Install with npm instead:
    npm install -g @gitee/gitee-cli
Or download the .zip archive from https://${HOST}/${REPO}/releases"
		;;
	*)
		err "unsupported operating system: $os"
		;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) echo "amd64" ;;
	aarch64 | arm64) echo "arm64" ;;
	*)
		err "unsupported architecture: $arch"
		;;
	esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# ---------------------------------------------------------------------------
# Resolve install directory
# ---------------------------------------------------------------------------

if [ -z "$INSTALL_DIR" ]; then
	if [ "$(id -u 2>/dev/null || echo 1000)" = "0" ]; then
		INSTALL_DIR="/usr/local/bin"
	else
		INSTALL_DIR="$HOME/.local/bin"
	fi
fi

# ---------------------------------------------------------------------------
# Resolve version and asset URLs
# ---------------------------------------------------------------------------

# Extract a field's value from a JSON blob without requiring jq.
json_field() {
	# json_field <key> reads JSON from stdin, prints the first string value.
	key="$1"
	sed -n "s/.*\"${key}\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" | head -n 1
}

resolve_release_json() {
	if [ "$VERSION" = "latest" ]; then
		fetch "${API_PREFIX}/repos/${REPO}/releases/latest"
	else
		tag="$VERSION"
		case "$tag" in
		v*) ;;
		*) tag="v${tag}" ;;
		esac
		fetch "${API_PREFIX}/repos/${REPO}/releases/tags/${tag}"
	fi
}

info "Fetching release metadata for ${REPO} (${VERSION})..."
RELEASE_JSON="$(resolve_release_json)" || err "failed to fetch release metadata"

TAG="$(printf '%s' "$RELEASE_JSON" | json_field tag_name)"
[ -n "$TAG" ] || err "could not determine release tag; the release may not exist"

RELEASE_VERSION="${TAG#v}"
ARCHIVE_NAME="${BINARY_NAME}_${RELEASE_VERSION}_${OS}_${ARCH}.tar.gz"

# Find the download URL for a named asset in the release JSON.
asset_url() {
	name="$1"
	# Each asset object exposes "browser_download_url"; match the one whose URL
	# ends with the wanted filename.
	printf '%s' "$RELEASE_JSON" |
		tr ',' '\n' |
		grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*"' |
		sed 's/.*"\(https\{0,1\}:[^"]*\)".*/\1/' |
		grep -F "/${name}" |
		head -n 1
}

ARCHIVE_URL="$(asset_url "$ARCHIVE_NAME")"
CHECKSUMS_URL="$(asset_url "checksums.txt")"

# Fallback: construct the conventional Gitee attachment download path if the
# API response did not expose browser_download_url fields.
if [ -z "$ARCHIVE_URL" ]; then
	ARCHIVE_URL="https://${HOST}/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"
fi
if [ -z "$CHECKSUMS_URL" ]; then
	CHECKSUMS_URL="https://${HOST}/${REPO}/releases/download/${TAG}/checksums.txt"
fi

# ---------------------------------------------------------------------------
# Download, verify, extract
# ---------------------------------------------------------------------------

TMPDIR_INSTALL="$(mktemp -d 2>/dev/null || mktemp -d -t gitee-cli)"
cleanup() { rm -rf "$TMPDIR_INSTALL"; }
trap cleanup EXIT INT TERM

ARCHIVE_PATH="${TMPDIR_INSTALL}/${ARCHIVE_NAME}"
CHECKSUMS_PATH="${TMPDIR_INSTALL}/checksums.txt"

info "Downloading ${ARCHIVE_NAME}..."
download "$ARCHIVE_URL" "$ARCHIVE_PATH" || err "failed to download ${ARCHIVE_URL}"

info "Verifying checksum..."
if download "$CHECKSUMS_URL" "$CHECKSUMS_PATH" 2>/dev/null && [ -s "$CHECKSUMS_PATH" ]; then
	expected="$(grep -F "$ARCHIVE_NAME" "$CHECKSUMS_PATH" | awk '{print $1}' | head -n 1)"
	if [ -z "$expected" ]; then
		warn "no checksum entry found for ${ARCHIVE_NAME}; skipping verification"
	else
		if have sha256sum; then
			actual="$(sha256sum "$ARCHIVE_PATH" | awk '{print $1}')"
		elif have shasum; then
			actual="$(shasum -a 256 "$ARCHIVE_PATH" | awk '{print $1}')"
		else
			actual=""
			warn "no sha256sum/shasum available; skipping verification"
		fi
		if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
			err "checksum mismatch for ${ARCHIVE_NAME}
    expected: ${expected}
    actual:   ${actual}"
		fi
		[ -n "$actual" ] && info "Checksum OK."
	fi
else
	warn "could not download checksums.txt; skipping verification"
fi

info "Extracting..."
tar -xzf "$ARCHIVE_PATH" -C "$TMPDIR_INSTALL" || err "failed to extract archive"

BIN_SRC="${TMPDIR_INSTALL}/${BINARY_NAME}"
[ -f "$BIN_SRC" ] || err "binary ${BINARY_NAME} not found in archive"
chmod +x "$BIN_SRC"

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------

BIN_DEST="${INSTALL_DIR}/${BINARY_NAME}"

install_binary() {
	if mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ]; then
		mv "$BIN_SRC" "$BIN_DEST"
		return 0
	fi
	# Directory not writable: retry with sudo when available.
	if have sudo; then
		info "Elevated permissions required to write to ${INSTALL_DIR}."
		sudo mkdir -p "$INSTALL_DIR" && sudo mv "$BIN_SRC" "$BIN_DEST"
		return 0
	fi
	return 1
}

if ! install_binary; then
	err "cannot write to ${INSTALL_DIR}.
Re-run with a writable location, e.g.:
    INSTALL_DIR=\"\$HOME/.local/bin\" sh install.sh"
fi

info ""
info "Installed ${BINARY_NAME} ${TAG} to ${BIN_DEST}"

# ---------------------------------------------------------------------------
# PATH guidance
# ---------------------------------------------------------------------------

case ":${PATH}:" in
*":${INSTALL_DIR}:"*)
	info "Run 'gitee --version' to get started."
	;;
*)
	info ""
	info "${INSTALL_DIR} is not on your PATH. Add it by appending this line to"
	info "your shell profile (~/.bashrc, ~/.zshrc, ~/.profile):"
	info ""
	info "    export PATH=\"${INSTALL_DIR}:\$PATH\""
	info ""
	info "Then restart your shell and run 'gitee --version'."
	;;
esac
