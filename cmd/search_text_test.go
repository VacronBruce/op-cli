package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func subjectFilterValue(t *testing.T, filters []api.Filter) (string, string) {
	t.Helper()
	if len(filters) != 1 {
		t.Fatalf("expected one filter, got %v", filters)
	}
	f := filters[0]["subject"]
	return f.Operator, strings.Join(f.Values, ",")
}

func TestSearchText_AllProjectsWhenNoProject(t *testing.T) {
	col := &api.WPCollection{Total: 1}
	col.Embedded.Elements = []api.WorkPackage{*briefWP(86156, "Paywall crash", "Done", "Ada", "", "")}
	mock := &testutil.MockClient{
		ListAllWorkPackagesFn: func(filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			op, val := subjectFilterValue(t, filters)
			if sortBy != `[["updatedAt","desc"]]` {
				t.Errorf("sortBy must be the API's JSON form, got %q", sortBy)
			}
			if op != "~" || val != "paywall" {
				t.Errorf("filter = %s %s", op, val)
			}
			return col, nil
		},
	}
	out, err := runSearchWith(t, mock, map[string]string{"text": "paywall"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "86156") || !strings.Contains(out, "Paywall crash") {
		t.Errorf("expected the shared table, got: %s", out)
	}
}

func TestSearchText_ScopedToProject(t *testing.T) {
	var gotProject string
	mock := &testutil.MockClient{
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotProject = project
			return &api.WPCollection{}, nil
		},
	}
	out, err := runSearchWith(t, mock, map[string]string{"text": "paywall", "project": "app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotProject != "app" || !strings.Contains(out, "No work packages found.") {
		t.Errorf("project %q, out %q", gotProject, out)
	}
}

func TestSearchText_WarnsWhenTruncated(t *testing.T) {
	col := &api.WPCollection{Total: 120}
	col.Embedded.Elements = []api.WorkPackage{*briefWP(1, "a", "Open", "", "", "")}
	mock := &testutil.MockClient{
		ListAllWorkPackagesFn: func([]api.Filter, string, int) (*api.WPCollection, error) { return col, nil },
	}
	stderr := testutil.CaptureStderr(func() {
		_, _ = runSearchWith(t, mock, map[string]string{"text": "a"}, nil)
	})
	if !strings.Contains(stderr, "120") {
		t.Errorf("expected truncation warning, got %q", stderr)
	}
}

func TestSearch_UnfoundWordSuggestsTextSearch(t *testing.T) {
	mock := &testutil.MockClient{
		SearchByJiraIDFn: func(string) (*api.WPCollection, error) { return &api.WPCollection{}, nil },
	}
	_, err := runSearchWith(t, mock, nil, []string{"paywall"})
	if err == nil || !strings.Contains(err.Error(), `op search --text "paywall"`) {
		t.Errorf("expected --text suggestion, got %v", err)
	}
	_, err = runSearchWith(t, mock, nil, []string{"WP-99"})
	if err == nil || strings.Contains(err.Error(), "--text") {
		t.Errorf("a key-shaped value must not suggest --text, got %v", err)
	}
}

func TestSearch_SeveralUnquotedWordsSuggestTextSearch(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().String("text", "", "")
	err := searchArgs(c, []string{"paywall", "tracking"})
	if err == nil || !strings.Contains(err.Error(), `op search --text "paywall tracking"`) {
		t.Errorf("expected --text suggestion, got %v", err)
	}
}

func TestSearch_TextWithUnquotedWordsSuggestsQuoting(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().String("text", "", "")
	_ = c.Flags().Set("text", "paywall")
	err := searchArgs(c, []string{"tracking"})
	if err == nil || !strings.Contains(err.Error(), `op search --text "paywall tracking"`) {
		t.Errorf("expected a quoted --text suggestion, got %v", err)
	}
}
