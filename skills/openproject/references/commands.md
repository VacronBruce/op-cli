# `op` command reference

Full command and flag reference. For custom-field values (components, products,
tech-areas, labels, priority), see `references/custom-fields.md`.

## Daily operations
```bash
op board                              # Sprint board (kanban view)
op board --status=blocked             # Board filtered by status (matches "in-progress" → "In progress")
op board --component=android          # Board filtered by component (also --label=...)
op board --no-sprint                  # Open items across all sprints, grouped by sprint
op my                                 # All my open items, any sprint (current project; see note on no project)
op my --sprint="App_05/19/2026"       # Scope to one sprint
op my --author --since=2w             # Items I created recently (2w/30d/3m)
op my --by-sprint                     # Group my items by sprint
op my --component=android [--all]     # Filter by component (--all includes closed)
op my team                            # Team items grouped by person
op overview                           # Cross-project dashboard: my open work, top projects x sprints
op overview --projects=8 --sprints=5  # Widen the dashboard (defaults 5 x 3)
op blocked                            # Blocked items in sprint
op projects                           # List all projects
op show <id>                          # Work package details + attachments + inline comment images
op show <id> --url                    # Print only the browser URL (no API call)
op show <id> --download [--out=DIR]   # Download attachments AND inline comment images (default: current dir)
op search <jira-id>                   # Map a JIRA ID (e.g. WP-23) to its OP number
op start <id>                         # Start work: create/checkout branch <project>-<id>-<slug>,
                                      #   move ticket to In Progress, assign it to you (run in a git repo)
```

> With neither `-p` nor `OP_PROJECT` set, `op my` auto-detects
> the project and sprint holding most of your recent open work, shows it, and
> suggests `op overview` (all projects) or `op my -p <id> --sprint "<name>"`
> to broaden or pin the view. Set `project:` in `~/.oprc` to fix a default.

> `op show` and `op check` read the **User Story** custom field (customField36)
> when present. `op check` counts a populated User Story field as satisfying the
> user-story requirement even if the description has no "As a…" text.

> **Inline comment images:** OpenProject stores screenshots pasted into comments in
> `Activity::Comment` containers, so they do NOT appear in the work package's
> `/attachments` list. `op comment` renders them as `[image #ID: filename]`
> markers (instead of raw `<img>` HTML). `op show` lists them under "Inline
> images in comments" and includes them in `--download` (named `<id>-<filename>`).
> `op check` counts them toward the "Has attachments" rule, so it doesn't flag a bug/feature
> whose only screenshots live in a comment as having none.

## Create & update
```bash
op create <type> "<subject>" [flags]  # Create work package
  # Types: task, bug, feature, epic, user-story, milestone
  # Flags:
  #   --assignee="Name"    --priority=P1   (see custom-fields.md for priority values)
  #   --epic="NTD+"        --component=android
  #   --product=entd       --tech-area=app   (--product repeatable: --product=eet --product=entd)
  #   --label=team#appandroid
  #   --points=5           --sprint="Sprint 1"
  #   --estimate="2d"      (Work estimate: 2d, 16h, "2d 4h"; days at 8h/day)
  #   --description="..."  --attach=screenshot.png
  #   --parent=81477       --start=2026-01-01   --due=2026-01-15

op update <id> [flags]                # Update work package
  # Flags: --status=in-progress --assignee="Name" --points=5 --estimate="2d 4h" --done=80
  #        --sprint="Sprint 1" --component=android --subject="..."
  #        --priority=P1 --description="..." --user-story="..." --release="[iOS][ETV] 1.0.9"
  #        --to-project=wp   # move to another project (then assign a sprint)

op link <id> --parent=81477           # Set parent work package
op link <id> --no-parent              # Remove parent link
op link <id> --relates-to=81483       # Create "relates" relation
op link <id> --blocks=81485           # Create "blocks" relation
  # All OpenProject relation types are available as flags:
  # --relates-to --blocks --blocked-by --duplicates --duplicated-by
  # --precedes --follows --includes --part-of --requires --required-by
  # `op unlink` takes the same flags and matches in either direction.

op attach <id> file.png [file2.jpg]   # Upload attachments
op attach <id> --list                 # List attachments with their IDs
op attach <id> --remove=<attachment-id>  # Remove an attachment (must belong to <id>)
op comment <id>                       # List comments (shows comment IDs; inline images as [image #ID: file])
op comment <id> "message"             # Post a comment
op comment <id> "message" --edit=<comment-id>  # Edit an existing comment's text
```

> **Ticket URLs:** `op create`, `op update`, and `op comment` (post/edit) print the
> work package's browser URL (`<base>/work_packages/<id>`) after the confirmation
> line. Include it when you report the result, so the ticket is one click away.

