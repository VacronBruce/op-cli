---
name: weekly-report
description: Generate a copy-paste-ready weekly status report for one or more projects — sprint completion, what's in progress, blockers to escalate, and at-risk items — from live sprint data.
when_to_use: When user says "weekly report", "status report", "what shipped this week", "sprint status writeup", "report for <project>", or wants a skimmable stakeholder summary of one or more projects.
user_invocable: true
argument-hint: "[-p <project>]... [--sprint <name>]"
allowed-tools: Bash(op *)
---

# Weekly Report

Produce a skimmable, copy-paste-ready weekly report for each project the user
names. One markdown block per project. A stakeholder should read each in ~30s.

## Parse arguments

From `$ARGUMENTS` extract:
- one or more `-p <val>` / `--project <val>` — the projects to report on. If none
  given, report on the default project (from `~/.oprc`).
- `--sprint <val>` — optional; sprint name (defaults to each project's active sprint).

Report on each project in turn, passing its `-p` (and `--sprint` if given) to every
`op` command below.

---

## Per project — gather

Run these for the project (stop and report if `op` errors, e.g. no active sprint):

1. `op sprint progress -v` — completion (items + points, %) and the full item list.
2. `op board --component=android` — in-progress and notable items (drop the
   `--component` filter if the user wants the whole project, not just Android).
3. `op blocked` — blocked items to escalate.

### Closed-this-week (best-effort — fail loud)

There is no date-ranged "closed this week" query for the whole team. Do the best
available and be explicit about the limit:

- Use the closed/done items already visible in `op sprint progress -v` as "Shipped".
- If that output does not distinguish *when* items closed, say so in the report:
  "Shipped = all done items in the sprint (not filtered to this week)."

Never imply the "Shipped" list is week-scoped if you couldn't scope it — state the
limitation in one line instead (fail loud; see the repo's CLAUDE.md rule 12).

---

## Per project — output

Emit one fenced markdown block per project so the user can copy it straight into a
report or chat. Keep `#id`s (clickable). Omit empty sections with a one-line "None."

```
## <Project> — week of <date>

**Sprint:** <name> · <X/Y> items · <P/Q> pts (<Z%>) · <days left> left

### Shipped
- #<id> <title>            <!-- note here if not week-scoped -->

### In progress
- #<id> <title> — <assignee>

### 🚧 Blocked (escalate)
- #<id> <title> — <assignee> — <what it's waiting on>

### ⚠ At risk
- #<id> <title> — <unestimated / unassigned / stale / over-due>

### Next
- <the 1–3 things the team should focus on next week>
```

If multiple projects were requested, print each block in sequence with a blank line
between, then a one-line roll-up: "<N> projects · <total done>/<total> items across
sprints."
