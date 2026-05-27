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
  #   --start=2026-01-01   --due=2026-01-15

op update <id> [flags]                # Update work package
  # Flags: --status=in-progress --assignee="Name" --points=5 --done=80

op assign <id> "Person Name"          # Quick reassign
op attach <id> file.png [file2.jpg]   # Upload attachments
```

### Sprint Management
```bash
op sprint plan                        # Show backlog items for planning
op sprint add <id> [<id>...]          # Move items to sprint
op sprint progress                    # Sprint progress summary
op sprint close                       # Sprint close summary + carryover list
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
5. **"sprint progress"** → `op sprint progress`
6. **"assign X to Y"** → `op assign <id> "Person"`
7. **"move to in progress"** → `op update <id> --status=in-progress`
8. **"attach this screenshot"** → Save image to temp file, then `op attach <id> /path/to/file`
9. **"plan next sprint"** → `op sprint plan` then `op sprint add`
10. **"close sprint"** → `op sprint close`
11. **"generate report"** → `op report`

## Custom Field Values

### Components: android, ios, ott, engineering, analytics
### Products: eet, entd, djy, cntd, others, competition
### Tech Areas: web, app, adtech, video, infra, portal, seo
### Labels: team#appios, team#appandroid, team#appall, team#web, ntd, seo, roku

## Global Flags
- `-p, --project <id>` — Override default project
- `--sprint <name>` — Override default sprint

## Output
Always show the raw CLI output to the user. Summarize if they asked a question rather than a command.
