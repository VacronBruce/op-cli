# Start rubric (`--start`)

Should I start this ticket now, and how? Audience: the developer about to run
`op start`. `op check` answers the mechanical part (fields, AC form by type, upstream
status of `relates` tickets, design evidence). This lens adds the judgment it can't
make: whether the work is still wanted, whether the real spec lives somewhere else,
which other repos it touches, whether a designer must look, and how much process the
ticket needs before code.

It only reads. Questions for the PM are drafted, never posted: posting is a comment
the user approves first.

## Extra fetches

On top of the shared fetch (`op show`, `op check`):
- **Parent.** `op show <id> --json | jq -r '._links.parent.href // empty'` gives
  `/api/v3/work_packages/<parent>`, or nothing. Then `op show <parent>`: its
  description and attachments are part of this ticket's spec. Judge the two together.
- **Siblings.** The parent's other children:
  `op show <parent> --json | jq -r '._links.children[]?.href'`, then
  `op show <ids> --brief`. With no parent, skip this.
- **Related.** Every relation from `op show <id>`, with its status:
  `op show <ids> --brief`. A `relates` ticket that is the feature's umbrella (same
  feature words, broader subject, often a platform-neutral tag) is read like a parent.
- **Duplicates.** `op search --text "<2–3 distinctive subject words>"` (all projects;
  duplicates often sit in another project).
- **Repos.** The ticket's own repo is the git repo you'll run `op start` in; outside
  one, infer it from the platform tag. `git fetch -q` each repo you search: a stale
  clone's "not found" means nothing.
- **Already done?** In each repo the ticket touches,
  `git log --all --oneline --grep "<id>"`, and the same for each sibling and related ID.

## Criteria

### 1. Worth doing (blocks)
Check in this order; the first that fires sets the value. When a match is partial,
use ASK and say what is unclear.
1. **DONE:** commits or a merged branch carry the ID, or a sibling's or related
   ticket's commits make the same change.
2. **DUP:** an open ticket with the same platform tag and overlapping subject.
3. **WAIT:** a related API, backend, web or CMS ticket isn't Done, Closed, Completed or
   Ready for Release. `op check`'s `no_blockers` warns on these; confirm the dependency
   is real and not just a `relates` link.
4. **ASK:** a sibling or related ticket marked Done may remove the need (e.g. it took
   the feature's entry point out). Ask before writing code.
5. **GO:** none of the above.

### 2. Info complete (blocks)
- The parent or an umbrella ticket holds the real spec (screenshots, video, wiki
  link): merge it in and say so. If the ticket can't be built from ticket + parent,
  it isn't complete.
- The text promises future input ("specs will be provided", "test cases to follow").
- The ticket's own status is Pending or Needs Update.
- A spec link exists only on a related ticket: name the link.
- No observable done condition: nobody could say what a tester would see.

### 3. Other repos (advice: adds search tasks)
Tickets that touch several repos often don't say so. Read the repo map, the file
named by `$OP_REPO_MAP` or else `~/.config/op/repo-map.md`: a table of areas
mentioned in ticket text (a CMS field, analytics events, paywall, push, a shared SDK)
and the repos to search for each. Also, map or not, add the repos implied by the
platform tags of related and sibling tickets.
- For each area that matches, run one search in that repo for the named field,
  event or screen (`git grep`, or a code index where one exists).
- Record each repo as touched or not touched, with the reason.
- Always open the other platform's sibling ticket and code: it is the cheapest
  reference implementation.

### 4. Designer (blocks T1+ UI work)
Touches UI = changes what a user sees: layout, a new or changed element, copy, a new
state. A behaviour fix behind an existing control with no visual change is not UI.
Designer needed when the ticket touches UI and any of:
- no design link, mockup or image (attached or inline in the description) on the
  ticket, its parent or its umbrella;
- the text defers design ("check UI/UX", "design to follow");
- an A/B test or a new visual pattern;
- the linked design disagrees with the ticket text.

### 5. Acceptance criteria (advice: grill, don't block)
`op check` judges the form by type. Judge the content:
- every criterion names something a tester or a test can observe;
- one outcome per criterion;
- at least one edge or error case (offline, empty, logged out, tablet, process death);
- out of scope is stated where a neighbouring change is tempting;
- a verification route is named.

### 6. Testable (advice)
Map each scenario, parity row or event row to a test level: unit, integration,
end-to-end, or manual with the route. A row that maps to nothing is a question.

### 7. Tier (decides the next step)

| Tier | When (any one) | Next step |
|---|---|---|
| **T0 Direct** | One-sentence change, location known, AC observable, one repo, no arch predictor | Test-first, straight away |
| **T1 Grill** | Approach clear, behaviour gaps (edge cases, unclear AC, a design question) | Short Q&A or `/op:ticket-refine`, then test-first |
| **T2 Spec first** | An arch predictor fires, 2+ repos, a new purchase or business flow, more than ~2 sprints | Write a spec and split it into tickets before code |
| **T3 Spike** | A core unknown ("research", "decide how", "can X be used") | Timeboxed spike; the output is a decision on the ticket |

Pick one tier: the highest whose trigger is confirmed. A trigger you can't confirm
from the tickets or the code becomes a PM question, not a tier.

Arch predictors in the subject: new screen / tab / page / redesign; journey, E2E,
regression or test harness; "replace"; upgrade, SDK, library; "all", "every",
"across"; a numbered "Task N" / "Wave" / "Step" series; notification, cache,
database, schema; "migrate". They are a reason to look, not a verdict: in one team's
history they caught about 60% of tickets whose MRs turned out architectural, and 60%
of matches were false alarms. A T2 predictor plus a linked plan ticket that already
holds the design counts as designed: tier it T1 or T0.

## Verdict
- Worth doing DONE → `ALREADY DONE`, DUP → `DUPLICATE OF #<n>`, WAIT →
  `WAIT UPSTREAM`, ASK → `ASK PM`.
- Worth doing GO, but Info FAIL or Designer YES on T1+ → `ASK PM`.
- Otherwise → `START`.

Only worth doing, info complete and designer block a start. The rest is advice: a
Definition of Ready used as a 100% gate drifts back toward waterfall.

## Output

```text
#<id> <subject> (<type>, <points> pts, <status>)
  Worth doing   GO|ASK|WAIT|DUP|DONE  <reason, naming the ticket that decides it>
  Info          PASS|FAIL        <what's missing, or "spec is on parent #<n>">
  Other repos   PASS|WARN        <repo: touched / not touched, why>
  Designer      NO|YES           <reason>
  AC            PASS|WARN|FAIL   <observable? edge case? out of scope?>
  Testable      PASS|WARN        <rows without a test level>
  Tier          T0|T1|T2|T3      <trigger> → <next step>
  op check      <passed>/<total> (<percent>%) <READY / NEEDS WORK>
Verdict: <verdict>

Questions for the PM (draft, not posted):
1. <specific question>
```

Leave out the questions block when there are none.

## Out of scope for this lens
Rewriting the ticket (that's `/op:ticket-refine`), posting anything, and changing
status or assignee (that's `op start`).
