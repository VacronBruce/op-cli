---
name: ticket-verify
description: Developer verification of ticket readiness before starting implementation. Checks technical gaps, ambiguities, dependencies, and team-specific requirements.
when_to_use: When user says "can I start on this ticket", "verify ticket", "what's missing from this ticket", "is this ready to develop", "estimate sanity check", or wants to verify a ticket has enough technical detail before coding.
user_invocable: true
argument-hint: "[ticket-id]"
allowed-tools: Bash(op *)
---

# Ticket Verify (Developer)

Verify ticket #$ARGUMENTS has enough technical detail to start implementation.

**Audience:** Developers, tech leads

## Process

0. If `$ARGUMENTS` is a JIRA ID (e.g. `WP-23`, `BUG-655`) rather than a numeric OpenProject ID, resolve it first with `op search $ARGUMENTS` and use the returned `#<number>` in the steps below.
1. Run `op show $ARGUMENTS` to fetch full ticket details and attachments
2. Run `op check $ARGUMENTS` to get the mechanical checklist score
3. Analyze with developer-focused judgment (see Evaluation Criteria below)
4. Detect team context from component/label (android, wordpress, web, ios)
5. Apply team-specific checks if team is detectable
6. Output the structured verification (see Output Format below)

## Evaluation Criteria

### 1. Implementability
Can a developer start coding from this description?
- Is the expected behavior unambiguous?
- Are inputs and outputs defined?
- Is the scope clear enough to estimate?

### 2. Technical Gaps
What specs are missing?
- API endpoint contracts (method, URL, request/response format)
- Data model changes (new fields, migrations, schema)
- UI specifications (layouts, states, transitions)
- Configuration or environment requirements

### 3. Ambiguities
Where would two developers interpret this differently?
- Undefined edge cases
- Vague requirements ("make it fast", "improve UX")
- Missing error/failure handling specs
- Unclear interaction with existing features

### 4. Dependencies
What's needed before or during implementation?
- Other tickets that must complete first
- External service changes or approvals
- Design assets not yet delivered
- Backend APIs not yet built

### 5. Risk Assessment
What could go wrong?
- Underestimated complexity vs story points
- Breaking changes to existing features
- Performance implications
- Security considerations

### 6. Estimation Sanity
Do story points match the actual work?
- Compare scope description to point value
- Flag if scope seems too large or too small for estimate

## Team-Specific Checks

Detect team from the ticket's component or label fields. If no team context is detectable, apply only the general checks from Evaluation Criteria above and skip team-specific items.

### Android (component=android or label=team#appandroid)
- API endpoint contract specified? (method, URL, request/response JSON)
- UI mockup or Figma link for new/changed screens?
- Data model changes documented? (Room entities, API DTOs)
- Navigation flow clear? (which screen to which screen)
- Offline behavior defined?
- Error states defined? (no network, server error, empty state)
- Min SDK / device compatibility noted?
- Backward compatibility addressed?

### WordPress (component or label contains web/wordpress)
- Affected templates/pages identified?
- WordPress hooks/filters to use or modify?
- Custom post types or meta fields needed?
- Database / wp_options changes?
- Plugin dependencies?
- SEO impact considered?
- Caching invalidation needed?
- Mobile responsive requirements?

### iOS (component=ios or label=team#appios)
- API endpoint contract specified?
- UI mockup or Figma link?
- Data model changes? (Core Data, Codable structs)
- Navigation flow clear?
- Min iOS version / device support noted?
- Accessibility requirements?

## Output Format

```markdown
## Developer Verification: #<id> <subject>

### Checklist (op check): <score>
<paste op check output>

### Can I Build This? <YES / NO — reason>

### Technical Gaps
1. **<gap title>** — <what's missing and why it matters>
2. **<gap title>** — <what's missing>
...

### Ambiguities
- <where two developers would disagree>
- <undefined edge case>
...

### Dependencies
- <team>: <what's needed>
- <external>: <what's needed>
...

### Edge Cases Not Covered
- <scenario not addressed in ticket>
...

### Risk Assessment
- Story points: <current> — <assessment: LOW/OK/HIGH for scope>
- <other risks>

### Questions to Ask PM Before Starting
1. <specific question>
2. <specific question>
...

### Verdict: <READY TO BUILD / BLOCKED / NEEDS CLARIFICATION>
<if not ready, list what must be resolved first>
```

## Key Principle

**Generate questions, not rewrites.** Unlike /ticket-prep (which helps PMs improve content), this skill identifies what a developer needs to know before they can start. The output is a list of specific questions to send back to the PM.

## What This Skill Does NOT Do

- Does NOT rewrite ticket descriptions (that's /ticket-prep)
- Does NOT evaluate business justification
- Does NOT judge writing clarity or formatting
- Does NOT make implementation decisions (that's the developer's job)
