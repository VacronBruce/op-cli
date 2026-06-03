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
sprint: "App_06/02/2026"    # Update this each sprint
```

### Custom fields (optional)

The `--component`, `--product`, `--tech-area`, and `--label` flags map to
instance-specific custom fields. Built-in defaults target this OpenProject
instance; override them — or add your own — under `custom_fields:` in `~/.oprc`.
Both the field key and the option set are configurable:

```yaml
custom_fields:
  component:                        # one of: component, product, tech-area, label
    field: customField12            # the OpenProject field key for this instance
    options:
      android: /api/v3/custom_options/42
      ios:     /api/v3/custom_options/43
  label:
    field: customField13
    options:
      mobile: /api/v3/custom_options/460
```

A field you don't list keeps its built-in default; listing one overrides its
field key and/or replaces its options. Look up option hrefs via the OpenProject
API (e.g. `/api/v3/custom_fields`).

Verify:

```bash
op projects
```

## Usage

### Daily commands

```bash
op board                           # Current sprint board (kanban view)
op board --status=blocked          # Board filtered by status
op board --component=android       # Board filtered by component
op my                              # My assigned items (current project + sprint)
op my --no-sprint                  # All my items (no sprint filter)
op my --author --since=2w          # Items I created in last 2 weeks
op my --component=android          # My Android items only
op my --by-sprint                  # My items grouped by sprint
op my team                         # Team items grouped by person
op overview                        # Cross-project dashboard of my open work
op overview --projects=8 --sprints=5  # Widen the dashboard
op blocked                         # Blocked items in sprint
op show 12345                      # View ticket details (includes JIRA ID + User Story)
op show 12345 --download           # Download attachments
op search WP-23                    # Map a JIRA ID to its OpenProject number
```

> **No project set?** With neither `-p` nor `OP_PROJECT`, `op my` auto-detects the
> project + sprint where most of your recent open work lives and shows it, then
> points you at `op overview` (everything) or `op my -p <id> --sprint "<name>"`.

### Create & update

```bash
op create task "Fix login page" --assignee="Ken Peng" --priority=P1
op create bug "Crash on save" --priority=SEV1 \
  --epic="NTD+" --component=android --product=entd \
  --tech-area=app --label=team#appandroid --attach=screenshot.png
op create feature "Dark mode" --points=3 --sprint="App_06/02/2026"
op update 12345 --status=in-progress
op update 12345 --assignee="Bruce Chen" --points=3
op update 12345 --description="Updated description here"
op update 12345 --subject="New title" --done=50
op update 12345 --assignee="Ken Peng"
op attach 12345 screenshot.png
```

Priority values: `P0`, `P1`, `P2`, `P3` (tasks/stories) | `SEV0`, `SEV1`, `SEV2`, `SEV3` (bugs)

### Comments

```bash
op comment 12345                   # List all comments (shows comment IDs)
op comment 12345 "LGTM"           # Post a comment
op comment 12345 "fixed typo" --edit=6789  # Edit comment #6789
```

### Links & relations

```bash
op link 81482 --parent=81477       # Set parent work package
op link 81482 --no-parent          # Remove parent link
op link 81482 --relates-to=81483   # Create a "relates" relation
op link 81482 --blocks=81485       # Create a "blocks" relation
```

### Sprint management

```bash
op sprint list                     # List all sprints in the project
op sprint add 101 102 103          # Move items to current sprint
op sprint add 101 --sprint="App_06/02/2026"  # Move to specific sprint
op sprint progress                 # Sprint progress summary (compact)
op sprint progress -v              # Full sprint report for stakeholders
op sprint close                    # Sprint close summary
```

### Backlog

```bash
op backlog                         # Items not in any sprint
op backlog --unestimated           # Unestimated items needing grooming
```

### Quality checks

```bash
op check 12345                     # Check ticket readiness
op check 12345 --strict            # Treat warnings as failures
op check 12345 --comment           # Post check results to ticket
op check --sprint                  # Check all tickets in current sprint
op check --sprint --component=android  # Check android tickets only
```

### Project & CLI info

```bash
op projects                        # List all accessible projects
op version                         # Print CLI version
op upgrade                         # Upgrade to latest version
```

### Global flags

```
-p, --project <id>    Override default project (e.g. -p web, -p bug, -p app)
--sprint <name>       Override default sprint
-h, --help            Help for any command
```

### Shell completion

`op` ships completion for all commands, and the enum flags complete their
values (honoring any `custom_fields:` overrides in `~/.oprc`):

```bash
source <(op completion zsh)     # add to ~/.zshrc
source <(op completion bash)    # add to ~/.bashrc
```

Then `op create --component <TAB>` suggests `android  ios  ott  engineering  analytics`
(likewise `--product`, `--tech-area`, `--label`).

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

### Claude Code skills

Three slash commands are available in Claude Code for natural language access:

**`/openproject`** — Translates natural language into `op` commands:

```
/openproject create a SEV1 bug "Crash on save" for NTD+, assign to Bruce, android component
/openproject show the sprint board filtered by blocked status
/openproject what's blocked in the current sprint?
/openproject show my team's work for this sprint
/openproject generate the sprint report
/openproject add tickets 101 102 103 to current sprint
/openproject what's in the backlog that needs estimation?
```

**`/ticket-prep`** — PM self-review for ticket quality before business review:

```
/ticket-prep 12345
```

Checks: completeness, clarity, business justification, acceptance criteria quality, visual assets, and scope definition. Outputs a structured review with rewrite suggestions.

**`/ticket-verify`** — Developer readiness check before starting implementation:

```
/ticket-verify 12345
```

Checks: implementability, technical gaps, ambiguities, dependencies, risk assessment, and estimation sanity. Detects team context (android/ios/web) for team-specific checks.

## Troubleshooting

**"missing config: set OP_URL and OP_API_KEY"**
Create `~/.oprc` or set environment variables (`OP_URL`, `OP_API_KEY`).

**"no active sprint found"**
The project has no open version. Use `--sprint="Name"` to specify one.

**"unknown type/status/priority"**
Names are case-insensitive with prefix match. The error message shows available options.

**"Version filter has invalid values"**
The default sprint in `~/.oprc` may be stale. Update the `sprint:` field or use `--sprint` flag. Run `op sprint list` to see available sprints.
