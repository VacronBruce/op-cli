---
name: sprint-close
description: Sprint-ending workflow — find "Ready for Release" tickets in any sprint, assign them a release version, and move them to Done.
when_to_use: When user says "close sprint", "sprint ending", "mark ready tickets as done", "release sprint tickets", "wrap up sprint", "end of sprint", or wants to push release-ready work packages to Done with a version assigned.
user_invocable: true
argument-hint: "[-p <project>] [--sprint <name>] [--release <name>] [--status <status>] [--auto]"
allowed-tools: Bash(op *)
---

# Sprint Close

End-of-sprint workflow: find all tickets in a "ready for release" state, assign
them a release version, and close them as Done. Works on the active sprint or
any named sprint.

## Parse arguments

From `$ARGUMENTS` extract:
- `-p <val>` or `--project <val>` — optional; the OpenProject project identifier
- `--sprint <val>` — optional; sprint name to target (defaults to active sprint)
- `--release <val>` — optional; release/version name to assign (prompted if omitted)
- `--status <val>` — optional; the ready status to filter on (default: `ready for release`)
- `--auto` — optional flag; skip all confirmation prompts

Set defaults:
```
TARGET_SPRINT_NAME = ""
TARGET_SPRINT_ID   = ""
RELEASE_NAME       = ""
READY_STATUS       = "--status" value if provided, else "ready for release"
CLOSED_TICKETS     = []   # {id, title}
FAILED_TICKETS     = []   # {id, title, error}
```

Pass `-p <project>` to every `op` command if `--project` was provided.

---

## Phase 0 — Identify target sprint

1. Run `op sprint list` (with `-p` if provided).
2. If `--sprint` was given:
   - Find the sprint by name in the list (case-insensitive substring match).
   - If not found: stop with "Sprint '<name>' not found. Run `op sprint list` to see available sprints."
   - Set `TARGET_SPRINT_NAME` and `TARGET_SPRINT_ID`.
3. If `--sprint` was NOT given:
   - Find the sprint whose status is open (active sprint).
   - If none found: stop with "No active sprint found. Use `--sprint <name>` to target a specific sprint."
   - Set `TARGET_SPRINT_NAME` and `TARGET_SPRINT_ID`.

Print: "Target sprint: `<TARGET_SPRINT_NAME>`"

---

## Phase 1 — Find ready-for-release tickets

1. Run `op board --status="<READY_STATUS>"` (with `-p` if provided).
   - The board command lists all items matching that status in the current/active
     sprint. If a non-active sprint is targeted, also run
     `op board --status="<READY_STATUS>" --no-sprint` and filter results whose
     sprint field matches `TARGET_SPRINT_NAME`.
2. Collect all returned work packages as `READY_TICKETS = [{id, title, assignee}]`.
3. If `READY_TICKETS` is empty:
   - Print: "No tickets with status '<READY_STATUS>' found in sprint '<TARGET_SPRINT_NAME>'."
   - Print: "Nothing to do. If tickets use a different status label, re-run with `--status <label>`."
   - Stop.
4. Print the list for review:
   ```
   Found <N> ticket(s) with status '<READY_STATUS>' in <TARGET_SPRINT_NAME>:
     #<id>  <title>  — <assignee>
     ...
   ```

---

## Phase 2 — Choose release version

1. Run `op release list` (with `-p` if provided) and show available releases.

2. If `--release` was given:
   - Check whether the name exists in the release list (exact match, case-insensitive).
   - **Exists**: set `RELEASE_NAME = <val>`. Print: "Using existing release: <RELEASE_NAME>"
   - **Does not exist**:
     - **Pause point** (skip if `--auto`):
       "Release '<val>' does not exist. Create it? [y/N]"
     - If confirmed (or `--auto`): run `op release create "<val>"`. Set `RELEASE_NAME = <val>`.
     - If declined: ask for an alternative name or stop.

3. If `--release` was NOT given:
   - Print the release list.
   - **Pause point** (always — release name is required input):
     "Enter the release name to assign (or type a new name to create it):"
   - Read the response as `RELEASE_NAME`.
   - Check if it exists in the list:
     - **Exists**: proceed.
     - **Does not exist**: run `op release create "<RELEASE_NAME>"`.

4. **Pause point** (skip if `--auto`):
   ```
   Ready to apply:
     Sprint:  <TARGET_SPRINT_NAME>
     Release: <RELEASE_NAME>
     Tickets: <N>

   Assign release '<RELEASE_NAME>' and set all <N> tickets to Done? [y/N]
   ```
   If user says no, stop cleanly.

---

## Phase 3 — Apply: assign release and close tickets

For each ticket in `READY_TICKETS`:

1. Assign the release:
   ```
   op update <id> --release="<RELEASE_NAME>"
   ```
   - On error: add to `FAILED_TICKETS` with reason "release assignment failed". Continue.

2. Move to Done:
   ```
   op update <id> --status=done
   ```
   - On error: add to `FAILED_TICKETS` with reason "status update failed". Continue.

3. On success: add to `CLOSED_TICKETS`. Print: `  ✓ #<id>  <title>`

If any failures occurred during a ticket, note them inline but keep processing the rest.

---

## Summary

```
═══════════════════════════════════════════
Sprint Close Summary: <TARGET_SPRINT_NAME>
═══════════════════════════════════════════

Release assigned: <RELEASE_NAME>

Closed (<N>):
  #<id>  <title>
  ...

Failed (<M>):
  #<id>  <title>  — <reason>
  ...

Next steps:
  • Run `op sprint close` to generate the carryover report.
  • Run `op release list` to confirm the release is populated.
═══════════════════════════════════════════
```

If `FAILED_TICKETS` is non-empty, print a reminder:
"Fix failed tickets manually: `op update <id> --release="<name>" --status=done`"
