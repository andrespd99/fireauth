#!/usr/bin/env bash
set -euo pipefail

REPO="cashea-bnpl/auth-devtools"
BINARY="cashea-auth"
INSTALL_DIR="/usr/local/bin"
API_BASE="https://api.github.com/repos/${REPO}"

# --- Parse arguments ---
TARGET_VERSION=""

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Install cashea-auth from GitHub releases.

Options:
  --version <ver>   Install a specific version (e.g. 0.3.0, 0.3.0-alpha.1)
                    If omitted, installs the latest stable release.
  --help            Show this help message.

Environment:
  GITHUB_TOKEN      Required (private repo). Or use gh CLI (gh auth login).

Examples:
  $0                          # Install latest stable
  $0 --version 0.3.0-stable    # Install a specific stable release
  $0 --version 0.3.0-alpha.1   # Install a pre-release
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      TARGET_VERSION="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

# --- GitHub token (required for private repos) ---
if [ -z "${GITHUB_TOKEN:-}" ]; then
  if command -v gh &>/dev/null; then
    GITHUB_TOKEN=$(gh auth token 2>/dev/null || true)
  fi
fi

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "Error: GITHUB_TOKEN is required (private repo)." >&2
  echo "" >&2
  echo "Option 1: export GITHUB_TOKEN=ghp_..." >&2
  echo "Option 2: install gh CLI and run 'gh auth login'" >&2
  exit 1
fi

AUTH_HEADER="Authorization: token ${GITHUB_TOKEN}"

# --- Detect OS and architecture ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

echo "Detected: ${OS}/${ARCH}"

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"

# --- Resolve release ---
# Helper: download a release asset by tag name.
# Sets ASSET_ID for the matching archive in the release.
fetch_release_assets() {
  local tag="$1"
  local release_json
  release_json=$(curl -fsSL -H "$AUTH_HEADER" \
    -H "Accept: application/vnd.github+json" \
    "${API_BASE}/releases/tags/${tag}")

  ASSET_ID=$(echo "$release_json" \
    | grep -B 3 "\"name\": \"${ARCHIVE}\"" \
    | grep '"id"' \
    | head -1 \
    | sed -E 's/.*: ([0-9]+).*/\1/')

  if [ -z "$ASSET_ID" ]; then
    echo "Error: could not find asset '${ARCHIVE}' in release ${tag}" >&2
    echo "Available assets:" >&2
    echo "$release_json" | grep '"name"' | sed -E 's/.*"name": "(.*)".*/  \1/' >&2
    exit 1
  fi
}

if [ -n "$TARGET_VERSION" ]; then
  # Strip leading 'v' if user included it.
  TARGET_VERSION="${TARGET_VERSION#v}"
  TAG="v${TARGET_VERSION}"
  echo "Fetching release ${TAG}..."
  fetch_release_assets "$TAG"
else
  # Find the latest stable release (tag matching v*.*.*-stable).
  echo "Fetching latest stable release..."
  # GitHub's /releases/latest returns the release marked as "latest", but
  # that may be a pre-release if it's the only one. We filter by listing all
  # releases and finding the newest non-prerelease with a -stable tag.
  RELEASES_JSON=$(curl -fsSL -H "$AUTH_HEADER" \
    -H "Accept: application/vnd.github+json" \
    "${API_BASE}/releases?per_page=30")

  TAG=$(echo "$RELEASES_JSON" \
    | grep -E '"tag_name": "v[0-9]+\.[0-9]+\.[0-9]+-stable"' \
    | head -1 \
    | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

  if [ -z "$TAG" ]; then
    echo "Error: no stable release found (tag matching v*.*.*-stable)." >&2
    echo "Available releases:" >&2
    echo "$RELEASES_JSON" | grep '"tag_name"' | sed -E 's/.*"tag_name": "(.*)".*/  \1/' >&2
    echo "" >&2
    echo "You can install a specific version with: $0 --version <version>" >&2
    exit 1
  fi

  echo "Latest stable version: ${TAG}"
  fetch_release_assets "$TAG"
fi

# --- Download ---
echo "Downloading ${ARCHIVE}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL \
  -H "$AUTH_HEADER" \
  -H "Accept: application/octet-stream" \
  "${API_BASE}/releases/assets/${ASSET_ID}" \
  -o "${TMP_DIR}/${ARCHIVE}"

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

# --- Install ---
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "cashea-auth ${TAG} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next step: run 'cashea-auth init' to set up your Firebase credentials."