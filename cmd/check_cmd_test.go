package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func newCheckTestCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().Bool("sprint", false, "")
	c.Flags().String("sprint-name", "", "")
	c.Flags().Bool("strict", false, "")
	c.Flags().Bool("comment", false, "")
	c.Flags().String("component", "", "")
	return c
}

// checkableWP returns a Task with a description but no assignee/points, so the
// report predictably contains both passes and warnings.
func checkableWP(id int) *api.WorkPackage {
	wp := &api.WorkPackage{ID: id, Subject: "Check me"}
	wp.Links.Type = api.Link{Title: "Task"}
	wp.Links.Status = api.Link{Title: "New"}
	wp.Description = &api.Formattable{Raw: "line1\nline2\nline3"}
	return wp
}

func checkMock(wp *api.WorkPackage) *testutil.MockClient {
	return &testutil.MockClient{
		ProjectValue:     "app",
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) { return wp, nil },
		GetFn:            func(path string, result interface{}) error { return nil }, // attachments: zero
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return &api.ActivityCollection{}, nil
		},
	}
}

func TestCheck_SingleID_PrintsReport(t *testing.T) {
	// `op check <id>` is the pre-review gate: the report must show the ticket,
	// its score, and each rule line so the user can see WHAT is missing.
	SetClient(checkMock(checkableWP(81321)))

	out := testutil.CaptureStdout(func() {
		if err := runCheck(newCheckTestCmd(), []string{"81321"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "#81321 Check me") {
		t.Errorf("expected report header, got: %s", out)
	}
	if !strings.Contains(out, "Score:") {
		t.Errorf("expected score line, got: %s", out)
	}
	if !strings.Contains(out, "No assignee set") {
		t.Errorf("expected assignee warning for unassigned WP, got: %s", out)
	}
}

func TestCheck_InvalidAndMissingID(t *testing.T) {
	SetClient(&testutil.MockClient{})

	err := runCheck(newCheckTestCmd(), []string{"abc"})
	if err == nil || !strings.Contains(err.Error(), "invalid work package ID") {
		t.Fatalf("expected invalid-ID error, got: %v", err)
	}

	err = runCheck(newCheckTestCmd(), nil)
	if err == nil || !strings.Contains(err.Error(), "provide a work package ID or use --sprint") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

func TestCheck_CommentFlagPostsMarkdownReport(t *testing.T) {
	// --comment is how the review bot leaves results on the ticket: it must
	// post the MARKDOWN rendering to the checked WP, then confirm on stdout.
	var gotWP int
	var gotMD string
	mock := checkMock(checkableWP(81321))
	mock.PostCommentFn = func(wpID int, markdown string) error {
		gotWP, gotMD = wpID, markdown
		return nil
	}
	SetClient(mock)

	cmd := newCheckTestCmd()
	_ = cmd.Flags().Set("comment", "true")
	out := testutil.CaptureStdout(func() {
		if err := runCheck(cmd, []string{"81321"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if gotWP != 81321 {
		t.Errorf("expected comment on #81321, got #%d", gotWP)
	}
	if !strings.Contains(gotMD, "|") {
		t.Errorf("expected markdown table in posted comment, got: %s", gotMD)
	}
	if !strings.Contains(out, "Posted check results as comment on #81321") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}

func TestCheck_StrictPromotesWarningsToFailures(t *testing.T) {
	// --strict is the sprint-gate mode: a ticket that only has warnings must
	// stop scoring as clean, otherwise strict mode changes nothing.
	SetClient(checkMock(checkableWP(81321)))

	relaxed := testutil.CaptureStdout(func() { _ = runCheck(newCheckTestCmd(), []string{"81321"}) })

	cmd := newCheckTestCmd()
	_ = cmd.Flags().Set("strict", "true")
	strict := testutil.CaptureStdout(func() { _ = runCheck(cmd, []string{"81321"}) })

	if strings.Count(strict, "FAIL") <= strings.Count(relaxed, "FAIL") {
		t.Errorf("strict mode must promote warnings to failures.\nrelaxed:\n%s\nstrict:\n%s", relaxed, strict)
	}
}

func TestCheckSprint_BatchSummaryForResolvedSprint(t *testing.T) {
	// --sprint --sprint-name checks a whole sprint by name: each item goes
	// through the runner and the summary table headlines the sprint.
	wp := checkableWP(7)
	mock := checkMock(wp)
	mock.ResolveVersionFn = func(project, name string) (*api.Version, error) {
		if name != "Sprint 25" {
			t.Errorf("expected Sprint 25 resolution, got %q", name)
		}
		return &api.Version{ID: 12, Name: "Sprint 25"}, nil
	}
	mock.ListWorkPackagesFn = func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
		if spec, ok := filters[0]["version"]; !ok || spec.Values[0] != "12" {
			t.Errorf("expected version=12 filter, got %v", filters)
		}
		return &api.WPCollection{Total: 1, Embedded: struct {
			Elements []api.WorkPackage `json:"elements"`
		}{Elements: []api.WorkPackage{*wp}}}, nil
	}
	SetClient(mock)

	cmd := newCheckTestCmd()
	_ = cmd.Flags().Set("sprint", "true")
	_ = cmd.Flags().Set("sprint-name", "Sprint 25")
	out := testutil.CaptureStdout(func() {
		if err := runCheck(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Sprint Readiness: Sprint 25") {
		t.Errorf("expected sprint readiness header, got: %s", out)
	}
	if !strings.Contains(out, "Check me") {
		t.Errorf("expected checked item in summary, got: %s", out)
	}
}

func TestCheckSprint_RunnerErrorAborts(t *testing.T) {
	mock := checkMock(checkableWP(7))
	mock.GetWorkPackageFn = func(id int) (*api.WorkPackage, error) {
		return nil, errors.New("gone")
	}
	mock.FindActiveSprintFn = func(project string) (*api.Version, error) {
		return &api.Version{ID: 12, Name: "Sprint 25"}, nil
	}
	mock.ListWorkPackagesFn = func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
		return &api.WPCollection{Total: 1, Embedded: struct {
			Elements []api.WorkPackage `json:"elements"`
		}{Elements: []api.WorkPackage{{ID: 7}}}}, nil
	}
	SetClient(mock)

	cmd := newCheckTestCmd()
	_ = cmd.Flags().Set("sprint", "true")
	var err error
	testutil.CaptureStdout(func() { err = runCheck(cmd, nil) })
	if err == nil || !strings.Contains(err.Error(), "checking #7") {
		t.Fatalf("expected per-item wrapped error, got: %v", err)
	}
}
