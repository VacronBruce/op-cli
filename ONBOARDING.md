# op-cli team onboarding

`op` is a CLI for our OpenProject (sprints, backlogs, work packages), plus a
set of **Claude Code skills** so you can *ask* in natural language instead of
remembering flags. This guide gets you set up in about 5 minutes.

---

## 1. Install

**Install op-cli and the `op` Claude Code plugin.** The repo is public, so you need no login or token:

```bash
bash <(curl -fsSL https://github.com/VacronBruce/op-cli/releases/latest/download/install.sh)
```

The script detects your platform, downloads the `op` binary from the latest GitHub
release, asks for your API key, and installs the **`op` plugin**, which puts every skill under
the `op:` prefix (`/op:openproject`, `/op:sprint-report`, `/op:file-bug`, `/op:ticket-review`, …).

---

## 2. Configure `~/.oprc`

The installer creates it. These are the keys that matter:

```yaml
url: https://openpr.epochbase.com
api_key: your-api-key-here     # OpenProject → My Account → Access Tokens
project: app                   # your default project identifier
sprint: "App_06/02/2026"       # default sprint for `op create` (optional)
```

Verify:

```bash
op version
op projects        # should list projects you can see
```

> **Optional extras** (see the repo README): a `custom_fields:` block to map
> component/product/label to your instance, and a `templates:` block so every
> `op create bug`/`feature` starts from your description template.

---

## 3. Updating later

The installer already set up the plugin in step 1. To update:

```bash
op upgrade               # just the binary, to the latest release
claude plugin update op  # just the skills, to the latest plugin version
# or re-run the install one-liner from step 1 — refreshes the binary AND the plugin
```

If you cloned the repo, you can install the plugin by hand with
`claude plugin marketplace add .` then `claude plugin install op@op`.

---

## 4. Use it in Claude Code

You rarely need raw commands. Invoke a skill or describe what you want.

| You want… | Say in Claude |
|---|---|
| Turn natural language into an `op` command | `/op:openproject add 101 102 to the current sprint` |
| What's on my plate | "what am I working on?" runs `op my` (auto-detects project/sprint if none set) |
| Cross-project view | `op overview` |
| Standup briefing | `/op:sprint-report` |
| Weekly status report | `/op:sprint-report --weekly -p app` |
| File a bug, guided | `/op:file-bug CC button crashes on publish (Android)` |
| PM ticket self-review | `/op:ticket-review 12345 --pm` |
| Dev readiness check | `/op:ticket-review 12345 --dev` |
| Full review, printed | `/op:ticket-review 12345` (add `--post` to post it as a comment) |
| Is it done? | `/op:ticket-review 12345 --done` |

---

## 5. Command cheat sheet

```bash
op board                       # current sprint, kanban view
op my                          # my open items (current sprint)
op overview                    # my open work across ALL projects (top 5 × 3)
op blocked                     # blocked items in the sprint
op show 12345                  # full ticket details + attachments
op search WP-23                # map a JIRA id → OpenProject number
op start 12345                 # branch <project>-12345-<slug> + In Progress + assign to you
op create bug "title" --component=android --product=entd --priority=SEV1
op update 12345 --status=in-progress --assignee="Bruce Chen"
op comment 12345 "LGTM"        # post a comment (--edit=<id> to revise)
op link 12345 --parent=12300   # parent / --relates-to / --blocks
op sprint progress             # sprint completion summary
op backlog --unestimated       # backlog items needing estimation
```

Tips:
- **Shell completion:** after `source <(op completion zsh)`, `--component=<TAB>` suggests values.
- With no project set, bare `op my` auto-detects the project and sprint where most of
  your open work lives and points you at `op overview`.
- Most flag values accept unique-prefix abbreviations (`--component=eng` resolves to engineering).
- `-p <project>` and `--sprint "<name>"` override your defaults on any command.

---

## 6. Help

- `op <command> --help` for any command.
- Repo: `https://github.com/VacronBruce/op-cli` (the README has the full docs).
- `op upgrade` to self-update to the latest release.
