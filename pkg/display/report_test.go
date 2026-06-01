package display

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

func TestSprintReport_Empty(t *testing.T) {
	out := captureOutput(func() {
		SprintReport(nil, "Sprint 1", "2026-01-01", "2026-01-14")
	})
	if !strings.Contains(out, "# Sprint Report: Sprint 1") {
		t.Errorf("expected report header, got: %s", out)
	}
	if !strings.Contains(out, "Period: 2026-01-01 to 2026-01-14") {
		t.Errorf("expected period, got: %s", out)
	}
}

func TestSprintReport_WithItems(t *testing.T) {
	pts3 := 3
	pts5 := 5
	wps := []api.WorkPackage{
		{ID: 1, Subject: "Done task", StoryPoints: &pts3, Links: api.WPLinks{
			Status:   api.Link{Title: "Closed"},
			Assignee: api.Link{Title: "Alice"},
		}},
		{ID: 2, Subject: "Active task", StoryPoints: &pts5, Links: api.WPLinks{
			Status:   api.Link{Title: "In progress"},
			Assignee: api.Link{Title: "Bob"},
		}},
		{ID: 3, Subject: "Blocked task", Links: api.WPLinks{
			Status:   api.Link{Title: "Blocked"},
			Assignee: api.Link{Title: "Carol"},
		}},
		{ID: 4, Subject: "New task", Links: api.WPLinks{
			Status: api.Link{Title: "New"},
		}},
	}

	out := captureOutput(func() {
		SprintReport(wps, "Sprint 1", "2026-01-01", "2026-01-14")
	})

	if !strings.Contains(out, "## Completed (1)") {
		t.Errorf("expected Completed section, got: %s", out)
	}
	if !strings.Contains(out, "## In Progress (1)") {
		t.Errorf("expected In Progress section, got: %s", out)
	}
	if !strings.Contains(out, "## Blocked (1)") {
		t.Errorf("expected Blocked section, got: %s", out)
	}
	if !strings.Contains(out, "## Not Started (1)") {
		t.Errorf("expected Not Started section, got: %s", out)
	}
	if !strings.Contains(out, "Progress:") {
		t.Errorf("expected progress bar, got: %s", out)
	}
	if !strings.Contains(out, "3/8 points") {
		t.Errorf("expected point tally, got: %s", out)
	}
	if !strings.Contains(out, "@Alice") {
		t.Errorf("expected @Alice in output, got: %s", out)
	}
	if !strings.Contains(out, "@unassigned") {
		t.Errorf("expected @unassigned for task 4, got: %s", out)
	}
}

func TestSprintReport_NoDates(t *testing.T) {
	out := captureOutput(func() {
		SprintReport(nil, "Sprint X", "", "")
	})
	if strings.Contains(out, "Period:") {
		t.Errorf("should not show period when dates are empty, got: %s", out)
	}
}
