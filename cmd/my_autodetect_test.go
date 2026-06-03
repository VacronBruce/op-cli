package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func myAutoWP(proj, sprint, status, updated, projHref string) api.WorkPackage {
	wp := api.WorkPackage{UpdatedAt: updated}
	wp.Links.Project = api.Link{Title: proj, Href: projHref}
	wp.Links.Version = api.Link{Title: sprint}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

func TestPickTopBucket_MostItemsWins(t *testing.T) {
	wps := []api.WorkPackage{
		myAutoWP("web", "W1", "New", "2026-06-09", "/api/v3/projects/9"),
		myAutoWP("app", "A1", "New", "2026-06-01", "/api/v3/projects/382"),
		myAutoWP("app", "A1", "New", "2026-06-02", "/api/v3/projects/382"),
	}
	b := pickTopBucket(wps)
	if b.ProjectTitle != "app" || b.SprintTitle != "A1" {
		t.Fatalf("expected app/A1 (2 items), got %s/%s", b.ProjectTitle, b.SprintTitle)
	}
	if len(b.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(b.Items))
	}
	if b.ProjectID != "382" {
		t.Errorf("expected projectID 382 parsed from href, got %q", b.ProjectID)
	}
}

// Equal counts fall back to most-recent activity.
func TestPickTopBucket_TieBreaksByRecency(t *testing.T) {
	wps := []api.WorkPackage{
		myAutoWP("app", "Old", "New", "2026-01-01", "/api/v3/projects/1"),
		myAutoWP("web", "Fresh", "New", "2026-06-09", "/api/v3/projects/2"),
	}
	b := pickTopBucket(wps)
	if b.SprintTitle != "Fresh" {
		t.Errorf("expected recency tie-break to pick 'Fresh', got %q", b.SprintTitle)
	}
}

func TestPickTopBucket_NoSprintBucket(t *testing.T) {
	wps := []api.WorkPackage{myAutoWP("app", "", "New", "2026-06-01", "/api/v3/projects/1")}
	b := pickTopBucket(wps)
	if b.HasSprint || b.SprintTitle != "(no sprint)" {
		t.Errorf("expected (no sprint) bucket, got hasSprint=%v title=%q", b.HasSprint, b.SprintTitle)
	}
}

func TestLastPathSegment(t *testing.T) {
	cases := map[string]string{
		"/api/v3/projects/382":  "382",
		"/api/v3/projects/382/": "382",
		"app":                   "app",
		"":                      "",
	}
	for in, want := range cases {
		if got := lastPathSegment(in); got != want {
			t.Errorf("lastPathSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

// op my with no project must auto-detect (not error), show the busiest sprint,
// and recommend the cross-project command.
func TestMy_NoProject_AutoDetects(t *testing.T) {
	mock := &testutil.MockClient{
		// ProjectValue empty → RequireProject errors → auto-detect path.
		GetMeFn: func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListAllWorkPackagesFn: func(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			if pageSize != autoDetectSample {
				t.Errorf("expected sample size %d, got %d", autoDetectSample, pageSize)
			}
			col := &api.WPCollection{Total: 3}
			col.Embedded.Elements = []api.WorkPackage{
				myAutoWP("app", "Sprint A", "Blocked", "2026-06-03", "/api/v3/projects/382"),
				myAutoWP("app", "Sprint A", "New", "2026-06-02", "/api/v3/projects/382"),
				myAutoWP("web", "Web 1", "New", "2026-06-01", "/api/v3/projects/9"),
			}
			return col, nil
		},
	}
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().Bool("author", false, "")

	out := captureStdout(func() {
		if err := runMy(c, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "app") || !strings.Contains(out, "Sprint A") {
		t.Errorf("expected inferred app/Sprint A (most items), got:\n%s", out)
	}
	if strings.Contains(out, "Web 1") {
		t.Errorf("did not expect the smaller web bucket in the table, got:\n%s", out)
	}
	if !strings.Contains(out, "op overview") {
		t.Errorf("expected a recommendation to op overview, got:\n%s", out)
	}
	if !strings.Contains(out, "op my -p 382") {
		t.Errorf("expected a -p 382 recommendation, got:\n%s", out)
	}
}

func TestMy_NoProject_NoItems(t *testing.T) {
	mock := &testutil.MockClient{
		GetMeFn:               func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListAllWorkPackagesFn: func([]api.Filter, string, int) (*api.WPCollection, error) { return &api.WPCollection{}, nil },
	}
	SetClient(mock)
	c := &cobra.Command{}
	c.Flags().Bool("author", false, "")

	out := captureStdout(func() {
		if err := runMy(c, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No open work") {
		t.Errorf("expected a friendly empty message, got:\n%s", out)
	}
}
