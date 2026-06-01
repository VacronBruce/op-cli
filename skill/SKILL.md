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
op comment <id>                       # List comments on ticket
op comment <id> "message"             # Post a comment
```

### Sprint Management
```bash
op sprint list                        # List all sprints (ID, status, dates)
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

### Version & Upgrade
```bash
op version                            # Show current version
op upgrade                            # Self-update to latest release
```

### Quality Checks
```bash
op check <id>                         # Check ticket readiness
op check <id> --strict                # Treat warnings as failures
op check <id> --comment               # Post results to ticket
op check --sprint                     # Check all sprint tickets
op check --sprint --component=android # Filter + check
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
12. **"is this ticket ready?"** → `op check <id>`
13. **"check sprint quality"** → `op check --sprint`
14. **"what's the discussion on X?"** → `op comment <id>`
15. **"leave a comment"** → `op comment <id> "message"`
16. **"list sprints"** → `op sprint list`
17. **"what version?"** → `op version`
18. **"update op"** → `op upgrade`
19. **"review as PM"** → invoke /ticket-prep skill
20. **"verify as developer"** → invoke /ticket-verify skill

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
