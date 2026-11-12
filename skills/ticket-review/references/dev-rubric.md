# Developer rubric (`--dev`)

Does the ticket carry enough technical detail to start coding? Audience: developers
and tech leads. Generate questions, not rewrites. The output is a list of specific
questions to send back to the PM (the PM lens is the one that rewrites content).

## Criteria

### 1. Implementability
- Is the expected behaviour unambiguous?
- Are inputs and outputs defined?
- Is the scope clear enough to estimate?

### 2. Technical gaps
- API endpoint contracts (method, URL, request/response format)
- Data model changes (new fields, migrations, schema)
- UI specifications (layouts, states, transitions)
- Configuration or environment requirements

### 3. Ambiguities
Where would two developers read this differently?
- Undefined edge cases
- Vague requirements ("make it fast", "improve UX")
- Missing error/failure handling
- Unclear interaction with existing features
- Acceptance criteria not in Given/When/Then form: name the scenarios (trigger, action,
  expected result) left undefined

### 4. Dependencies
- Tickets that must finish first
- External service changes or approvals
- Design assets not yet delivered
- Backend APIs not yet built

### 5. Risk
- Complexity underestimated against the story points
- Breaking changes to existing features
- Performance and security implications

### 6. Estimation sanity
Compare the scope to the point value; flag a scope that looks too large or too small.

## Team-specific checks

Detect the team from the ticket's component or label. With no team detectable, apply
only the general criteria above.

**Android** (component=android or label=team#appandroid)
- API endpoint contract (method, URL, request/response JSON)?
- UI mockup or Figma link for new/changed screens?
- Data model changes documented (Room entities, API DTOs)?
- Navigation flow clear (which screen to which)?
- Offline behaviour defined?
- Error states defined (no network, server error, empty state)?
- Min SDK / device compatibility noted?
- Backward compatibility addressed?

**WordPress** (component or label contains web/wordpress)
- Affected templates/pages identified?
- Hooks/filters to use or modify?
- Custom post types or meta fields?
- Database / wp_options changes?
- Plugin dependencies?
- SEO impact? Cache invalidation? Mobile responsive requirements?

**iOS** (component=ios or label=team#appios)
- API endpoint contract?
- UI mockup or Figma link?
- Data model changes (Core Data, Codable structs)?
- Navigation flow clear?
- Min iOS version / device support?
- Accessibility requirements?

## Verdict
`READY TO BUILD` / `BLOCKED` / `NEEDS CLARIFICATION`

## Output (when `--dev` runs alone)

```markdown
## Developer Verification: #<id> <subject>

### Checklist (op check): <score>
<op check output>

### Can I Build This? <YES / NO — reason>

### Technical Gaps
1. **<gap>** — <what's missing and why it matters>

### Ambiguities
- <where two developers would disagree>

### Dependencies
- <team or external>: <what's needed>

### Edge Cases Not Covered
- <scenario>

### Risk Assessment
- Story points: <current> — <LOW / OK / HIGH for the scope>
- <other risks>

### Questions to Ask PM Before Starting
1. <specific question>

### Verdict: <READY TO BUILD / BLOCKED / NEEDS CLARIFICATION>
<if not ready, what must be resolved first>
```

## Out of scope for this lens
Rewriting the description, business justification, writing style, and implementation
decisions (those belong to the developer who picks the ticket up).
