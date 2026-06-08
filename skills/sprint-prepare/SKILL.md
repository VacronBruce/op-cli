---
name: sprint-prepare
description: Prepare the next sprint — create the sprint, intake tickets from provided IDs/PRD/Figma, auto-move high-priority backlog and bugs, and produce a full summary report.
when_to_use: When dev lead says "prepare next sprint", "sprint prep", "set up next sprint", or wants to create a sprint and populate it from tickets, a PRD, Figma designs, or backlog.
user_invocable: true
argument-hint: "--project <id-or-name> [--tickets 123,456] [--figma <url>] [--prd ./prd.md] [--auto]"
allowed-tools: Bash(op *), Read
---

# Sprint Prepare

Prepare the next sprint for the given project. Run all phases in sequence; pause
for confirmation between phases unless `--auto` is set.

## Parse arguments

From `$ARGUMENTS` extract:
- `--project <val>` — required; the OpenProject project identifier
- `--tickets <val>` — optional; comma-separated work package IDs
- `--figma <val>` — optional; Figma file URL
- `--prd <val>` — optional; local path to a Markdown PRD file
- `--auto` — optional flag; skip all confirmation prompts

If `--project` is missing, stop and tell the user: `--project is required.`

## Shared state

Track these across all phases (update as work proceeds):

```
NEXT_SPRINT_NAME = ""
NEXT_SPRINT_ID   = ""
CREATED_TICKETS  = []   # {id, title, source}
MOVED_TICKETS    = []   # {id, title}
CANDIDATES       = []   # {id, title, reason} — for manual follow-up
```

---

## Pre-phase: Detect active sprint and create next sprint

1. Run `op sprint list -p <project>` (pass `-p <project>` to all `op` calls if project is not the default).
2. Identify the currently active sprint (status=open, date range contains today). If none, stop: "No active sprint found. Cannot infer next sprint dates."
3. Compute suggested next sprint name: `Sprint <end+1day>` where `<end+1day>` is the active sprint's end date + 1 day in `YYYY-MM-DD` format.
4. Check whether a sprint with that name already exists in the list.
   - If it exists: set `NEXT_SPRINT_NAME` and `NEXT_SPRINT_ID` to the existing one, skip creation, and note "(already exists)".
   - If it does not exist: show the user the suggested name and dates, then either confirm (non-auto) or proceed (auto).
     After confirmation: `op sprint create "<name>" --start=<start> -p <project>`
     This sets a 2-week end date automatically. Capture the returned sprint ID as `NEXT_SPRINT_ID`.

**Pause point** (skip if `--auto`): "Sprint `<name>` created (#`<id>`). Proceed to Phase 1 — Ticket Intake? [y/N]"
If user says no, stop cleanly and show the summary so far.

---

## Phase 1 — Ticket Intake

Skip this phase entirely if neither `--tickets` nor `--prd` was provided.

### 1a. Tickets from `--tickets`

For each ID in the comma-separated list:

1. `op show <id> -p <project>` to fetch the ticket.
   - On 403/404: log "Skipped #<id>: not accessible" and continue.
2. Check the project in the response:
   - **Same project**: run `op sprint add <id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`. Add to `MOVED_TICKETS`.
   - **Different project**: create a linked copy:
     a. `op create task "<title>" --description "Linked from #<id> (<source project>)" -p <project>`
     b. `op link <new-id> --relates-to=<id>` to link back to the source.
     c. `op sprint add <new-id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`
     d. Apply inferred fields (see Field Inference below).
     e. Add new ticket to `CREATED_TICKETS`.

### 1b. PRD intake

Read the file at `--prd`. Extract one user story per ticket using this heuristic:
- Each `##` heading or numbered item that describes a user-facing capability is one ticket.
- Title = the heading or first sentence of the item.
- Description = the full body of that section.

Print the extracted story list for the user to review. **Pause point** (skip if `--auto`): "Found <N> stories. Create all as tickets? [y/N]"

