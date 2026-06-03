package display

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// makeActivity is a helper that builds an Activity with the given comment text,
// date string, and user title.
func makeActivity(id int, raw, createdAt, user string) api.Activity {
	a := api.Activity{
		ID:        id,
		CreatedAt: createdAt,
	}
	if raw != "" {
		a.Comment = &api.Formattable{Format: "markdown", Raw: raw}
	}
	a.Links.User = api.Link{Title: user}
	return a
}

func TestActivities_EmptyCollection(t *testing.T) {
	ac := &api.ActivityCollection{}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "No comments.") {
		t.Errorf("expected 'No comments.' for empty collection, got: %s", out)
	}
}

func TestActivities_AllActivitiesLackComment(t *testing.T) {
	// Journal / attribute-change activities have no comment; they must be filtered.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		{ID: 1, Comment: nil},
		{ID: 2, Comment: &api.Formattable{Format: "markdown", Raw: ""}},
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "No comments.") {
		t.Errorf("expected 'No comments.' when all activities lack a comment, got: %s", out)
	}
}

func TestActivities_SingleComment(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		makeActivity(1, "Looks good to me", "2024-05-10T09:00:00Z", "Alice"),
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "Comments (1):") {
		t.Errorf("expected 'Comments (1):' in output, got: %s", out)
	}
	if !strings.Contains(out, "Looks good to me") {
		t.Errorf("expected comment text in output, got: %s", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected username 'Alice' in output, got: %s", out)
	}
}

func TestActivities_MultipleComments(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		makeActivity(1, "First review", "2024-06-01T08:00:00Z", "Alice"),
		makeActivity(2, "Addressed review", "2024-06-02T12:00:00Z", "Bob"),
		makeActivity(3, "Ship it", "2024-06-03T15:00:00Z", "Carol"),
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "Comments (3):") {
		t.Errorf("expected 'Comments (3):' in output, got: %s", out)
	}
	for _, want := range []string{"First review", "Addressed review", "Ship it", "Alice", "Bob", "Carol"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestActivities_MixedCommentAndNonComment(t *testing.T) {
	// Non-comment activities (nil or empty raw) must be filtered out; only
	// activities with a non-empty raw comment should appear.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		{ID: 1, Comment: nil}, // journal entry
		makeActivity(2, "Real comment", "2024-07-01T10:00:00Z", "Dave"), // only this counts
		{ID: 3, Comment: &api.Formattable{Format: "markdown", Raw: ""}}, // empty raw
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "Comments (1):") {
		t.Errorf("expected 'Comments (1):' after filtering, got: %s", out)
	}
	if !strings.Contains(out, "Real comment") {
		t.Errorf("expected 'Real comment' in output, got: %s", out)
	}
}

func TestActivities_DateFormattingFullTimestamp(t *testing.T) {
	// createdAt is a full ISO-8601 timestamp; only the date portion (first 10
	// characters) should appear in the output.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		makeActivity(1, "Review done", "2024-12-25T23:59:59Z", "Eve"),
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "2024-12-25") {
		t.Errorf("expected date '2024-12-25' in output, got: %s", out)
	}
	// The time portion must not bleed into the date bracket.
	if strings.Contains(out, "23:59:59") {
		t.Errorf("time portion should not appear in output, got: %s", out)
	}
}

func TestActivities_DateFormattingShortTimestamp(t *testing.T) {
	// When createdAt is shorter than 10 characters, an empty date string is used.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		makeActivity(1, "Short date comment", "2024-01", "Frank"),
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	// The comment still renders; the date bracket is just empty.
	if !strings.Contains(out, "Short date comment") {
		t.Errorf("expected comment text in output, got: %s", out)
	}
}

func TestActivities_DateFormattingEmpty(t *testing.T) {
	// When createdAt is empty, an empty date bracket is used.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		makeActivity(1, "No date comment", "", "Grace"),
	}

	out := captureOutput(func() {
		Activities(ac)
	})

	if !strings.Contains(out, "No date comment") {
		t.Errorf("expected comment text in output, got: %s", out)
	}
}

func TestActivities_HeaderCount(t *testing.T) {
	// Verify the count in the header reflects only comment activities, not total
	// elements including non-comment ones.
	tests := []struct {
		name     string
		elements []api.Activity
		wantN    int
	}{
		{
			name: "two comments out of three elements",
			elements: []api.Activity{
				makeActivity(1, "comment one", "2024-01-01T00:00:00Z", "U1"),
				{ID: 2, Comment: nil},
				makeActivity(3, "comment two", "2024-01-02T00:00:00Z", "U2"),
			},
			wantN: 2,
		},
		{
			name: "single comment",
			elements: []api.Activity{
				makeActivity(1, "only comment", "2024-01-01T00:00:00Z", "U1"),
			},
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &api.ActivityCollection{}
			ac.Embedded.Elements = tt.elements

			out := captureOutput(func() {
				Activities(ac)
			})

			wantHeader := fmt.Sprintf("Comments (%d):", tt.wantN)
			if !strings.Contains(out, wantHeader) {
				t.Errorf("expected %q in output, got: %s", wantHeader, out)
			}
		})
	}
}
