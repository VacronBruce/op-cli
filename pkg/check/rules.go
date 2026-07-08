package check

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// CheckInput carries the out-of-band facts a check may need beyond the work
// package itself — data the runner fetches from separate endpoints (attachment
// counts, relations). Grouping them in a struct keeps the CheckFunc signature
// stable as new signals are added.
type CheckInput struct {
	AttachmentCount int
	// BlockedByCount is how many other work packages this one depends on
	// (is blocked by). The runner computes it from the relations collection.
	BlockedByCount int
}

// CheckFunc is the signature for individual check functions.
type CheckFunc func(wp *api.WorkPackage, in CheckInput) Result

// CheckDescription verifies the description exists and has substance.
func CheckDescription(wp *api.WorkPackage, _ CheckInput) Result {
	if wp.Description == nil || strings.TrimSpace(wp.Description.Raw) == "" {
		return Result{
			Name:    "Description has substance",
			Level:   Fail,
			Message: "No description provided",
		}
	}
	lines := countNonEmptyLines(wp.Description.Raw)
	if lines < 3 {
		return Result{
			Name:    "Description has substance",
			Level:   Fail,
			Message: "Description too short (less than 3 lines of content)",
		}
	}
	return Result{
		Name:    "Description has substance",
		Level:   Pass,
		Message: "",
	}
}

// CheckAcceptanceCriteria checks acceptance criteria and rewards the BDD
// Given/When/Then form — the shared artifact business, PM, and dev all read.
// Pass when all three of given/when/then are present; Warn when criteria exist in
// some other shape (nudge toward G/W/T without blocking); Fail when there are none.
func CheckAcceptanceCriteria(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Acceptance criteria (Given/When/Then)"
	if wp.Description == nil {
		return Result{Name: name, Level: Fail, Message: "No description to check"}
	}
	raw := strings.ToLower(wp.Description.Raw)
	if strings.Contains(raw, "given") && strings.Contains(raw, "when") && strings.Contains(raw, "then") {
		return Result{Name: name, Level: Pass}
	}
	keywords := []string{"acceptance criteria", "ac:", "- [ ]", "- [x]", "given", "when", "then", "scenario"}
	for _, kw := range keywords {
		if strings.Contains(raw, kw) {
			return Result{Name: name, Level: Warn, Message: "Acceptance criteria present but not in Given/When/Then form"}
		}
	}
	return Result{Name: name, Level: Fail, Message: "No acceptance criteria section found"}
}

// CheckBusinessValue looks for the Impact Map "why/who" — a beneficiary and the
// outcome they gain — so the ticket reads meaningfully to business and product
// owners, not just engineers. Checks the User Story field and the description.
// Advisory only (Warn), never a hard failure.
func CheckBusinessValue(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Business value stated (who benefits / why)"
	var texts []string
	if wp.UserStory != nil {
		texts = append(texts, wp.UserStory.Raw)
	}
	if wp.Description != nil {
		texts = append(texts, wp.Description.Raw)
	}
	raw := strings.ToLower(strings.Join(texts, "\n"))
	keywords := []string{"so that", "in order to", "as a ", "why:", "impact", "benefit"}
	for _, kw := range keywords {
		if strings.Contains(raw, kw) {
			return Result{Name: name, Level: Pass}
		}
	}
	return Result{Name: name, Level: Warn, Message: "No business value / 'so that' clause found"}
}

// CheckUseCase looks for a user story, either in the dedicated User Story
// custom field (customField36) or as use-case/user-story text in the description.
func CheckUseCase(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Use case / user story present"
	if wp.UserStory != nil && strings.TrimSpace(wp.UserStory.Raw) != "" {
		return Result{Name: name, Level: Pass}
	}
	if wp.Description == nil {
		return Result{Name: name, Level: Fail, Message: "No description to check"}
	}
	raw := strings.ToLower(wp.Description.Raw)
	keywords := []string{"use case", "as a ", "user story", "scenario", "user flow"}
	for _, kw := range keywords {
		if strings.Contains(raw, kw) {
			return Result{Name: name, Level: Pass}
		}
	}
	return Result{Name: name, Level: Fail, Message: "No use case or user story section found"}
}

// CheckReproductionSteps looks for reproduction steps in bug descriptions.
func CheckReproductionSteps(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Reproduction steps present"
	if wp.Description == nil {
		return Result{Name: name, Level: Fail, Message: "No description to check"}
	}
	raw := strings.ToLower(wp.Description.Raw)
	keywords := []string{"steps to reproduce", "reproduce", "step 1", "expected behavior", "actual behavior", "expected:", "actual:"}
	for _, kw := range keywords {
		if strings.Contains(raw, kw) {
			return Result{Name: name, Level: Pass}
		}
	}
	return Result{Name: name, Level: Fail, Message: "No reproduction steps found"}
}

