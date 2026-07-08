---
name: ticket-review
description: Automated full review of an OpenProject ticket. Runs op check, applies the ticket-prep (PM) and ticket-verify (dev) rubrics in one pass, and posts ONE combined review comment that notifies the creator. Idempotent. Drives the op-agent reviewer daemon and is also runnable by hand.
when_to_use: When you want a complete, postable review of a single ticket in one pass — used by the op-agent ticket-reviewer daemon, or manually when you say "fully review ticket", "review and comment on ticket", or "run the bot review on <id>".
user_invocable: true
argument-hint: "[ticket-id] [--dry-run] | --detect"
allowed-tools: Bash(op *), Read, Write
---

# Ticket Review (combined PM + Dev, postable)

Produce ONE combined review of ticket #$ARGUMENTS and post it as a comment, idempotently.
This is the engine the **op-agent** reviewer daemon dispatches per ticket; it is also
runnable by hand. It reuses the rubrics of `/op:ticket-prep` (PM quality) and `/op:ticket-verify`
(developer readiness) in a single pass and posts one combined comment.

## Modes (parse from $ARGUMENTS)

- `<id>` — review the ticket and post the combined comment (idempotent).
- `<id> --dry-run` — compose and print the comment; do NOT post.
- `--detect` — one-time setup: determine what the BR-/AR-/APP- prefixes map to and write
  the daemon config (see Prefix Detection below).

## Defaults

Override via a config file if present at `$OP_REVIEWER_CONFIG` or
`~/.claude/state/ticket-reviewer.config.json`:

