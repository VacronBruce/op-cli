# op-cli

A lean CLI for managing OpenProject sprints, backlogs, and work packages. Built for team leads.

## Install

### Quick Install (recommended)

```bash
bash <(curl -fsSL https://github.com/VacronBruce/op-cli/releases/latest/download/install.sh)
```

The repo is public, so this needs no login or token. The script will:
1. Auto-detect your platform (macOS/Linux, ARM/Intel)
2. Download the correct binary from the latest GitHub release
3. Ask for your OpenProject API key
4. Install the `op` Claude Code plugin (`/op:openproject`, `/op:sprint-report`, `/op:file-bug`, `/op:ticket-review`, …)

**Windows (PowerShell):** no Git Bash needed — run this in PowerShell:

```powershell
irm https://github.com/VacronBruce/op-cli/releases/latest/download/install.ps1 | iex
```

It downloads `op.exe`, adds it to your PATH, and sets up config + completion.
(Git Bash / WSL users can use the `curl` one-liner above instead.)

**Updating:** existing users re-run the same one-liner (refreshes the binary **and**
the plugin), or run `op upgrade` for just the binary. Already on the plugin?
`claude plugin update op` pulls the latest skills.

### Alternative: Clone + Build (needs Go)

```bash
git clone https://github.com/VacronBruce/op-cli.git
cd op-cli && bash install.sh
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

### Ticket templates (optional)

`op create <type>` fills the description from a `templates.<type>` block in
`~/.oprc` when you don't pass `--description`, so every bug/feature starts with
the right scaffold. The key is the lowercased type name.

```yaml
templates:
  bug: |
    ## Steps to reproduce
    1.
    ## Expected

    ## Actual

    ## Acceptance criteria
    - [ ]
  feature: |
    ## User story
    As a <role>, I want <capability> so that <benefit>.

    ## Acceptance criteria
    - [ ]
  task: |
    ## Goal

    ## Acceptance criteria
    - [ ]
```

### Verify & maintain

`op setup` prints an `[ok]`/`[--]` checklist (config, credentials, connection,
project, sprint, completion) with the exact fix for anything missing, and its
flags update single keys in `~/.oprc` without touching the rest of the file:

```bash
op setup                              # health check — run anytime
op setup --sprint="App_06/15/2026"    # rotate the sprint each cycle
op setup --project=app                # change default project
op setup --api-key=<key>              # store a new API key
```

## Usage

### Daily commands

```bash
op board                           # Current sprint board (kanban view)
op board --status=blocked          # Board filtered by status
op board --component=android       # Board filtered by component
op my                              # All my open items (any sprint, current project)
op my --sprint="App_05/19/2026"    # Scope to one sprint
op my --author --since=2w          # Items I created in last 2 weeks
op my --component=android          # My Android items only
op my --by-sprint                  # My items grouped by sprint
op my team                         # Team items grouped by person
op overview                        # Cross-project dashboard of my open work
op overview --projects=8 --sprints=5  # Widen the dashboard
op blocked                         # Blocked items in sprint
op show 12345                      # View ticket details (includes JIRA ID + User Story + Release + relations + inline comment images)
op show 12345 12346 --brief        # One line per ticket: #id [status] subject | assignee | sprint | release
op show 12345 --json               # Raw API JSON (an array for several IDs)
op show 12345 --url                # Print only the browser URL (no API call)
op show 12345 --download           # Download attachments and inline comment images
op search WP-23                    # Map a JIRA ID to its OpenProject number
op search AR-178 --field key       # Search a different custom field (substring match)
op search AR-178 --scan --project app  # Scan activity journals when a key only exists in history
op search --text "paywall"         # Subject search across projects (add -p app to scope)
op activity 12345                  # Full history: status/field changes and comments (aliases: activities, history)
op activity 12345 --description    # How the description changed, before/after per edit (needs web_user; add --json for scripts)
op api statuses                    # Call any API endpoint with your configured key (GET by default)
op start 12345                     # Start work: branch <project>-12345-<slug>, In Progress, assign to you
```

> **No project set?** With neither `-p` nor `OP_PROJECT`, `op my` auto-detects the
> project + sprint where most of your recent open work lives and shows it, then
> points you at `op overview` (everything) or `op my -p <id> --sprint "<name>"`.

### Create & update

> `op create bug` files to the Bug Backlog board (`bug`) by default, with **no sprint** (the `.oprc`/`OP_SPRINT` sprint belongs to the ambient project and is not applied — triage assigns one). Pass `-p <board>` to file it elsewhere, or `--sprint="<name>"` to place it on a bug-board sprint.

```bash
op create task "Fix login page" --assignee="Ken Peng" --priority=P1
op create bug "Crash on save" --priority=SEV1 --attach=screenshot.png   # -> Bug Backlog board
op create feature "Dark mode" --points=3 --estimate="2d" --sprint="App_06/02/2026" \
  --component=android --product=entd --tech-area=app --label=team#appandroid
