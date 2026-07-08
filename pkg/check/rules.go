package check

import (
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// CheckFunc is the signature for individual check functions.
type CheckFunc func(wp *api.WorkPackage, attachmentCount int) Result

// CheckDescription verifies the description exists and has substance.
func CheckDescription(wp *api.WorkPackage, _ int) Result {
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
func CheckAcceptanceCriteria(wp *api.WorkPackage, _ int) Result {
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
func CheckBusinessValue(wp *api.WorkPackage, _ int) Result {
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
func CheckUseCase(wp *api.WorkPackage, _ int) Result {
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
func CheckReproductionSteps(wp *api.WorkPackage, _ int) Result {
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
func CheckStoryPoints(wp *api.WorkPackage, _ int) Result {
	name := "Story points estimated"
	if wp.StoryPoints != nil && *wp.StoryPoints > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "Story points not estimated"}
}

// CheckAssignee verifies an assignee is set.
func CheckAssignee(wp *api.WorkPackage, _ int) Result {
	name := "Assignee set"
	if wp.Links.Assignee.Href != "" {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No assignee set"}
}

// CheckPriority verifies priority is explicitly set (not default "Normal").
func CheckPriority(wp *api.WorkPackage, _ int) Result {
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
func CheckAttachments(_ *api.WorkPackage, attachmentCount int) Result {
	name := "Has attachments"
	if attachmentCount > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No attachments (mockups/screenshots recommended)"}
}

// CheckParentEpic verifies a parent work package is linked.
func CheckParentEpic(wp *api.WorkPackage, _ int) Result {
	name := "Parent epic linked"
	if wp.Links.Parent.Href != "" {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No parent epic linked"}
}

// CheckComponent verifies at least one component is assigned.
func CheckComponent(wp *api.WorkPackage, _ int) Result {
	name := "Component assigned"
	if len(wp.Links.Component) > 0 {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "No component assigned (android/ios/ott/engineering/analytics)"}
}

// RulesForType returns the appropriate check functions for a work package type.
func RulesForType(typeName string) []CheckFunc {
	t := strings.ToLower(typeName)
	switch {
	case t == "bug":
		return []CheckFunc{
			CheckDescription,
			CheckReproductionSteps,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckAttachments,
			CheckParentEpic,
			CheckComponent,
		}
	case t == "feature" || t == "user story" || t == "story":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
			CheckUseCase,
			CheckBusinessValue,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckAttachments,
			CheckParentEpic,
			CheckComponent,
		}
	case t == "task":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckParentEpic,
			CheckComponent,
		}
	case t == "epic":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
			CheckBusinessValue,
			CheckComponent,
		}
	default:
		// Fallback: basic checks for unknown types
		return []CheckFunc{
			CheckDescription,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckComponent,
		}
	}
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
