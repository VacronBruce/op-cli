---
name: file-bug
description: "Files an OpenProject bug: gathers repro steps, expected vs actual, acceptance criteria and fields, shows the final ticket, and creates it after approval. Use when the user wants to log a bug."
user_invocable: true
argument-hint: "[short description of the bug]"
allowed-tools: Bash(op *)
---

# Guided bug filing

Turn a bug report into a clean `op create bug`. Treat `$ARGUMENTS` as the initial
description and ask only for what's missing.

`op create bug` files to the Bug Backlog board (`bug`) by default, with no sprint. It
ignores the `.oprc` / `OP_SPRINT` sprint, because triage assigns one. Pass
`-p <board>` only to file elsewhere, or `--sprint="<name>"` for a sprint that exists on
that board.

## Two approvals, every time

A ticket notifies people and lands on a shared board, so create it only after two
separate approvals from the user:

1. **Intent**: what bug you'll file and on which board. The user's own explicit request
   to file this bug ("file a bug: …", `/op:file-bug …`) counts. A bug you noticed while
   doing something else, a plan, or "continue" does not, so ask first.
2. **Exact content**: after collecting, show the complete final ticket (below) and wait
   for an explicit yes. This applies even when you guessed nothing. After any edit the user
   asks for, show the result again.

## Collect

- **Title**: concise, with a platform tag when relevant, e.g. `[Android] CC button crashes on publish`.
- **Steps to reproduce**: numbered. Ask; don't invent them.
- **Expected** vs **Actual** behaviour.
- **Acceptance criteria**: how we know it's fixed.
- **Classification**: suggest values from the title, and the user confirms them at approval 2:
  - `--component` (android, ios, ott, engineering, analytics)
  - `--product` (eet, entd, djy, cntd, competition, others)
  - `--tech-area` (web, app, adtech, video, infra, portal, seo)
  - `--label` (team#appios, team#appandroid, team#appall, team#web, ntd, seo, roku)
  - Values accept unique-prefix abbreviations (e.g. `--component=eng`); `op fields <name>`
    lists the live values.
- **Priority**: `P0`–`P3` / `SEV0`–`SEV3` (default `SEV2` for a bug).
- Optional: `--assignee`, `--sprint`, `--attach <screenshot path>`.

If `~/.oprc` defines a `templates.bug` body, `op create bug` without `--description`
pre-fills that template; you can fill it in instead.

## Show the final ticket

Description as markdown:

```
## Steps to reproduce
1. ...
## Expected
...
## Actual
...
## Acceptance criteria
- [ ] ...
```

Present board, title, type (bug), priority, every field value, attachments and the
full description, then the exact command you'll run:

```
op create bug "<title>" --priority=<P> --component=<c> --product=<p> --label=<l> \
  [--tech-area=<t>] [--assignee=<a>] [--sprint="<s>"] [--attach=<path>] \
  --description="<markdown>"
```

Ask: "Create this ticket?" and wait.

## After creating

Run the approved command once. `op create` prints the new `#id`, its URL and the board
it filed on; report them. To verify, query that board (`op my --author -p <board>`).
An empty result from another project is not a failed create, and re-running the create
on that assumption is how duplicate tickets happened before.
