package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// --- backlog --unestimated tests ---

func TestBacklog_Unestimated_FiltersCorrectly(t *testing.T) {
	pts5 := 5
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 3,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{
					Elements: []api.WorkPackage{
						{ID: 1, Subject: "Has points", StoryPoints: &pts5, Links: api.WPLinks{
							Type:   api.Link{Title: "Task"},
							Status: api.Link{Title: "New"},
						}},
						{ID: 2, Subject: "No points", Links: api.WPLinks{
							Type:   api.Link{Title: "Task"},
							Status: api.Link{Title: "New"},
						}},
						{ID: 3, Subject: "Zero points", StoryPoints: new(int), Links: api.WPLinks{
							Type:   api.Link{Title: "Bug"},
							Status: api.Link{Title: "New"},
						}},
					},
				},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().Set("unestimated", "true")

	out := captureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Unestimated backlog items (2)") {
		t.Errorf("expected 2 unestimated items, got: %s", out)
	}
	if strings.Contains(out, "Has points") {
		t.Errorf("should not include item with points, got: %s", out)
	}
	if !strings.Contains(out, "No points") {
		t.Errorf("should include item without points, got: %s", out)
	}
}

func TestBacklog_Unestimated_AllEstimated(t *testing.T) {
	pts3 := 3
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 1,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{
					Elements: []api.WorkPackage{
						{ID: 1, Subject: "Estimated", StoryPoints: &pts3, Links: api.WPLinks{
							Status: api.Link{Title: "New"},
						}},
					},
				},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().Set("unestimated", "true")

	out := captureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "All backlog items have estimates!") {
		t.Errorf("expected all-estimated message, got: %s", out)
	}
}

// --- board --status tests ---

func TestBoard_StatusFilter(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return &api.Version{
				ID:   1,
				Name: "Sprint 1",
				Links: struct {
					Self            api.Link `json:"self"`
					DefiningProject api.Link `json:"definingProject"`
				}{Self: api.Link{Href: "/api/v3/versions/1"}},
			}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			return &api.WPCollection{
				Total: 3,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{
					Elements: []api.WorkPackage{
						{ID: 1, Subject: "Blocked task", Links: api.WPLinks{
							Status:   api.Link{Title: "Blocked"},
							Assignee: api.Link{Title: "Alice"},
							Priority: api.Link{Title: "High"},
						}},
						{ID: 2, Subject: "Open task", Links: api.WPLinks{
							Status:   api.Link{Title: "New"},
							Assignee: api.Link{Title: "Bob"},
							Priority: api.Link{Title: "Normal"},
						}},
						{ID: 3, Subject: "Another blocked", Links: api.WPLinks{
							Status:   api.Link{Title: "Blocked"},
							Assignee: api.Link{Title: "Carol"},
							Priority: api.Link{Title: "Normal"},
						}},
					},
				},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().String("label", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Set("status", "blocked")

	out := captureStdout(func() {
		err := runBoard(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Blocked task") {
		t.Errorf("expected 'Blocked task' in output, got: %s", out)
	}
	if !strings.Contains(out, "Another blocked") {
		t.Errorf("expected 'Another blocked' in output, got: %s", out)
	}
	if strings.Contains(out, "Open task") {
		t.Errorf("should not include 'Open task' when filtering by blocked, got: %s", out)
	}
}

// --- assign deprecation test ---

func TestAssign_IsDeprecated(t *testing.T) {
	if !assignCmd.Hidden {
		t.Error("assign command should be hidden")
	}
	if assignCmd.Deprecated == "" {
		t.Error("assign command should have deprecation message")
	}
	if !strings.Contains(assignCmd.Deprecated, "op update") {
		t.Errorf("deprecation message should mention 'op update', got: %s", assignCmd.Deprecated)
	}
}

// --- report deprecation test ---

func TestReport_IsDeprecated(t *testing.T) {
	if !reportCmd.Hidden {
		t.Error("report command should be hidden")
	}
	if reportCmd.Deprecated == "" {
		t.Error("report command should have deprecation message")
	}
	if !strings.Contains(reportCmd.Deprecated, "sprint progress") {
		t.Errorf("deprecation message should mention 'sprint progress', got: %s", reportCmd.Deprecated)
	}
}

// --- sprint plan deprecation test ---

func TestSprintPlan_IsDeprecated(t *testing.T) {
	if !sprintPlanCmd.Hidden {
		t.Error("sprint plan command should be hidden")
	}
	if sprintPlanCmd.Deprecated == "" {
		t.Error("sprint plan command should have deprecation message")
	}
	if !strings.Contains(sprintPlanCmd.Deprecated, "op backlog") {
		t.Errorf("deprecation message should mention 'op backlog', got: %s", sprintPlanCmd.Deprecated)
	}
}

// --- my team subcommand test ---

func TestMyTeam_IsSubcommand(t *testing.T) {
	// Check that 'team' is a subcommand of 'my'
	found := false
	for _, sub := range myCmd.Commands() {
		if sub.Use == "team" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'team' should be a subcommand of 'my'")
	}
}

func TestMyTeamAlias_IsDeprecated(t *testing.T) {
	if !myTeamAliasCmd.Hidden {
		t.Error("my-team alias should be hidden")
	}
	if myTeamAliasCmd.Deprecated == "" {
		t.Error("my-team alias should have deprecation message")
	}
	if !strings.Contains(myTeamAliasCmd.Deprecated, "op my team") {
		t.Errorf("deprecation message should mention 'op my team', got: %s", myTeamAliasCmd.Deprecated)
	}
}
