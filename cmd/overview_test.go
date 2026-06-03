package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func mkOverviewWP(proj, sprint, status, updated string) api.WorkPackage {
	wp := api.WorkPackage{UpdatedAt: updated}
	wp.Links.Project = api.Link{Title: proj}
	wp.Links.Version = api.Link{Title: sprint}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

func TestOverview_RendersCrossProject(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		GetMeFn: func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListAllWorkPackagesFn: func(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			col := &api.WPCollection{Total: 2}
			col.Embedded.Elements = []api.WorkPackage{
				mkOverviewWP("app", "S1", "Blocked", "2026-06-03"),
				mkOverviewWP("web", "W1", "New", "2026-06-02"),
			}
			return col, nil
		},
	}
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().Int("projects", 5, "")
	c.Flags().Int("sprints", 3, "")

	out := captureStdout(func() {
		if err := runOverview(c, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"app", "web", "1 blocked"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}

	// The dashboard must query my work via the global endpoint (assignee=me).
	foundAssignee := false
	for _, f := range gotFilters {
		if spec, ok := f["assignee"]; ok {
			for _, v := range spec.Values {
				if v == "7" {
					foundAssignee = true
				}
			}
		}
	}
	if !foundAssignee {
		t.Errorf("expected an assignee=7 filter, got %+v", gotFilters)
	}
}

func TestOverview_RejectsZeroLimits(t *testing.T) {
	SetClient(&testutil.MockClient{})
	c := &cobra.Command{}
	c.Flags().Int("projects", 0, "")
	c.Flags().Int("sprints", 3, "")
	if err := runOverview(c, nil); err == nil {
		t.Fatal("expected error for --projects=0")
	}
}
