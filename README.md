# op-cli

A lean CLI for managing OpenProject sprints, backlogs, and work packages. Built for team leads.

## Install

### Prerequisites

Install `glab` (GitLab CLI) if you don't have it:

```bash
# macOS
brew install glab

# Linux
brew install glab
# or: sudo apt install glab
```

Authenticate to our GitLab (one-time):

```bash
GITLAB_HOST=gitlab-tw.ddns.net glab auth login
```

### Quick Install (recommended)

```bash
mkdir -p /tmp/op-cli && cd /tmp/op-cli && GITLAB_HOST=gitlab-tw.ddns.net glab release download --repo gmedtn/op-cli --include-external --asset-name="install.sh" && bash install.sh
```

The script will:
1. Auto-detect your platform (macOS/Linux, ARM/Intel)
2. Download the correct binary via glab
3. Ask for your OpenProject API key
4. Install Claude Code `/openproject` skill

### Alternative: Clone + Build (needs Go)

```bash
git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git
cd op-cli && git checkout develop && bash install.sh
```

### Alternative: curl + GitLab Token

```bash
curl -fsSH "PRIVATE-TOKEN: your-token" \
  "https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/latest/install.sh" | bash
```

## Setup

Your OpenProject API key is required. Get it from:

1. Log in to https://openpr.epochbase.com
2. Go to **My Account** > **Access Tokens**
3. Create a new API token

The install script creates `~/.oprc` automatically. To edit manually:

```yaml
url: https://openpr.epochbase.com
api_key: your-api-key-here
project: app
sprint: "App_05/19/2026"
```

Verify:

```bash
op projects
```

## Usage

### Daily commands

```bash
op board                           # Current sprint board (kanban view)
op my                              # My assigned items
op my-team                         # Team items grouped by person
op blocked                         # Blocked items in sprint
op show 12345                      # View ticket details
op show 12345 --download           # Download attachments
```

### Create & update

```bash
op create task "Fix login page" --assignee="Ken Peng" --priority=high
op create bug "Crash on save" --priority=immediate \
  --epic="NTD+" --component=android --product=entd \
  --tech-area=app --label=team#appandroid --attach=screenshot.png
op update 12345 --status=in-progress
op assign 12345 "Ken Peng"
op attach 12345 screenshot.png
```

### Sprint management

```bash
op sprint plan                     # Show backlog items for planning
op sprint add 101 102 103          # Move items to current sprint
op sprint progress                 # Sprint progress summary
op sprint close                    # Sprint close summary
```

### Backlog & reporting

```bash
op backlog                         # Items not in any sprint
op backlog groom                   # Unestimated items
op report                          # Sprint report for stakeholders
op projects                        # List all projects
```

### Global flags

```
-p, --project <id>    Override default project (e.g. -p web, -p bug, -p app)
--sprint <name>       Override default sprint
-h, --help            Help for any command
```

### Claude Code (Docker / container mode)

When Claude Code runs inside a container, it cannot execute `op` directly and does not
have access to host-side `~/.claude/skills/`. Two things are needed: installing the skill
into the container, and bridging `op` commands back to the host.

#### 1. Install the skill into the container

The container's `~/.claude/` directory is mounted from the **project root**'s `.claude/`
folder. Copy the skill there so the containerized Claude Code can see it:

```bash
# From the project root (run on host, one-time setup)
mkdir -p .claude/skills/openproject
cp ~/.claude/skills/openproject/SKILL.md .claude/skills/openproject/SKILL.md
```

> **Tip:** `.claude/` in the project root is already gitignored. If it isn't, add it.

After this, `/openproject` will be available inside the container.

#### 2. Start the host bridge

The `.op-bridge/` scripts relay `op` commands from the container to the host binary
via shared files:

1. Start the watcher **on the host** (once per session):
   ```bash
   bash .op-bridge/host-watcher.sh
   ```
2. Inside the container, call `op` via the bridge:
   ```bash
   .op-bridge/op.sh show 123
   .op-bridge/op.sh update 456 --status=in-progress
   ```

The watcher reads requests from `.op-bridge/request.txt`, runs them against the host `op`
binary, and writes the result to `.op-bridge/result.txt`.

### Claude Code (skill)

Use `/openproject` in Claude Code for natural language access:

```
/openproject create a high priority bug for NTD+, assign to Bruce
/openproject show the sprint board
/openproject what's blocked?
```

## Troubleshooting

**"missing config: set OP_URL and OP_API_KEY"**
Create `~/.oprc` or set environment variables (`OP_URL`, `OP_API_KEY`).

**"no active sprint found"**
The project has no open version. Use `--sprint="Name"` to specify one.

**"unknown type/status/priority"**
Names are case-insensitive with prefix match. The error message shows available options.
