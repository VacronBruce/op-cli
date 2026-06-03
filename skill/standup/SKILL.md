---
name: standup
description: Lead's standup / daily digest — sprint progress, the team's work by person, blockers, and at-risk items for the current sprint, in one briefing.
when_to_use: When user says "standup", "daily digest", "sprint health", "team status", "prep for standup", or wants a one-shot summary of the current sprint's state.
user_invocable: true
argument-hint: "[-p <project>] [--sprint <name>]"
allowed-tools: Bash(op *)
---

# Standup / Daily Digest

Produce a concise, skimmable standup briefing for the current sprint. Pass any
`-p <project>` / `--sprint "<name>"` from `$ARGUMENTS` through to each command.

## Gather

Run these (stop and report if `op` errors, e.g. no active sprint):

1. `op sprint progress` — overall completion (items + points, % done).
2. `op blocked` — blocked items (surface these first; they need action).
3. `op my team` — work grouped by assignee (who is on what).
4. If no project is set, run `op overview` instead of 1–3 and summarize the
   cross-project rollup.

## Output

A short digest, not raw dumps:

- **Sprint:** name · `X/Y` items done · `P/Q` points (%) · days left if known.
- **🚧 Blocked (N):** one line each — `#id subject — assignee`. List first.
- **By person:** one line per assignee — in-progress vs. todo, the key item(s).
- **⚠ Risks:** unestimated items, unassigned open items, anything stale/over-due.
- **Focus:** the single most important thing for the team to discuss today.

Keep `#id`s in the output (they're clickable). Don't pad — a lead should read it
in 15 seconds. If a section is empty (e.g. nothing blocked), say so in one line.
