#!/bin/bash
# Don't use set -e — install scripts need to handle partial failures gracefully

# op-cli installer
#
# Method 1 (curl + GitLab token):
#   GITLAB_TOKEN=your-token bash <(curl -fsSH "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/0.3.0/install.sh)
#
# Method 2 (clone + build):
#   git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git && cd op-cli && git checkout develop && bash install.sh

VERSION="0.8.0"
GITLAB_URL="https://gitlab-tw.ddns.net"
PKG_URL="${GITLAB_URL}/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/latest"
INSTALL_DIR="/usr/local/bin"
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
echo "1/3 Installing op binary..."

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

echo "    Installing to ${INSTALL_DIR}/op..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/op-install "${INSTALL_DIR}/op"
else
  sudo mv /tmp/op-install "${INSTALL_DIR}/op"
fi
if ! command -v op &>/dev/null; then
  echo "    Warning: op not found in PATH. You may need to add ${INSTALL_DIR} to your PATH."
else
  echo "    Done: $(which op)"
fi
echo ""

# Step 2: Config
echo "2/3 Config setup"
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

# Step 3: Claude Code skills (openproject + ticket-*, standup, file-bug)
echo "3/3 Claude Code skills"
SKILLS_ROOT="$HOME/.claude/skills"
if command -v claude &>/dev/null || [ -d "$HOME/.claude" ]; then
  GOT_SKILLS=""

  if [ -d "skill" ] && [ -f "skill/SKILL.md" ]; then
    # Clone mode: install straight from the repo's skill/ directory.
    mkdir -p "${SKILLS_ROOT}/openproject"
    cp skill/SKILL.md "${SKILLS_ROOT}/openproject/SKILL.md"
    for d in skill/*/; do
      [ -f "${d}SKILL.md" ] || continue
      name=$(basename "$d")
      mkdir -p "${SKILLS_ROOT}/${name}"
      cp "${d}SKILL.md" "${SKILLS_ROOT}/${name}/SKILL.md"
    done
    GOT_SKILLS=true
  else
    # Download mode: fetch the skills bundle (glab, then token).
    SKILLS_TGZ="/tmp/op-skills.tar.gz"
    rm -f "$SKILLS_TGZ"
    if command -v glab &>/dev/null; then
      GITLAB_HOST=gitlab-tw.ddns.net glab release download \
        --repo gmedtn/op-cli --include-external \
        --asset-name="op-skills.tar.gz" -D /tmp 2>/dev/null
    fi
    if [ ! -f "$SKILLS_TGZ" ] && [ -n "$GITLAB_TOKEN" ]; then
      curl -fsSL -o "$SKILLS_TGZ" -H "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "${PKG_URL}/op-skills.tar.gz" 2>/dev/null
    fi
    if [ -f "$SKILLS_TGZ" ]; then
      mkdir -p "$SKILLS_ROOT"
      tar -xzf "$SKILLS_TGZ" -C "$SKILLS_ROOT" && GOT_SKILLS=true
    fi
  fi

  if [ -n "$GOT_SKILLS" ]; then
    echo "    Installed skills to ${SKILLS_ROOT}/ (openproject, ticket-prep/verify/review, standup, file-bug)"
  else
    echo "    Could not install the skills bundle; skipping."
    echo "    To add them later: clone the repo and copy skill/* into ${SKILLS_ROOT}/."
  fi
else
  echo "    Claude Code not detected, skipping."
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
