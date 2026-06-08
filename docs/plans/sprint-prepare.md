# Plan: `op sprint-prepare` Skill

## Problem & Success Criteria

**Problem:** Dev leads spend manual effort every sprint preparing the next sprint — finding backlog candidates, triaging bugs, creating tickets from designs and PRDs, and setting up the sprint itself. There is no structured workflow for this today.

**Success:** A dev lead can run `op sprint prepare` a few days before sprint end and get:
1. The next sprint created in OpenProject
2. High-priority backlog and bug tickets automatically moved to it
3. New tickets created from provided ticket references, PRD, and Figma designs
4. A full summary report of everything created/moved, plus a candidate list for manual follow-up

## Non-Goals

- Codebase refactor analysis (future: dev lead will provide refactor tasks as input)
- Interactive ticket selection within the skill (dev lead uses separate `op` commands)
- Automated sprint scheduling or recurring setup
- Modifying tickets in other projects

## Design Decisions

| Decision | Rationale |
|---|---|
| Implemented as a Claude Code skill, not a Go command | Most logic requires LLM judgment (Figma parsing, PRD decomposition, field inference). Only API gaps get new Go code. |
| All flags optional | Dev lead may not have Figma or PRD every sprint. Each phase only runs if its input is present. |
| Pause between phases by default; `--auto` skips | Gives dev lead a checkpoint to abort or review before destructive API calls. |
| Sprint name = `Sprint YYYY-MM-DD` (inferred from current sprint end + 1 day) | Team uses date-based names; suggestion reduces typing while allowing override. |
| `--end` defaults to `--start` + 13 days | Team uses 2-week sprints; making `--end` optional reduces typing while still allowing override. |
| Single `--priority` flag covers both priority and severity | P0–P3 values are priority; SEV0–SEV3 (or Sev0–Sev3) are severity. One flag, one filter, no separate `--severity` flag. |
| Field inference from last 3–5 sprints | Provides data-driven defaults (avg story points per type, typical assignee per component) instead of requiring manual entry. |
| Tickets from other projects → create linked child, don't move | Cross-project moves break ownership. A linked ticket in the current project preserves traceability. |

## Command Shape

```
op sprint prepare \
  --project <id-or-name> \
  --tickets 123,456,789 \
  --figma <url> \
  --prd ./prd.md \
  [--auto]
```

## Five Phases

### Phase 1 — Ticket Intake (from `--tickets` and `--prd`)

**`--tickets` logic:**
- For each ticket ID: resolve it via `op show <id>`
- If it belongs to the current project → `op sprint add <id>` (move to next sprint)
- If it belongs to another project → `op create` a new ticket in current project with:
  - Title from source ticket
  - Description referencing source ticket
  - `op link <new-id> --relates-to=<source-id>` to link back
- Set fields (type, priority, points, assignee, component) inferred from last 3–5 sprints

**`--prd` logic:**
- Read the local `.md` file
- Claude extracts one user story per ticket
- For each story: `op create` with title, description, type, priority, points, assignee, component (inferred from sprint history)
- `op sprint add <new-id>` to add to next sprint

### Phase 2 — Backlog Review

- Query backlog for the current project: all open tickets with no sprint assignment
- Filter with `op backlog --priority p0,p1,sev1,sev2` (single `--priority` flag handles both priority and severity values)
- Auto-move qualifying tickets → `op sprint add`
- Output remaining backlog tickets as a candidate list for manual follow-up

### Phase 3 — Bug Review (backlog bugs only)

- Query open bugs in the backlog for the current project: `op backlog --type bug --priority sev1,sev2`
- Scope: backlog-only (no cross-sprint bug query in this iteration)
- Auto-move qualifying bugs → `op sprint add`
- Output remaining backlog bugs as a candidate list

### Phase 4 — Figma Analysis (only if `--figma` provided)

- Use Figma MCP to fetch the design file
- Extract: screen/component names → ticket titles, design notes → descriptions, acceptance criteria
- For each extracted item:
  - Search existing tickets in project for a title match
  - If match found → attach Figma link to description, confirm sprint assignment
  - If no match → `op create` new ticket with all extracted fields + Figma link
  - `op sprint add` to next sprint

### Phase 5 — Codebase Refactor

**Out of scope for this iteration.** Placeholder phase; skill will note it and skip.

### Sprint Creation (runs before Phase 1 if next sprint doesn't exist)

- Find active sprint via existing `FindActiveSprint()` / `op sprint progress`
- Suggest next sprint name: `Sprint YYYY-MM-DD` where date = active sprint end + 1 day
- Suggest `--start` = active sprint end + 1 day; `--end` defaults to `--start` + 13 days
- Call `op sprint create <name> --start=<date> --end=<date>` *(new CLI command needed)*

## op-cli Changes Needed

