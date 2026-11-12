---
name: sprint-triage
description: Triages a sprint's tickets. `components` fills missing Component fields from labels and titles; `assignees` balances unassigned tickets across the team roster by load. Shows the plan first. Use when the user wants to tag or dispatch sprint work.
user_invocable: true
argument-hint: "components|assignees [-p <project>] [--sprint <name>] [--component <c>] [--dry-run] [--auto]"
allowed-tools: Bash(op *), Bash(awk *)
---

# Sprint triage

Both modes read the sprint, propose a plan as a table, and write after approval.
- **`components`**: tickets with no Component get one inferred from label, title and type.
- **`assignees`**: unassigned tickets of one component go to the lightest-loaded
  teammate on that component's roster. Built for a recurring (e.g. twice-weekly)
  dispatch.

With no mode given, pick `components` for "missing components" / "tag tickets" and
`assignees` for "dispatch" / "hand out tickets" / "who should take these". If it's unclear, ask.

## Arguments

- `-p <project>` / `--project <project>`: pass to every `op` call.
- `--sprint "<name>"`: the sprint. The default is the current one, resolved below.
- `--component <c>`: for `assignees` only, which component to dispatch (default `android`).
- `--dry-run`: print the plan and write nothing.
- `--auto`: apply the plan without the confirm step. These are field updates on
  existing tickets, which are recoverable and notify no one new.

## Resolve the sprint

`op board` with no `--sprint` picks the first open version whose dates cover today. In
`app` that is the year-long `App_2026_Done_Story/Tasks`, not the current two-week
sprint. So with no `--sprint`, run `op sprint list -p <project>` and take the sprint
that follows the project's dated naming pattern (`App_MM/DD/YYYY`, `Web - MM/DD/YYYY`)
with start ≤ today ≤ end (with no end date, as in web, the latest start ≤ today). Print
`Sprint: <name>` and pass `--sprint="<name>"` to `op board` and `op my team`.

---

## Mode `components`

Valid components: `android`, `ios`, `ott`, `engineering`, `analytics` (`op fields
component` lists the live set).

### Collect
1. Run `op board --sprint="<name>"` to get every ticket in the sprint.
2. For a ticket with no component (or where the board leaves it unclear), run `op show <id>` for title,
   type, labels, assignee and description. Collect them as `UNTAGGED`.
3. If `UNTAGGED` is empty, print "All tickets already have a component." and stop.

### Infer, in priority order

**1. Label** (HIGH confidence)

| Label | Component |
|---|---|
| `team#appandroid` | `android` |
| `team#appios` | `ios` |
| `team#appall` | `android,ios` (both) |
| `team#web` | `engineering` |
| `roku` | `ott` |

**2. Title keywords**, case-insensitive (MED)

| Keyword | Component |
|---|---|
| `[android]`, `android`, `google play` | `android` |
| `[ios]`, `ios`, `iphone`, `ipad`, `swift`, `xcode`, `app store` | `ios` |
| `[ott]`, `ott`, `roku`, `apple tv`, `fire tv`, `smart tv`, `tv app` | `ott` |
| `[web]`, `web`, `website`, `wordpress`, `frontend`, `backend`, `api`, `server` | `engineering` |
| `analytics`, `tracking`, `gtm`, `ga4`, `event`, `pixel` | `analytics` |

**3. Assignee** (LOW), only when 1 and 2 found nothing. If the assignee is on exactly one
roster in `~/.oprc` (see Roster below), use that component. Otherwise mark `?`.

**4. Type fallback** (LOW): a `bug` or `task` with no other signal gets `engineering`.

Tickets left at `?` go to a "needs manual review" list.

### Plan

```
Component Assignment Plan — <sprint>
  #id    Title                          Suggested    Confidence  Reason
  12345  [iOS] Fix login crash          ios          HIGH        label: team#appios
  12350  Share sheet on both apps       android,ios  HIGH        label: team#appall
  12346  Android camera permission      android      MED         keyword: "android"
  12347  Update CI pipeline             engineering  LOW         type: task fallback
Needs manual review (1):
  #12349  Some vague ticket title       (no signal found)
```

With `--dry-run`, print and stop. Otherwise, unless `--auto`, ask: "Apply these <N>? [y/N]
(or list IDs to skip)".

### Apply
Group tickets by suggested value and update each group in one call:

```
op update <id> <id> ... --component=android
op update <id> ... --component=android,ios
```

Component is a multi-value field and each `op update --component` replaces the whole
value, so a `team#appall` ticket needs both values in one call; two separate calls
(`android`, then `ios`) leave only `ios`. Bulk `op update` reports each ID and continues
past failures.

Finish with Applied / Needs manual review (`op update <id> --component=<value>`) /
Errors, then suggest `op check --sprint --sprint-name="<name>"` to confirm coverage.

---

## Mode `assignees`

### Roster
The candidates come from a roster in `~/.oprc`, not from who already holds tickets. A
teammate with zero tickets in the sprint, the lightest load of all, never appears in
`op my team`, and `op my team` also lists other teams (iOS devs show up next to Android
ones). Read only the `team:` block, so the API key in the same file stays out of the
transcript:

```
awk '/^team:/{f=1;next} f&&/^[^[:space:]#]/{exit} f' ~/.oprc
```

Expected shape (names exactly as `op my team` prints them; see `.oprc.example`):

```yaml
team:
  android:
    - Alice Example
    - Bob Example
```

If there is no roster for `<component>`, say so, offer the `.oprc` snippet, and fall back to the
people in `op my team` who already hold `<component>` tickets, warning that teammates
with no tickets can't be seen this way.

### Collect
1. Run `op board --sprint="<name>" --component=<component>`.
2. Put tickets whose card shows `@unassigned` into `UNASSIGNED = [{id, title, type, priority,
   points}]`. If a card is unclear, run `op show <id>`.
3. If it's empty, print "No unassigned <component> tickets in <sprint>." and stop.
4. Pull from the backlog only if the user asked. Add `op backlog --type task` items after the
   sprint tickets.

### Load
Build `LOAD[person] = {in_progress, todo, points}` from `op my team --sprint="<name>"` across all
components, since that is the person's real capacity. Every roster member missing from
the output gets load 0. People not on the roster are not candidates; if the user wants
someone else, they can name them at the confirm step.

### Plan
For each ticket, pick the roster member with the lightest load (fewest in-progress
items, then fewest points; ties go to whoever has related recent work). Update `LOAD`
as you go so the batch stays balanced.

```
Dispatch Plan — <sprint> (<component>)
  #id    Title                          → Assignee   Load    Reason
  82360  [Android] Offline cache        → Alice      0 wip   lightest load (no tickets)
  82361  [Android] Push opt-in dialog   → Bob        1 wip   lightest load
  82365  [Android] Fix ANR on launch    → Alice      1 wip   balanced (2nd)
Load after this plan:  Alice: 2 wip   Bob: 2 wip   Carol: 3 wip
```

With `--dry-run`, print and stop. Otherwise, unless `--auto`, ask: "Apply this dispatch?
[y/N] — or edit like `82361:Carol`, or `skip 82365`", and apply the edits.

### Apply
Make one call per assignee:

```
op update <id> <id> ... --assignee="<Name>"
```

Print `✓ <Name> ← #<id>, #<id>` per assignee, `✗ <Name> ← #<id> — <error>` on failure,
and continue. Finish with Assigned / Skipped / Errors and suggest `/op:sprint-report`
to share the updated board.
