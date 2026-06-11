---
name: file-bug
description: Guided bug filing — collects repro / expected / actual / acceptance criteria and the right component, product, label, and tech-area, then runs `op create bug` with a well-formed markdown description.
when_to_use: When user says "file a bug", "report a bug", "create a bug ticket", "log a defect", or describes a bug they want captured in OpenProject.
user_invocable: true
argument-hint: "[short description of the bug]"
allowed-tools: Bash(op *)
---

# Guided Bug Filing

Turn a bug report into a clean `op create bug`. Treat `$ARGUMENTS` as the initial
description and **only ask for what's missing** — don't interrogate.

> `op create bug` files to the Bug Backlog board (`bug`) by default, with **no sprint** — the `.oprc`/`OP_SPRINT` sprint is not applied (triage assigns one). Pass `-p <board>` only to file it elsewhere, or `--sprint="<name>"` to put it on a bug-board sprint (it must exist there).

## Collect

- **Title** — concise, with platform tag if relevant, e.g. `[Android] CC button crashes on publish`.
- **Steps to reproduce** — numbered. **Do not invent these — ask.**
- **Expected** vs **Actual** behavior.
- **Acceptance criteria** — how we know it's fixed.
- **Classification** — suggest from the title, then confirm:
  - `--component` (android, ios, ott, engineering, analytics)
  - `--product` (eet, entd, djy, cntd, competition, others)
  - `--tech-area` (web, app, adtech, video, infra, portal, seo)
  - `--label` (team#appios, team#appandroid, team#appall, team#web, ntd, seo, roku)
  - Values accept unique-prefix abbreviations (e.g. `--component=eng`).
- **Priority** — `P0`–`P3` / `SEV0`–`SEV3` (default `SEV2` for a bug).
- Optional: `--assignee`, `--sprint`, `--attach <screenshot path>`.

## Build & run

Compose the description as markdown:

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

Then assemble:

```
op create bug "<title>" --priority=<P> --component=<c> --product=<p> --label=<l> \
  [--tech-area=<t>] [--assignee=<a>] [--sprint="<s>"] [--attach=<path>] \
  --description="<markdown>"
```

If you guessed any classification field, **show the command and confirm before
running**. After creating, print the new `#id` and suggest `op show <id>` to verify.

> Tip: if `~/.oprc` defines a `templates.bug` body, `op create bug` without
> `--description` pre-fills that scaffold — you can fill it in instead.