For each approved story:
1. `op create task "<title>" --description "<description>" -p <project>`
2. Apply inferred fields (see Field Inference below).
3. `op sprint add <new-id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`
4. Add to `CREATED_TICKETS`.

**Pause point** (skip if `--auto`): "Phase 1 complete — <N> tickets created/moved. Proceed to Phase 2 — Backlog Review? [y/N]"

---

## Phase 2 — Backlog Review

1. `op backlog --priority p0,p1,sev1,sev2 -p <project>`
2. For each returned ticket: `op sprint add <id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`. Add to `MOVED_TICKETS`.
3. `op backlog -p <project>` (no filter) to get all remaining backlog items.
4. Add remaining items (not already moved) to `CANDIDATES` with reason "backlog candidate".
5. Print: "Auto-moved <N> high-priority backlog items. <M> remaining candidates."

**Pause point** (skip if `--auto`): "Proceed to Phase 3 — Bug Review? [y/N]"

---

## Phase 3 — Bug Review (backlog only)

1. `op backlog --type bug --priority sev1,sev2 -p <project>`
2. For each returned ticket not already in `MOVED_TICKETS`: `op sprint add <id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`. Add to `MOVED_TICKETS`.
3. `op backlog --type bug -p <project>` (no priority filter) to get all backlog bugs.
4. Add remaining bugs (not already moved) to `CANDIDATES` with reason "backlog bug candidate".
5. Print: "Auto-moved <N> Sev1/Sev2 backlog bugs. <M> remaining bug candidates."

**Pause point** (skip if `--auto`): "Proceed to Phase 4 — Figma Analysis? [y/N]"

---

## Phase 4 — Figma Analysis

Skip this phase if `--figma` was not provided.

Check whether the Figma MCP tool is available. If not, print: "Figma MCP unavailable — skipping Phase 4. Attach Figma link manually." and continue to the summary.

If available:
1. Fetch the Figma file using the Figma MCP.
2. For each top-level frame or component:
   - Extract: frame name → ticket title; frame description / annotations → description; any "Acceptance Criteria" section in annotations → AC.
3. For each extracted item:
   - Search existing tickets in the project for a title match (`op search "<title>" -p <project>` or fuzzy check).
   - **Match found**: update the description to include the Figma link. Add to `MOVED_TICKETS` (or note as already in sprint).
   - **No match**: `op create task "<title>" --description "<description>\n\nFigma: <url>" -p <project>`, then `op sprint add <new-id> --sprint="<NEXT_SPRINT_NAME>"`. Add to `CREATED_TICKETS`.

**Pause point** (skip if `--auto`): "Phase 4 complete — <N> Figma tickets created/linked. Proceed to summary? [y/N]"

---

## Phase 5 — Codebase Refactor

Out of scope for this iteration. Skipping.

---

## Field Inference

When creating new tickets, infer default field values from the last 3–5 completed sprints:

1. `op sprint list -p <project>` → identify the 3–5 most recent closed sprints.
2. For each sprint, list its work packages and collect: type, component, story points, assignee.
3. Compute:
   - `avg_points[type]` = mean story points for tickets of that type
   - `typical_assignee[component]` = most frequent assignee for that component
4. Apply as defaults when creating tickets via `--points` and `--assignee` flags on `op create`.
5. If fewer than 3 completed sprints exist, skip inference and leave fields blank.

---

## Summary Report

Print after all phases complete:

```
═══════════════════════════════════════
Sprint Prepare Summary: <NEXT_SPRINT_NAME>
═══════════════════════════════════════

Tickets moved to sprint (<N>):
  #<id>  <title>
  ...

Tickets created in sprint (<N>):
  #<id>  <title>  [from: <source>]
  ...

Candidates for manual follow-up (<N>):
  #<id>  <title>  [<reason>]
  ...

Next: use 'op sprint add <id> --sprint="<name>"' to move any candidates.
═══════════════════════════════════════
```

If any phase was skipped or errored, note it in the summary.
