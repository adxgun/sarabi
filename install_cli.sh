#!/bin/bash

set -e

install_dir="$HOME/.sarabi-cli"
bin_name="sarabi"

# Detect OS and Architecture
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

echo "Detected OS: $OS"
echo "Detected Architecture: $ARCH"

# Get latest release URL
DOWNLOAD_URL=$(curl -s https://api.github.com/repos/adxgun/sarabi/releases/latest | \
  grep "browser_download_url" | \
  grep "${OS}_${ARCH}" | \
  cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "❌ Could not find a matching release for ${OS}_${ARCH}"
  exit 1
fi

mkdir -p "$install_dir"
cd "$install_dir"

echo "⬇️ Downloading Sarabi CLI from $DOWNLOAD_URL"
curl -L "$DOWNLOAD_URL" -o sarabi.tar.gz

echo "📦 Extracting..."
tar -xzf sarabi.tar.gz
chmod +x "$bin_name"
rm sarabi.tar.gz

# Add to PATH if not already
if [[ ":$PATH:" != *":$install_dir:"* ]]; then
  echo "📌 Adding $install_dir to PATH"
  echo "export PATH=\"\$PATH:$install_dir\"" >> "$HOME/.bashrc"
  echo "export PATH=\"\$PATH:$install_dir\"" >> "$HOME/.zshrc"
fi

echo "✅ Sarabi CLI installed at $install_dir/$bin_name"
"$bin_name" --version || echo "Run 'source ~/.bashrc' or 'source ~/.zshrc' to start using Sarabi"
