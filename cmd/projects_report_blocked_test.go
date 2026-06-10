package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// --- projects ---

func TestProjects_RendersIdentifierAndActiveColumns(t *testing.T) {
	// The IDENTIFIER column is what users feed to -p; ACTIVE distinguishes
	// archived boards that would reject writes.
	mock := &testutil.MockClient{
		ListProjectsFn: func() (*api.ProjectCollection, error) {
			col := &api.ProjectCollection{Total: 2}
			col.Embedded.Elements = []api.Project{
				{ID: 1, Name: "App", Identifier: "app", Active: true},
				{ID: 2, Name: "Old Board", Identifier: "old", Active: false},
			}
			return col, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runProjects(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "app") || !strings.Contains(out, "yes") {
		t.Errorf("expected active project row, got: %s", out)
	}
	if !strings.Contains(out, "old") || !strings.Contains(out, "no") {
		t.Errorf("expected inactive project marked 'no', got: %s", out)
	}
}

func TestProjects_EmptyAndError(t *testing.T) {
	mock := &testutil.MockClient{
		ListProjectsFn: func() (*api.ProjectCollection, error) {
			return &api.ProjectCollection{Total: 0}, nil
		},
	}
	SetClient(mock)
	out := testutil.CaptureStdout(func() {
		if err := runProjects(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No projects found.") {
		t.Errorf("expected empty message, got: %s", out)
	}

	mock = &testutil.MockClient{
		ListProjectsFn: func() (*api.ProjectCollection, error) {
			return nil, errors.New("boom")
		},
	}
	SetClient(mock)
	err := runProjects(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "listing projects") {
		t.Fatalf("expected wrapped error, got: %v", err)
	}
}

// --- report (deprecated, delegates to display.SprintReport) ---

func TestReport_RendersSprintReportForActiveSprint(t *testing.T) {
	// Deprecated alias of `sprint progress --verbose`: until removed it must
	// still resolve the ACTIVE sprint and render the report for its items.
	mock := &testutil.MockClient{
		ProjectValue: "app",
		FindActiveSprintFn: func(project string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24", StartDate: "2026-06-01", EndDate: "2026-06-14"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			if spec, ok := filters[0]["version"]; !ok || spec.Values[0] != "11" {
				t.Errorf("expected version=11 filter, got %v", filters)
			}
			return &api.WPCollection{
				Total: 1,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{sprintWP(1, "report item", "Closed", intPtr(2))}},
			}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runReport(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Sprint 24") || !strings.Contains(out, "report item") {
		t.Errorf("expected sprint report content, got: %s", out)
	}
}

// --- blocked ---

func TestBlocked_FiltersBlockedStatusCaseInsensitively(t *testing.T) {
	// "Blocked" is matched on the human status title, which differs in casing
	// across OP instances; the filter must not depend on exact case.
	mock := &testutil.MockClient{
		ProjectValue: "app",
		FindActiveSprintFn: func(project string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 3,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{
					sprintWP(1, "stuck", "BLOCKED", nil),
					sprintWP(2, "fine", "In progress", nil),
					sprintWP(3, "also stuck", "blocked", nil),
				}},
			}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runBlocked(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Blocked items in Sprint 24 (2):") {
		t.Errorf("expected 2 blocked items, got: %s", out)
	}
	if strings.Contains(out, "fine") {
		t.Errorf("non-blocked item leaked into output: %s", out)
	}
}

func TestBlocked_NoneBlocked(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		FindActiveSprintFn: func(project string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{Total: 1, Embedded: struct {
				Elements []api.WorkPackage `json:"elements"`
			}{Elements: []api.WorkPackage{sprintWP(1, "fine", "In progress", nil)}}}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runBlocked(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No blocked items in Sprint 24") {
		t.Errorf("expected no-blocked message, got: %s", out)
	}
}
