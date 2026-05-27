package display

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

func TestBoard_Empty(t *testing.T) {
	out := captureOutput(func() {
		Board(nil)
	})
	if !strings.Contains(out, "No work packages in current sprint") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestBoard_StatusGrouping(t *testing.T) {
	wps := []api.WorkPackage{
		{ID: 1, Subject: "Task A", Links: api.WPLinks{
			Status:   api.Link{Title: "New"},
			Assignee: api.Link{Title: "Alice"},
			Priority: api.Link{Title: "High"},
		}},
		{ID: 2, Subject: "Task B", Links: api.WPLinks{
			Status:   api.Link{Title: "In progress"},
			Assignee: api.Link{Title: "Bob"},
			Priority: api.Link{Title: "Normal"},
		}},
		{ID: 3, Subject: "Task C", Links: api.WPLinks{
			Status:   api.Link{Title: "New"},
			Assignee: api.Link{Title: "Alice"},
			Priority: api.Link{Title: "Low"},
		}},
	}

	out := captureOutput(func() {
		Board(wps)
	})

	if !strings.Contains(out, "New (2)") {
		t.Errorf("expected 'New (2)', got: %s", out)
	}
	if !strings.Contains(out, "In progress (1)") {
		t.Errorf("expected 'In progress (1)', got: %s", out)
	}
}

func TestBoard_Summary(t *testing.T) {
	pts3 := 3
	pts5 := 5
	wps := []api.WorkPackage{
		{ID: 1, Subject: "Done task", StoryPoints: &pts3, Links: api.WPLinks{
			Status:   api.Link{Title: "Closed"},
			Assignee: api.Link{Title: "Alice"},
			Priority: api.Link{Title: "High"},
		}},
		{ID: 2, Subject: "Open task", StoryPoints: &pts5, Links: api.WPLinks{
			Status:   api.Link{Title: "New"},
			Assignee: api.Link{Title: "Bob"},
			Priority: api.Link{Title: "Normal"},
		}},
	}

	out := captureOutput(func() {
		Board(wps)
	})

	if !strings.Contains(out, "1/2 items done") {
		t.Errorf("expected '1/2 items done', got: %s", out)
	}
	if !strings.Contains(out, "3/8 points") {
		t.Errorf("expected '3/8 points', got: %s", out)
	}
}
