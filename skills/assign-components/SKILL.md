---
name: assign-components
description: Find tickets missing a component and assign the correct one based on title, description, type, and labels. Works on the active sprint or any named sprint.
when_to_use: When user says "check missing components", "assign components", "tickets without component", "tag untagged tickets", "missing component", or wants to bulk-assign components to work packages that have none set.
user_invocable: true
argument-hint: "[-p <project>] [--sprint <name>] [--dry-run] [--auto]"
allowed-tools: Bash(op *)
---

# Assign Components

Find every ticket in a sprint that has no component set, infer the right
component from its content, and apply it — with confirmation before writing.

Valid components: `android`, `ios`, `ott`, `engineering`, `analytics`

## Parse arguments

From `$ARGUMENTS` extract:
- `-p <val>` or `--project <val>` — optional; the OpenProject project identifier
- `--sprint <val>` — optional; sprint name to target (defaults to active sprint)
- `--dry-run` — optional; show suggestions without writing any changes
- `--auto` — optional; apply all suggestions without per-ticket confirmation

Pass `-p <project>` to every `op` command if provided.

---

## Phase 0 — Identify target sprint

1. Run `op sprint list` (with `-p` if provided).
2. If `--sprint` given: find it by name (case-insensitive substring). Stop if not found.
3. Otherwise: use the open/active sprint. Stop if none found.

Print: "Sprint: `<name>`"

---

## Phase 1 — Collect tickets without a component

1. Run `op board` (with `-p` and/or `--sprint` as appropriate) to get all sprint tickets.
2. For each ticket whose component field is empty (visible as no `[android]`/`[ios]`/… tag
   in board output, or confirmed with `op show <id>` if unsure):
   - Fetch full details: `op show <id>`
   - Add to `UNTAGGED = [{id, title, type, labels, assignee, description_excerpt}]`
3. If `UNTAGGED` is empty: print "All tickets already have a component. Nothing to do." and stop.

Print: "Found <N> ticket(s) without a component."

---

## Phase 2 — Infer components

For each ticket in `UNTAGGED`, infer the component using these signals in priority order:

### Signal 1 — Label
| Label | → Component |
|-------|-------------|
| `team#appandroid` | `android` |
| `team#appios` | `ios` |
| `team#appall` | *(both android AND ios — split into two update calls)* |
| `team#web` | `engineering` |
| `roku` | `ott` |

### Signal 2 — Title / subject keywords (case-insensitive)
| Keyword pattern | → Component |
|-----------------|-------------|
| `[android]`, `android`, `google play` | `android` |
| `[ios]`, `ios`, `iphone`, `ipad`, `swift`, `xcode`, `app store` | `ios` |
| `[ott]`, `ott`, `roku`, `apple tv`, `fire tv`, `smart tv`, `tv app` | `ott` |
| `[web]`, `web`, `website`, `wordpress`, `frontend`, `backend`, `api`, `server` | `engineering` |
| `analytics`, `tracking`, `gtm`, `ga4`, `event`, `pixel` | `analytics` |

### Signal 3 — Assignee heuristic
Use only if signals 1 and 2 are both absent. Base it on the assignee's known team
(from team context you may have built up across this session). If no team context is
known for the assignee, leave component as `?` and flag for manual review.

### Signal 4 — Type fallback
| Type | → Component |
|------|-------------|
| `bug` with no other signal | `engineering` (generic fallback) |
| `task` with no other signal | `engineering` (generic fallback) |

### Confidence levels
- **HIGH**: signal 1 (label) matched
- **MED**: signal 2 (keyword) matched
- **LOW**: signal 3 or 4 applied

Build `SUGGESTIONS = [{id, title, component, confidence, reason}]`.

Tickets with component `?` (no signal found) go into `NEEDS_MANUAL` for separate reporting.

---

## Phase 3 — Review and confirm

Print the suggestion table:

```
Component Assignment Plan
─────────────────────────────────────────────────────────────────
  #id    Title                          Suggested   Confidence  Reason
  ─────────────────────────────────────────────────────────────────
  12345  [iOS] Fix login crash          ios         HIGH        label: team#appios
  12346  Android camera permission      android     MED         keyword: "android"
  12347  Update CI pipeline             engineering LOW         type: task fallback
  12348  Add tracking for share button  analytics   MED         keyword: "analytics"
─────────────────────────────────────────────────────────────────
Needs manual review (<M>):
  #12349  Some vague ticket title       (no signal found)
─────────────────────────────────────────────────────────────────
```

**Pause point** (skip if `--auto` or `--dry-run`):
"Apply these <N> suggestions? [y/N] (or enter comma-separated IDs to skip)"

- If `--dry-run`: print the table, print "Dry-run mode — no changes written." and stop.
- If `--auto`: proceed without asking.
- Otherwise: read user response. If they list IDs to skip, remove those from `SUGGESTIONS`.

---

## Phase 4 — Apply

For each ticket in `SUGGESTIONS`:

```
op update <id> --component=<component>
```

For tickets with component `team#appall` (both android AND ios):
```
op update <id> --component=android
op update <id> --component=ios
```

Print: `  ✓ #<id>  <title>  → <component>`

On error: print `  ✗ #<id>  <title>  — <error>` and continue.

---

## Summary

```
═══════════════════════════════════════════════
Component Assignment: <SPRINT_NAME>
═══════════════════════════════════════════════

Applied (<N>):
  #<id>  <title>  → <component>
  ...

Needs manual review (<M>):
  #<id>  <title>  (run: op update <id> --component=<value>)
  ...

Errors (<E>):
  #<id>  <title>  — <error>
  ...

Run `op check --sprint` to verify component coverage.
═══════════════════════════════════════════════
```
