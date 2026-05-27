# op-cli

A lean CLI for managing OpenProject sprints, backlogs, and work packages. Built for team leads.

## Install

### Option 1: Download binary

Download from the [Releases](../../-/releases) page, then:

```bash
chmod +x op-*
sudo mv op-* /usr/local/bin/op
```

### Option 2: Build from source

```bash
git clone git@gitlab-tw.ddns.net:gmedtn/op-cli.git
cd op-cli
go build -o op .
sudo mv op /usr/local/bin/
```

## Setup

### 1. Get your API key

1. Log in to OpenProject: https://openpr.epochbase.com
2. Go to **My Account** > **Access Tokens**
3. Create a new API token

### 2. Create config file

```bash
cp .oprc.example ~/.oprc
```

Edit `~/.oprc`:

```yaml
url: https://openpr.epochbase.com
api_key: your-api-key-here
project: web  # optional default project
```

Or use environment variables:

```bash
export OP_URL=https://openpr.epochbase.com
export OP_API_KEY=your-api-key-here
export OP_PROJECT=web
```

### 3. Verify

```bash
op projects
```

## Usage

### Daily commands

```bash
op board                           # Current sprint board (kanban view)
op board -p web                    # Board for specific project
op my                              # My assigned items
op my-team                         # Team items grouped by person
op blocked                         # Blocked items in sprint
```

### Create & update

```bash
op create task "Fix login page" --assignee="Ken Peng" --priority=high
op create bug "Crash on save" --priority=immediate
op update 12345 --status=in-progress
op update 12345 --assignee="Ken Peng" --points=5 --done=80
op assign 12345 "Ken Peng"
```

### Sprint management

```bash
op sprint plan                     # Show backlog items for planning
op sprint add 101 102 103          # Move items to current sprint
op sprint add 101 --points=5       # Add with story points
op sprint progress                 # Sprint progress summary
op sprint close                    # Sprint close summary
```

### Backlog & reporting

```bash
op backlog                         # Items not in any sprint
op backlog groom                   # Unestimated items
op report                          # Sprint report for stakeholders
```

### Global flags

```
-p, --project <id>    Override default project (e.g. -p web, -p bug, -p app)
-h, --help            Help for any command
```

## Troubleshooting

**"missing config: set OP_URL and OP_API_KEY"**
Create `~/.oprc` or set environment variables.

**"no active sprint found"**
The project has no open version. Use `--sprint="Name"` to specify one.

**"unknown type/status/priority"**
Names are case-insensitive with prefix match. The error message shows available options.

## Requirements

- Go 1.22+ (to build from source only)
- OpenProject account with API token
