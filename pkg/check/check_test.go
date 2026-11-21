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
		{"bulleted bold given/when/then", "- **Given** a user\n- **When** they click\n- **Then** it works", Pass, ""},
		{"numbered given/when/then", "1. Given a user\n2. When they click\n3. Then it works", Pass, ""},
		{"quoted given/when/then, mixed case", "> GIVEN a user\n> when they click\n> Then it works", Pass, ""},
		{"given/when/then words only in prose", "The user is given a card when they click, then it flips.", Warn, "Acceptance criteria present but not in Given/When/Then form"},
		{"labelled inline scenario", "- **Happy path**: Given a user, when they click, then it works", Pass, ""},
		{"when/then before given on a line", "When ready, then ship it, given time", Warn, "Acceptance criteria present but not in Given/When/Then form"},
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
		{"line-anchored given/when/then", "- **Given** a logged-in user\n- **When** they open settings\n- **Then** the app crashes", Pass},
		{"expected and actual in prose", "I expected the list to refresh but the actual result was a blank screen", Pass},
		{"only expected marker", "I expected the list to refresh", Fail},
		{"inline scenario sentence", "Given a user, when they tap it, then it hangs", Pass},
		{"labelled inline scenarios", "## Scenarios\n- **Cold start**: Given a signed-out user, when they open the app, then the home feed loads.\n- **Offline**: Given no network, when they pull to refresh, then an error banner shows.", Pass},
		{"given/when/then words mid-sentence", "The user is given a card when they tap, then it hangs", Fail},
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
// work, parked in Pending/Needs Update, or waiting on an unfinished upstream
// (API/backend/web) ticket is advisory-Warn, never Fail.
func TestCheckIndependent(t *testing.T) {
	withStatus := func(title string) func(*api.WorkPackage) {
		return func(wp *api.WorkPackage) { wp.Links.Status.Title = title }
	}
	tests := []struct {
		name    string
		opts    []func(*api.WorkPackage)
		in      CheckInput
		level   Level
		message string
	}{
		{"no blockers", nil, CheckInput{}, Pass, ""},
		{"one blocker", nil, CheckInput{BlockedByCount: 1}, Warn, "Blocked by 1 other work package(s)"},
		{"several blockers", nil, CheckInput{BlockedByCount: 3}, Warn, "Blocked by 3 other work package(s)"},
		{"own status pending", []func(*api.WorkPackage){withStatus("Pending")}, CheckInput{}, Warn, "Pending"},
		{"own status needs update, any case", []func(*api.WorkPackage){withStatus("needs update")}, CheckInput{}, Warn, "needs update"},
		{"own status in progress", []func(*api.WorkPackage){withStatus("In Progress")}, CheckInput{}, Pass, ""},
		{"open upstream names each", nil, CheckInput{OpenUpstream: []Upstream{{ID: 201, Status: "To Do"}, {ID: 202, Status: "Needs Update"}}}, Warn, "Related upstream not done: #201 To Do, #202 Needs Update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckIndependent(makeWP("Task", "l1\nl2\nl3", tt.opts...), tt.in)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
			if !strings.Contains(r.Message, tt.message) {
				t.Errorf("message %q does not contain %q", r.Message, tt.message)
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
		{"Bug", 10},     // + INVEST no_blockers + design_link (advisory)
		{"Feature", 13}, // + QUS well_formed + no_blockers + design_link (advisory)
		{"User Story", 13},
		{"Story", 13},
		{"Task", 9}, // done_criteria replaces acceptance_criteria; + no_blockers + design_link
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
		GetFn: func(path string, result interface{}) error { return nil },
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			return &api.AttachmentCollection{Total: 1}, nil
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
		GetFn: func(path string, result interface{}) error { return nil },
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			return &api.AttachmentCollection{}, nil
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

func TestCheckDoneCriteria(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"no description", "", Fail},
		{"nothing", "Just some text here", Fail},
		{"line-anchored given/when/then", "Given a user\nWhen they click\nThen it works", Pass},
		{"done when heading", "## Done when\nthe screen matches", Pass},
		{"parity", "Reach parity with the other platform", Pass},
		{"regression", "No regression in login", Pass},
		{"markdown table", "| Field | Old | New |\n| --- | --- | --- |\n| Title | a | b |", Pass},
		{"table without separator row", "| Field | Old | New |\n| Title | a | b |", Fail},
		{"two checklist items", "- [ ] one\n- [x] two", Pass},
		{"one checklist item", "- [ ] one", Warn},
		{"generic acceptance criteria heading", "## Acceptance Criteria\n- item", Warn},
		{"given/when/then words only in prose", "A user is given a card when they click, then it flips.", Warn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckDoneCriteria(wp, CheckInput{})
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

func TestDoneCriteriaRegistered(t *testing.T) {
	if _, ok := CheckByID("done_criteria"); !ok {
		t.Error("done_criteria not registered")
	}
	if _, ok := CheckByID("design_link"); !ok {
		t.Error("design_link not registered")
	}
}

func TestDefaultDoR_TaskUsesDoneCriteria(t *testing.T) {
	ids := defaultDoR.Types["task"]
	has := func(id string) bool {
		for _, x := range ids {
			if x == id {
				return true
			}
		}
		return false
	}
	if has("acceptance_criteria") || !has("done_criteria") {
		t.Errorf("task default = %v, want done_criteria instead of acceptance_criteria", ids)
	}
	for _, typ := range []string{"bug", "feature", "task"} {
		found := false
		for _, id := range defaultDoR.Types[typ] {
			found = found || id == "design_link"
		}
		if !found {
			t.Errorf("%s default lacks design_link", typ)
		}
	}
}

func TestCheckAttachments_CountsParent(t *testing.T) {
	r := CheckAttachments(makeWP("Task", "d"), CheckInput{ParentAttachmentCount: 2})
	if r.Level != Pass {
		t.Errorf("got %s (%s), want Pass: parent mockups count", r.Level, r.Message)
	}
}

func TestCheckDesignLink(t *testing.T) {
	subject := func(s string) func(*api.WorkPackage) {
		return func(wp *api.WorkPackage) { wp.Subject = s }
	}
	figma := []string{"figma.com"}
	tests := []struct {
		name    string
		desc    string
		subject string
		in      CheckInput
		level   Level
	}{
		{"figma link in description", "see https://www.figma.com/design/abc/file", "Add banner", CheckInput{DesignHosts: figma}, Pass},
		{"figma link in parent description", "no link", "Add banner", CheckInput{DesignHosts: figma, ParentDescription: "Design: https://figma.com/design/x"}, Pass},
		{"custom design host", "mock at https://zeplin.io/p/1", "Add banner", CheckInput{DesignHosts: []string{"zeplin.io"}}, Pass},
		{"other host is not a design", "see https://example.com/figma.com", "Add banner", CheckInput{DesignHosts: figma}, Warn},
		{"inline markdown image", "layout: ![mock](/api/v3/attachments/1/content)", "Add banner", CheckInput{DesignHosts: figma}, Pass},
		{"inline html image", `<figure class="image"><img class="op-uc-image" src="/api/v3/attachments/1/content"></figure>`, "Add banner", CheckInput{DesignHosts: figma}, Pass},
		{"inline html image in parent", "no link", "Add banner", CheckInput{DesignHosts: figma, ParentDescription: `<p><img src="/api/v3/attachments/2/content"></p>`}, Pass},
		{"image or video attachment", "no link", "Add banner", CheckInput{DesignHosts: figma, HasVisualAttachment: true}, Pass},
		{"ui subject with nothing", "no link", "Update settings screen", CheckInput{DesignHosts: figma}, Warn},
		{"cjk ui subject", "no link", "[App] 調整頁面", CheckInput{DesignHosts: figma}, Warn},
		{"dark mode subject", "no link", "Support dark mode", CheckInput{DesignHosts: figma}, Warn},
		{"non-ui subject", "no link", "Fix crash when syncing data", CheckInput{DesignHosts: figma}, Pass},
		{"table is not tab", "no link", "Migrate the orders table", CheckInput{DesignHosts: figma}, Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", tt.desc, subject(tt.subject))
			r := CheckDesignLink(wp, tt.in)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
	r := CheckDesignLink(makeWP("Task", "x", subject("Fix crash")), CheckInput{DesignHosts: figma})
	if r.Message != "no UI change detected" {
		t.Errorf("message = %q, want %q", r.Message, "no UI change detected")
	}
}

func resultByName(t *testing.T, rep *Report, name string) Result {
	t.Helper()
	for _, r := range rep.Results {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no result named %q in %v", name, rep.Results)
	return Result{}
}

func relatesRel(from, to int) api.Relation {
	rel := api.Relation{Type: "relates"}
	rel.Links.From.Href = fmt.Sprintf("/api/v3/work_packages/%d", from)
	rel.Links.To.Href = fmt.Sprintf("/api/v3/work_packages/%d", to)
	return rel
}

func statusWP(id int, subject, status string) api.WorkPackage {
	return api.WorkPackage{ID: id, Subject: subject, Links: api.WPLinks{Status: api.Link{Title: status}}}
}

func upstreamMock(related []api.WorkPackage, listCalls *int) *testutil.MockClient {
	return &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return makeWP("Task", "l1\nl2\nl3"), nil
		},
		GetFn:            func(path string, result interface{}) error { return nil },
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) { return &api.ActivityCollection{}, nil },
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) {
			rc := &api.RelationCollection{}
			for _, w := range related {
				rc.Embedded.Elements = append(rc.Embedded.Elements, relatesRel(wpID, w.ID))
			}
			return rc, nil
		},
		ListAllWorkPackagesFn: func(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			*listCalls++
			c := &api.WPCollection{}
			c.Embedded.Elements = related
			return c, nil
		},
	}
}

func TestRunner_WarnsOnUnfinishedRelatedUpstream(t *testing.T) {
	calls := 0
	mock := upstreamMock([]api.WorkPackage{
		statusWP(201, "[App/Web] Add endpoint", "To Do"),
		statusWP(202, "[App][Backend] Add field", "Needs Update"),
		statusWP(203, "Server API change", "Ready for Release"),
		statusWP(204, "Polish the app icon", "To Do"),
		statusWP(205, "WordPress template", "Done"),
	}, &calls)
	rep, err := (&Runner{Client: mock}).Run(100)
	if err != nil {
		t.Fatal(err)
	}
	r := resultByName(t, rep, "Independent (no blocking dependencies)")
	if r.Level != Warn {
		t.Fatalf("level = %s (%s), want Warn", r.Level, r.Message)
	}
	if !strings.Contains(r.Message, "Related upstream not done: #201 To Do, #202 Needs Update") {
		t.Errorf("message %q does not name the open upstream tickets", r.Message)
	}
	for _, unwanted := range []string{"#203", "#204", "#205"} {
		if strings.Contains(r.Message, unwanted) {
			t.Errorf("message %q should not mention %s", r.Message, unwanted)
		}
	}
	if calls != 1 {
		t.Errorf("related statuses fetched with %d list calls, want 1", calls)
	}
}

func TestRunner_RelatedUpstreamDonePasses(t *testing.T) {
	calls := 0
	mock := upstreamMock([]api.WorkPackage{statusWP(203, "Backend API change", "Completed")}, &calls)
	rep, err := (&Runner{Client: mock}).Run(100)
	if err != nil {
		t.Fatal(err)
	}
	if r := resultByName(t, rep, "Independent (no blocking dependencies)"); r.Level != Pass {
		t.Errorf("level = %s (%s), want Pass", r.Level, r.Message)
	}
}

func TestRunner_UpstreamFetchFailureIsNonFatal(t *testing.T) {
	calls := 0
	mock := upstreamMock([]api.WorkPackage{statusWP(201, "Backend API change", "To Do")}, &calls)
	mock.ListAllWorkPackagesFn = func([]api.Filter, string, int) (*api.WPCollection, error) {
		return nil, fmt.Errorf("boom")
	}
	rep, err := (&Runner{Client: mock}).Run(100)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if r := resultByName(t, rep, "Independent (no blocking dependencies)"); r.Level != Pass {
		t.Errorf("level = %s (%s), want Pass", r.Level, r.Message)
	}
}

func parentMock(parentDesc string, parentAtts, ownAtts []api.Attachment) *testutil.MockClient {
	atts := func(list []api.Attachment) *api.AttachmentCollection {
		c := &api.AttachmentCollection{Total: len(list)}
		c.Embedded.Elements = list
		return c
	}
	return &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			if id == 10 {
				return makeWP("Epic", parentDesc), nil
			}
			return makeWP("Task", "l1\nl2\nl3", func(wp *api.WorkPackage) {
				wp.Subject = "Fix data sync"
				wp.Links.Parent = api.Link{Href: "/api/v3/work_packages/10"}
			}), nil
		},
		GetFn:            func(path string, result interface{}) error { return nil },
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) { return &api.ActivityCollection{}, nil },
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			if wpID == 10 {
				return atts(parentAtts), nil
			}
			return atts(ownAtts), nil
		},
	}
}

