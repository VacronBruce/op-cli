# PM rubric (`--pm`)

Is the ticket complete and clear enough for business review? Audience: product
managers, project managers, business analysts. Give rewrites, not just criticism:
when acceptance criteria are vague, write better ones the PM can copy-paste; when the
business justification is missing, draft one they can fill in.

## Criteria

### 1. Completeness
All template sections hold real content, not placeholders:
- User Story / Summary
- Acceptance criteria (not empty checkboxes)
- Implementation notes
- UI/UX section (for visual changes)
- Out of scope

### 2. Clarity
Would someone outside the team understand the need?
- Is the problem statement clear?
- Is the scope obvious?
- Could two people read this and agree on what to build?

### 3. Business justification (Impact Map: Why / Who)
Is the WHY clear in language a business reader understands?
- **Why (goal):** what business outcome does this serve? Prefer a "so that <outcome>" clause.
- **Who (users):** who benefits?
- Why now? (data, user feedback, competitor pressure, compliance)
- What happens if we don't do this?

### 4. Acceptance criteria quality (prefer Given/When/Then)
Specific, testable, and readable by business, PM and dev alike. Given/When/Then is the
one artifact all roles read.
- BAD: "Dark mode works"
- OK: "User can toggle dark mode from Settings > Display"
- GOOD:
  - Given the app is in light mode
  - When the user enables Dark Mode in Settings > Display
  - Then all screens switch to the dark palette and the choice persists across restarts
- Can QA write test cases straight from these criteria?

### 5. Visual assets
For UI changes: mockups or Figma links attached? Affected screens identified? Mobile
and tablet layouts covered where they apply?

### 6. Scope
Is "out of scope" stated? Any ambiguity that invites scope creep?

## Verdict
`READY FOR REVIEW` / `NEEDS REFINEMENT` / `NEEDS REWRITE`

## Output (when `--pm` runs alone)

```markdown
## Ticket Prep Review: #<id> <subject>

### In plain terms
- **Why (goal):** <one plain sentence — the business outcome>
- **Who (users):** <who benefits>

### Checklist (op check): <score>
<op check output>

### Completeness: <COMPLETE / NEEDS WORK / INCOMPLETE>
- [ok/missing] User story section
- [ok/missing] Acceptance criteria (real content, not placeholders)
- [ok/missing] Implementation notes
- [ok/missing] UI/UX assets
- [ok/missing] Out of scope

### Clarity: <CLEAR / GOOD / VAGUE>
<explanation>

### Business Justification: <STRONG / ADEQUATE / VAGUE / MISSING>
<explanation; if vague/missing, a suggested rewrite>

### Acceptance Criteria Quality: <TESTABLE / NEEDS REWRITE / MISSING>
<assessment; if it needs a rewrite, suggested criteria in Given/When/Then form>

### Missing Attachments
<what should be attached>

### Action Items for PM
1. <specific action>

### Verdict: <READY FOR REVIEW / NEEDS REFINEMENT / NEEDS REWRITE>
<if not ready, count of items to fix>
```

## Out of scope for this lens
Technical feasibility and implementation depth (the dev lens), story-point estimates.
