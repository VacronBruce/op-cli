#!/bin/bash
# Don't use set -e — install scripts need to handle partial failures gracefully

# op-cli installer
#
# Method 1 (curl + GitLab token):
#   GITLAB_TOKEN=your-token bash <(curl -fsSH "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/0.3.0/install.sh)
#
# Method 2 (clone + build):
#   git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git && cd op-cli && git checkout develop && bash install.sh

VERSION="0.12.0"
GITLAB_URL="https://gitlab-tw.ddns.net"
PKG_URL="${GITLAB_URL}/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/latest"
INSTALL_DIR="${INSTALL_DIR:-}"  # override via env var; auto-detected below
OP_URL="https://openpr.epochbase.com"

echo "================================"
echo "  op-cli installer (v${VERSION})"
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
echo "Platform: ${OS}/${ARCH}"
echo ""

# Step 1: Get the binary
echo "1/4 Installing op binary..."

if [ -f "go.mod" ] && [ -f "main.go" ]; then
  # Mode: clone — build from source
  if command -v go &>/dev/null; then
    echo "    Building from source..."
    if ! go build -ldflags "-X github.com/chenhuijun/op-cli/cmd.Version=${VERSION}" -o /tmp/op-install .; then
      echo "    Error: build failed."
      exit 1
    fi
    echo "    Built successfully."
  else
    echo "    Error: Go is required to build from source."
    echo "    Install Go from https://go.dev/dl/ or use the curl method instead."
    exit 1
  fi
else
  # Mode: download pre-built binary
  # Try to get a GitLab token from: env var → glab CLI → prompt user

  if [ -z "$GITLAB_TOKEN" ]; then
    # Try glab CLI first
    if command -v glab &>/dev/null; then
      echo "    Found glab CLI, downloading via glab..."
      GITLAB_HOST=gitlab-tw.ddns.net glab release download \
        --repo gmedtn/op-cli --include-external \
        --asset-name="${BINARY}" -D /tmp 2>/dev/null
      if [ -f "/tmp/${BINARY}" ]; then
        mv "/tmp/${BINARY}" /tmp/op-install
        echo "    Downloaded via glab."
        DOWNLOADED=true
      else
        echo "    glab download failed. You may need to authenticate:"
        echo "      GITLAB_HOST=gitlab-tw.ddns.net glab auth login"
        echo ""
      fi
    fi
  fi

  # If glab didn't work, try curl with token
  if [ -z "$DOWNLOADED" ]; then
    if [ -z "$GITLAB_TOKEN" ]; then
      echo ""
      echo "    GitLab authentication required to download the binary."
      echo ""
      echo "    You need a GitLab Personal Access Token (PAT):"
      echo "    1. Go to: https://gitlab-tw.ddns.net/-/user_settings/personal_access_tokens"
      echo "    2. Create a token with 'read_api' scope"
      echo ""
      read -p "    Paste your GitLab token (or press Enter to skip): " GITLAB_TOKEN < /dev/tty

      if [ -z "$GITLAB_TOKEN" ]; then
        echo ""
        echo "    No token provided. Alternative install methods:"
        echo ""
        echo "    Option A — set token and retry:"
        echo "      export GITLAB_TOKEN=your-gitlab-token"
        echo "      bash install.sh"
        echo ""
        echo "    Option B — use glab CLI:"
        echo "      brew install glab"
        echo "      GITLAB_HOST=gitlab-tw.ddns.net glab auth login"
        echo "      bash install.sh"
        echo ""
        echo "    Option C — clone and build (needs Go):"
        echo "      git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git"
        echo "      cd op-cli && git checkout develop && bash install.sh"
        exit 1
      fi
    fi

    echo "    Downloading ${BINARY}..."
    if ! curl -fsSL -o /tmp/op-install -H "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "${PKG_URL}/${BINARY}"; then
      echo "    Error: download failed. Check your token."
      exit 1
    fi
    echo "    Downloaded."
  fi
fi

chmod +x /tmp/op-install

# Auto-detect install dir: env override → ~/.local/bin (if in PATH) → /usr/local/bin
if [ -z "$INSTALL_DIR" ]; then
  LOCAL_BIN="$HOME/.local/bin"
  case ":$PATH:" in
    *":${LOCAL_BIN}:"*) INSTALL_DIR="$LOCAL_BIN" ;;
    *)                  INSTALL_DIR="/usr/local/bin" ;;
  esac
fi
mkdir -p "$INSTALL_DIR"

echo "    Installing to ${INSTALL_DIR}/op..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/op-install "${INSTALL_DIR}/op"
else
  sudo mv /tmp/op-install "${INSTALL_DIR}/op"
fi
case ":$PATH:" in
  *":${INSTALL_DIR}:"*)
    echo "    Done: $(which op)" ;;
  *)
    echo "    Installed to ${INSTALL_DIR}/op"
    echo "    Warning: ${INSTALL_DIR} is not in PATH. Add this to your shell profile:"
    echo "      export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac
