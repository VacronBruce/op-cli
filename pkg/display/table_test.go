package display

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestWorkPackageTable_Empty(t *testing.T) {
	out := captureOutput(func() {
		WorkPackageTable(nil)
	})
	if !strings.Contains(out, "No work packages found") {
		t.Errorf("expected 'No work packages found', got: %s", out)
	}
}

func TestWorkPackageTable_SingleItem(t *testing.T) {
	wps := []api.WorkPackage{
		{
			ID:      42,
			Subject: "Fix login bug",
			Links: api.WPLinks{
				Type:     api.Link{Title: "Bug"},
				Status:   api.Link{Title: "In Progress"},
				Priority: api.Link{Title: "High"},
				Assignee: api.Link{Title: "Alice"},
			},
		},
	}

	out := captureOutput(func() {
		WorkPackageTable(wps)
	})

	if !strings.Contains(out, "42") {
		t.Error("expected ID 42 in output")
	}
	if !strings.Contains(out, "Bug") {
		t.Error("expected 'Bug' type in output")
	}
	if !strings.Contains(out, "In Progress") {
		t.Error("expected 'In Progress' status in output")
	}
	if !strings.Contains(out, "Alice") {
		t.Error("expected 'Alice' assignee in output")
	}
	if !strings.Contains(out, "Fix login bug") {
		t.Error("expected subject in output")
	}
}

func TestGroupByAssignee(t *testing.T) {
	wps := []api.WorkPackage{
		{ID: 1, Subject: "Task A", Links: api.WPLinks{
			Assignee: api.Link{Title: "Alice"},
			Status:   api.Link{Title: "Open"},
			Priority: api.Link{Title: "High"},
		}},
		{ID: 2, Subject: "Task B", Links: api.WPLinks{
			Assignee: api.Link{Title: "Bob"},
			Status:   api.Link{Title: "Open"},
			Priority: api.Link{Title: "Normal"},
		}},
		{ID: 3, Subject: "Task C", Links: api.WPLinks{
			Assignee: api.Link{Title: "Alice"},
			Status:   api.Link{Title: "Done"},
			Priority: api.Link{Title: "Low"},
		}},
	}

	out := captureOutput(func() {
		GroupByAssignee(wps)
	})

	if !strings.Contains(out, "Alice (2 items)") {
		t.Errorf("expected 'Alice (2 items)', got: %s", out)
	}
	if !strings.Contains(out, "Bob (1 items)") {
		t.Errorf("expected 'Bob (1 items)', got: %s", out)
	}
}

func TestGroupByAssignee_Empty(t *testing.T) {
	out := captureOutput(func() {
		GroupByAssignee(nil)
	})
	if !strings.Contains(out, "No work packages found") {
		t.Errorf("expected 'No work packages found', got: %s", out)
	}
}

func TestWorkPackageDetail_ShowsRelease(t *testing.T) {
	// Release lives on customField50 (a version link array). It is distinct from
	// the sprint version, so `op show` must surface it in the detail view.
	wp := &api.WorkPackage{
		ID:      81321,
		Subject: "Offline reading",
		Links: api.WPLinks{
			Type:     api.Link{Title: "Feature"},
			Status:   api.Link{Title: "New"},
			Priority: api.Link{Title: "Normal"},
			Version:  api.Link{Title: "Sprint 42"},
			Release:  []api.Link{{Title: "[iOS][ETV] 1.0.9"}},
		},
	}

	out := captureOutput(func() {
		WorkPackageDetail(wp)
	})

	if !strings.Contains(out, "Release:") || !strings.Contains(out, "[iOS][ETV] 1.0.9") {
		t.Errorf("expected Release line with the release name, got: %s", out)
	}
}

func TestWorkPackageDetail_NoReleaseWhenAbsent(t *testing.T) {
	// Tickets without a release must not print an empty Release line.
	wp := &api.WorkPackage{
		ID:      81322,
		Subject: "No release yet",
		Links: api.WPLinks{
			Type:     api.Link{Title: "Task"},
			Status:   api.Link{Title: "New"},
			Priority: api.Link{Title: "Normal"},
		},
	}

	out := captureOutput(func() {
		WorkPackageDetail(wp)
	})

	if strings.Contains(out, "Release:") {
		t.Errorf("expected no Release line when unset, got: %s", out)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exact len!", 10, "exact len!"},
		{"this is way too long", 10, "this is..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
		}
	}
}
