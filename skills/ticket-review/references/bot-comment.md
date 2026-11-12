# Bot comment, RESULT line and `--detect`

The skill prints the combined PM + Dev review in this format and, with `--post`, posts it. The
op-agent ticket-reviewer daemon (gitlab-tw `gmedtn/op-agent`, `ticket-reviewer/`) runs
`claude -p "/ticket-review <id> --post" --allowedTools "Bash(op *),Read,Write"` per
ticket and parses the RESULT line; before plugin 0.26.0 it ran the same command without
`--post`, because posting was the default then. `/ticket-review --detect` writes
its config.

## Config

Read `$OP_REVIEWER_CONFIG`, else `~/.claude/state/ticket-reviewer.config.json`, if it
exists. Defaults when it doesn't:
- Marker (hidden HTML comment that identifies the bot's own comments): `<!-- op-ticket-reviewer:v1 -->`
- Trigger phrase (how a creator asks for a re-review): `@bot review ticket again`

## Idempotency guard (`--post` only)

Run this on the `op comment <id>` listing before posting, so the bot never posts twice
for the same ticket version:
- Find the most recent comment containing the marker (the bot's last review).
- If there is none, post (first review).
- If one exists and no later comment contains the trigger phrase, don't post. Print
  `RESULT id=<id> posted=skipped reason=already-reviewed` and stop.
- If one exists and a later comment from a non-bot author contains the trigger phrase,
  post (re-review requested).

Without `--post`, skip the guard, because a person asking for a review wants to read it
even when the bot already commented.

## Comment format

The comment has a section per role, so business, product and dev each find their part fast. The
plain-terms block comes first after the title so a non-technical reader gets the whole
picture without scrolling into jargon.

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

### ✅ Product / PM (PM rubric)
- Verdict: <READY FOR REVIEW / NEEDS REFINEMENT / NEEDS REWRITE>
- <top fixes; for vague acceptance criteria or missing justification, a copy-pasteable
  rewrite in Given/When/Then form>

### 👩‍💻 Developers (developer rubric)
- Verdict: <READY TO BUILD / BLOCKED / NEEDS CLARIFICATION>
- <technical gaps and specific questions the PM must answer before coding>

### Mechanical check
<one-line op check summary: PASS / WARN / FAIL counts>

### What to do next
1. <actionable item>
2. <actionable item>
```

Rules:
- The marker is the first line, because the daemon and later runs find the bot's own
  comment by it.
- The plain-terms block stays first after the title, one plain sentence per line. It is
  the shared, jargon-free view (Impact Map Why / Who / What) that any role can read.
- Use plain headed sections, not `<details>`, because OpenProject sanitizes comment HTML, so
  `<details>` can render as raw text.
- Status maps to overall: 🟢 means overall `READY`, and 🟡/🔴 mean `NEEDS WORK` (🔴 when PM is NEEDS
  REWRITE or Dev is BLOCKED, 🟡 otherwise). The RESULT line keeps the binary token; the
  traffic light is presentation only.
- List only the items that matter.

## RESULT line

It is the last line of output for a combined PM + Dev review, posted or not: one line of
space-separated `key=value` tokens. Every value is a single token with no spaces (the
underscore forms below), because the daemon splits on whitespace and reports these
fields per project to a status webhook.

```
RESULT id=<id> posted=<yes|no> overall=<READY|NEEDS_WORK> pm=<pm> dev=<dev> score=<0-100>
RESULT id=<id> posted=skipped reason=already-reviewed
```

- `posted`: `yes` (posted with `--post`), `no` (printed only), `skipped` (guard stopped it)
- `overall`: `READY` | `NEEDS_WORK`
- `pm`: `READY_FOR_REVIEW` | `NEEDS_REFINEMENT` | `NEEDS_REWRITE`
- `dev`: `READY_TO_BUILD` | `BLOCKED` | `NEEDS_CLARIFICATION`
- `score`: the `NN%` figure from `op check`, copied verbatim. It is the deterministic
  Definition-of-Ready completeness (Pass=100, Warn=50, Fail=0, averaged), not a
  judgment; the judgment lives in `pm`, `dev` and `overall`.

## Prefix detection (`--detect`)

One-time daemon setup: work out what the BR- / AR- / APP- prefixes map to.
1. Run `op projects` to list project identifiers.
2. If identifiers match BR / AR / APP (case-insensitive), set `prefixMapping: "projects"` and
   record the matching identifiers in `projects`.
3. Otherwise run `op show` on a few recent tickets and inspect subjects for `BR-/AR-/APP-` and
   the JIRA ID field to choose `subject-prefix` or `jira-id`.
4. If that is inconclusive, stop and report exactly what you found. Don't guess, because the
   daemon enumerates tickets from this file.
5. Write the result to the config file (default `~/.claude/state/ticket-reviewer.config.json`):
   ```json
   { "prefixMapping": "projects|subject-prefix|jira-id",
     "projects": ["..."],
     "subjectPrefixes": ["BR-", "AR-", "APP-"],
     "triggerPhrase": "@bot review ticket again",
     "marker": "<!-- op-ticket-reviewer:v1 -->",
     "perRunCap": 5,
     "rereviewScanWindowDays": 30 }
   ```