echo ""

# Step 2: Config
echo "2/4 Config setup"
if [ -f "$HOME/.oprc" ]; then
  echo "    Already exists at ~/.oprc, skipping."
else
  echo ""
  echo "    You need an OpenProject API key from: ${OP_URL}"
  echo "    Go to: My Account > Access Tokens > Create new token"
  echo ""
  read -p "    Paste your API key: " API_KEY < /dev/tty

  if [ -z "$API_KEY" ]; then
    echo "    No API key provided. Edit ~/.oprc later."
    cat > "$HOME/.oprc" <<EOF
url: ${OP_URL}
api_key: YOUR_API_KEY_HERE
# project: app
# sprint: "App_05/19/2026"
EOF
  else
    read -p "    Default project (leave empty to skip): " DEFAULT_PROJECT < /dev/tty
    read -p "    Default sprint  (leave empty to skip): " DEFAULT_SPRINT < /dev/tty

    cat > "$HOME/.oprc" <<EOF
url: ${OP_URL}
api_key: ${API_KEY}
EOF
    [ -n "$DEFAULT_PROJECT" ] && echo "project: ${DEFAULT_PROJECT}" >> "$HOME/.oprc"
    [ -n "$DEFAULT_SPRINT" ] && echo "sprint: \"${DEFAULT_SPRINT}\"" >> "$HOME/.oprc"
  fi

  chmod 600 "$HOME/.oprc"
  echo "    Saved to ~/.oprc"
fi
echo ""

# Step 3: Shell completion — add a source line to the user's shell rc (idempotent)
echo "3/4 Shell completion"
SHELL_NAME=$(basename "${SHELL:-}")
case "$SHELL_NAME" in
  zsh)  COMPLETION_RC="$HOME/.zshrc" ;;
  bash) COMPLETION_RC="$HOME/.bashrc" ;;
  *)    COMPLETION_RC="" ;;
esac

if [ -n "$COMPLETION_RC" ]; then
  if [ -f "$COMPLETION_RC" ] && grep -qF "op completion $SHELL_NAME" "$COMPLETION_RC"; then
    echo "    Already enabled in ${COMPLETION_RC}, skipping."
  else
    {
      echo ""
      echo "# op-cli shell completion"
      echo "command -v op &>/dev/null && source <(op completion $SHELL_NAME)"
    } >> "$COMPLETION_RC"
    echo "    Enabled in ${COMPLETION_RC} — restart your shell or run: source ${COMPLETION_RC}"
  fi
else
  echo "    Shell '${SHELL_NAME}' not auto-configured. Enable it manually, e.g.:"
  echo "      op completion --help"
fi
echo ""

# Step 4: Claude Code plugin (OpenProject skills under the op: prefix)
echo "4/4 Claude Code plugin (op:)"
if command -v claude &>/dev/null; then
  # Marketplace source: the local repo dir in clone mode, the git URL otherwise.
  if [ -d ".claude-plugin" ] && [ -f ".claude-plugin/marketplace.json" ]; then
    MP_SRC="$(pwd)"
  else
    MP_SRC="git@gitlab-tw.ddns.net:gmedtn/op-cli.git"
  fi

  # Register the marketplace (or refresh it if already present), then install.
  if claude plugin marketplace add "$MP_SRC" 2>/dev/null \
     || claude plugin marketplace update op 2>/dev/null; then
    if claude plugin install op@op --scope user 2>/dev/null; then
      # Migrate: drop any old loose skill copies so they don't shadow the plugin.
      SKILLS_ROOT="$HOME/.claude/skills"
      for s in openproject op-bridge ticket-prep ticket-verify ticket-review standup file-bug; do
        [ -d "${SKILLS_ROOT}/${s}" ] && rm -rf "${SKILLS_ROOT}/${s}"
      done
      echo "    Installed the op plugin — use /op:openproject, /op:standup, /op:file-bug, /op:ticket-* ..."
      echo "    (Removed any old loose copies under ${SKILLS_ROOT}/ to avoid duplicates.)"
    else
      echo "    Marketplace added but plugin install failed. Run: claude plugin install op@op"
    fi
  else
    echo "    Could not add the op marketplace. Add it manually:"
    echo "      claude plugin marketplace add ${MP_SRC}"
    echo "      claude plugin install op@op"
  fi
else
  echo "    Claude Code (claude CLI) not detected, skipping."
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
  echo "  op projects       # List all projects"
  echo "  op board          # Sprint board"
  echo "  op my             # My items"
  echo "  op show <id>      # View ticket"
  echo "  op --help         # All commands"
else
  echo ""
  echo "Binary installed but could not connect."
  echo "Check your API key in ~/.oprc"
fi
