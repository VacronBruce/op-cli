---
name: ticket-prep
description: PM/project manager self-review for ticket quality before business review. Checks completeness, clarity, business justification, and acceptance criteria.
when_to_use: When user says "review my ticket", "is my ticket ready", "check ticket quality", "prep ticket for review", "help me improve ticket", or wants to self-review before submitting for business review or sprint planning.
user_invocable: true
argument-hint: "[ticket-id]"
allowed-tools: Bash(op *)
---

# Ticket Prep Review (PM / Project Manager)

Self-review ticket #$ARGUMENTS for completeness and clarity before business review.

**Audience:** Product managers, project managers, business analysts

## Process

1. Run `op show $ARGUMENTS` to fetch full ticket details and attachments
2. Run `op check $ARGUMENTS` to get the mechanical checklist score
3. Analyze with PM-focused judgment (see Evaluation Criteria below)
4. Output the structured review (see Output Format below)

## Evaluation Criteria

### 1. Completeness
Check if all template sections are filled in with real content (not just placeholders):
- User Story / Summary section
- Acceptance criteria (not empty checkboxes)
- Implementation notes
- UI/UX section (for visual changes)
- Out of scope section

### 2. Clarity
Would someone outside your team understand the need?
- Is the problem statement clear?
- Is the scope obvious?
- Could two people read this and agree on what to build?

### 3. Business Justification
Is the WHY clear?
- Who benefits from this change?
- Why now? (data, user feedback, competitor pressure, compliance)
- What happens if we don't do this?

### 4. Acceptance Criteria Quality
Are criteria specific and testable?
- BAD: "Dark mode works"
- GOOD: "User can toggle dark mode from Settings > Display"
- Can a QA engineer write test cases from these criteria?

### 5. Visual Assets
For UI changes:
- Are mockups or Figma links attached?
- Are affected screens identified?
- Are both mobile and tablet layouts covered (if applicable)?

### 6. Scope Definition
- Is "out of scope" explicitly stated?
- Are there ambiguities that could lead to scope creep?

## Output Format

```markdown
## Ticket Prep Review: #<id> <subject>

### Checklist (op check): <score>
<paste op check output>

### Completeness: <COMPLETE / NEEDS WORK / INCOMPLETE>
- [ok/missing] User story section
- [ok/missing] Acceptance criteria (with real content, not placeholders)
- [ok/missing] Implementation notes
- [ok/missing] UI/UX assets
- [ok/missing] Out of scope

### Clarity: <CLEAR / GOOD / VAGUE>
<explanation>

### Business Justification: <STRONG / ADEQUATE / VAGUE / MISSING>
<explanation>
<if vague/missing, provide suggested rewrite>

### Acceptance Criteria Quality: <TESTABLE / NEEDS REWRITE / MISSING>
<current criteria assessment>
<if needs rewrite, provide specific suggested criteria>

### Missing Attachments
<list what should be attached>

### Action Items for PM
1. <specific action>
2. <specific action>
...

### Verdict: <READY FOR REVIEW / NEEDS REFINEMENT / NEEDS REWRITE>
<if not ready, count of items to fix>
```

## Key Principle

**Provide rewrite suggestions, not just criticism.** When acceptance criteria are vague, write better ones the PM can copy-paste. When business justification is missing, draft a template they can fill in.

## What This Skill Does NOT Do

- Does NOT evaluate technical feasibility (that's /ticket-verify)
- Does NOT check implementation details depth (developer concern)
- Does NOT estimate story points or complexity