// CheckStoryPoints verifies story points are estimated.
func CheckStoryPoints(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Story points estimated"
	if wp.StoryPoints != nil && *wp.StoryPoints > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "Story points not estimated"}
}

// CheckAssignee verifies an assignee is set.
func CheckAssignee(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Assignee set"
	if wp.Links.Assignee.Href != "" {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No assignee set"}
}

// CheckPriority verifies priority is explicitly set (not default "Normal").
func CheckPriority(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Priority explicitly set"
	title := wp.Links.Priority.Title
	if title == "" {
		return Result{Name: name, Level: Warn, Message: "Priority not set"}
	}
	if strings.EqualFold(title, "Normal") {
		return Result{Name: name, Level: Warn, Message: "Priority is default (Normal)"}
	}
	return Result{Name: name, Level: Pass}
}

// CheckAttachments verifies attachments are present.
func CheckAttachments(_ *api.WorkPackage, in CheckInput) Result {
	name := "Has attachments"
	if in.AttachmentCount > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No attachments (mockups/screenshots recommended)"}
}

// CheckParentEpic verifies a parent work package is linked.
func CheckParentEpic(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Parent epic linked"
	if wp.Links.Parent.Href != "" {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No parent epic linked"}
}

// CheckComponent verifies at least one component is assigned.
func CheckComponent(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Component assigned"
	if len(wp.Links.Component) > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No component assigned (android/ios/ott/engineering/analytics)"}
}

// CheckWellFormed applies the QUS "well-formed" criterion: a user story should
// name a role and a means — the canonical "As a <role>, I want <means>" shape
// (the core check in the AQUSA tool). Advisory only (Pass/Warn, never Fail):
// plenty of valid features are not phrased as stories, so this nudges the writer
// toward the shared form rather than blocking readiness.
func CheckWellFormed(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Well-formed user story (role + means)"
	raw := storyText(wp)
	role := containsAny(raw, "as a ", "as an ", "as the ")
	means := containsAny(raw, "i want", "i need", "i'd like", "i would like", "would like to", "should be able to", "i can ", "wants to")
	switch {
	case role && means:
		return Result{Name: name, Level: Pass}
	case role || means:
		return Result{Name: name, Level: Warn, Message: "Only part of the 'As a <role>, I want <means>' form is present"}
	default:
		return Result{Name: name, Level: Warn, Message: "No 'As a <role>, I want <means>' user-story form found"}
	}
}

// CheckAtomic applies the QUS "atomic" criterion: a story should describe one
// feature. A coordinating conjunction in the story text often signals two
// features bundled together (AQUSA flags this heuristically). Advisory only —
// conjunctions are frequently innocent ("log in and out"), so it warns, never
// fails. Noisy by nature, so it is opt-in via DoR config, not a default check.
func CheckAtomic(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Atomic — one feature per story"
	raw := storyText(wp)
	if containsAny(raw, " and ", " & ", " and/or ", " or ") {
		return Result{Name: name, Level: Warn, Message: "Story text contains a conjunction — confirm it is not two features in one"}
	}
	return Result{Name: name, Level: Pass}
}

// CheckIndependent applies the INVEST "Independent" criterion — Atlassian's DoR
// "zero/minimal dependencies": a ticket blocked by other unfinished work is not
// truly ready to start. Advisory only (Warn, never Fail) — a dependency is a
// scheduling signal, not proof the ticket is malformed. The blocked-by count is
// computed by the runner from the work package's relations (see blockedByCount).
func CheckIndependent(_ *api.WorkPackage, in CheckInput) Result {
	name := "Independent (no blocking dependencies)"
	if in.BlockedByCount > 0 {
		return Result{Name: name, Level: Warn, Message: fmt.Sprintf("Blocked by %d other work package(s) — confirm they are done, not prerequisites", in.BlockedByCount)}
	}
	return Result{Name: name, Level: Pass}
}

// RulesForType returns the check functions for a work package type per the
// baked-in default Definition of Ready. Callers wanting a tuned rule set load a
// DoRConfig (see LoadDoR) and call its Rules method instead.
func RulesForType(typeName string) []CheckFunc {
	return defaultDoR.Rules(typeName)
}

// storyText joins the User Story field and the description (lowercased) — the
// text QUS criteria inspect.
func storyText(wp *api.WorkPackage) string {
	var parts []string
	if wp.UserStory != nil {
		parts = append(parts, wp.UserStory.Raw)
	}
	if wp.Description != nil {
		parts = append(parts, wp.Description.Raw)
	}
	return strings.ToLower(strings.Join(parts, "\n"))
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// countNonEmptyLines counts lines that have non-whitespace content.
func countNonEmptyLines(s string) int {
	n := 0
	for line := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}
