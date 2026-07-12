#!/usr/bin/env bash
set -euo pipefail

REPO="andrespd99/fireauth"
BINARY="fireauth"
INSTALL_DIR="/usr/local/bin"
API_BASE="https://api.github.com/repos/${REPO}"

# --- Parse arguments ---
TARGET_VERSION=""

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Install fireauth from GitHub releases.

Options:
  --version <ver>   Install a specific version (e.g. 0.3.0, 0.3.0-alpha.1)
                    If omitted, installs the latest stable release.
  --help            Show this help message.

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

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"

# --- Resolve release ---
if [ -n "$TARGET_VERSION" ]; then
  # Strip leading 'v' if user included it.
  TARGET_VERSION="${TARGET_VERSION#v}"
  TAG="v${TARGET_VERSION}"
  echo "Fetching release ${TAG}..."
  ASSET_URL=$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    "${API_BASE}/releases/tags/${TAG}" \
    | grep -o "\"browser_download_url\": \"https://[^\"]*${ARCHIVE}\"" \
    | head -1 \
    | sed -E 's/.*"browser_download_url": "(.*)"/\1/')

  if [ -z "$ASSET_URL" ]; then
    echo "Error: could not find asset '${ARCHIVE}' in release ${TAG}" >&2
    echo "Available assets:" >&2
    curl -fsSL "${API_BASE}/releases/tags/${TAG}" \
      | grep '"name"' | sed -E 's/.*"name": "(.*)".*/  \1/' >&2
    exit 1
  fi
else
  # Find the latest stable release (tag matching v*.*.*-stable).
  echo "Fetching latest stable release..."
  RELEASES_JSON=$(curl -fsSL \
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
  ASSET_URL=$(echo "$RELEASES_JSON" \
    | grep -o "\"browser_download_url\": \"https://[^\"]*${ARCHIVE}\"" \
    | head -1 \
    | sed -E 's/.*"browser_download_url": "(.*)"/\1/')

  if [ -z "$ASSET_URL" ]; then
    echo "Error: could not find asset '${ARCHIVE}' in release ${TAG}" >&2
    exit 1
  fi
fi

# --- Download ---
echo "Downloading ${ARCHIVE}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "$ASSET_URL"

# --- Verify checksum ---
# Fetch checksums.txt from the same release and verify the archive.
if [ -n "$RELEASES_JSON" ]; then
  CHECKSUMS_URL=$(echo "$RELEASES_JSON" | grep -o "\"browser_download_url\": \"https://[^\"]*checksums.txt\"" | head -1 | sed -E 's/.*"browser_download_url": "(.*)"/\1/')
else
  CHECKSUMS_URL=$(curl -fsSL -H "Accept: application/vnd.github+json" "${API_BASE}/releases/tags/${TAG}" | grep -o "\"browser_download_url\": \"https://[^\"]*checksums.txt\"" | head -1 | sed -E 's/.*"browser_download_url": "(.*)"/\1/')
fi

if [ -n "$CHECKSUMS_URL" ]; then
  echo "Verifying checksum..."
  curl -fsSL -o "${TMP_DIR}/checksums.txt" "$CHECKSUMS_URL"
  EXPECTED=$(grep " ${ARCHIVE}$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    ACTUAL=$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      echo "Error: checksum mismatch for ${ARCHIVE}" >&2
      echo "  Expected: $EXPECTED" >&2
      echo "  Actual:   $ACTUAL" >&2
      exit 1
    fi
    echo "Checksum verified."
  fi
fi

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
echo "fireauth ${TAG} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Next step: run 'fireauth init' to set up your Firebase credentials."