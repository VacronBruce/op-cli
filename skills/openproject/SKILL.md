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
9a. **"prepare / set up next sprint"** → invoke /op:sprint-prepare skill (`--project`, `--tickets`, `--figma`, `--prd`, `--auto`)
9b. **"create a sprint"** → `op sprint create "<name>" --start=YYYY-MM-DD` (`--end` defaults to start+13 days)
9c. **"show high-priority backlog"** → `op backlog --priority p0,p1` (accepts P0–P3 and SEV0–SEV3 values)
9d. **"show backlog bugs"** → `op backlog --type bug` (combine with `--priority` to filter further)
10. **"close sprint / end of sprint"** → `op sprint close` for carryover report; for closing ready tickets → invoke /op:sprint-close skill (`-p`, `--sprint`, `--release`, `--status`, `--auto`)
10a. **"check/assign missing components"** → invoke /op:assign-components skill (`-p`, `--sprint`, `--dry-run`, `--auto`)
11. **"generate report"** → `op sprint progress -v`
12. **"is this ticket ready?"** → `op check <id>` (Definition-of-Ready gate: prints a deterministic completeness percent and READY / NEEDS WORK; a FAIL blocks, a WARN is advisory). Teams can tune which checks apply per type via a DoR config file pointed to by `OP_DOR_CONFIG` (see `references/commands.md`).
13. **"check sprint quality"** → `op check --sprint`
14. **"what's the discussion on X?"** → `op comment <id>`
15. **"leave a comment"** → `op comment <id> "message"`
15a. **"edit/fix my comment"** → `op comment <id>` to find the comment ID, then `op comment <id> "new text" --edit=<comment-id>`
16. **"list sprints"** → `op sprint list`
16a. **"list releases / versions"** → `op release list`
16b. **"create a release / version"** → `op release create "<name>"` (`--status=open|locked|closed`, `--start`, `--end`)
16c. **"set a ticket's release / assign to a version"** → `op update <id> --release="<name>"`
16d. **"change epic/parent/dates/product/label after creation"** → `op update <id> --epic="<name>"`, `--parent=<id>`, `--start`/`--due` (YYYY-MM-DD), `--product=<p>`, `--label=<l>` (same values as create)
16e. **"bulk update / move several tickets at once"** → `op update <id> <id> <id> --<flag>=...` (same change per ID; continues past failures and summarizes)
17. **"what version?"** → `op version`
18. **"update op"** → `op upgrade`
19. **"show blocked items"** → `op blocked` or `op board --status=blocked`
20. **"unestimated backlog"** → `op backlog --unestimated`
21. **"show ticket details"** → `op show <id>`
21a. **"get the ticket web link / browser URL / shareable link"** → `op show <id> --url` (prints only the URL, no API call)
22. **"what's the OP number for WP-23 / look up a JIRA ID"** → `op search <jira-id>` (maps the JIRA ID custom field to the OpenProject work package number); use `--field <name>` to search a different custom field (e.g. `op search AR-178 --field key` when tickets were renamed and the key is in a separate field); if the key only appears in historical activity (e.g. a renamed ticket's old key in a journal note), use `op search AR-178 --scan --project <id>` to scan activity journals
23. **"set parent / link tickets"** → `op link <id> --parent=X` (or `--relates-to`, `--blocks`, `--no-parent`)
23a. **"start work on / start a ticket"** → `op start <id>` (creates branch `<project>-<id>-<slug>`, moves it to In Progress, assigns to you; run inside the git repo)
23b. **"what's linked to this ticket / show relations"** → `op link <id> --list`
23c. **"remove a relation / unlink tickets"** → `op unlink <id> --relates-to=X` or `--blocks=X` (the relation ID is resolved for you)
24. **"review as PM"** → invoke /op:ticket-prep skill
25. **"verify as developer"** → invoke /op:ticket-verify skill
26. **"fully review / bot-review a ticket"** → invoke /op:ticket-review skill (combined PM + Dev, posts one comment)
26a. **"is this done / definition of done / can I close this"** → invoke /op:ticket-dod skill (Definition of Done exit gate; reports DONE / NOT DONE, does not change status)
27. **"what components/products/labels are valid?"** → `op fields` (overview) or `op fields component` (values)
28. **"remove an attachment"** → `op attach <id> --list` to find the attachment ID, then `op attach <id> --remove=<attachment-id>`
29. **"op isn't working / check my config"** → `op setup` (health checklist with fixes); **"new sprint started"** → `op setup --sprint="<name>"`

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
