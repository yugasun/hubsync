#!/bin/bash
set -e

REPO="yugasun/hubsync"
LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f4)
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [[ "$ARCH" == "x86_64" ]]; then
  ARCH=amd64
elif [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
  ARCH=arm64
else
  echo "Unsupported architecture: $ARCH"
  exit 1
fi

BINARY="hubsync-${OS}-${ARCH}"
URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY"

TMP=$(mktemp -d)
cd $TMP

if ! curl -fLO $URL; then
  echo "Download failed: $URL"
  exit 1
fi
chmod +x $BINARY
sudo mv $BINARY /usr/local/bin/hubsync
cd -
rm -rf $TMP

echo "hubsync installed to /usr/local/bin/hubsync"
echo "Please run 'hubsync --help' to see usage instructions."
