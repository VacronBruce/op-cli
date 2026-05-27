#!/bin/bash
set -e

# op-cli installer
#
# Method 1 (curl + GitLab token):
#   GITLAB_TOKEN=your-token bash <(curl -fsSH "PRIVATE-TOKEN: $GITLAB_TOKEN" https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/0.3.0/install.sh)
#
# Method 2 (clone + build):
#   git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git && cd op-cli && git checkout develop && bash install.sh

VERSION="0.3.0"
GITLAB_URL="https://gitlab-tw.ddns.net"
PKG_URL="${GITLAB_URL}/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/${VERSION}"
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
    go build -o /tmp/op-install .
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
    # Try glab CLI
    if command -v glab &>/dev/null; then
      echo "    Found glab CLI, checking auth..."
      GLAB_TOKEN=$(GITLAB_HOST=gitlab-tw.ddns.net glab auth status -t 2>&1 | grep "Token:" | head -1 | awk '{print $NF}')
      if [ -n "$GLAB_TOKEN" ] && [ "$GLAB_TOKEN" != "**************************" ]; then
        GITLAB_TOKEN="$GLAB_TOKEN"
        echo "    Using token from glab."
      else
        # glab is authenticated but token is masked — use glab to download instead
        echo "    Using glab to download ${BINARY}..."
        GITLAB_HOST=gitlab-tw.ddns.net glab release download v${VERSION} \
          --repo gmedtn/op-cli --include-external \
          --asset-name="${BINARY}" -D /tmp 2>/dev/null
        if [ -f "/tmp/${BINARY}" ]; then
          mv "/tmp/${BINARY}" /tmp/op-install
          echo "    Downloaded via glab."
          DOWNLOADED=true
        fi
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
    curl -fsSL -o /tmp/op-install -H "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "${PKG_URL}/${BINARY}"
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
echo "    Done: $(which op)"
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

# Step 3: Claude Code skill (embedded)
echo "3/3 Claude Code skill"
SKILL_DIR="$HOME/.claude/skills/openproject"
if command -v claude &>/dev/null || [ -d "$HOME/.claude" ]; then
  mkdir -p "$SKILL_DIR"
  cat > "${SKILL_DIR}/SKILL.md" <<'SKILL_EOF'
---
name: openproject
description: Manage OpenProject work packages, sprints, and backlogs via the `op` CLI
user_invocable: true
---

# OpenProject CLI Skill

Translate natural language requests into `op` CLI commands and execute them.

## Command Reference

```bash
op board                    # Sprint board
op my                       # My items
op my-team                  # Team items by person
op blocked                  # Blocked items
op show <id>                # Ticket details
op show <id> --download     # Download attachments
op projects                 # List projects

op create <type> "subject"  # Create (task/bug/feature/story)
  --assignee="Name" --priority=P1 --epic="NTD+"
  --component=android --product=entd --tech-area=app
  --label=team#appandroid --attach=file.png
op update <id> --status=in-progress --assignee="Name"
op assign <id> "Name"
op attach <id> file.png

op sprint plan/add/progress/close
op backlog / op backlog groom
op report
```

## How to Handle Requests

1. "create a bug" → `op create bug "subject" --flags`
2. "show board" → `op board`
3. "what am I working on?" → `op my`
4. "prep standup" → `op my-team` + `op blocked`
5. "assign X to Y" → `op assign <id> "Name"`
6. "attach screenshot" → save image, `op attach <id> /path`
7. "show ticket" → `op show <id> --download --out=/tmp`, read images

## Custom Fields
Components: android, ios, ott, engineering, analytics
Products: eet, entd, djy, cntd, others
Tech Areas: web, app, adtech, video, infra, portal, seo
Labels: team#appios, team#appandroid, team#appall, team#web, ntd, seo
SKILL_EOF
  echo "    Installed /openproject skill to ${SKILL_DIR}"
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
