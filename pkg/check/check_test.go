package check

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
)

func intPtr(v int) *int { return &v }

func makeWP(typeName string, desc string, opts ...func(*api.WorkPackage)) *api.WorkPackage {
	wp := &api.WorkPackage{
		ID:      100,
		Subject: "Test ticket",
		Links: api.WPLinks{
			Type: api.Link{Title: typeName},
		},
	}
	if desc != "" {
		wp.Description = &api.Formattable{Raw: desc}
	}
	for _, opt := range opts {
		opt(wp)
	}
	return wp
}

func TestCheckDescription(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"nil description", "", Fail},
		{"empty description", "", Fail},
		{"one line", "Short desc", Fail},
		{"two lines", "Line one\nLine two", Fail},
		{"three lines", "Line one\nLine two\nLine three", Pass},
		{"with blank lines", "Line one\n\nLine two\n\nLine three", Pass},
		{"only blank lines", "\n\n\n", Fail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckDescription(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

// Acceptance criteria are the shared BDD artifact every role reads, so the check
// rewards the full Given/When/Then form (Pass), only nudges when criteria exist in
// some other shape (Warn), and fails when there are none at all (Fail). A partial
// G/W/T (missing "then") must Warn, not Pass — an incomplete scenario is exactly
// the ambiguity BDD is meant to remove.
func TestCheckAcceptanceCriteria(t *testing.T) {
	tests := []struct {
		name    string
		desc    string
		level   Level
		message string
	}{
		{"no description", "", Fail, "No description to check"},
		{"no criteria", "Just some text here", Fail, "No acceptance criteria section found"},
		{"AC heading only, no GWT", "## Acceptance Criteria\n- item", Warn, "Acceptance criteria present but not in Given/When/Then form"},
		{"checkbox only, no GWT", "## Done when\n- [ ] thing works", Warn, "Acceptance criteria present but not in Given/When/Then form"},
		{"partial given/when, no then", "Given a user\nWhen they click", Warn, "Acceptance criteria present but not in Given/When/Then form"},
		{"full given/when/then", "Given a user\nWhen they click\nThen it works", Pass, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Feature", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckAcceptanceCriteria(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
			if r.Message != tt.message {
				t.Errorf("got message %q, want %q", r.Message, tt.message)
			}
		})
	}
}

// The business "why/who" (Impact Map) must be legible to non-technical readers, so
// the check passes when the ticket names a beneficiary or the outcome they gain
// ("so that", "in order to", "as a"), whether that lives in the description or the
// User Story field, and warns (advisory, never blocks) when neither is present.
func TestCheckBusinessValue(t *testing.T) {
	tests := []struct {
		name      string
		desc      string
		userStory *api.Formattable
		level     Level
	}{
		{"nil description, no field", "", nil, Warn},
		{"plain text, no value", "Fix the login page", nil, Warn},
		{"has 'so that'", "As a user I want X so that Y", nil, Pass},
		{"has 'in order to'", "In order to reduce churn, add X", nil, Pass},
		{"value in user story field", "Fix login", &api.Formattable{Raw: "As a visitor so that checkout is faster"}, Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Feature", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			wp.UserStory = tt.userStory
			r := CheckBusinessValue(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

func TestCheckUseCase(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"no description", "", Fail},
		{"no use case", "Fix the login page", Fail},
		{"has 'as a'", "As a user, I want to log in", Pass},
		{"has 'use case'", "## Use Case\n1. Open app", Pass},
		{"has 'scenario'", "Scenario: user logs in", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Feature", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckUseCase(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

// The User Story custom field (customField36) satisfies the check even when the
// description has no user-story text, and an empty field falls back to the description.
func TestCheckUseCaseField(t *testing.T) {
	tests := []struct {
		name      string
		desc      string
		userStory *api.Formattable
		level     Level
	}{
		{"field set, weak desc", "Fix the login page", &api.Formattable{Raw: "As a visitor, I want X so that Y"}, Pass},
		{"field set, nil desc", "", &api.Formattable{Raw: "As a visitor, I want X"}, Pass},
		{"field empty, weak desc", "Fix the login page", &api.Formattable{Raw: "   "}, Fail},
		{"field whitespace, desc has story", "As a user, I want to log in", &api.Formattable{Raw: ""}, Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Story", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			wp.UserStory = tt.userStory
			r := CheckUseCase(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckReproductionSteps(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"no description", "", Fail},
		{"no steps", "App crashes sometimes", Fail},
		{"has steps to reproduce", "## Steps to Reproduce\n1. Open app", Pass},
		{"has expected/actual", "Expected: no crash\nActual: crash", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Bug", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckReproductionSteps(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckStoryPoints(t *testing.T) {
	tests := []struct {
		name   string
		points *int
		level  Level
	}{
		{"nil points", nil, Warn},
		{"zero points", intPtr(0), Warn},
		{"positive points", intPtr(5), Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", "desc\nline2\nline3")
			wp.StoryPoints = tt.points
			r := CheckStoryPoints(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckAssignee(t *testing.T) {
	tests := []struct {
		name  string
		href  string
		level Level
	}{
		{"no assignee", "", Warn},
		{"has assignee", "/api/v3/users/5", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", "desc\nline2\nline3")
			wp.Links.Assignee = api.Link{Href: tt.href}
			r := CheckAssignee(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckPriority(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		level   Level
		message string
	}{
		{"default Normal", "Normal", Warn, "Priority is default (Normal)"},
		// An unset priority must not be reported as "default (Normal)" —
		// the user needs to know the field is missing, not defaulted.
		{"empty", "", Warn, "Priority not set"},
		{"High", "High", Pass, ""},
		{"Immediate", "Immediate", Pass, ""},
		{"normal lowercase", "normal", Warn, "Priority is default (Normal)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", "desc\nline2\nline3")
			wp.Links.Priority = api.Link{Title: tt.title}
			r := CheckPriority(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
			if r.Message != tt.message {
				t.Errorf("got message %q, want %q", r.Message, tt.message)
			}
		})
	}
}

func TestCheckAttachments(t *testing.T) {
	tests := []struct {
		name  string
		count int
		level Level
	}{
		{"no attachments", 0, Warn},
		{"has attachments", 2, Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Bug", "desc\nline2\nline3")
			r := CheckAttachments(wp, CheckInput{AttachmentCount: tt.count})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckParentEpic(t *testing.T) {
	tests := []struct {
		name  string
		href  string
		level Level
	}{
		{"no parent", "", Warn},
		{"has parent", "/api/v3/work_packages/50", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", "desc\nline2\nline3")
			wp.Links.Parent = api.Link{Href: tt.href}
			r := CheckParentEpic(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

// CheckIndependent implements INVEST "Independent": a ticket blocked by other
// work is advisory-Warn (never Fail — a dependency is a scheduling fact, not a
// malformed ticket). Zero blockers passes.
func TestCheckIndependent(t *testing.T) {
	tests := []struct {
		name    string
		blocked int
		level   Level
	}{
		{"no blockers", 0, Pass},
		{"one blocker", 1, Warn},
		{"several blockers", 3, Warn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckIndependent(makeWP("Task", "l1\nl2\nl3"), CheckInput{BlockedByCount: tt.blocked})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

// blockedByCount must count only dependencies OF the queried ticket: the "to" end
// of a "blocks" relation, or the "from" end of a "blocked" one. A relation where
// the ticket is the blocker (it blocks others) must NOT count — that does not
// make the ticket itself un-ready — and unrelated relation types are ignored.
func TestBlockedByCount(t *testing.T) {
	const self = 42
	mkRel := func(typ string, from, to int) api.Relation {
		var rel api.Relation
		rel.Type = typ
		rel.Links.From.Href = fmt.Sprintf("/api/v3/work_packages/%d", from)
		rel.Links.To.Href = fmt.Sprintf("/api/v3/work_packages/%d", to)
		return rel
	}
	rc := &api.RelationCollection{}
	rc.Embedded.Elements = []api.Relation{
		mkRel("blocks", 7, self),   // 7 blocks self -> self is blocked (count)
		mkRel("blocks", self, 9),   // self blocks 9 -> does NOT count
		mkRel("blocked", self, 11), // self blocked by 11 -> count
		mkRel("relates", 3, self),  // unrelated type -> does NOT count
	}
	if got := blockedByCount(self, rc); got != 2 {
		t.Errorf("blockedByCount = %d, want 2", got)
	}
	if got := blockedByCount(self, nil); got != 0 {
		t.Errorf("blockedByCount(nil) = %d, want 0", got)
	}
}

// The runner must thread the blocked-by count from relations into the checks, so
// a ticket blocked by another surfaces the INVEST independence Warn.
func TestRunner_FlagsBlockedDependency(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return makeWP("Task", "l1\nl2\nl3"), nil
		},
		GetFn: func(path string, result interface{}) error { return nil },
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return &api.ActivityCollection{}, nil
		},
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) {
			rc := &api.RelationCollection{}
			rel := api.Relation{Type: "blocks"}
			rel.Links.From.Href = "/api/v3/work_packages/999"
			rel.Links.To.Href = fmt.Sprintf("/api/v3/work_packages/%d", wpID)
			rc.Embedded.Elements = []api.Relation{rel}
			return rc, nil
		},
	}
	runner := &Runner{Client: mock}
	report, err := runner.Run(100)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	var found bool
	for _, res := range report.Results {
		if res.Name == "Independent (no blocking dependencies)" {
			found = true
			if res.Level != Warn {
				t.Errorf("independence check = %v (%s), want Warn — ticket is blocked", res.Level, res.Message)
			}
		}
	}
	if !found {
		t.Fatal("independence check not present in task report")
	}
}

func TestRulesForType(t *testing.T) {
	tests := []struct {
		typeName string
		count    int
	}{
		{"Bug", 9},      // + INVEST no_blockers (advisory)
		{"Feature", 12}, // + QUS well_formed + INVEST no_blockers (advisory)
		{"User Story", 12},
		{"Story", 12},
		{"Task", 8}, // + INVEST no_blockers (advisory)
		{"Epic", 4},
		{"Unknown", 5},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			rules := RulesForType(tt.typeName)
			if len(rules) != tt.count {
				t.Errorf("got %d rules, want %d", len(rules), tt.count)
			}
		})
	}
}

func TestReportScore(t *testing.T) {
	r := &Report{
		Results: []Result{
			{Level: Pass},
			{Level: Pass},
			{Level: Fail},
			{Level: Warn},
			{Level: Pass},
		},
	}
	if r.Passed() != 3 {
		t.Errorf("Passed() = %d, want 3", r.Passed())
	}
	if r.Total() != 5 {
		t.Errorf("Total() = %d, want 5", r.Total())
	}
	if r.Score() != "3/5" {
		t.Errorf("Score() = %s, want 3/5", r.Score())
	}
	if !r.HasFailures() {
		t.Error("HasFailures() = false, want true")
	}
	// Deterministic Definition-of-Ready percent: Pass=100, Warn=50, Fail=0,
	// averaged. 3 Pass + 1 Warn + 1 Fail over 5 → (300 + 50) / 5 = 70.
	if r.ScorePercent() != 70 {
		t.Errorf("ScorePercent() = %d, want 70", r.ScorePercent())
	}
	// A Fail is a blocker: the DoR gate is NEEDS WORK.
	if r.Readiness() != "NEEDS WORK" {
		t.Errorf("Readiness() = %q, want NEEDS WORK", r.Readiness())
	}
}

func TestReportScorePercentAndReadiness(t *testing.T) {
	cases := []struct {
		name    string
		results []Result
		percent int
		ready   string
	}{
		{"all pass", []Result{{Level: Pass}, {Level: Pass}}, 100, "READY"},
		{"warn-only does not block gate", []Result{{Level: Pass}, {Level: Warn}}, 75, "READY"},
		{"any fail blocks gate", []Result{{Level: Pass}, {Level: Fail}}, 50, "NEEDS WORK"},
		{"empty report is ready at 0", nil, 0, "READY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{Results: tc.results}
			if got := r.ScorePercent(); got != tc.percent {
				t.Errorf("ScorePercent() = %d, want %d", got, tc.percent)
			}
			if got := r.Readiness(); got != tc.ready {
				t.Errorf("Readiness() = %q, want %q", got, tc.ready)
			}
		})
	}
}

func TestRunner(t *testing.T) {
	pts := 5
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return &api.WorkPackage{
				ID:      id,
				Subject: "Test bug",
				Description: &api.Formattable{
					Raw: "## Steps to Reproduce\n1. Open app\n2. Click button\n3. Crash\n\nExpected: no crash\nActual: crash",
				},
				StoryPoints: &pts,
				Links: api.WPLinks{
					Type:     api.Link{Title: "Bug"},
					Priority: api.Link{Title: "High"},
					Assignee: api.Link{Href: "/api/v3/users/5", Title: "Dev"},
					Parent:   api.Link{Href: "/api/v3/work_packages/10"},
				},
			}, nil
		},
		GetFn: func(path string, result interface{}) error {
			// Mock attachments response
			if att, ok := result.(*attachmentCollection); ok {
				att.Total = 1
			}
			return nil
		},
	}

	runner := &Runner{Client: mock}
	report, err := runner.Run(123)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if report.WPID != 123 {
		t.Errorf("WPID = %d, want 123", report.WPID)
	}
	if report.Type != "Bug" {
		t.Errorf("Type = %s, want Bug", report.Type)
	}
	// All checks should pass for this well-formed bug
	for _, r := range report.Results {
		if r.Level == Fail {
			t.Errorf("check %q failed: %s", r.Name, r.Message)
		}
	}
}

// TestRunner_CountsInlineCommentImages verifies the attachments check passes when
// the only screenshots live inline in a comment (Activity::Comment container),
// which the work package /attachments endpoint reports as zero.
func TestRunner_CountsInlineCommentImages(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return &api.WorkPackage{
				ID:      id,
				Subject: "Bug with screenshot in a comment",
				Links:   api.WPLinks{Type: api.Link{Title: "Bug"}},
			}, nil
		},
		GetFn: func(path string, result interface{}) error {
			if att, ok := result.(*attachmentCollection); ok {
				att.Total = 0 // no work-package-level attachments
			}
			return nil
		},
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			ac := &api.ActivityCollection{}
			ac.Embedded.Elements = []api.Activity{
				{ID: 1, Comment: &api.Formattable{
					Raw: `repro <img src="/api/v3/attachments/43221/content">`,
				}},
			}
			return ac, nil
		},
	}

	runner := &Runner{Client: mock}
	report, err := runner.Run(123)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	var found bool
	for _, r := range report.Results {
		if r.Name == "Has attachments" {
			found = true
			if r.Level != Pass {
				t.Errorf("attachments check = %v (%s), want Pass — inline comment image should count", r.Level, r.Message)
			}
		}
	}
	if !found {
		t.Fatal("attachments check not present in bug report")
	}
}

func TestRunnerError(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	runner := &Runner{Client: mock}
	_, err := runner.Run(999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- RunBatch concurrency ---

func batchMock(getWP func(id int) (*api.WorkPackage, error)) *testutil.MockClient {
	return &testutil.MockClient{
		GetWorkPackageFn: getWP,
		GetFn:            func(path string, result interface{}) error { return nil },
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return &api.ActivityCollection{}, nil
		},
	}
}

// Checks fetch concurrently, but the reports must come back in INPUT order —
// the sprint table reads top-to-bottom and diffs between runs must be stable.
func TestRunBatch_PreservesInputOrder(t *testing.T) {
	mock := batchMock(func(id int) (*api.WorkPackage, error) {
		wp := makeWP("Task", "l1\nl2\nl3")
		wp.ID = id
		wp.Subject = fmt.Sprintf("ticket-%d", id)
		return wp, nil
	})
	r := &Runner{Client: mock}

	wps := make([]api.WorkPackage, 20)
	for i := range wps {
		wps[i] = api.WorkPackage{ID: 1000 + i}
	}
	reports, err := r.RunBatch(wps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 20 {
		t.Fatalf("expected 20 reports, got %d", len(reports))
	}
	for i, rep := range reports {
		if rep.WPID != 1000+i {
			t.Fatalf("report %d is for #%d — order not preserved", i, rep.WPID)
		}
	}
}

// RunBatch must actually overlap requests: two checks block until BOTH have
// started; a sequential implementation deadlocks here (caught by the timeout).
func TestRunBatch_RunsConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	mock := batchMock(func(id int) (*api.WorkPackage, error) {
		started <- struct{}{}
		once.Do(func() {
			go func() {
				<-started
				<-started
				close(release)
			}()
		})
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			return nil, fmt.Errorf("timed out waiting for a second concurrent check — RunBatch is sequential")
		}
		wp := makeWP("Task", "l1\nl2\nl3")
		wp.ID = id
		return wp, nil
	})
	r := &Runner{Client: mock}

	_, err := r.RunBatch([]api.WorkPackage{{ID: 1}, {ID: 2}})
	if err != nil {
		t.Fatal(err)
	}
}

// The error contract is unchanged: any failed check fails the batch, naming
// the FIRST failing ticket in input order regardless of completion order.
func TestRunBatch_FirstErrorInInputOrderWins(t *testing.T) {
	mock := batchMock(func(id int) (*api.WorkPackage, error) {
		if id == 2 || id == 4 {
			return nil, fmt.Errorf("boom-%d", id)
		}
		wp := makeWP("Task", "l1\nl2\nl3")
		wp.ID = id
		return wp, nil
	})
	r := &Runner{Client: mock}

	_, err := r.RunBatch([]api.WorkPackage{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}})
	if err == nil || !strings.Contains(err.Error(), "checking #2") {
		t.Fatalf("expected first input-order error (#2), got: %v", err)
	}
}
