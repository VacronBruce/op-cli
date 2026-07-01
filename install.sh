#!/bin/bash
# Don't use set -e — install scripts need to handle partial failures gracefully

# op-cli installer
#
# Method 1 (curl — downloads the latest release binary, no auth needed):
#   bash <(curl -fsSL https://github.com/VacronBruce/op-cli/releases/latest/download/install.sh)
#
# Method 2 (clone + build):
#   git clone https://github.com/VacronBruce/op-cli.git && cd op-cli && bash install.sh

VERSION="0.21.1"
REPO="VacronBruce/op-cli"
REPO_URL="https://github.com/${REPO}"
DL_URL="${REPO_URL}/releases/latest/download"  # GitHub always serves the newest release here
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
  # Mode: download pre-built binary from the latest GitHub release.
  # The repo is public, so the release asset is a plain download — no auth needed.
  echo "    Downloading ${BINARY} from the latest GitHub release..."
  if ! curl -fsSL -o /tmp/op-install "${DL_URL}/${BINARY}"; then
    echo "    Error: download failed."
    echo ""
    echo "    Alternative — clone and build (needs Go):"
    echo "      git clone ${REPO_URL}.git"
    echo "      cd op-cli && bash install.sh"
    exit 1
  fi
  echo "    Downloaded."
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
    MP_SRC="${REPO_URL}.git"
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

# Verify — op setup prints an [ok]/[--] checklist with a fix for each gap
echo "--- Verifying ---"
if op setup; then
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
  echo "Some checks failed — each [--] line above shows the fix."
  echo "Re-run 'op setup' anytime to re-check."
fi
