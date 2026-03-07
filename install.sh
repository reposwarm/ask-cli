#!/bin/sh
# ask CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/reposwarm/ask-cli/main/install.sh | sh
#
# Installs the latest ask binary to /usr/local/bin (or ~/.local/bin if no sudo).
# Supports Linux and macOS, amd64 and arm64.

set -e

REPO="reposwarm/ask-cli"
BINARY="ask"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)   ARCH="amd64" ;;
  aarch64|arm64)   ARCH="arm64" ;;
  *)               echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
echo "🔍 Finding latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "❌ Could not find latest release. Check https://github.com/${REPO}/releases"
  exit 1
fi
echo "📦 Latest version: ${LATEST}"

# Build download URL
ASSET="${BINARY}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  ASSET="${ASSET}.exe"
fi
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"

# Download
TMPDIR=$(mktemp -d)
TMPFILE="${TMPDIR}/${BINARY}"
echo "⬇️  Downloading ${URL}..."
if ! curl -fsSL -o "$TMPFILE" "$URL"; then
  echo "❌ Download failed. Release may not have a binary for ${OS}/${ARCH}."
  echo "   Check: https://github.com/${REPO}/releases/tag/${LATEST}"
  rm -rf "$TMPDIR"
  exit 1
fi
chmod +x "$TMPFILE"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  echo "📁 Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
else
  # Fall back to user-local directory
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
  echo ""
  echo "⚠️  Installed to ${INSTALL_DIR}/${BINARY}"
  echo "   Make sure ${INSTALL_DIR} is in your PATH:"
  echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

rm -rf "$TMPDIR"

echo ""
echo "✅ ask ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "   Get started:"
echo "     ask setup                          # Set up local askbox"
echo "     ask \"how does auth work?\"           # Ask a question"
echo "     ask results list                   # Browse architecture docs"
echo ""
