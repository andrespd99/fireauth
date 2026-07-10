#!/usr/bin/env bash
set -euo pipefail

REPO="andrespd99/fireauth"
BINARY="fireauth"
INSTALL_DIR="/usr/local/bin"

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
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: could not determine latest release." >&2
  echo "Check https://github.com/${REPO}/releases" >&2
  exit 1
fi

echo "Latest version: ${LATEST}"

# Find the asset download URL for this OS/arch.
ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
ASSET_URL=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep -o "\"browser_download_url\": \"https://[^\"]*${ARCHIVE}\"" \
  | head -1 \
  | sed -E 's/.*"browser_download_url": "(.*)"/\1/')

if [ -z "$ASSET_URL" ]; then
  echo "Error: could not find release asset '${ARCHIVE}'" >&2
  echo "Available assets for ${LATEST}:" >&2
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"name"' | sed -E 's/.*"name": "(.*)".*/  \1/' >&2
  exit 1
fi

echo "Downloading ${ARCHIVE}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "$ASSET_URL"

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
echo "fireauth ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next step: run 'fireauth init' to set up your Firebase credentials."