op update 12345 --status=in-progress
op update 12345 --assignee="Bruce Chen" --points=3
op update 12345 --estimate="2d 4h"             # set Work estimate (days at 8h/day)
op update 12345 --description="Updated description here"  # refuses to drop inline images; --force to override
op update 12345 --user-story="As a reader, I want X so that Y"  # edit the User Story field
op update 12345 --subject="New title" --done=50
op update 12345 --assignee="Ken Peng"
op update 12345 --release="[iOS][ETV] 1.0.9"   # assign to a release (see Releases below)
op update 12345 --to-project=wp                # move to another project (then assign a sprint)
op update 12345 --epic="NTD+" --parent=12000   # re-parent / link to an epic after creation
op update 12345 --start=2026-07-01 --due=2026-07-15
op update 12345 --product=entd --label=team#appandroid
op update 101 102 103 --status=done            # bulk: same change applied to every ID
op attach 12345 screenshot.png
op attach 12345 --list                         # list attachments with their IDs
op attach 12345 --get=318 -o /tmp              # download attachment #318 as /tmp/318-<name> (inline comment images too)
op attach 12345 --remove=318                   # remove attachment #318 (must belong to #12345)
```

`--product` is repeatable to tag multiple products: `--product=eet --product=entd`.

Priority values: `P0`, `P1`, `P2`, `P3` (tasks/stories) | `SEV0`, `SEV1`, `SEV2`, `SEV3` (bugs)

### Comments

```bash
op comment 12345                   # List all comments (shows comment IDs; inline images as [image #ID: file])
op comment list 12345              # Same as above (alias: op comments 12345)
op comment 12345 "LGTM"           # Post a comment
op comment 12345 "fixed typo" --edit=6789  # Edit comment #6789
```

### Links & relations

```bash
op link 81482 --parent=81477       # Set parent work package
op link 81482 --no-parent          # Remove parent link
op link 81482 --relates-to=81483   # Create a "relates" relation
op link 81482 --blocks=81485       # Create a "blocks" relation
op link 81482 --list               # List existing relations (op show lists them too)
op unlink 81482 --relates-to=81483 # Remove the "relates" relation to #81483
op unlink 81482 --blocks=81485     # Remove the "blocks" relation to #81485
```

All OpenProject relation types are available as flags on both `op link` and
`op unlink`: `--relates-to`, `--blocks`, `--blocked-by`, `--duplicates`,
`--duplicated-by`, `--precedes`, `--follows`, `--includes`, `--part-of`,
`--requires`, `--required-by`. `op unlink` matches the relation in either
direction (e.g. `--blocked-by` also removes a relation stored as "blocks"
from the other side).

### Sprint management

```bash
op sprint list                     # List all sprints in the project
op sprint create "Sprint 2026-07-07" --start=2026-07-07  # Create sprint (--end defaults to start+13d)
op sprint create "Sprint 2026-07-07" --start=2026-07-07 --end=2026-07-20  # explicit end
op sprint add 101 102 103          # Move items to current sprint
op sprint add 101 --sprint="App_06/02/2026"  # Move to specific sprint
op sprint progress                 # Sprint progress summary (compact)
op sprint progress -v              # Full sprint report for stakeholders
op sprint close                    # Sprint close summary
```

### Releases

```bash
op release list                    # List all releases (versions) for the project
op release create "[iOS][ETV] 1.0.9"             # Create a release (status open)
op release create "[iOS][EET] 3.2.0" --status=locked   # open (default) | locked | closed
op release create "v2.0" --start=2026-06-10 --end=2026-06-30  # optional date range
op release update 1821 --name="[iOS][EET] 3.2.1" --end=2026-06-25  # rename / reschedule (id or exact name)
op release show "[Android][EET] 3.3.1"           # Every ticket in the release (any status) + its notes
op release update "[Android][EET] 3.3.1" --description="$(cat whats-new.txt)"  # Write the release notes
op update 12345 --release="[iOS][ETV] 1.0.9"     # Assign a ticket to a release
```

Release descriptions are plain text in OpenProject, so write release notes without Markdown.

`--release` resolves the name against existing releases (and tab-completes); an
unknown name fails with the list of available releases.

### Backlog

```bash
op backlog                         # Items not in any sprint
op backlog --unestimated           # Unestimated items needing grooming
op backlog --priority p0,p1        # Filter by priority (accepts P0–P3 and SEV0–SEV3)
op backlog --type bug              # Filter by type (e.g. bug, task, feature)
op backlog --type bug --priority sev1,sev2  # Combine filters
```

### Quality checks

```bash
op check 12345                     # Check ticket readiness
op check 12345 --strict            # Treat warnings as failures
op check 12345 --comment           # Post check results to ticket
op check --sprint                  # Check all tickets in current sprint
op check --sprint --component=android  # Check android tickets only
```

`op check` is the **Definition-of-Ready gate**. It prints a deterministic score —
`Score: 5/8 (63%) — NEEDS WORK` — where the percent is `Pass=100, Warn=50, Fail=0`
averaged (reproducible, not a model guess) and the gate reads `READY` unless a
check FAILs (a WARN is advisory). Tune which checks apply per work-package type
with a JSON file pointed to by `OP_DOR_CONFIG` — including the opt-in QUS checks
`well_formed` and `atomic`. See `skills/openproject/references/commands.md` for the
check IDs and config format.

### Project & CLI info

```bash
op projects                        # List all accessible projects
op fields                          # List custom fields (component, product, ...)
op fields component                # List the allowed --component values
op version                         # Print CLI version (same: op --version)
op upgrade                         # Upgrade to latest version
```

When a newer release is out, a failed command and `op version` end with
`op-cli vX is out (this is Y); update with: op upgrade` on stderr. op checks GitHub
at most once a day; set `OP_NO_UPDATE_CHECK=1` to turn it off.

### Global flags

```
-p, --project <id>    Override default project (e.g. -p web, -p bug, -p app)
--sprint <name>       Override default sprint
-h, --help            Help for any command
```

### Shell completion

`op` ships completion for all commands, and the enum flags complete their
values (honoring any `custom_fields:` overrides in `~/.oprc`):

`install.sh` enables this automatically for your shell (zsh/bash). To set it up
manually, or for other shells:

```bash
source <(op completion zsh)     # add to ~/.zshrc
source <(op completion bash)    # add to ~/.bashrc
```

Then `op create --component <TAB>` suggests `android  ios  ott  engineering  analytics`
(likewise `--product`, `--tech-area`, `--label`).

### Claude Code (Docker / container mode)

When Claude Code runs inside a container, it cannot execute `op` directly and the host's
`op` plugin isn't visible. Two things are needed: installing the skill into the container
as a loose skill, and bridging `op` commands back to the host.

#### 1. Install the skill into the container

The container's `~/.claude/` directory is mounted from the **project root**'s `.claude/`
folder. Copy the skill there (from a clone of this repo) so the containerized Claude Code
can see it:

```bash
# From the project root (run on host, one-time setup); adjust the path to your op-cli clone
mkdir -p .claude/skills/openproject
cp /path/to/op-cli/skills/openproject/SKILL.md .claude/skills/openproject/SKILL.md
```

> **Tip:** `.claude/` in the project root is already gitignored. If it isn't, add it.

After this, `/openproject` will be available inside the container (loose skill — no `op:`
prefix, since the plugin isn't installed in the container).

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

The skills ship as the **`op` plugin** — namespaced under an `op:` prefix
(`/op:openproject`, `/op:sprint-report`, …) so they never collide with other skills.
`install.sh` registers the marketplace and installs the plugin for you; to do it
by hand:

```
/plugin marketplace add https://github.com/VacronBruce/op-cli.git
/plugin install op@op
```

Slash commands available in Claude Code for natural language access:

**`/op:openproject`** — Translates natural language into `op` commands:

```
/op:openproject create a SEV1 bug "Crash on save" for NTD+, assign to Bruce, android component
/op:openproject show the sprint board filtered by blocked status
/op:openproject what's blocked in the current sprint?
/op:openproject show my team's work for this sprint
/op:openproject generate the sprint report
/op:openproject add tickets 101 102 103 to current sprint
/op:openproject what's in the backlog that needs estimation?
```

**`/op:ticket-review`** — Review a ticket through one or more lenses and print the result:

```
/op:ticket-review 12345            # PM + developer readiness (default)
/op:ticket-review 12345 --pm       # PM self-review before business review
/op:ticket-review 12345 --dev      # developer readiness before coding
/op:ticket-review 12345 --start    # start now? worth doing, parent spec, other repos, designer, tier
/op:ticket-review 12345 --done     # Definition of Done before closing
/op:ticket-review WP-23            # a JIRA ID is resolved first
```

The PM lens checks completeness, clarity, business justification, acceptance criteria, visual assets and scope, with rewrite suggestions. The developer lens checks implementability, technical gaps, ambiguities, dependencies, risk and estimation, plus team-specific checks (android/ios/web). The start lens runs before `op start`: it reads the parent, siblings and related tickets to judge whether the work is still wanted, whether the real spec lives elsewhere, which other repos to search (from an optional repo map at `$OP_REPO_MAP` or `~/.config/op/repo-map.md`), whether a designer must look, and which tier of process the ticket needs; PM questions are drafted, never posted. The DoD lens mirrors the `op check` Definition-of-Ready gate and marks anything the ticket can't prove `UNKNOWN — confirm`; it never changes ticket status. The review is printed; `--post` posts the combined PM + Dev review as one bot comment (used by the op-agent reviewer daemon).

**`/op:ticket-refine`** — Interactive refinement that writes the improved fields back:

```
/op:ticket-refine 12345
```

Unlike `/op:ticket-review`, this one *fixes* the ticket. It finds weak or missing acceptance criteria and user story, talks through each gap one question at a time, shows a before/after, and — only after you confirm — writes the corrected description (AC) and User Story field back via `op update`. Posts no comment; the field history records the change.

**`/op:sprint-report`** — Standup digest, or a weekly stakeholder report:

```
/op:sprint-report
/op:sprint-report -p web --sprint "Web - 11/02/2026"
/op:sprint-report --weekly -p app -p web --component android
```

The default is a 15-second standup briefing (progress, blockers, work by person, risks). `--weekly` prints one copy-paste block per project (shipped / in progress / blocked / at risk / next). `--component` narrows either to one component.

**`/op:file-bug`** — Guided bug filing:

```
/op:file-bug CC button crashes when publishing on Android
```

Collects repro/expected/actual/acceptance criteria + the right component/product/label, shows the complete final ticket, and creates it only after you approve that exact content.

**`/op:sprint-prepare`** — Full next-sprint preparation for dev leads:

```
/op:sprint-prepare --project app --tickets 123,456 --prd ./prd.md
/op:sprint-prepare --project app --figma https://figma.com/... --auto
```

Finds or creates the next sprint under the project's naming convention (`App_MM/DD/YYYY`, `Web - MM/DD/YYYY`), auto-moves P0/P1 and Sev1/Sev2 backlog items, drafts tickets from IDs / PRD / Figma, and produces a summary. Pauses between phases (`--auto` skips the pauses); new tickets always need your approval of the intent and of the exact content.

**`/op:sprint-triage`** — Tag or dispatch a sprint's tickets:

```
/op:sprint-triage components --dry-run
/op:sprint-triage components --sprint "App_06/02/2026" --auto
/op:sprint-triage assignees --component android
```

`components` infers a missing Component from labels (`team#appandroid` → android, `team#appall` → android + ios, `roku` → ott, `team#web` → engineering), then title keywords, then type. `assignees` spreads unassigned tickets across the `team:` roster in `~/.oprc` (see `.oprc.example`) by current load, including teammates with no tickets yet. Both show a plan before writing; `--dry-run` only shows it.

