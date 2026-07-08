---
name: ticket-dod
description: Definition of Done review before a ticket is closed. Checks that implementation-side obligations (tests, review, docs, deploy, acceptance criteria met, no known regressions) are satisfied, and posts or prints a DONE / NOT DONE verdict with the unmet items.
when_to_use: When user says "is this done", "definition of done", "verify done", "can I close this ticket", "DoD check", or wants to confirm a ticket meets the done bar before moving it to Done / Ready for Release.
user_invocable: true
argument-hint: "[ticket-id] [--dry-run]"
allowed-tools: Bash(op *), Read
---

# Ticket Definition of Done (Developer / QA)

Verify ticket #$ARGUMENTS meets the **Definition of Done** before it is closed.

**Audience:** developers, tech leads, QA.

This is the exit gate, the mirror image of `op check` (which is the Definition of
**Ready** entry gate). Where DoR is about ticket *content* — all of which lives in
OpenProject — **most DoD items are delivery facts that live outside the tracker**
(CI status, code review, deploy). So this is a **guided review**, not a fully
automated gate: verify what the ticket and its comments can prove, and for the
rest state the item as **UNKNOWN — confirm** rather than guessing it passed.

## Process

0. If `$ARGUMENTS` is a JIRA ID (e.g. `WP-23`), resolve it with `op search $ARGUMENTS` first.
1. `op show $ARGUMENTS` — read subject, type, status, acceptance criteria, assignee, and description.
2. `op comment $ARGUMENTS` — read comments for evidence (PR links, test/QA notes, deploy confirmations).
3. Evaluate each DoD item below. Mark it `DONE`, `NOT DONE`, or `UNKNOWN — confirm`.
4. Gate: **DONE** only if every *required* item is DONE and none is NOT DONE. Otherwise **NOT DONE**.
5. If `--dry-run`, print the review and stop. Otherwise print it (this skill does
   **not** change ticket status — moving to Done is a human/`sprint-close` action).

## Definition of Done checklist

Required (a NOT DONE blocks the gate):
- **Acceptance criteria met** — every criterion in the ticket is satisfied. *(Verifiable from the ticket.)*
- **Code merged** — the change is merged to the main branch. *(Look for a merged PR/MR link in comments; else UNKNOWN.)*
- **Tests added and passing** — unit/integration tests cover the change and CI is green. *(UNKNOWN unless a comment shows it.)*
- **Code reviewed** — reviewed and approved. *(UNKNOWN unless a comment/PR shows it.)*
- **No known regressions** — QA/verification passed; no open blocking bugs linked. *(Partly verifiable.)*

Advisory (report but do not block):
- **Docs / changelog updated** where the change is user- or API-facing.
- **Deployed** to staging or shipped behind a flag.
- **Story points reconciled** — actual effort noted if it diverged from the estimate.

## Output Format

```markdown
## Definition of Done: #<id> <subject>

### Status: <✅ DONE | ❌ NOT DONE>  (ticket status: <current OpenProject status>)

### Required
- [DONE/NOT DONE/UNKNOWN] Acceptance criteria met — <evidence or what is missing>
- [DONE/NOT DONE/UNKNOWN] Code merged — <PR link or "no evidence in ticket">
- [DONE/NOT DONE/UNKNOWN] Tests added and passing — <evidence>
- [DONE/NOT DONE/UNKNOWN] Code reviewed — <evidence>
- [DONE/NOT DONE/UNKNOWN] No known regressions — <evidence>

### Advisory
- [ok/missing] Docs / changelog
- [ok/missing] Deployed
- [ok/missing] Story points reconciled

### To close this ticket
1. <specific unmet item to resolve or confirm>
2. ...
```

## Key Principle

**Do not infer DONE from a closed status, and do not guess.** A ticket can be
marked closed without the work truly being done. If the ticket cannot prove an
item, say `UNKNOWN — confirm` and ask — surfacing the gap is the whole point
(fail loud). This skill reports; it never flips the ticket to Done itself.

## What This Skill Does NOT Do

- Does NOT change ticket status, fields, or assignees.
- Does NOT evaluate readiness to *start* (that is `op check` / /op:ticket-verify).
- Does NOT run CI, tests, or deploys — it reads the evidence already recorded.
