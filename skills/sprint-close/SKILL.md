---
name: sprint-close
description: "Closes out a sprint: finds its Ready for Release tickets, matches each to its product's release (e.g. [Android][EET] 3.2.1), and after approval sets the release and Done in one update. Use at the end of a sprint."
user_invocable: true
argument-hint: "[-p <project>] [--sprint <name>] [--release <name>] [--status <status>] [--auto]"
allowed-tools: Bash(op *)
---

# Sprint close

End-of-sprint workflow: find the sprint's tickets in a "ready for release" state, give
each the release of its own platform and product, and move them to Done.

## Arguments

- `-p <project>` / `--project <project>`: pass to every `op` call.
- `--sprint "<name>"`: the sprint. The default is the current one, resolved below.
- `--release "<name>"`: use this one release for every ticket instead of matching per
  product.
- `--status "<status>"`: the ready status to look for (default `ready for release`).
- `--auto`: skip the confirm steps. These are field updates on existing tickets.

## Phase 0: Resolve the sprint

Run `op sprint list -p <project>`.
- If `--sprint` is given, find it by name (case-insensitive substring). If it isn't found, stop with
  "Sprint '<name>' not found. Run `op sprint list` to see available sprints."
- Otherwise take the open sprint that follows the project's dated naming pattern
  (`App_MM/DD/YYYY`, `Web - MM/DD/YYYY`) with start ≤ today ≤ end (with no end date,
  the latest start ≤ today). Don't rely on the auto-detected sprint, because in `app` it is the
  year-long `App_2026_Done_Story/Tasks`. If none matches, stop and ask for `--sprint`.

Print: "Target sprint: `<name>`"

## Phase 1: Find the ready tickets

`op board -p <project> --sprint="<name>" --status="<READY_STATUS>"` lists exactly that
sprint's tickets in that status, so you don't need to scan the whole board.

If it's empty, print "No tickets with status '<READY_STATUS>' in '<name>'. If the team uses another
label, re-run with `--status <label>`." and stop. Otherwise list them:
`#<id>  <title>  — <assignee>`.

## Phase 2: Match each ticket to its release

Releases are per platform and product: `op release list -p <project>` shows names like
`[Android][EET] …`, `[Android][DJY] …`, `[iOS][NTD+] 1.6.7`. One sprint's ready tickets
usually belong to several releases, so a single release for the whole sprint is wrong.

1. Run `op release list -p <project>`.
2. For each ticket take its `[Platform][Product]` from the title tag (`[Android][DJY]
   Enable resource shrinking…`). With no tag, run `op show <id>` and use its component and product.
3. Propose the open release with that prefix (open means not shipped yet; take the newest if several).
   If there's no tag, or no open release for the pair, ask the user to pick one, create one, or skip the ticket.
4. With `--release`, every ticket gets it, but if the tickets span more than one
   `[Platform][Product]`, say so and ask before applying one release to all.
5. To create a release, run `op release create "<name>" -p <project>` after a yes (or `--auto`).
6. Note each release's ID from the list. Some names carry a double space
   (`[Android][NTD+]  1.2.0`), and `--release` accepts the numeric ID, which avoids that problem.

Confirm (unless `--auto`):

```
Ready to apply — <sprint>
  [Android][DJY] 3.13.1 (2101):  #85612, #85794, #85872
  [Android][EET] 3.2.14 (2102):  #85594
  Skipped:                       #85874 (no release chosen)
Set these releases and move <N> tickets to Done? [y/N]
```

## Phase 3: Apply

Make one call per release group, with release and status in the same update:

```
op update <id> <id> ... --release=<release-id> --status=done -p <project>
```

Bulk `op update` applies the same change to each ID, reports each result and continues
past failures. Collect `CLOSED` and `FAILED (id, error)`.

## Summary

```
═══════════════════════════════════════════
Sprint Close Summary: <sprint>
═══════════════════════════════════════════
[Android][DJY] 3.13.1:  #85612, #85794, #85872
[Android][EET] 3.2.14:  #85594
Failed (<M>):
  #<id>  <title>  — <error>
Next:
  • op release show "<name>" to confirm each release's tickets.
  • op sprint close for the carryover report; it reports on the auto-detected
    sprint only, so check the Sprint: line it prints.
═══════════════════════════════════════════
```

For failures, print "Fix by hand: `op update <id> --release=<release-id> --status=done`".