func TestRunner_CountsParentAttachments(t *testing.T) {
	mock := parentMock("epic", []api.Attachment{{ID: 1, ContentType: "application/pdf"}}, nil)
	inner := mock.GetWorkPackageFn
	mock.GetWorkPackageFn = func(id int) (*api.WorkPackage, error) {
		wp, err := inner(id)
		wp.Links.Type.Title = "Bug"
		return wp, err
	}
	rep, err := (&Runner{Client: mock}).Run(100)
	if err != nil {
		t.Fatal(err)
	}
	if r := resultByName(t, rep, "Has attachments"); r.Level != Pass {
		t.Errorf("attachments = %s (%s), want Pass from parent", r.Level, r.Message)
	}
}

func TestRunner_ParentFetchFailureIsNonFatal(t *testing.T) {
	mock := parentMock("epic", nil, nil)
	mock.ListAttachmentsFn = func(int) (*api.AttachmentCollection, error) { return nil, fmt.Errorf("boom") }
	if _, err := (&Runner{Client: mock}).Run(100); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
}

func TestRunner_DesignLinkSources(t *testing.T) {
	tests := []struct {
		name   string
		runner func(*testutil.MockClient) *Runner
		mock   *testutil.MockClient
		level  Level
	}{
		{"parent description has figma link by default", func(m *testutil.MockClient) *Runner { return &Runner{Client: m} },
			parentMock("Mockups: https://www.figma.com/design/abc", nil, nil), Pass},
		{"parent video attachment", func(m *testutil.MockClient) *Runner { return &Runner{Client: m} },
			parentMock("epic", []api.Attachment{{ID: 1, ContentType: "video/mp4"}}, nil), Pass},
		{"own image attachment", func(m *testutil.MockClient) *Runner { return &Runner{Client: m} },
			parentMock("epic", nil, []api.Attachment{{ID: 2, ContentType: "image/png"}}), Pass},
		{"configured host", func(m *testutil.MockClient) *Runner { return &Runner{Client: m, DesignHosts: []string{"zeplin.io"}} },
			parentMock("see https://zeplin.io/p/1", nil, nil), Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep, err := tt.runner(tt.mock).Run(100)
			if err != nil {
				t.Fatal(err)
			}
			if r := resultByName(t, rep, "Design reference"); r.Level != tt.level {
				t.Errorf("design = %s (%s), want %s", r.Level, r.Message, tt.level)
			}
		})
	}
}

func TestRunner_UIWorkWithoutDesignWarns(t *testing.T) {
	mock := parentMock("epic", nil, nil)
	inner := mock.GetWorkPackageFn
	mock.GetWorkPackageFn = func(id int) (*api.WorkPackage, error) {
		wp, err := inner(id)
		if id != 10 {
			wp.Subject = "Redesign the settings screen"
		}
		return wp, err
	}
	rep, err := (&Runner{Client: mock}).Run(100)
	if err != nil {
		t.Fatal(err)
	}
	if r := resultByName(t, rep, "Design reference"); r.Level != Warn {
		t.Errorf("design = %s (%s), want Warn", r.Level, r.Message)
	}
}