## Sprint management
```bash
op sprint list                        # List all sprints (ID, status, dates)
op sprint create "<name>" --start=YYYY-MM-DD  # Create a sprint; --end defaults to start+13d
op sprint create "<name>" --start=2026-07-07 --end=2026-07-20  # explicit end date
op sprint add <id> [<id>...]          # Move items to active sprint
op sprint add <id> --sprint="Sprint 2" # Move items to a named sprint (e.g. carryover)
op sprint progress                    # Sprint progress summary (compact)
op sprint progress -v                 # Full sprint report for stakeholders
op sprint close                       # Sprint close summary + carryover list
```

## Releases / versions
```bash
op release list                       # List all releases (versions) for the project
op release create "<name>"            # Create a release, e.g. "[iOS][ETV] 1.0.9" (status open)
op release create "<name>" --status=locked      # open (default) | locked | closed
op release create "<name>" --start=2026-06-10 --end=2026-06-30  # optional date range
op release update <id-or-name> --name="<new>"   # Rename a release (id from `op release list`, or exact name)
op release update <id-or-name> --end=2026-06-25 # Reschedule; also --start, --status=open|locked|closed
op release show "<name>"              # The release + every ticket in it (any status), with status counts
op release show 2054                  # ...or by release ID
op release update "<name>" --description="..."  # Replace the release description (release notes)
op release update 2054 -d "$(cat whats-new.txt)" # Long notes from a file
op update <id> --release="<name>"     # Assign a work package to a release (resolved by name)
```

> A ticket's release is custom field 50, not the built-in version field (that one is
> the sprint), which is why `op board`/`op my` filters can't list a release. Use
> `op release show` instead. Release descriptions are **plain text** in OpenProject, so Markdown
> such as `## Heading` or `**bold**` shows literally.

> `--release` resolves the name against the project's existing releases (kind
> `release`). An unknown name fails with the list of available releases, and the
> flag tab-completes. `op release create` makes a *new* release.

## Backlog
```bash
op backlog                            # All items not in a sprint
op backlog --unestimated              # Unestimated items needing grooming
op backlog --priority p0,p1           # Filter by priority (accepts P0–P3 and SEV0–SEV3)
op backlog --type bug                 # Filter by work package type (e.g. bug, task)
op backlog --type bug --priority sev1,sev2  # Combine filters
```

## Version & upgrade
```bash
op version                            # Show current version
op upgrade                            # Self-update to latest release
```

## Quality checks
```bash
op check <id>                         # Definition-of-Ready gate for one ticket
op check <id> --strict                # Treat warnings as failures
op check <id> --comment               # Post results to ticket
op check --sprint                     # Check all sprint tickets
op check --sprint --component=android # Filter + check
op check --sprint --sprint-name="App_11/03/2026"  # A named sprint instead of the auto-detected one
```

`op check` is the **Definition-of-Ready gate**. It prints a deterministic
completeness score such as `Score: 5/8 (63%) — NEEDS WORK`. The percent is
`Pass=100, Warn=50, Fail=0` averaged (reproducible, not a guess), and the gate is
`READY` unless any check FAILs (a WARN is advisory and never blocks).

**Tuning the checks (`OP_DOR_CONFIG`).** By default each work-package type uses a
built-in check set. Point `OP_DOR_CONFIG` at a JSON file to add/drop checks per
type without a code change. Check IDs: `description`, `acceptance_criteria`,
`use_case`, `business_value`, `reproduction_steps`, `story_points`, `assignee`,
`priority`, `attachments`, `parent_epic`, `component`, `well_formed` (QUS role+means),
`atomic` (QUS one-feature-per-story; opt-in, advisory), `no_blockers` (INVEST
Independent, an advisory Warn when other work blocks the ticket; default for
bug/feature/task, not epic). Story/User Story fold onto
`feature`; the `""` key is the fallback for unknown types. An unknown ID, missing
file, or bad JSON fails loudly rather than silently dropping a check.
```json
{ "types": { "bug": ["description", "reproduction_steps", "story_points"],
             "feature": ["description", "acceptance_criteria", "well_formed", "atomic"] } }
```

> **Definition of Done** is the matching exit gate. Invoke /op:ticket-review
> --done before closing a ticket. Most DoD items (tests, review, deploy) live
> outside OpenProject, so it is a guided review, not an `op` subcommand.

## Setup & config
```bash
op setup                              # [ok]/[--] health check: config, key, connection, project, sprint, completion
op setup --sprint="<name>"            # update one ~/.oprc key in place (also --project, --api-key, --url)
```
`op setup` works without a valid config, so use it to diagnose config problems.

## Custom fields
```bash
op fields                             # List custom fields (name, field key, option count)
op fields component                   # List the allowed --component values
```
`op fields` shows the same registry the `--component`/`--product`/`--tech-area`/`--label`/
`--jira-id` flags resolve against, including `~/.oprc` overrides. It works without
URL/API-key config.

## Global flags
- `-p, --project <id>`: override the default project
- `--sprint <name>` is not global: `board`, `my`, `my team`, `sprint add`, `create` and
  `update` take it (`check --sprint` uses `--sprint-name`). `sprint progress`, `blocked`
  and `sprint close` have no sprint flag and always use the first open version whose
  dates cover today.