- Marker (hidden HTML comment that identifies the bot's own comments): `<!-- op-ticket-reviewer:v1 -->`
- Trigger phrase (how a creator requests re-review): `@bot review ticket again`

## Process (single ticket)

1. `op show <id>` — capture subject, **author (the creator to notify)**, type, component,
   label, and description.
2. `op check <id>` — the **Definition-of-Ready gate**. It prints a deterministic
   completeness percent and a `READY | NEEDS WORK` gate (a FAIL is a blocker; a
   WARN is advisory and does not block). Use its `NN%` figure verbatim as the
   `score` in the RESULT line — do NOT invent your own number.
3. `op comment <id>` — read existing comments and apply the **idempotency guard**:
   - Find the most recent comment containing the marker (the bot's own last review).
   - If none → proceed (first review).
   - If one exists and **no later** comment contains the trigger phrase → STOP; print
     `RESULT id=<id> posted=skipped reason=already-reviewed` and exit (do not post).
     (See **RESULT line** below for the exact machine-readable format.)
   - If one exists and a **later** comment (from a non-bot author) contains the trigger
     phrase → proceed (re-review requested).
4. Apply the **/op:ticket-prep rubric** (completeness, clarity, business justification,
   acceptance-criteria quality, visual assets, scope) → PM verdict
   (READY FOR REVIEW / NEEDS REFINEMENT / NEEDS REWRITE) with concrete rewrite suggestions.
5. Apply the **/op:ticket-verify rubric** (implementability, technical gaps, ambiguities,
   dependencies, risk, estimation; plus team-specific checks by component/label) →
   Dev verdict (READY TO BUILD / BLOCKED / NEEDS CLARIFICATION) with specific questions.
6. Compose ONE combined comment (see Output Format). The marker line MUST be first.
7. If `--dry-run`: print the comment and exit. Otherwise `op comment <id> "<comment>"`,
   then print the **RESULT line** (see below).

## Gate

- **READY** only if PM = READY FOR REVIEW **and** Dev = READY TO BUILD.
- Otherwise **NEEDS WORK**.

## RESULT line (machine-readable — the op-agent daemon parses this)

Print exactly ONE `RESULT` line as the **last** line of output. It MUST be a single line
of space-separated `key=value` tokens, and every value MUST be a single token with **no
spaces** (use the UNDERSCORE forms below, not the pretty spaced verdicts used in the comment
body). The daemon reports these fields per project to a status webhook.

- Posted a review (new or re-review):
  ```
  RESULT id=<id> posted=yes overall=<READY|NEEDS_WORK> pm=<pm> dev=<dev> score=<0-100>
  ```
- Skipped (already reviewed, no fresh trigger):
  ```
  RESULT id=<id> posted=skipped reason=already-reviewed
  ```

Token vocabularies (underscore forms; map 1:1 to the spaced verdicts in the comment body):

- `overall` : `READY` | `NEEDS_WORK`
- `pm`      : `READY_FOR_REVIEW` | `NEEDS_REFINEMENT` | `NEEDS_REWRITE`
- `dev`     : `READY_TO_BUILD` | `BLOCKED` | `NEEDS_CLARIFICATION`
- `score`   : integer 0–100 — the **deterministic Definition-of-Ready completeness
  percent** reported by `op check` (its `NN%` figure). Copy it verbatim; it is
  reproducible (Pass=100, Warn=50, Fail=0, averaged), so it is objective ticket
  completeness, NOT a judgment. The qualitative call lives in the `pm`/`dev`
  verdicts and the `overall` gate, not in this number.

## Output Format (the posted comment)

The comment is **role-layered** so business, product, and dev each find their part fast.
The plain-terms block comes **first** (after the title) so a non-technical reader gets the
whole picture without scrolling into jargon.

```markdown
<!-- op-ticket-reviewer:v1 -->
## 🤖 Review: #<id> <subject>

Hi @<creator> — automated readiness review below. Please address the items, then comment
**"@bot review ticket again"** and I'll re-review.

**Status: <🟢 Ready to build | 🟡 Needs work before build | 🔴 Not ready>**  (PM: <verdict> · Dev: <verdict>)

### 📋 In plain terms — Business / Product Owner
- **Why (goal):** <one plain sentence: the business outcome this ticket serves>
- **Who (users):** <who benefits>
- **What's the call:** <"Ready to build", or in non-technical words what's missing and why it matters>

### ✅ Product / PM (ticket-prep rubric)
- Verdict: <READY FOR REVIEW / NEEDS REFINEMENT / NEEDS REWRITE>
- <top fixes; for vague acceptance criteria or missing justification, provide a
  copy-pasteable rewrite in Given/When/Then form>

### 👩‍💻 Developers (ticket-verify rubric)
- Verdict: <READY TO BUILD / BLOCKED / NEEDS CLARIFICATION>
- <technical gaps and specific questions the PM must answer before coding>

### Mechanical check
<one-line op check summary: PASS / WARN / FAIL counts>

### What to do next
1. <actionable item>
2. <actionable item>
```

Rules:
- The hidden marker MUST be the **first line** so the daemon and future runs recognize the
  bot's own comment.
- The **📋 In plain terms** block is mandatory and must stay first after the title — it is
  the shared, jargon-free view (Impact-Map *Why / Who / What*) that lets any role read the
  outcome. Keep each line to one plain sentence.
- Use **plain headed sections** (as above). Do NOT rely on collapsible `<details>`:
  OpenProject sanitizes comment HTML, so `<details>` may render as raw text rather than
  collapse.
- **Status line ↔ overall:** 🟢 = overall `READY`; 🟡/🔴 = overall `NEEDS WORK` (use 🔴 when
  PM = NEEDS REWRITE or Dev = BLOCKED, 🟡 otherwise). The machine-readable `RESULT` line
  below still uses the binary `READY|NEEDS_WORK` token — the traffic light is presentation
  only.
- Keep it concise — list only items that matter.

## Prefix Detection (`--detect`)

1. `op projects` — list project identifiers.
2. If identifiers match BR / AR / APP (case-insensitive) → `prefixMapping: "projects"` and
   record the matching identifiers in `projects`.
3. Else `op show` a few recent tickets and inspect subjects for `BR-/AR-/APP-` and the
   JIRA ID field → `subject-prefix` or `jira-id`.
4. If inconclusive, STOP and report exactly what was found — do **not** guess.
5. Write the conclusion to the config file (default
   `~/.claude/state/ticket-reviewer.config.json`):
   ```json
   { "prefixMapping": "projects|subject-prefix|jira-id",
     "projects": ["..."],
     "subjectPrefixes": ["BR-", "AR-", "APP-"],
     "triggerPhrase": "@bot review ticket again",
     "marker": "<!-- op-ticket-reviewer:v1 -->",
     "perRunCap": 5,
     "rereviewScanWindowDays": 30 }
   ```

## Key Principle

ONE comment, idempotent. Combine PM rewrites and developer questions; never post twice for
the same ticket version. If anything is uncertain (auth failure, unparseable output,
ambiguous prefix mapping), **fail loud** — print the problem and stop rather than guessing.

## What This Skill Does NOT Do

- Does NOT change ticket status, fields, or assignees — it only comments.
- Does NOT discover tickets or run the poll loop (that is the op-agent daemon / `poll.sh`).
- Does NOT replace `/op:ticket-prep` or `/op:ticket-verify` for deep, interactive single-lens review.
