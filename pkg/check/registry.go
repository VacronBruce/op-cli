package check

// registry maps stable check IDs to their check functions. The IDs are the
// vocabulary a DoRConfig uses to select which checks apply to a work-package
// type — the tunable layer over the baked-in defaults. Adding a new check means
// registering it here so a config can reference it by ID.
var registry = map[string]CheckFunc{
	"description":         CheckDescription,
	"acceptance_criteria": CheckAcceptanceCriteria,
	"use_case":            CheckUseCase,
	"business_value":      CheckBusinessValue,
	"reproduction_steps":  CheckReproductionSteps,
	"story_points":        CheckStoryPoints,
	"assignee":            CheckAssignee,
	"priority":            CheckPriority,
	"attachments":         CheckAttachments,
	"parent_epic":         CheckParentEpic,
	"component":           CheckComponent,
	"well_formed":         CheckWellFormed, // QUS: role + means
	"atomic":              CheckAtomic,     // QUS: one feature per story (opt-in)
}

// CheckByID returns the check function registered under id, and whether it exists.
func CheckByID(id string) (CheckFunc, bool) {
	c, ok := registry[id]
	return c, ok
}
