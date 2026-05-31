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

// CheckAcceptanceCriteria looks for acceptance criteria in the description.
func CheckAcceptanceCriteria(wp *api.WorkPackage, _ int) Result {
	name := "Acceptance criteria present"
	if wp.Description == nil {
		return Result{Name: name, Level: Fail, Message: "No description to check"}
	}
	raw := strings.ToLower(wp.Description.Raw)
	keywords := []string{"acceptance criteria", "ac:", "- [ ]", "- [x]", "given", "when", "then"}
	for _, kw := range keywords {
		if strings.Contains(raw, kw) {
			return Result{Name: name, Level: Pass}
		}
	}
	return Result{Name: name, Level: Fail, Message: "No acceptance criteria section found"}
}

// CheckUseCase looks for use case or user story format in the description.
func CheckUseCase(wp *api.WorkPackage, _ int) Result {
	name := "Use case / user story present"
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
	if title != "" && !strings.EqualFold(title, "Normal") {
		return Result{Name: name, Level: Pass}
	}
	return Result{Name: name, Level: Warn, Message: "Priority is default (Normal)"}
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
		}
	case t == "feature" || t == "user story":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
			CheckUseCase,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckAttachments,
			CheckParentEpic,
		}
	case t == "task":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
			CheckParentEpic,
		}
	case t == "epic":
		return []CheckFunc{
			CheckDescription,
			CheckAcceptanceCriteria,
		}
	default:
		// Fallback: basic checks for unknown types
		return []CheckFunc{
			CheckDescription,
			CheckStoryPoints,
			CheckAssignee,
			CheckPriority,
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
