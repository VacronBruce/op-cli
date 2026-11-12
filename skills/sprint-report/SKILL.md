---
name: sprint-report
description: Summarises a sprint from live OpenProject data, as a 15-second standup digest by default or, with --weekly, a copy-paste stakeholder report per project. Use when the user wants standup prep, sprint health or a weekly status report.
user_invocable: true
argument-hint: "[--weekly] [-p <project>]... [--sprint <name>] [--component <c>]"
allowed-tools: Bash(op *)
---

# Sprint report

Two outputs from the same data:
- **default**: a standup digest a lead reads in 15 seconds.
- **`--weekly`**: one copy-paste markdown block per project; a stakeholder reads each
  in about 30 seconds.

## Arguments

- `--weekly`: the weekly report instead of the standup digest.
- `-p <project>` / `--project <project>`: repeatable with `--weekly` (one block per
  project). If none is given, use the `project:` in `~/.oprc`.
- `--sprint "<name>"`: the sprint to report. The default is the current one, resolved below.
- `--component <c>`: only that component's tickets (`android`, `ios`, `ott`,
  `engineering`, `analytics`). Without it, the whole project.

## Resolve the sprint first

`op sprint progress`, `op blocked` and `op sprint close` take no `--sprint` flag. They
use the first open version whose dates cover today. In `app` that is the year-long
`App_2026_Done_Story/Tasks`, not the two-week `App_11/03/2026`, so their numbers can
describe the wrong sprint. Resolve the sprint yourself and pass it explicitly:

1. If `--sprint` is given, use it.
2. Otherwise run `op sprint list -p <project>` and take the sprint that follows the project's
   dated naming pattern (`App_MM/DD/YYYY` in app, `Web - MM/DD/YYYY` in web) with
   start ≤ today ≤ end. Where the project leaves the end date blank (web), take the
   latest start ≤ today.
3. Print `Sprint: <name>`.

## Gather, per project

Stop and report if an `op` command errors.

1. `op board -p <p> --sprint="<name>" [--component=<c>]` lists items by status. The footer
   reads `X/Y items done | P/Q points`. Cards show `@unassigned` when nobody owns one.
2. `op board -p <p> --sprint="<name>" --status=blocked [--component=<c>]` lists blocked items.
3. Standup only: `op my team -p <p> --sprint="<name>"` groups work by person. It has
   no component filter, so with `--component` keep only the tickets that appeared in
   step 1.
4. `op sprint progress -v` and `op blocked` are fine as a cross-check only when the
   `Sprint:` line they print matches the resolved sprint.

With no project anywhere (no `-p`, no `project:` in `~/.oprc`), run `op overview` and
summarise its cross-project rollup instead.

## Standup output

A short digest, not raw dumps:

- **Sprint:** name · `X/Y` items done · `P/Q` points (%) · days left (from the sprint's
  end date).
- **🚧 Blocked (N):** one line each, `#id subject — assignee`. Put it first, because these
  need action.
- **By person:** one line per assignee with in progress vs. to do and the key item(s).
- **⚠ Risks:** unestimated items, unassigned open items, anything stale or overdue.
- **Focus:** the single most important thing for the team to discuss today.

Keep the `#id`s (they're clickable). An empty section gets one line ("Nothing blocked.").

## Weekly output

**Shipped is not week-scoped.** No `op` query returns "closed this week" for the team,
so Shipped is the done/closed items on the sprint board. Say so in one line when that
list covers the whole sprint ("Shipped = all done items in the sprint, not filtered to
this week"), because a stakeholder would otherwise read sprint-wide numbers as this
week's output.

One fenced block per project. Keep `#id`s; an empty section gets "None.".

```
## <Project> — week of <date>

**Sprint:** <name> · <X/Y> items · <P/Q> pts (<Z%>) · <days left> left

### Shipped
- #<id> <title>

### In progress
- #<id> <title> — <assignee>

### 🚧 Blocked (escalate)
- #<id> <title> — <assignee> — <what it's waiting on>

### ⚠ At risk
- #<id> <title> — <unestimated / unassigned / stale / overdue>

### Next
- <the 1–3 things the team should focus on next week>
```

With several projects, print the blocks in sequence, then one roll-up line:
"<N> projects · <total done>/<total> items across sprints."
