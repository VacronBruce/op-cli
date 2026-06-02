package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func wpWithJira(id int, jira, subject, status string) api.WorkPackage {
	wp := api.WorkPackage{ID: id, JiraID: jira, Subject: subject}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

func runSearchWith(t *testing.T, mock *testutil.MockClient, args []string) (string, error) {
	t.Helper()
	SetClient(mock)
	var err error
	out := captureStdout(func() {
		err = runSearch(&cobra.Command{}, args)
	})
	return out, err
}

func TestSearch_ExactMatch(t *testing.T) {
	col := &api.WPCollection{}
	col.Embedded.Elements = []api.WorkPackage{
		wpWithJira(81271, "WP-23", "Cannot publish in Collab", "Done"),
	}
	mock := &testutil.MockClient{
		SearchByJiraIDFn: func(jiraID string) (*api.WPCollection, error) {
			if jiraID != "WP-23" {
				t.Errorf("expected query WP-23, got %q", jiraID)
			}
			return col, nil
		},
	}
	out, err := runSearchWith(t, mock, []string{"WP-23"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "#81271") || !strings.Contains(out, "WP-23") {
		t.Errorf("expected op number mapping in output, got: %q", out)
	}
}

// A substring query like "WP-2" must not return WP-23 when an exact match for
// "WP-2" is absent — but if the field only contains partial matches, list them.
func TestSearch_PrefersExactOverPartial(t *testing.T) {
	col := &api.WPCollection{}
	col.Embedded.Elements = []api.WorkPackage{
		wpWithJira(1, "WP-2", "Exact", "To Do"),
		wpWithJira(2, "WP-23", "Partial", "To Do"),
		wpWithJira(3, "WP-200", "Partial", "To Do"),
	}
	mock := &testutil.MockClient{
		SearchByJiraIDFn: func(string) (*api.WPCollection, error) { return col, nil },
	}
	out, err := runSearchWith(t, mock, []string{"WP-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "#1") {
		t.Errorf("expected exact match #1, got: %q", out)
	}
	if strings.Contains(out, "#2") || strings.Contains(out, "#3") {
		t.Errorf("expected only the exact match, got partials too: %q", out)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	mock := &testutil.MockClient{
		SearchByJiraIDFn: func(string) (*api.WPCollection, error) { return &api.WPCollection{}, nil },
	}
	_, err := runSearchWith(t, mock, []string{"NOPE-1"})
	if err == nil {
		t.Fatal("expected error for no match, got nil")
	}
	if !strings.Contains(err.Error(), "NOPE-1") {
		t.Errorf("expected error to mention the JIRA ID, got: %v", err)
	}
}
