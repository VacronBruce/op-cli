package check

import (
	"fmt"
	"regexp"
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
	BlockedByCount        int
	ParentAttachmentCount int
	// HasVisualAttachment is true when the ticket or its parent carries an image
	// or video attachment.
	HasVisualAttachment bool
	ParentDescription   string
	DesignHosts         []string
	// OpenUpstream lists related work packages that look upstream (API, backend,
	// web) and are not yet ready.
	OpenUpstream []Upstream
}

type Upstream struct {
	ID     int
	Status string
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
	if hasGWT(wp.Description.Raw) {
		return Result{Name: name, Level: Pass}
	}
	if hasGenericAC(wp.Description.Raw) {
		return Result{Name: name, Level: Warn, Message: "Acceptance criteria present but not in Given/When/Then form"}
	}
	return Result{Name: name, Level: Fail, Message: "No acceptance criteria section found"}
}

var (
	gwtLine = func(word string) *regexp.Regexp {
		return regexp.MustCompile(`(?im)^[ \t>*_+\-\d.)]*` + word + `\b`)
	}
	givenRe, whenRe, thenRe = gwtLine("given"), gwtLine("when"), gwtLine("then")
	expectedRe              = regexp.MustCompile(`(?i)\bexpected\b`)
	actualRe                = regexp.MustCompile(`(?i)\bactual\b`)
	checklistRe             = regexp.MustCompile(`(?im)^\s*[-*]\s+\[[ x]\]`)
	tableSepRe              = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?\s*$`)
	inlineImageRe           = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	uiWordsRe               = regexp.MustCompile(`(?i)\b(ui|ux|design|layout|screen|page|button|icon|colou?r|font|banner|dark mode|tablet|ipad|carousel|menu|tab|header|footer|dialog|style|spacing|card|widget|onboarding)s?\b`)
	uiCJK                   = []string{"頁面", "設計", "设计", "樣式", "样式", "按鈕", "按钮", "版面", "介面", "界面"}
)

// hasGWT reports whether given, when and then each start a line, tolerating
// bullets, numbering, quote markers and markdown emphasis before the word.
func hasGWT(raw string) bool {
	return givenRe.MatchString(raw) && whenRe.MatchString(raw) && thenRe.MatchString(raw)
}

func hasGenericAC(raw string) bool {
	return containsAny(strings.ToLower(raw), "acceptance criteria", "ac:", "- [ ]", "- [x]", "given", "when", "then", "scenario")
}

func hasMarkdownTable(raw string) bool {
	lines := strings.Split(raw, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i-1], "|") && tableSepRe.MatchString(lines[i]) {
			return true
		}
	}
	return false
}

// CheckDoneCriteria is the task-shaped counterpart of CheckAcceptanceCriteria:
// a "Done when" list, parity table, regression note or checklist specifies a
// task as well as Given/When/Then does. Generic acceptance-criteria wording
// alone is a Warn.
func CheckDoneCriteria(wp *api.WorkPackage, _ CheckInput) Result {
	name := "Done criteria"
	if wp.Description == nil {
		return Result{Name: name, Level: Fail, Message: "No description to check"}
	}
	raw := wp.Description.Raw
	lower := strings.ToLower(raw)
	if hasGWT(raw) || containsAny(lower, "done when", "parity", "regression") ||
		hasMarkdownTable(raw) || len(checklistRe.FindAllString(raw, -1)) >= 2 {
		return Result{Name: name, Level: Pass}
	}
	if hasGenericAC(raw) {
		return Result{Name: name, Level: Warn, Message: "Criteria present but no Given/When/Then, 'Done when' list, table or checklist"}
	}
	return Result{Name: name, Level: Fail, Message: "No done criteria found"}
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
	raw := wp.Description.Raw
	if containsAny(strings.ToLower(raw), "steps to reproduce", "reproduce", "step 1", "expected behavior", "actual behavior", "expected:", "actual:") ||
		hasGWT(raw) || (expectedRe.MatchString(raw) && actualRe.MatchString(raw)) {
		return Result{Name: name, Level: Pass}
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
	if in.AttachmentCount+in.ParentAttachmentCount > 0 {
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
func CheckIndependent(wp *api.WorkPackage, in CheckInput) Result {
	name := "Independent (no blocking dependencies)"
	var msgs []string
	if in.BlockedByCount > 0 {
		msgs = append(msgs, fmt.Sprintf("Blocked by %d other work package(s) — confirm they are done, not prerequisites", in.BlockedByCount))
	}
	if status := wp.Links.Status.Title; strings.EqualFold(status, "Pending") || strings.EqualFold(status, "Needs Update") {
		msgs = append(msgs, fmt.Sprintf("Ticket status is %s", status))
	}
	if len(in.OpenUpstream) > 0 {
		parts := make([]string, len(in.OpenUpstream))
		for i, u := range in.OpenUpstream {
			parts[i] = fmt.Sprintf("#%d %s", u.ID, u.Status)
		}
		msgs = append(msgs, "Related upstream not done: "+strings.Join(parts, ", "))
	}
	if len(msgs) > 0 {
		return Result{Name: name, Level: Warn, Message: strings.Join(msgs, "; ")}
	}
	return Result{Name: name, Level: Pass}
}

// CheckDesignLink asks whether a UI change has a design: a link to a design
// host, an inline image, or an image/video attachment on the ticket or its
// parent. Advisory only — a missing design warns only when the subject reads
// like UI work.
func CheckDesignLink(wp *api.WorkPackage, in CheckInput) Result {
	name := "Design reference"
	text := in.ParentDescription
	if wp.Description != nil {
		text += "\n" + wp.Description.Raw
	}
	if in.HasVisualAttachment || inlineImageRe.MatchString(text) || linksToHost(text, in.DesignHosts) {
		return Result{Name: name, Level: Pass}
	}
	if uiWordsRe.MatchString(wp.Subject) || containsAny(wp.Subject, uiCJK...) {
		return Result{Name: name, Level: Warn, Message: "UI change with no design link, image or video attached"}
	}
	return Result{Name: name, Level: Pass, Message: "no UI change detected"}
}

func linksToHost(text string, hosts []string) bool {
	for _, h := range hosts {
		re := regexp.MustCompile(`(?i)https?://(?:[\w-]+\.)*` + regexp.QuoteMeta(h) + `\b`)
		if re.MatchString(text) {
			return true
		}
	}
	return false
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