| Gap | What to Add | Location |
|---|---|---|
| No `op sprint create` CLI command | Add `sprint create <name> --start --end` subcommand; `--end` defaults to `--start` + 13 days; wraps existing `CreateVersion()` | `cmd/sprint.go` |
| No priority filter in backlog query | Expose `--priority` flag on `op backlog`; accepts P0–P3 and SEV0–SEV3 values; pass to `ListWorkPackages()` filter | `cmd/backlog.go` |
| No type filter in backlog query | Expose `--type` flag on `op backlog`; pass to `ListWorkPackages()` filter | `cmd/backlog.go` |

*(Gap previously listed as "bulk sprint add" has been resolved — `op sprint add` already supports multiple IDs.)*

## Ticket Field Inference Algorithm

```
1. Fetch all work packages from the last 3–5 completed sprints in the project
2. Group by (type, component)
3. avg_points[type][component] = mean of storyPoints across the group
4. typical_assignee[component] = mode of assignee across the group
5. Apply as defaults when creating new tickets; Claude may override based on source content
```

## Vertical Slices (Implementation Order)

Each slice is end-to-end: CLI change (if any) → skill logic → verified output.

| # | Slice | Verifies |
|---|---|---|
| 1 | **op sprint create** — add `sprint create` subcommand to Go CLI; `--end` defaults to `--start` + 13 days | `op sprint create "Sprint 2026-07-07" --start 2026-07-07` creates a 2-week sprint; override with `--end` works |
| 2 | **Backlog/bug filtering** — add `--priority` (P0–P3, SEV0–SEV3) and `--type` flags to `op backlog` | `op backlog --priority p0,p1` returns only P0/P1 tickets; `op backlog --type bug --priority sev1,sev2` returns sev1/sev2 bugs |
| 3 | **Skill scaffold** — create `skills/sprint-prepare/SKILL.md`, parse all flags, detect active sprint, suggest next sprint name, pause/confirm flow | Skill loads, prints suggested sprint name, pauses |
| 4 | **Sprint creation in skill** — call `op sprint create` with suggested name, allow dev lead to override | Next sprint created in OpenProject |
| 5 | **Phase 1a: ticket intake from --tickets** — resolve each ID, move same-project tickets, create cross-project linked tickets | Tickets moved or created with correct links |
| 6 | **Phase 1b: PRD intake** — read .md file, decompose into stories, create tickets with inferred fields | One ticket per user story created in next sprint |
| 7 | **Sprint history inference** — fetch last 3–5 sprints, compute avg points and typical assignee per component | Inferred fields applied to created tickets |
| 8 | **Phase 2 & 3: backlog + bug auto-move** — `op backlog --priority p0,p1,sev1,sev2` for backlog; `op backlog --type bug --priority sev1,sev2` for backlog bugs; move qualifying tickets, output candidates | P0/P1 backlog tickets and sev1/sev2 backlog bugs moved; candidate list printed |
| 9 | **Phase 4: Figma analysis** — fetch design via Figma MCP, extract titles/descriptions/AC, create or link tickets | Figma-derived tickets created in next sprint |
| 10 | **Summary report** — full list of all created/moved ticket IDs+titles, counts per category | Report printed at end |

## Testing Strategy

Per slice:
- **Slices 1–2 (CLI):** Unit test flag parsing; integration test against a test project in OpenProject (or mock HTTP)
- **Slices 3–4 (skill scaffold):** Manual run: confirm suggested name, confirm sprint creation
- **Slices 5–6 (ticket intake):** Provide 2 same-project IDs + 1 cross-project ID + a sample PRD; verify created/moved tickets in OpenProject
- **Slices 7 (inference):** Verify inferred fields match historical data (spot-check 3 tickets)
- **Slices 8 (backlog/bug):** Seed test project with P0 and P2 backlog tickets + sev1 and sev3 backlog bugs; verify only P0 and sev1 auto-move
- **Slice 9 (Figma):** Use a real Figma file URL; verify ticket titles match frame names
- **Slice 10 (report):** Verify all created/moved IDs appear in the final summary

## Open Risks / Assumptions

| Risk | Mitigation |
|---|---|
| Figma MCP availability — not guaranteed in all environments | Skill checks for Figma MCP before Phase 4; skips with a warning if unavailable |
| OpenProject API rate limits during bulk ticket creation | Add small delay between `op create` calls; log progress |
| PRD structure varies — Claude may miss stories or hallucinate | Skill prints extracted stories before creating tickets; dev lead confirms in non-auto mode |
| Sprint history may be < 3 sprints (new project) | Fall back to no inference; leave fields blank for dev lead to fill |
| `--tickets` IDs from other projects may not be accessible (permissions) | Catch 403/404 and report clearly; skip that ticket |
| Sprint name conflict (sprint with same name already exists) | Check before creating; abort with clear message if collision |
| OpenProject may not support severity as a standard filter field | Investigate API filter options during Slice 2; fall back to client-side filtering if needed |
