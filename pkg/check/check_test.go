package check

import (
	"fmt"
	"testing"

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
			r := CheckDescription(wp, 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

func TestCheckAcceptanceCriteria(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"no description", "", Fail},
		{"no criteria", "Just some text here", Fail},
		{"has AC keyword", "## Acceptance Criteria\n- item", Pass},
		{"has checkbox", "## Done when\n- [ ] thing works", Pass},
		{"has given/when/then", "Given a user\nWhen they click\nThen it works", Pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Feature", tt.desc)
			if tt.desc == "" {
				wp.Description = nil
			}
			r := CheckAcceptanceCriteria(wp, 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
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
			r := CheckUseCase(wp, 0)
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
			r := CheckUseCase(wp, 0)
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
			r := CheckReproductionSteps(wp, 0)
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
			r := CheckStoryPoints(wp, 0)
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
			r := CheckAssignee(wp, 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestCheckPriority(t *testing.T) {
	tests := []struct {
		name  string
		title string
		level Level
	}{
		{"default Normal", "Normal", Warn},
		{"empty", "", Warn},
		{"High", "High", Pass},
		{"Immediate", "Immediate", Pass},
		{"normal lowercase", "normal", Warn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := makeWP("Task", "desc\nline2\nline3")
			wp.Links.Priority = api.Link{Title: tt.title}
			r := CheckPriority(wp, 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
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
			r := CheckAttachments(wp, tt.count)
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
			r := CheckParentEpic(wp, 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s", r.Level, tt.level)
			}
		})
	}
}

func TestRulesForType(t *testing.T) {
	tests := []struct {
		typeName string
		count    int
	}{
		{"Bug", 7},
		{"Feature", 8},
		{"User Story", 8},
		{"Story", 8},
		{"Task", 6},
		{"Epic", 2},
		{"Unknown", 4},
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
