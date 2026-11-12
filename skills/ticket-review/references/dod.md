# Definition of Done (`--done`)

This is the exit gate, the counterpart of `op check` (the Definition of Ready entry gate).
Audience: developers, tech leads, QA. This lens prints its verdict; it never posts a
comment and never changes the ticket's status (moving to Done is a person's call or
`/op:sprint-close`).

DoR is about ticket content, all of which lives in OpenProject. Most DoD items are
delivery facts that live outside the tracker (CI, code review, deploy), so this is a
guided review: verify what the ticket and its comments prove, and mark the rest
`UNKNOWN — confirm` rather than guessing it passed. A closed status proves nothing on
its own, because a ticket can be closed without the work being done.

## Evidence
Read `op show <id>` (acceptance criteria, status, assignee) and `op comment <id>` (MR/PR
links, test or QA notes, deploy confirmations).

## Checklist

Mark each `DONE`, `NOT DONE` or `UNKNOWN — confirm`.

Required (a NOT DONE fails the gate):
- **Acceptance criteria met**: every criterion is satisfied. *(Verifiable from the ticket.)*
- **Code merged**: merged to the main branch. *(Needs a merged MR/PR link in comments; otherwise UNKNOWN.)*
- **Tests added and passing**: tests cover the change and CI is green. *(UNKNOWN unless a comment shows it.)*
- **Code reviewed**: reviewed and approved. *(UNKNOWN unless a comment or MR shows it.)*
- **No known regressions**: QA/verification passed; no open blocking bugs linked. *(Partly verifiable.)*

Advisory (report, don't block):
- **Docs / changelog updated** where the change is user- or API-facing.
- **Deployed** to staging, or shipped behind a flag.
- **Story points reconciled**: actual effort noted if it diverged from the estimate.

Gate: **DONE** only if every required item is DONE. Otherwise **NOT DONE**.

## Output

```markdown
## Definition of Done: #<id> <subject>

### Status: <✅ DONE | ❌ NOT DONE>  (ticket status: <current OpenProject status>)

### Required
- [DONE/NOT DONE/UNKNOWN] Acceptance criteria met — <evidence or what is missing>
- [DONE/NOT DONE/UNKNOWN] Code merged — <MR link or "no evidence in ticket">
- [DONE/NOT DONE/UNKNOWN] Tests added and passing — <evidence>
- [DONE/NOT DONE/UNKNOWN] Code reviewed — <evidence>
- [DONE/NOT DONE/UNKNOWN] No known regressions — <evidence>

### Advisory
- [ok/missing] Docs / changelog
- [ok/missing] Deployed
- [ok/missing] Story points reconciled

### To close this ticket
1. <unmet item to resolve or confirm>
```

## Out of scope for this lens
Readiness to start (that is `op check` and the dev lens); running CI, tests or deploys
(it reads the evidence already recorded).