**`/op:sprint-close`** — End-of-sprint workflow: give each "Ready for Release" ticket its product's release and move it to Done:

```
/op:sprint-close
/op:sprint-close --sprint "App_10/20/2026"
/op:sprint-close --release "[iOS][ETV] 1.2.0" --auto
```

Matches each ticket's `[Platform][Product]` tag to the open release with that prefix (e.g. `[Android][DJY] …`), confirms the grouped plan, then sets `--release` and `--status=done` in one update per release. `--release` forces one release for all; `--status` overrides the default "ready for release" filter.

### What changed in plugin 0.28.0

`op show` lists relations, `op attach --get` downloads one attachment, reads that
get no response are retried, a wrong project identifier names the closest real
ones, `op view` and `op project` work, and op says when a newer release is out.

### What changed in plugin 0.27.0

`op activity`, `op show --brief/--json` with several IDs, `op search --text`, `op api`,
`op comment list`, `op --version`, and real author names in `op comment` output.
The `openproject` skill now routes history, multi-ticket lookups, scripting and raw
API calls to these instead of curl or `op show | grep`.

### Skill renames in plugin 0.26.0

Fourteen skills became eight. Old commands map to:

| Before | Now |
|---|---|
| `/op:ticket-prep <id>` | `/op:ticket-review <id> --pm` |
| `/op:ticket-verify <id>` | `/op:ticket-review <id> --dev` |
| `/op:ticket-dod <id>` | `/op:ticket-review <id> --done` |
| `/op:ticket-review <id>` (posted) | `/op:ticket-review <id> --post` — without `--post` it now only prints |
| `/op:standup` | `/op:sprint-report` |
| `/op:weekly-report -p …` | `/op:sprint-report --weekly -p …` |
| `/op:assign-components` | `/op:sprint-triage components` |
| `/op:android-dispatch` | `/op:sprint-triage assignees` (needs a `team:` roster in `~/.oprc`) |
| `/op:op-bridge` | read by `/op:openproject` automatically when `.op-bridge/` exists |

`/op:openproject`, `/op:file-bug`, `/op:ticket-refine`, `/op:sprint-prepare` and `/op:sprint-close` keep their names. Re-run `install.sh` (or `claude plugin marketplace update op` then `claude plugin install op@op`) to pick them up.

## Troubleshooting

**"missing config: set OP_URL and OP_API_KEY"**
Create `~/.oprc` or set environment variables (`OP_URL`, `OP_API_KEY`).

**"no active sprint found"**
The project has no open version. Use `--sprint="Name"` to specify one.

**"unknown type/status/priority"**
Names are case-insensitive with prefix match. The error message shows available options.

**"no response from openpr.epochbase.com after 3 tries"**
op sent the read again after 2 s and 5 s and never got an answer. Check the
VPN or network; writes are never retried, so check whether an update went through
before sending it again.

**"Version filter has invalid values"**
The default sprint in `~/.oprc` may be stale. Update the `sprint:` field or use `--sprint` flag. Run `op sprint list` to see available sprints.
