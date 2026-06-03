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
op board --status=blocked             # Board filtered by status (matches "in-progress" → "In progress")
op board --component=android          # Board filtered by component (also --label=...)
op board --no-sprint                  # Open items across all sprints, grouped by sprint
op my                                 # My assigned items (current sprint)
op my --author --since=2w             # Items I created recently (2w/30d/3m; implies --no-sprint)
op my --by-sprint                     # Group my items by sprint
op my --component=android [--all]     # Filter by component (--all includes closed, --no-sprint drops the sprint filter)
op my team                            # Team items grouped by person
op blocked                            # Blocked items in sprint
op projects                           # List all projects
op show <id>                          # Work package details + attachments
op show <id> --download [--out=DIR]   # Download attachments (default: current dir)
op search <jira-id>                   # Map a JIRA ID (e.g. WP-23) to its OP number
```

> `op show` and `op check` read the **User Story** custom field (customField36)
> when present; `op check` counts a populated User Story field as satisfying the
> user-story requirement even if the description has no "As a…" text.

### Create & Update
```bash
op create <type> "<subject>" [flags]  # Create work package
  # Types: task, bug, feature, epic, user-story, milestone
  # Flags:
  #   --assignee="Name"    --priority=P1   (see priority values below)
  #   --epic="NTD+"        --component=android
  #   --product=entd       --tech-area=app
  #   --label=team#appandroid
  #   --points=5           --sprint="Sprint 1"
  #   --description="..."  --attach=screenshot.png
  #   --parent=81477       --start=2026-01-01   --due=2026-01-15

op update <id> [flags]                # Update work package
  # Flags: --status=in-progress --assignee="Name" --points=5 --done=80
  #        --sprint="Sprint 1" --component=android --subject="..."
  #        --priority=P1 --description="..."

op link <id> --parent=81477           # Set parent work package
op link <id> --no-parent              # Remove parent link
op link <id> --relates-to=81483       # Create "relates" relation
op link <id> --blocks=81485           # Create "blocks" relation

op attach <id> file.png [file2.jpg]   # Upload attachments
op comment <id>                       # List comments on ticket (shows comment IDs)
op comment <id> "message"             # Post a comment
op comment <id> "message" --edit=<comment-id>  # Edit an existing comment's text
```

> **Priority values** (use these, NOT the "Low/Normal/High" labels in `--help`):
> `P0`, `P1`, `P2`, `P3` (standard) and `SEV0`, `SEV1`, `SEV2`, `SEV3` (severity/bugs).

### Sprint Management
```bash
op sprint list                        # List all sprints (ID, status, dates)
op sprint add <id> [<id>...]          # Move items to active sprint
op sprint add <id> --sprint="Sprint 2" # Move items to a named sprint (e.g. carryover)
op sprint progress                    # Sprint progress summary (compact)
op sprint progress -v                 # Full sprint report for stakeholders
op sprint close                       # Sprint close summary + carryover list
```

### Backlog
```bash
op backlog                            # All items not in a sprint
op backlog --unestimated              # Unestimated items needing grooming
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
3a. **"what did I create / file recently?"** → `op my --author --since=2w`
4. **"prep for standup"** → Run `op my team` then `op blocked`, summarize
5. **"sprint progress"** → `op sprint progress`
6. **"assign X to Y"** → `op update <id> --assignee="Person"`
7. **"move to in progress"** → `op update <id> --status=in-progress`
8. **"attach this screenshot"** → Save image to file, then `op attach <id> /path/to/file` (see Attaching Images below)
9. **"plan next sprint"** → `op backlog` then `op sprint add`
10. **"close sprint"** → `op sprint close`
11. **"generate report"** → `op sprint progress -v`
12. **"is this ticket ready?"** → `op check <id>`
13. **"check sprint quality"** → `op check --sprint`
14. **"what's the discussion on X?"** → `op comment <id>`
15. **"leave a comment"** → `op comment <id> "message"`
15a. **"edit/fix my comment"** → `op comment <id>` to find the comment ID, then `op comment <id> "new text" --edit=<comment-id>`
16. **"list sprints"** → `op sprint list`
17. **"what version?"** → `op version`
18. **"update op"** → `op upgrade`
19. **"show blocked items"** → `op blocked` or `op board --status=blocked`
20. **"unestimated backlog"** → `op backlog --unestimated`
21. **"show ticket details"** → `op show <id>`
22. **"what's the OP number for WP-23 / look up a JIRA ID"** → `op search <jira-id>` (maps the JIRA ID custom field to the OpenProject work package number)
23. **"set parent / link tickets"** → `op link <id> --parent=X` (or `--relates-to`, `--blocks`, `--no-parent`)
24. **"review as PM"** → invoke /ticket-prep skill
25. **"verify as developer"** → invoke /ticket-verify skill
26. **"fully review / bot-review a ticket"** → invoke /ticket-review skill (combined PM + Dev, posts one comment)

## Custom Field Values

### Components: android, ios, ott, engineering, analytics
### Products: eet, entd, djy, cntd, others, competition
### Tech Areas: web, app, adtech, video, infra, portal, seo
### Labels: team#appios, team#appandroid, team#appall, team#web, ntd, seo, roku

> These are built-in defaults. Both the field key and the options are
> overridable per instance via a `custom_fields:` section in `~/.oprc` (see
> README). Values accept unique-prefix abbreviations (e.g. `--component=eng`),
> and shell completion (`op completion zsh|bash`) suggests them.

## Global Flags
- `-p, --project <id>` — Override default project
- `--sprint <name>` — Override default sprint

## Attaching Images

When the user provides an image (pasted or screenshot) to attach to a ticket:

1. Save the image to a file first (use `/tmp/` or current directory)
2. Then run `op attach <id> /path/to/saved-image.png`

**Container mode (`.op-bridge/`):** The bridge automatically handles file transfer.
When `op.sh attach` detects file arguments, it copies them to the shared `.op-bridge/`
directory so the host watcher can access them. No special handling needed — just call
`.op-bridge/op.sh attach <id> /path/to/file` and the bridge handles the rest.

## Output
Always show the raw CLI output to the user. Summarize if they asked a question rather than a command.
