#!/bin/bash
# dexfinder installer — downloads the latest release for your platform
set -euo pipefail

REPO="JuneLeGency/dexfinder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest release..."
VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')
if [ -z "$VERSION" ]; then
    echo "Failed to get latest version. Install manually from https://github.com/${REPO}/releases"
    exit 1
fi

FILENAME="dexfinder-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    FILENAME="${FILENAME}.zip"
else
    FILENAME="${FILENAME}.tar.gz"
fi

URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"
echo "Downloading dexfinder v${VERSION} for ${OS}/${ARCH}..."
curl -sL "$URL" -o "/tmp/${FILENAME}"

# Extract
cd /tmp
if [ "$OS" = "windows" ]; then
    unzip -o "$FILENAME"
else
    tar xzf "$FILENAME"
fi

# Install
BINARY=$(ls dexfinder-${OS}-${ARCH}* 2>/dev/null | grep -v '.tar.gz' | grep -v '.zip' | head -1)
if [ -z "$BINARY" ]; then
    echo "Failed to find binary after extraction"; exit 1
fi

echo "Installing to ${INSTALL_DIR}/dexfinder..."
sudo install -m 755 "$BINARY" "${INSTALL_DIR}/dexfinder"
rm -f "/tmp/${FILENAME}" "/tmp/${BINARY}"

echo "dexfinder v${VERSION} installed successfully!"
echo "Run 'dexfinder --help' to get started."
