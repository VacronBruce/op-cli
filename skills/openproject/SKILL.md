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

## Reference files

Consult these before constructing a command — don't guess flags or values:

- **`references/commands.md`** — full command + flag reference (board, my, create,
  update, link, comment, sprint, release, backlog, check, version) plus global flags
  and the no-project / user-story notes.
- **`references/custom-fields.md`** — valid values for `--component`, `--product`,
  `--tech-area`, `--label`, and `--priority`.

## How to Handle Requests

1. **"create a task/bug"** → `op create task/bug "subject" --flags`
2. **"show board"** → `op board`
3. **"what am I working on?"** → `op my`
3a. **"what did I create / file recently?"** → `op my --author --since=2w`
3b. **"what's on my plate across all projects / everything I own?"** → `op overview` (cross-project dashboard; `--projects`/`--sprints` to resize)
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
16a. **"list releases / versions"** → `op release list`
16b. **"create a release / version"** → `op release create "<name>"` (`--status=open|locked|closed`, `--start`, `--end`)
16c. **"set a ticket's release / assign to a version"** → `op update <id> --release="<name>"`
17. **"what version?"** → `op version`
18. **"update op"** → `op upgrade`
19. **"show blocked items"** → `op blocked` or `op board --status=blocked`
20. **"unestimated backlog"** → `op backlog --unestimated`
21. **"show ticket details"** → `op show <id>`
22. **"what's the OP number for WP-23 / look up a JIRA ID"** → `op search <jira-id>` (maps the JIRA ID custom field to the OpenProject work package number)
23. **"set parent / link tickets"** → `op link <id> --parent=X` (or `--relates-to`, `--blocks`, `--no-parent`)
23a. **"start work on / start a ticket"** → `op start <id>` (creates branch `<project>-<id>-<slug>`, moves it to In Progress, assigns to you; run inside the git repo)
24. **"review as PM"** → invoke /op:ticket-prep skill
25. **"verify as developer"** → invoke /op:ticket-verify skill
26. **"fully review / bot-review a ticket"** → invoke /op:ticket-review skill (combined PM + Dev, posts one comment)

For exact flags and field values on any of the above, read `references/commands.md`
and `references/custom-fields.md` first.

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
