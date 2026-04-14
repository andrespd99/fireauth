#!/usr/bin/env bash
set -euo pipefail

REPO="cashea-bnpl/auth-devtools"
BINARY="cashea-auth"
INSTALL_DIR="/usr/local/bin"

# --- GitHub token (required for private repos) ---
if [ -z "${GITHUB_TOKEN:-}" ]; then
  # Try gh CLI token as fallback.
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

# Detect OS and architecture.
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

# Get the latest release tag.
echo "Fetching latest release..."
LATEST=$(curl -fsSL -H "$AUTH_HEADER" "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: could not determine latest release." >&2
  echo "Check https://github.com/${REPO}/releases or verify your GITHUB_TOKEN has repo access." >&2
  exit 1
fi

echo "Latest version: ${LATEST}"

# Find the asset download URL for this OS/arch.
ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
ASSET_URL=$(curl -fsSL -H "$AUTH_HEADER" "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep -o "\"url\": \"https://api.github.com/repos/${REPO}/releases/assets/[0-9]*\"" \
  | head -1 \
  | sed -E 's/"url": "(.*)"/\1/')

# For private repos, we need to get the asset ID and use the API to download.
ASSET_ID=$(curl -fsSL -H "$AUTH_HEADER" "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep -B 3 "\"name\": \"${ARCHIVE}\"" \
  | grep '"id"' \
  | head -1 \
  | sed -E 's/.*: ([0-9]+).*/\1/')

if [ -z "$ASSET_ID" ]; then
  echo "Error: could not find release asset '${ARCHIVE}'" >&2
  echo "Available assets for ${LATEST}:" >&2
  curl -fsSL -H "$AUTH_HEADER" "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"name"' | sed -E 's/.*"name": "(.*)".*/  \1/' >&2
  exit 1
fi

echo "Downloading ${ARCHIVE}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL \
  -H "$AUTH_HEADER" \
  -H "Accept: application/octet-stream" \
  "https://api.github.com/repos/${REPO}/releases/assets/${ASSET_ID}" \
  -o "${TMP_DIR}/${ARCHIVE}"

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

# Install.
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "cashea-auth ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next step: run 'cashea-auth init' to set up your Firebase credentials."
