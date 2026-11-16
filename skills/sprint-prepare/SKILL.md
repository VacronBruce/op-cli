---
name: sprint-prepare
description: "Prepares the next sprint: finds or creates it under the project's naming convention, moves high-priority backlog and bugs in, and drafts tickets from IDs, a PRD or Figma for approval. Use when a lead asks to prepare or set up the next sprint."
user_invocable: true
argument-hint: "--project <id> [--tickets 123,456] [--figma <url>] [--prd ./prd.md] [--auto]"
allowed-tools: Bash(op *), Read
---

# Sprint Prepare

Prepare the next sprint for one project. Run the phases in order and pause between
them unless `--auto` is set.

## Arguments

- `--project <id>` — required; pass `-p <project>` to every `op` call. Missing → stop
  with "--project is required."
- `--tickets <ids>` — comma-separated work package IDs to bring in.
- `--figma <url>` — a Figma file to turn into tickets.
- `--prd <path>` — a local Markdown PRD to turn into tickets.
- `--auto` — skip the pause points between phases and before moves. It never skips the
  two approvals for creating tickets (see below): a ticket notifies people and lands on
  a shared board, and a flag typed at the start is not approval of content nobody has
  seen yet.

Track across phases: `NEXT_SPRINT_NAME`, `MOVED_TICKETS`, `CREATED_TICKETS`,
`CANDIDATES` (for manual follow-up).

## Creating tickets: two approvals, every time

Phases 1 and 4 may create tickets. Collect the drafts first, then:
1. **Intent** — list what you propose to create (title, source, why) and wait for a yes.
   The user can drop items.
2. **Exact content** — for each remaining draft show subject, type, every field
   (including inferred `--points` / `--assignee`), the sprint, and the full description,
   plus the `op create` command. Wait for an explicit yes; edits mean showing it again.

Create only the approved drafts, each once. `op create` prints the new `#id` and the
board it filed on; verify there, and don't re-run a create because a query in another
project came back empty (that has produced duplicates).

---

## Pre-phase: find or create the next sprint

Sprint names follow a per-project convention; check `op sprint list -p <project>`
rather than inventing one. Seen so far: `App_MM/DD/YYYY` in `app` (two weeks, start and
end set) and `Web - MM/DD/YYYY` in `web` (start only). The date is the sprint's start.

1. `op sprint list -p <project>`.
2. The current sprint is the open one that follows the dated pattern with
   start ≤ today ≤ end; with no end dates (web), the latest start ≤ today. Skip
   catch-all versions such as `App_2026_Done_Story/Tasks`, which is open and spans the
   whole year. None found → stop: "No active sprint found. Cannot infer next sprint dates."
3. Next start = current end + 1 day, or current start + 14 days when there is no end.
   Next name = the same pattern with that start date (e.g. `App_11/17/2026`).
4. Already in the list → use it and note "(already exists)". `app` sprints are created
   months ahead, so this is the usual case.
5. Otherwise show the name and dates; after a yes (or with `--auto`):
   `op sprint create "<name>" --start=<YYYY-MM-DD> -p <project>` (end defaults to
   start + 13 days).

Pause (unless `--auto`): "Sprint `<name>` ready. Proceed to Phase 1 — Ticket Intake? [y/N]"

---

## Phase 1 — Ticket intake

Skip when neither `--tickets` nor `--prd` was given.

### 1a. `--tickets`
For each ID, `op show <id>` (403/404 → "Skipped #<id>: not accessible", continue).
- **Same project** → collect for one move:
  `op sprint add <id> <id> ... --sprint="<NEXT_SPRINT_NAME>" -p <project>`; add to `MOVED_TICKETS`.
- **Other project** → draft a linked copy (title, "Linked from #<id> (<project>)",
  inferred fields). After the two approvals:
  `op create task "<title>" --description="<description>" -p <project>`, then
  `op link <new-id> --relates-to=<id>` and `op sprint add <new-id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`.
  Add to `CREATED_TICKETS`.

### 1b. `--prd`
Read the file. One ticket per `##` heading or numbered item that describes a
user-facing capability: title = the heading or first sentence, description = that
section's body. These are drafts; run the two approvals, then for each approved one
`op create task ...`, apply inferred fields, `op sprint add`, and add to `CREATED_TICKETS`.

Pause (unless `--auto`): "Phase 1 complete — <N> moved, <M> created. Proceed to Phase 2 — Backlog Review? [y/N]"

---

## Phase 2 — Backlog review

1. `op backlog --priority p0,p1,sev1,sev2 -p <project>` → move them all in one
   `op sprint add ... --sprint="<NEXT_SPRINT_NAME>" -p <project>`; add to `MOVED_TICKETS`.
2. `op backlog -p <project>` → the rest go to `CANDIDATES` ("backlog candidate").
3. Print: "Auto-moved <N> high-priority backlog items. <M> remaining candidates."

Pause (unless `--auto`): "Proceed to Phase 3 — Bug Review? [y/N]"

## Phase 3 — Bug review (backlog only)

1. `op backlog --type bug --priority sev1,sev2 -p <project>` → move those not already
   moved, in one `op sprint add`; add to `MOVED_TICKETS`.
2. `op backlog --type bug -p <project>` → the rest go to `CANDIDATES` ("backlog bug candidate").
3. Print: "Auto-moved <N> Sev1/Sev2 backlog bugs. <M> remaining bug candidates."

Pause (unless `--auto`): "Proceed to Phase 4 — Figma Analysis? [y/N]"

---

## Phase 4 — Figma analysis

Skip without `--figma`. Without the Figma MCP tools, print "Figma MCP unavailable —
skipping Phase 4. Attach the Figma link manually." and go to the summary.

1. Fetch the file with the Figma MCP. For each top-level frame or component: frame name
   → title; description/annotations → description; an "Acceptance Criteria" annotation → AC.
2. Look for an existing ticket by title with `op search --text "<title words>" -p <project>`,
   which matches subjects. Plain `op search <value>` matches the JIRA ID, not subjects.
3. **Match** → add the Figma link to that ticket's description. `op update --description`
   replaces the whole field, so rebuild it from `op show` with every existing section and
   inline image kept (see `/op:ticket-refine` for how), and show the before/after first.
   Note it under `MOVED_TICKETS` or as already in the sprint.
4. **No match** → a draft (description + "Figma: <url>"). After the two approvals:
   `op create task ...`, then `op sprint add <new-id> --sprint="<NEXT_SPRINT_NAME>" -p <project>`.
   Add to `CREATED_TICKETS`.

---

## Field inference

Defaults for new tickets, from the last 3–5 closed sprints:
1. From `op sprint list -p <project>`, take the 3–5 most recent closed/locked dated sprints.
2. For each, `op board -p <project> --sprint="<name>"` and collect type, component,
   points and assignee.
3. `avg_points[type]` and `typical_assignee[component]` become `--points` / `--assignee`
   on the drafts, shown in approval 2 like any other field.
4. Fewer than 3 closed sprints → leave those fields blank.

---

## Summary

```
═══════════════════════════════════════
Sprint Prepare Summary: <NEXT_SPRINT_NAME>
═══════════════════════════════════════
Tickets moved to sprint (<N>):
  #<id>  <title>
Tickets created in sprint (<N>):
  #<id>  <title>  [from: <source>]
Candidates for manual follow-up (<N>):
  #<id>  <title>  [<reason>]

Next: op sprint add <id> --sprint="<name>" -p <project> to move any candidates.
═══════════════════════════════════════
```

Note any phase that was skipped, declined or errored.
