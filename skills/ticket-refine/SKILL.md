---
name: ticket-refine
description: Interactive ticket refinement for PMs and developers. Finds weak or missing acceptance criteria and user story, talks through fixes one question at a time, then writes the corrected fields back to the ticket after you confirm.
when_to_use: When user says "refine ticket", "fix the AC", "improve the user story", "help me rewrite acceptance criteria", "clean up this ticket", or wants to interactively improve a ticket's AC / user story and save the changes back (not just get a review comment).
user_invocable: true
argument-hint: "[ticket-id]"
allowed-tools: Bash(op *)
---

# Ticket Refine (interactive, writes back)

Interactively refine ticket #$ARGUMENTS and **write the improved fields back**.

**Audience:** PMs and developers who want to *fix* a ticket, not just get a review.

This is the write-capable counterpart to `/op:ticket-prep` and `/op:ticket-verify`
(which only analyze and comment). It focuses on the two things that most often
block readiness: **acceptance criteria** and the **user story**. It talks through
each gap one question at a time, shows a before/after, and only writes after you
say so.

## What it can write

- **Acceptance criteria** live *inside the description* — there is no separate AC
  field. Refining AC means rewriting the description via `op update --description`,
  preserving every other section untouched — **including inline images**. Screenshots
  are embedded in the description as markdown like `![](/api/v3/attachments/N/content)`;
  copy every such reference verbatim into the new description or the image is lost.
  As a safety net, `op update --description` refuses a rewrite that drops an inline
  image the ticket currently shows (override with `--force` only if the removal is
  intentional).
- **User story** is a dedicated field — write it via `op update --user-story`.

## Process

0. If `$ARGUMENTS` is a JIRA ID (e.g. `WP-23`), resolve it with `op search $ARGUMENTS` first.
1. `op show $ARGUMENTS` — read subject, type, the current description, the User
   Story field, and any existing acceptance criteria.
2. `op check $ARGUMENTS` — the Definition-of-Ready signal. Note which content
   checks Warn/Fail (acceptance criteria, well-formed user story, use case).
3. Decide the refinement targets. Focus on **acceptance criteria** and the **user
   story**; touch the wider description only if its clarity is what blocks the AC.
   If both AC and user story are already strong, say so and stop — do not invent work.
4. **Refine interactively — ONE question at a time.** For each gap:
   - Name what's weak in plain terms (e.g. *"'should be 100% showing up' isn't
     testable — how do we verify it?"*).
   - Propose a concrete rewrite as a starting point.
   - Ask the user to confirm or adjust, then **wait for their answer** before moving
     to the next gap. Never dump all questions at once.
   - If an answer is still vague, push back with a follow-up. Do **not** fabricate
     testable criteria the user did not give you (see Rules).
5. Assemble the final field values:
   - **Description:** rebuild the *full* description, keeping all existing sections
     verbatim and replacing/inserting only the acceptance-criteria section.
   - **User story:** the corrected story text.
6. **Confirmation gate.** Show a clear before → after for each field you intend to
   write. Ask for an explicit go-ahead (e.g. "apply"). If the user says no or wants
   changes, loop back to step 4 — do not write.
7. Write only the approved fields, in one call where possible:
   - `op update $ARGUMENTS --description "<full markdown>"`
   - `op update $ARGUMENTS --user-story "<markdown>"`
   (Both flags can be combined in a single `op update`.)
8. Re-run `op check $ARGUMENTS` and report the score **before → after** so the
   improvement is verified, not assumed.

## Preferred forms

- **Acceptance criteria:** Given / When / Then, one scenario per behavior.
  > **Acceptance criteria**
  > 1. Given `<context>`, when `<action>`, then `<observable result>`.
- **User story:** `As a <role>, I want <capability> so that <benefit>.`

## Rules

- **Interactive, one question at a time.** Wait for each answer before the next.
- **Never write without explicit confirmation.** Always show the before/after first.
- **Preserve the description.** Only the section you refined may change; if you are
  unsure you can rebuild it without dropping content, stop and show the user the
  full new description for approval rather than guessing. **Never drop inline image
  markdown** (`![](/api/v3/attachments/N/content)`) — reproduce it exactly. `op show`
  prints the raw markdown including these references; carry them across intact.
- **Do not invent facts.** If the user cannot answer a question, leave a clearly
  marked `TODO:` in the field rather than fabricating a plausible-but-wrong AC —
  fail loud, don't paper over the gap.
- **No comment is posted.** The field history already records what changed; keep
  the ticket free of review noise. (Use `/op:ticket-review` when you want a posted
  comment instead.)
- Write **only** the fields the user approved — do not touch status, assignee,
  points, or anything else here.

## What This Skill Does NOT Do

- Does NOT change status, assignee, points, sprint, or other fields (use `op update`
  directly for those).
- Does NOT post comments or drive the reviewer bot (that is `/op:ticket-review`).
- Does NOT decide readiness — it improves content; `op check` reports the score.
