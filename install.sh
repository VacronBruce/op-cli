#!/bin/bash
set -e

# op-cli installer
# Usage: curl -sL https://gitlab-tw.ddns.net/gmedtn/op-cli/-/raw/main/install.sh | bash

GITLAB_URL="https://gitlab-tw.ddns.net"
PROJECT="gmedtn/op-cli"
VERSION="v0.2.0"
INSTALL_DIR="/usr/local/bin"
OP_URL="https://openpr.epochbase.com"

echo "================================"
echo "  op-cli installer ($VERSION)"
echo "================================"
echo ""

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)      echo "Error: unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64|amd64)  ARCH="amd64" ;;
  *)             echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="op-${OS}-${ARCH}"
DOWNLOAD_URL="${GITLAB_URL}/${PROJECT}/-/releases/${VERSION}/downloads/${BINARY}"

echo "Platform: ${OS}/${ARCH}"
echo "Binary:   ${BINARY}"
echo ""

# Download
echo "Downloading ${BINARY}..."
if command -v curl &>/dev/null; then
  curl -fSL -o /tmp/op "$DOWNLOAD_URL"
elif command -v wget &>/dev/null; then
  wget -q -O /tmp/op "$DOWNLOAD_URL"
else
  echo "Error: curl or wget required"
  exit 1
fi

chmod +x /tmp/op

# Install
echo "Installing to ${INSTALL_DIR}/op..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/op "${INSTALL_DIR}/op"
else
  sudo mv /tmp/op "${INSTALL_DIR}/op"
fi

echo "Installed: $(which op)"
echo ""

# Config setup
if [ -f "$HOME/.oprc" ]; then
  echo "Config already exists at ~/.oprc, skipping."
else
  echo "--- OpenProject Config Setup ---"
  echo ""
  echo "You need an API key from: ${OP_URL}"
  echo "Go to: My Account > Access Tokens > Create new token"
  echo ""
  # Read from /dev/tty so it works even when piped (curl | bash)
  read -p "Paste your API key: " API_KEY < /dev/tty

  if [ -z "$API_KEY" ]; then
    echo "No API key provided. You can set it later in ~/.oprc"
    cat > "$HOME/.oprc" <<EOF
url: ${OP_URL}
api_key: YOUR_API_KEY_HERE
# project: app
# sprint: "App_05/19/2026"
EOF
  else
    read -p "Default project (leave empty to skip): " DEFAULT_PROJECT < /dev/tty
    read -p "Default sprint (leave empty to skip): " DEFAULT_SPRINT < /dev/tty

    cat > "$HOME/.oprc" <<EOF
url: ${OP_URL}
api_key: ${API_KEY}
EOF

    if [ -n "$DEFAULT_PROJECT" ]; then
      echo "project: ${DEFAULT_PROJECT}" >> "$HOME/.oprc"
    fi
    if [ -n "$DEFAULT_SPRINT" ]; then
      echo "sprint: \"${DEFAULT_SPRINT}\"" >> "$HOME/.oprc"
    fi
  fi

  chmod 600 "$HOME/.oprc"
  echo "Config saved to ~/.oprc"
fi

echo ""

# Verify
echo "--- Verifying ---"
if op projects 2>/dev/null | head -3; then
  echo ""
  echo "================================"
  echo "  Setup complete!"
  echo "================================"
  echo ""
  echo "Try these commands:"
  echo "  op projects          # List all projects"
  echo "  op board -p web      # Sprint board"
  echo "  op my -p web         # My items"
  echo "  op --help            # All commands"
else
  echo ""
  echo "Installed but could not connect."
  echo "Check your API key in ~/.oprc"
fi
