---
name: android-dispatch
description: Team-lead work dispatch — find unassigned Android tickets in the sprint, show each teammate's current load, propose a balanced assignment, and bulk-assign after confirmation. Built for a recurring (e.g. twice-weekly) dispatch.
when_to_use: When user says "dispatch work", "assign android work", "hand out tickets", "who should take these", "distribute the sprint", "twice-weekly assignment", or wants to spread unassigned Android tickets across the team by current load.
user_invocable: true
argument-hint: "[-p <project>] [--sprint <name>] [--dry-run] [--auto]"
allowed-tools: Bash(op *)
---

# Android Dispatch

Find every unassigned Android ticket in the sprint, balance it across the team by
each person's current load, and assign — with confirmation before writing.

## Parse arguments

From `$ARGUMENTS` extract:
- `-p <val>` or `--project <val>` — optional; the OpenProject project identifier
- `--sprint <val>` — optional; sprint name to target (defaults to active sprint)
- `--dry-run` — optional; show the plan without writing any changes
- `--auto` — optional; apply all suggestions without per-ticket confirmation

Pass `-p <project>` and/or `--sprint <name>` to every `op` command if provided.

---

## Phase 0 — Identify target sprint

1. If `--sprint` given, use it; otherwise the active sprint is used by default.
2. Print: "Sprint: `<name>`" (from the first command's output).

---

## Phase 1 — Collect unassigned Android tickets

1. Run `op board --component=android` (with `-p`/`--sprint` as appropriate).
2. Collect every ticket with **no assignee**. Board output shows the assignee per
   card; a blank/empty assignee means unassigned. If a card's assignee is unclear,
   confirm with `op show <id>` before deciding (same as `assign-components` does).
3. Build `UNASSIGNED = [{id, title, type, priority, estimate}]`.
4. If `UNASSIGNED` is empty: print "No unassigned Android tickets in this sprint.
   Nothing to dispatch." and stop.

Print: "Found <N> unassigned Android ticket(s)."

> Backlog pull (optional): if the user asked to also pull from the backlog, add
> `op backlog --type task` items too, but keep sprint tickets first.

---

## Phase 2 — Read the team's current load

1. Run `op my team` — items grouped by assignee (in-progress vs. todo).
2. Build `LOAD = {person: {in_progress: n, todo: n, points: p}}` from the output.
   Points may be absent; fall back to item counts.
3. The set of candidate assignees is the people already appearing in `op my team`
   for this sprint. Do **not** invent names — if the user wants someone not listed,
   they can name them at the confirm step.

This is the capacity view — no separate command needed.

---

## Phase 3 — Propose a balanced assignment

For each ticket in `UNASSIGNED`, pick the assignee with the **lightest current
load** (fewest in-progress items, then fewest points), breaking ties toward whoever
has relevant recent work. Update the running `LOAD` as you assign so the spread
stays balanced across the batch.

Build `PLAN = [{id, title, assignee, load_before, reason}]`.

Print the plan table:

```
Dispatch Plan — <SPRINT_NAME>
─────────────────────────────────────────────────────────────────
  #id    Title                          → Assignee     Load    Reason
  ─────────────────────────────────────────────────────────────────
  82360  [Android] Offline cache        → Alice        2 wip   lightest load
  82361  [Android] Push opt-in dialog   → Bob          1 wip   lightest load
  82365  [Android] Fix ANR on launch    → Alice        3 wip   balanced (2nd)
─────────────────────────────────────────────────────────────────
Current load after this plan:
  Alice: 4 wip   Bob: 2 wip   Carol: 3 wip
─────────────────────────────────────────────────────────────────
```

**Pause point** (skip if `--auto` or `--dry-run`):
"Apply this dispatch? [y/N] — or type edits like `82361:Carol` to reassign, or
`skip 82365` to drop a ticket."

- If `--dry-run`: print the plan, print "Dry-run — no changes written." and stop.
- If `--auto`: proceed without asking.
- Otherwise: apply the user's inline edits/skips to `PLAN`, then continue.

---

## Phase 4 — Apply

Group `PLAN` by assignee and bulk-assign each group in one call:

```
op update <id> <id> ... --assignee="<Name>"
```

Print per assignee: `  ✓ <Name> ← #<id>, #<id> …`
On error: print `  ✗ <Name> ← #<id> — <error>` and continue with the rest.

---

## Summary

```
═══════════════════════════════════════════════
Dispatch: <SPRINT_NAME>
═══════════════════════════════════════════════

Assigned (<N>):
  <Name> ← #<id> <title>
  ...

Skipped (<S>):
  #<id> <title>
  ...

Errors (<E>):
  #<id> — <error>
  ...

Next: run /op:standup to broadcast the updated board to the team.
═══════════════════════════════════════════════
```
