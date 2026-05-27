#!/bin/bash
set -e

# op-cli installer
# Usage: curl -fsSL https://gitlab-tw.ddns.net/gmedtn/op-cli/uploads/5c6f8b8930a81dd63249a3a37ae8e7fe/install.sh | bash

GITLAB_URL="https://gitlab-tw.ddns.net"
PROJECT="gmedtn/op-cli"
VERSION="v0.3.0"
INSTALL_DIR="/usr/local/bin"
OP_URL="https://openpr.epochbase.com"

# Direct upload URLs (these work for logged-in GitLab users without API auth)
declare -A BINARY_URLS
BINARY_URLS[op-darwin-arm64]="${GITLAB_URL}/${PROJECT}/uploads/27966c315689016fe7d27eae2bf7a879/op-darwin-arm64"
BINARY_URLS[op-darwin-amd64]="${GITLAB_URL}/${PROJECT}/uploads/2bf708f7797fa91830513678ff037332/op-darwin-amd64"
BINARY_URLS[op-linux-amd64]="${GITLAB_URL}/${PROJECT}/uploads/e0951c2c56d7d985151555aa0c4082b9/op-linux-amd64"

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
DOWNLOAD_URL="${BINARY_URLS[$BINARY]}"

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: no binary available for ${BINARY}"
  exit 1
fi

echo "Platform: ${OS}/${ARCH}"
echo "Binary:   ${BINARY}"
echo ""

# Download binary
echo "1/3 Downloading ${BINARY}..."
if command -v curl &>/dev/null; then
  curl -fSL -o /tmp/op "$DOWNLOAD_URL"
elif command -v wget &>/dev/null; then
  wget -q -O /tmp/op "$DOWNLOAD_URL"
else
  echo "Error: curl or wget required"
  exit 1
fi

chmod +x /tmp/op

# Install binary
echo "    Installing to ${INSTALL_DIR}/op..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/op "${INSTALL_DIR}/op"
else
  sudo mv /tmp/op "${INSTALL_DIR}/op"
fi
echo "    Done: $(which op)"
echo ""

# Config setup
echo "2/3 Config setup"
if [ -f "$HOME/.oprc" ]; then
  echo "    Config already exists at ~/.oprc, skipping."
else
  echo ""
  echo "    You need an API key from: ${OP_URL}"
  echo "    Go to: My Account > Access Tokens > Create new token"
  echo ""
  read -p "    Paste your API key: " API_KEY < /dev/tty

  if [ -z "$API_KEY" ]; then
    echo "    No API key provided. You can set it later in ~/.oprc"
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

    if [ -n "$DEFAULT_PROJECT" ]; then
      echo "project: ${DEFAULT_PROJECT}" >> "$HOME/.oprc"
    fi
    if [ -n "$DEFAULT_SPRINT" ]; then
      echo "sprint: \"${DEFAULT_SPRINT}\"" >> "$HOME/.oprc"
    fi
  fi

  chmod 600 "$HOME/.oprc"
  echo "    Saved to ~/.oprc"
fi
echo ""

# Install Claude Code skill (embedded — no extra download needed)
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

## Prerequisites
- `op` CLI installed (see install script in repo)
- Config at `~/.oprc` with `url`, `api_key`, `project`, and `sprint` fields

## Command Reference

### Daily Operations
```bash
op board                              # Sprint board (kanban view)
op my                                 # My assigned items
op my-team                            # Team items grouped by person
op blocked                            # Blocked items in sprint
op projects                           # List all projects
op show <id>                          # View ticket details
op show <id> --download               # Download attachments
```

### Create & Update
```bash
op create <type> "<subject>" [flags]  # Create work package
  # Types: task, bug, feature, epic, user-story, milestone
  # Flags:
  #   --assignee="Name"    --priority=P1
  #   --epic="NTD+"        --component=android
  #   --product=entd       --tech-area=app
  #   --label=team#appandroid
  #   --points=5           --sprint="Sprint 1"
  #   --description="..."  --attach=screenshot.png

op update <id> [flags]                # Update work package
op assign <id> "Person Name"          # Quick reassign
op attach <id> file.png [file2.jpg]   # Upload attachments
```

### Sprint Management
```bash
op sprint plan                        # Show backlog items for planning
op sprint add <id> [<id>...]          # Move items to sprint
op sprint progress                    # Sprint progress summary
op sprint close                       # Sprint close summary
```

### Backlog & Reporting
```bash
op backlog                            # All items not in a sprint
op backlog groom                      # Unestimated items
op report                             # Sprint report for stakeholders
```

## How to Handle Requests

1. **"create a task/bug"** → `op create task/bug "subject" --flags`
2. **"show board"** → `op board`
3. **"what am I working on?"** → `op my`
4. **"prep for standup"** → Run `op my-team` then `op blocked`, summarize
5. **"assign X to Y"** → `op assign <id> "Person"`
6. **"attach this screenshot"** → Save image, then `op attach <id> /path/to/file`
7. **"show ticket 123"** → `op show 123 --download --out=/tmp`, then read images

## Custom Field Values

### Components: android, ios, ott, engineering, analytics
### Products: eet, entd, djy, cntd, others
### Tech Areas: web, app, adtech, video, infra, portal, seo
### Labels: team#appios, team#appandroid, team#appall, team#web, ntd, seo

## Global Flags
- `-p, --project <id>` — Override default project
- `--sprint <name>` — Override default sprint
SKILL_EOF
  echo "    Installed /openproject skill to ${SKILL_DIR}"
else
  echo "    Claude Code not detected, skipping skill."
  echo "    To install later: mkdir -p ~/.claude/skills/openproject && copy SKILL.md from repo"
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
  echo ""
  echo "  Claude Code: /openproject create a bug for NTD+"
else
  echo ""
  echo "Binary installed but could not connect to OpenProject."
  echo "Check your API key in ~/.oprc"
fi
