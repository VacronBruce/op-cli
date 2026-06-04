package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// newMyCmd builds a cobra command carrying every flag runMy reads, so tests can
// exercise it the way the real `op my` is wired up in init().
func newMyCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().Bool("author", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().Bool("by-sprint", false, "")
	return cmd
}

func myWP(id int, subject string) api.WorkPackage {
	wp := api.WorkPackage{ID: id, Subject: subject}
	wp.Links.Status = api.Link{Title: "New"}
	return wp
}

// `op my` with no flags must show all my open work across every sprint: it must
// NOT resolve or apply a sprint filter, so work outside the active sprint shows
// up instead of the old confusing empty result.
func TestMy_DefaultShowsAllSprints(t *testing.T) {
	resolveCalled := false
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			resolveCalled = true
			return &api.Version{ID: 1, Name: "Sprint 1"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 2,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{myWP(1, "Task A"), myWP(2, "Task B")}},
			}, nil
		},
	}
	SetClient(mock)

	out := captureStdout(func() {
		if err := runMy(newMyCmd(), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if resolveCalled {
		t.Error("default `op my` must not resolve a sprint version")
	}
	if strings.Contains(out, "Sprint:") {
		t.Errorf("default `op my` must not print a Sprint: line, got: %s", out)
	}
	if !strings.Contains(out, "My items (2)") {
		t.Errorf("expected all 2 items, got: %s", out)
	}
}

// --sprint opts back in to sprint scoping: it resolves the version and prints
// the sprint header.
func TestMy_SprintFlagScopesToSprint(t *testing.T) {
	resolveCalled := false
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			resolveCalled = true
			return &api.Version{ID: 1, Name: "App_05/19/2026"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 1,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{myWP(1, "Task A")}},
			}, nil
		},
	}
	SetClient(mock)

	cmd := newMyCmd()
	_ = cmd.Flags().Set("sprint", "App_05/19/2026")

	out := captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !resolveCalled {
		t.Error("--sprint must resolve a sprint version")
	}
	if !strings.Contains(out, "Sprint: App_05/19/2026") {
		t.Errorf("expected sprint header, got: %s", out)
	}
}
