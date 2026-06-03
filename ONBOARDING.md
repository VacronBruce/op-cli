# op-cli — Team Onboarding

`op` is a fast CLI for our OpenProject (sprints, backlogs, work packages), plus a
set of **Claude Code skills** so you can just *ask* in natural language instead of
remembering flags. This guide gets you from zero to productive in ~5 minutes.

---

## 1. Install

**Prerequisite — `glab` (GitLab CLI), authenticated once:**

```bash
brew install glab          # macOS/Linux  (or: sudo apt install glab)
GITLAB_HOST=gitlab-tw.ddns.net glab auth login
```

**Install op-cli + the `op` Claude Code plugin:**

```bash
mkdir -p /tmp/op-cli && cd /tmp/op-cli && \
  GITLAB_HOST=gitlab-tw.ddns.net glab release download --repo gmedtn/op-cli \
  --include-external --asset-name="install.sh" && bash install.sh
```

The script detects your platform, installs the `op` binary, asks for your API key,
and installs the **`op` plugin** — all skills under the `op:` prefix
(`/op:openproject`, `/op:standup`, `/op:file-bug`, `/op:ticket-*`).

---

## 2. Configure `~/.oprc`

The installer creates it; the essentials:

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

> **Optional power-ups** (see the repo README): a `custom_fields:` block to map
> component/product/label to your instance, and a `templates:` block so every
> `op create bug`/`feature` starts with the right description scaffold.

---

## 3. Updating later

The installer already set up the plugin in step 1. To update:

```bash
op upgrade               # just the binary, to the latest release
claude plugin update op  # just the skills, to the latest plugin version
# or re-run the install one-liner from step 1 — refreshes the binary AND the plugin
```

(Manual fallback if you cloned the repo:
`claude plugin marketplace add .` then `claude plugin install op@op`.)

---

## 4. Use it in Claude Code

You rarely need raw commands — invoke a skill or just describe what you want.

| You want… | Say in Claude |
|---|---|
| Map NL → `op` command | `/op:openproject add 101 102 to the current sprint` |
| What's on my plate | "what am I working on?" → `op my` (auto-detects project/sprint if none set) |
| Cross-project view | `op overview` |
| Standup briefing | `/op:standup` |
| File a bug, guided | `/op:file-bug CC button crashes on publish (Android)` |
| PM ticket self-review | `/op:ticket-prep 12345` |
| Dev readiness check | `/op:ticket-verify 12345` |
| Full bot review (posts a comment) | `/op:ticket-review 12345` |

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
- **Shell completion:** `source <(op completion zsh)` → `--component=<TAB>` suggests values.
- **No project set?** bare `op my` auto-detects the project+sprint where most of
  your open work lives and points you at `op overview`.
- Most flag values accept **unique-prefix abbreviations** (`--component=eng` → engineering).
- `-p <project>` and `--sprint "<name>"` override your defaults on any command.

---

## 6. Help

- `op <command> --help` for any command.
- Repo: `git@gitlab-tw.ddns.net:gmedtn/op-cli.git` (README has full docs).
- `op upgrade` to self-update to the latest release.
