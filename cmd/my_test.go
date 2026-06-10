package cmd

import (
	"errors"
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

func TestMy_ByAuthor(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	cmd := newMyCmd()
	_ = cmd.Flags().Set("author", "true")

	out := captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	found := false
	for _, f := range gotFilters {
		if _, ok := f["author"]; ok {
			found = true
		}
	}
	if !found {
		t.Errorf("expected author filter, got %v", gotFilters)
	}

	if !strings.Contains(out, "Created by me") {
		t.Errorf("expected 'Created by me' in output, got: %s", out)
	}
}

func TestMy_ComponentFilter(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	cmd := newMyCmd()
	_ = cmd.Flags().Set("component", "android")

	captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	found := false
	for _, f := range gotFilters {
		if _, ok := f["customField12"]; ok { // customField12 is component
			found = true
		}
	}
	if !found {
		t.Errorf("expected customField12 filter, got %v", gotFilters)
	}
}

func TestMy_EmptyResult(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	out := captureStdout(func() {
		if err := runMy(newMyCmd(), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "My items (0)") {
		t.Errorf("expected 'My items (0)', got: %s", out)
	}
}

func TestMy_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return nil, errors.New("network timeout")
		},
	}
	SetClient(mock)

	captureStdout(func() {
		err := runMy(newMyCmd(), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "listing work packages") || !strings.Contains(err.Error(), "network timeout") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestMy_AllFlag(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	cmd := newMyCmd()
	_ = cmd.Flags().Set("all", "true")

	captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, f := range gotFilters {
		if _, ok := f["status"]; ok {
			t.Errorf("expected no status filter with --all, got %v", gotFilters)
		}
	}
}

func TestMy_BySprintFlag(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
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
	_ = cmd.Flags().Set("by-sprint", "true")

	out := captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Task A") {
		t.Errorf("expected output to contain 'Task A', got: %s", out)
	}
}

func TestMy_SinceFlag(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn:      func() (*api.User, error) { return &api.User{ID: 7}, nil },
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	cmd := newMyCmd()
	_ = cmd.Flags().Set("since", "2d")

	captureStdout(func() {
		if err := runMy(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	found := false
	for _, f := range gotFilters {
		if _, ok := f["createdAt"]; ok {
			found = true
		}
	}
	if !found {
		t.Errorf("expected createdAt filter with --since, got %v", gotFilters)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  int
		err   bool
	}{
		{"2d", 2, false},
		{"1w", 7, false},
		{"3m", 90, false},
		{"10x", 0, true},
		{"xyz", 0, true},
	}

	for _, tt := range tests {
		got, err := parseDuration(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseDuration(%q) error = %v, want err %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func newMyTeamCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("sprint", "", "")
	return cmd
}

func TestMyTeam_Success(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			wp := api.WorkPackage{ID: 100, Subject: "Team Task"}
			wp.Links.Assignee = api.Link{Title: "Alice"}
			return &api.WPCollection{
				Total: 1,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{wp}},
			}, nil
		},
	}
	SetClient(mock)

	cmd := newMyTeamCmd()
	_ = cmd.Flags().Set("sprint", "Sprint 24")

	out := captureStdout(func() {
		if err := runMyTeam(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Sprint: Sprint 24") {
		t.Errorf("expected sprint header, got: %s", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected assignee header, got: %s", out)
	}
	if !strings.Contains(out, "Team Task") {
		t.Errorf("expected task subject, got: %s", out)
	}
}

func TestMyTeam_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return nil, errors.New("internal error")
		},
	}
	SetClient(mock)

	cmd := newMyTeamCmd()
	_ = cmd.Flags().Set("sprint", "Sprint 24")

	captureStdout(func() {
		err := runMyTeam(cmd, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "listing work packages") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
