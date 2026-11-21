---
name: ticket-review
description: Reviews an OpenProject ticket for readiness (PM and developer lenses), whether to start it now (start lens) or completion (Definition of Done) and prints the review; --post adds one bot comment. Use for 'is this ticket ready / done?' or 'should I start this?'.
user_invocable: true
argument-hint: "<id|JIRA-ID> [--pm] [--dev] [--start] [--done] [--post] | --detect"
allowed-tools: Bash(op *), Bash(jq *), Bash(git log *), Bash(git grep *), Bash(git fetch *), Read, Write
---

# Ticket Review

Review ticket `$ARGUMENTS` through one or more lenses and print the result. Posting is
opt-in with `--post`: a comment notifies the creator and watchers and can't be cleanly
taken back, so by default a person reads the exact text first and decides.

## Arguments

- `<id>` — OpenProject number, or a JIRA ID such as `WP-23` / `BUG-655`.
- Lens, default `--pm --dev`:
  - `--pm` — quality before business review; read `references/pm-rubric.md`.
  - `--dev` — readiness before coding; read `references/dev-rubric.md`.
  - `--start` — whether to start now and how (worth doing, parent context, other repos,
    designer, tier); read `references/start-lens.md`. Runs alone, before `op start`.
  - `--done` — Definition of Done before closing; read `references/dod.md`. Runs alone.
- `--post` — post the combined PM + Dev review as one bot comment. It skips the
  show-first step, so it is meant for the unattended op-agent daemon. By hand, run
  without it, read the review, then ask to post it.
- `--dry-run` — accepted from older callers; same as the default (print only).
- `--detect` — one-time setup for the daemon; see `references/bot-comment.md`.

## Fetch once, for every lens

0. A JIRA ID → `op search <JIRA-ID>` and continue with the returned `#<number>`.
   `op search` matches the JIRA ID custom field across all projects, not titles.
1. `op show <id>` — subject, author (the creator to address), type, status, component,
   label, points, description, User Story field, attachments.
2. `op check <id>` (PM / Dev / Start lenses) — the Definition-of-Ready gate: a deterministic
   `NN%` and READY / NEEDS WORK. A FAIL blocks; a WARN is advisory.
3. `op comment <id>` (no text: lists comments, read-only) — for `--done` (evidence) and `--post` (idempotency guard).

Stop and say what failed if any of these errors (auth, unknown ID, unparseable output)
rather than reviewing a ticket you couldn't read.

## Review

Read the rubric file for each lens that runs and apply it.

- **Default (`--pm --dev`)** — one combined review. Gate: READY only if PM = READY FOR
  REVIEW and Dev = READY TO BUILD; otherwise NEEDS WORK. Print it in the comment format
  from `references/bot-comment.md`, so what the user reads is exactly what `--post`
  would post, and end with the RESULT line from that file.
- **`--pm` or `--dev` alone, `--start`, or `--done`** — print that rubric's own output format.
  These are never posted.

## Posting

- Without `--post`: print, end with `RESULT … posted=no …`, and don't call
  `op comment <id> "<text>"`. If the user then approves posting, post exactly the text
  they read (marker line included).
- With `--post` (combined review only): apply the idempotency guard in
  `references/bot-comment.md`, then `op comment <id> "<comment>"`, then print
  `RESULT … posted=yes …` as the last line.

## Limits

- It only reads, and comments with `--post`. Status, fields and assignees stay as they
  are; use `/op:ticket-refine` to rewrite AC or the user story.
- It doesn't discover tickets or run the poll loop; that is the op-agent daemon.
