## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

<!-- Extended Rules -->

## 5. Use the model only for judgment calls
Use Claude for: classification, drafting, summarization, extraction from unstructured text.
Do NOT use Claude for: routing, retries, status-code handling, deterministic transforms.
If a status code already answers the question, plain code answers the question.

## 6. Token budgets are not advisory
Per-task budget: 4,000 tokens.
Per-session budget: 30,000 tokens.
If a task is approaching budget, summarize and start fresh. Do not push through.
Surfacing the breach > silently overrunning.

## 7. Surface conflicts, don't average them
If two existing patterns in the codebase contradict, don't blend them.
Pick one (the more recent / more tested), explain why, and flag the other for cleanup.
"Average" code that satisfies both rules is the worst code.

## 8. Read before you write
Before adding code in a file, read the file's exports, the immediate caller, and any obvious shared utilities.
If you don't understand why existing code is structured the way it is, ask before adding to it.
"Looks orthogonal to me" is the most dangerous phrase in this codebase.

## 9. Tests verify intent, not just behavior
Every test must encode WHY the behavior matters, not just WHAT it does.
A test like `expect(getUserName()).toBe('John')` is worthless if the function takes a hardcoded ID.
If you can't write a test that would fail when business logic changes, the function is wrong.

## 10. Checkpoint after every significant step
After completing each step in a multi-step task: summarize what was done, what's verified, what's left.
Don't continue from a state you can't describe back to me.
If you lose track, stop and restate.

## 11. Match the codebase's conventions, even if you disagree
If the codebase uses snake_case and you'd prefer camelCase: snake_case.
If the codebase uses class-based components and you'd prefer hooks: class-based.
Disagreement is a separate conversation. Inside the codebase, conformance > taste.
If you genuinely think the convention is harmful, surface it. Don't fork it silently.

## 12. Fail loud
If you can't be sure something worked, say so explicitly.
"Migration completed" is wrong if 30 records were skipped silently.
"Tests pass" is wrong if you skipped any.
"Feature works" is wrong if you didn't verify the edge case I asked about.
Default to surfacing uncertainty, not hiding it.

## 13. Keep docs in sync after CLI changes
After any change to `op` behavior (new/renamed command, flag, output format, or field),
check and update the docs that demonstrate it — they drift silently otherwise:
- The skills under `skills/` (especially `skills/openproject/SKILL.md` command reference
  and how-to list, plus `ticket-prep` / `ticket-verify` / `ticket-review`) and their
  installed copies under `~/.claude/skills/` (loose) or via the `op:` plugin.
- `README.md` examples.
Every command/flag/output example shown in those files must still match real `op` output.
If a change touches them, update them in the same commit; if you can't verify an example, say so.

## Release

### GitLab release asset URLs

**Use Generic Package Registry URLs** for release links. Do NOT use `/uploads/` paths.

```
# CORRECT — works with glab, curl + token, and browser cookie
https://gitlab-tw.ddns.net/api/v4/projects/gmedtn%2Fop-cli/packages/generic/op-cli/latest/<filename>

# WRONG — only works with browser cookie, fails with glab and curl
https://gitlab-tw.ddns.net/gmedtn/op-cli/uploads/<hash>/<filename>
```

### Release process

```bash
export GITLAB_TOKEN=your-token
bash release.sh v0.5.0
```

The script uploads to both `<version>/` and `latest/` in the package registry. Release links always point to `latest/` so developers get the newest version.

### Browser download limitation

The `/api/v4/` package registry URLs return 401 when clicked in browser (browser sends cookies, not API token). For browser downloads, the release page description tells users to use glab or clone instead. This is a GitLab limitation for internal repos — no workaround exists.
