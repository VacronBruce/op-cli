package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// priorityCollection returns a mock /priorities API response with the given names+IDs.
func priorityCollection(items []struct {
	id   int
	name string
}) interface{} {
	type elem struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
	}
	type resp struct {
		Embedded struct {
			Elements []json.RawMessage `json:"elements"`
		} `json:"_embedded"`
		Total int `json:"total"`
	}

	var r resp
	r.Total = len(items)
	for _, item := range items {
		e := elem{ID: item.id, Name: item.name}
		e.Links.Self.Href = fmt.Sprintf("/api/v3/priorities/%d", item.id)
		raw, _ := json.Marshal(e)
		r.Embedded.Elements = append(r.Embedded.Elements, raw)
	}
	return r
}

// typeCollection returns a mock /types API response with the given names+IDs.
func typeCollection(items []struct {
	id   int
	name string
}) interface{} {
	type elem struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"_links"`
	}
	type resp struct {
		Embedded struct {
			Elements []json.RawMessage `json:"elements"`
		} `json:"_embedded"`
		Total int `json:"total"`
	}

	var r resp
	r.Total = len(items)
	for _, item := range items {
		e := elem{ID: item.id, Name: item.name}
		e.Links.Self.Href = fmt.Sprintf("/api/v3/types/%d", item.id)
		raw, _ := json.Marshal(e)
		r.Embedded.Elements = append(r.Embedded.Elements, raw)
	}
	return r
}

func runBacklogWith(t *testing.T, mock *testutil.MockClient, flagMap map[string]string) ([]api.Filter, error) {
	t.Helper()
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().Bool("unestimated", false, "")
	c.Flags().StringSlice("priority", nil, "")
	c.Flags().StringSlice("type", nil, "")

	for k, v := range flagMap {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("setting flag --%s=%s: %v", k, v, err)
		}
	}

	var capturedFilters []api.Filter
	mock.ListWorkPackagesFn = func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
		capturedFilters = filters
		return &api.WPCollection{}, nil
	}

	testutil.CaptureStdout(func() {
		_ = runBacklog(c, nil)
	})
	return capturedFilters, nil
}

func TestBacklog_NoPriorityType_NoExtraFilters(t *testing.T) {
	mock := &testutil.MockClient{ProjectValue: "app"}

	filters, err := runBacklogWith(t, mock, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the two baseline filters (version=!*, status=o) must be present.
	if len(filters) != 2 {
		t.Errorf("expected 2 filters without flags, got %d", len(filters))
	}
}

func TestBacklog_PriorityFlag_AddsFilterWithResolvedID(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn: func(path string, result interface{}) error {
			if strings.Contains(path, "/priorities") {
				data, _ := json.Marshal(priorityCollection([]struct {
					id   int
					name string
				}{
					{7, "P0"}, {8, "P1"}, {9, "P2"},
				}))
				return json.Unmarshal(data, result)
			}
			return fmt.Errorf("unexpected GET %s", path)
		},
	}

	filters, err := runBacklogWith(t, mock, map[string]string{"priority": "p0,p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must have baseline 2 + priority filter = 3 total.
	if len(filters) != 3 {
		t.Fatalf("expected 3 filters, got %d", len(filters))
	}

	// The priority filter must contain the resolved numeric IDs, not the names.
	pf := filters[2]
	spec, ok := pf["priority"]
	if !ok {
		t.Fatal("expected 'priority' filter key")
	}
	if spec.Operator != "=" {
		t.Errorf("expected operator '=', got %q", spec.Operator)
	}
	if len(spec.Values) != 2 {
		t.Fatalf("expected 2 priority IDs, got %d", len(spec.Values))
	}
	// IDs must be "7" (P0) and "8" (P1), not the names.
	if spec.Values[0] != "7" || spec.Values[1] != "8" {
		t.Errorf("expected IDs [7 8], got %v", spec.Values)
	}
}

func TestBacklog_TypeFlag_AddsFilterWithResolvedID(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn: func(path string, result interface{}) error {
			if strings.Contains(path, "/types") {
				data, _ := json.Marshal(typeCollection([]struct {
					id   int
					name string
				}{
					{1, "Task"}, {3, "Bug"}, {5, "Epic"},
				}))
				return json.Unmarshal(data, result)
			}
			return fmt.Errorf("unexpected GET %s", path)
		},
	}

	filters, err := runBacklogWith(t, mock, map[string]string{"type": "bug"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(filters) != 3 {
		t.Fatalf("expected 3 filters, got %d", len(filters))
	}

	tf := filters[2]
	spec, ok := tf["type"]
	if !ok {
		t.Fatal("expected 'type' filter key")
	}
	if len(spec.Values) != 1 || spec.Values[0] != "3" {
		t.Errorf("expected type ID [3] for 'bug', got %v", spec.Values)
	}
}

func TestBacklog_PriorityAndType_BothFiltersPresent(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn: func(path string, result interface{}) error {
			switch {
			case strings.Contains(path, "/priorities"):
				data, _ := json.Marshal(priorityCollection([]struct {
					id   int
					name string
				}{
					{7, "P0"}, {10, "Sev1"},
				}))
				return json.Unmarshal(data, result)
			case strings.Contains(path, "/types"):
				data, _ := json.Marshal(typeCollection([]struct {
					id   int
					name string
				}{
					{3, "Bug"},
				}))
				return json.Unmarshal(data, result)
			}
			return fmt.Errorf("unexpected GET %s", path)
		},
	}

	filters, err := runBacklogWith(t, mock, map[string]string{
		"priority": "p0,sev1",
		"type":     "bug",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 baseline + priority + type = 4 filters total.
	if len(filters) != 4 {
		t.Fatalf("expected 4 filters, got %d", len(filters))
	}
}

func TestBacklog_UnknownPriority_ReturnsError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn: func(path string, result interface{}) error {
			if strings.Contains(path, "/priorities") {
				data, _ := json.Marshal(priorityCollection([]struct {
					id   int
					name string
				}{
					{7, "P0"},
				}))
				return json.Unmarshal(data, result)
			}
			return fmt.Errorf("unexpected GET %s", path)
		},
	}

	SetClient(mock)
	c := &cobra.Command{}
	c.Flags().Bool("unestimated", false, "")
	c.Flags().StringSlice("priority", nil, "")
	c.Flags().StringSlice("type", nil, "")
	_ = c.Flags().Set("priority", "nonexistent")

	err := runBacklog(c, nil)
	if err == nil || !strings.Contains(err.Error(), "resolving priority") {
		t.Errorf("expected 'resolving priority' error, got: %v", err)
	}
}
