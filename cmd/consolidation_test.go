package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func hasFilter(filters []api.Filter, field, operator string, values ...string) bool {
	for _, f := range filters {
		spec, ok := f[field]
		if !ok {
			continue
		}
		if spec.Operator != operator {
			continue
		}
		if len(spec.Values) != len(values) {
			continue
		}
		match := true
		for i := range values {
			if spec.Values[i] != values[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

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

	out := testutil.CaptureStdout(func() {
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

	out := testutil.CaptureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "All backlog items have estimates!") {
		t.Errorf("expected all-estimated message, got: %s", out)
	}
}

// --- backlog --type tests ---

func TestBacklog_TypeFilter_Lowercase(t *testing.T) {
	var gotFilters []api.Filter
	getCalls := 0

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			getCalls++
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().StringSlice("type", nil, "")
	_ = cmd.Flags().Set("type", "bug")

	_ = testutil.CaptureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if getCalls != 1 {
		t.Errorf("expected 1 /types fetch, got %d", getCalls)
	}
	if !hasFilter(gotFilters, "type", "=", "7") {
		t.Errorf("expected type filter in filters: %#v", gotFilters)
	}
}

func TestBacklog_TypeFilter_Uppercase(t *testing.T) {
	var gotFilters []api.Filter

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().StringSlice("type", nil, "")
	_ = cmd.Flags().Set("type", "Bug")

	_ = testutil.CaptureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "type", "=", "7") {
		t.Errorf("expected type filter in filters: %#v", gotFilters)
	}
}

func TestBacklog_TypeFilter_Invalid(t *testing.T) {
	listCalls := 0
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}},{"id":8,"name":"Task","_links":{"self":{"href":"/api/v3/types/8"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			listCalls++
			return &api.WPCollection{}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().StringSlice("type", nil, "")
	_ = cmd.Flags().Set("type", "invalid")

	var err error
	_ = testutil.CaptureStdout(func() {
		err = runBacklog(cmd, nil)
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `resolving type "invalid": unknown "invalid"`) {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("unexpected error: %v", err)
	}
	if listCalls != 0 {
		t.Errorf("expected ListWorkPackages not to be called, got %d calls", listCalls)
	}
}

func TestBacklog_TypeFilter_MultipleTypes(t *testing.T) {
	var gotFilters []api.Filter

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}},{"id":8,"name":"Task","_links":{"self":{"href":"/api/v3/types/8"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("unestimated", false, "")
	cmd.Flags().StringSlice("type", nil, "")
	_ = cmd.Flags().Set("type", "bug,task")

	_ = testutil.CaptureStdout(func() {
		err := runBacklog(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "type", "=", "7", "8") {
		t.Errorf("expected multi-type filter in filters: %#v", gotFilters)
	}
}

// --- board --status tests ---

func boardStatusCmd(status string, noSprint bool) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().String("label", "", "")
	cmd.Flags().String("status", "", "")
	_ = cmd.Flags().Set("status", status)
	if noSprint {
		_ = cmd.Flags().Set("no-sprint", "true")
	}
	return cmd
}

// --status must filter on the SERVER, resolved like `op update --status`
// ("in-progress" matches "In progress"): a client-side filter only saw the
// fetched page and silently dropped matches beyond it.
func TestBoard_StatusFilter_IsServerSide(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn:        resolverCollections, // serves /statuses: New(1), In progress(7)
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return &api.Version{ID: 1, Name: "Sprint 1"}, nil
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 1,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{Elements: []api.WorkPackage{
					{ID: 1, Subject: "Active task", Links: api.WPLinks{Status: api.Link{Title: "In progress"}}},
				}},
			}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runBoard(boardStatusCmd("in-progress", false), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "status", "=", "7") {
		t.Errorf("expected server-side status=7 filter, got: %#v", gotFilters)
	}
	if !strings.Contains(out, "Active task") {
		t.Errorf("expected 'Active task' in output, got: %s", out)
	}
}

// With --no-sprint the board normally adds an open-only filter; an explicit
// --status must REPLACE it, not AND with it — otherwise asking for a closed
// status would always return nothing.
func TestBoard_StatusFilter_ReplacesOpenOnlyFilter(t *testing.T) {
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetFn:        resolverCollections,
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{}, nil
		},
	}
	SetClient(mock)

	testutil.CaptureStdout(func() {
		if err := runBoard(boardStatusCmd("new", true), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "status", "=", "1") {
		t.Errorf("expected status=1 filter, got: %#v", gotFilters)
	}
	if hasFilter(gotFilters, "status", "o", "") {
		t.Errorf("open-only filter must be dropped when --status is explicit, got: %#v", gotFilters)
	}
}

// --- fail-loud exit code tests ---

// sprint add must exit non-zero when any item fails, so a script doing
// `op sprint add ... && next` doesn't proceed on a partial failure.
func TestSprintAdd_PartialFailure_ReturnsError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "test",
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return &api.Version{ID: 1, Name: "Sprint 1"}, nil
		},
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			if id == 999 {
				return nil, errors.New("not found")
			}
			return &api.WorkPackage{ID: id, Subject: "ok"}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Int("points", 0, "")
	cmd.Flags().String("sprint", "", "")

	var err error
	_ = testutil.CaptureStdout(func() { err = runSprintAdd(cmd, []string{"101", "999"}) })
	if err == nil {
		t.Fatal("expected an error when an item fails, got nil")
	}
	if !strings.Contains(err.Error(), "1 of 2") {
		t.Errorf("expected '1 of 2' summary, got: %v", err)
	}
}

// attach must exit non-zero when any file fails to upload.
func TestAttach_PartialFailure_ReturnsError(t *testing.T) {
	mock := &testutil.MockClient{
		UploadAttachmentFn: func(wpID int, filePath, desc string) (*api.Attachment, error) {
			if strings.Contains(filePath, "bad") {
				return nil, errors.New("upload failed")
			}
			return &api.Attachment{FileName: filePath, FileSize: 10}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().String("desc", "", "")

	var err error
	_ = testutil.CaptureStdout(func() { err = runAttach(cmd, []string{"123", "good.png", "bad.png"}) })
	if err == nil {
		t.Fatal("expected an error when an attachment fails, got nil")
	}
	if !strings.Contains(err.Error(), "1 of 2") {
		t.Errorf("expected '1 of 2' summary, got: %v", err)
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

// --- my --type tests ---

func TestMy_TypeFilter_Lowercase(t *testing.T) {
	var gotFilters []api.Filter
	getCalls := 0

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn: func() (*api.User, error) {
			return &api.User{ID: 123, Name: "Me"}, nil
		},
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			getCalls++
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().Bool("author", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().Bool("by-sprint", false, "")

	_ = cmd.Flags().Set("no-sprint", "true")
	_ = cmd.Flags().Set("type", "bug")

	_ = testutil.CaptureStdout(func() {
		err := runMy(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if getCalls != 1 {
		t.Errorf("expected 1 /types fetch, got %d", getCalls)
	}
	if !hasFilter(gotFilters, "type", "=", "7") {
		t.Errorf("expected type filter in filters: %#v", gotFilters)
	}
}

func TestMy_TypeFilter_Uppercase(t *testing.T) {
	var gotFilters []api.Filter

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn: func() (*api.User, error) {
			return &api.User{ID: 123, Name: "Me"}, nil
		},
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().Bool("author", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().Bool("by-sprint", false, "")

	_ = cmd.Flags().Set("no-sprint", "true")
	_ = cmd.Flags().Set("type", "Bug")

	_ = testutil.CaptureStdout(func() {
		err := runMy(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "type", "=", "7") {
		t.Errorf("expected type filter in filters: %#v", gotFilters)
	}
}

func TestMy_TypeFilter_Invalid(t *testing.T) {
	listCalls := 0
	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn: func() (*api.User, error) {
			return &api.User{ID: 123, Name: "Me"}, nil
		},
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}},{"id":8,"name":"Task","_links":{"self":{"href":"/api/v3/types/8"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			listCalls++
			return &api.WPCollection{}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().Bool("author", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().Bool("by-sprint", false, "")

	_ = cmd.Flags().Set("no-sprint", "true")
	_ = cmd.Flags().Set("type", "invalid")

	var err error
	_ = testutil.CaptureStdout(func() {
		err = runMy(cmd, nil)
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `resolving type "invalid": unknown "invalid"`) {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("unexpected error: %v", err)
	}
	if listCalls != 0 {
		t.Errorf("expected ListWorkPackages not to be called, got %d calls", listCalls)
	}
}

func TestMy_TypeFilter_MultipleTypes(t *testing.T) {
	var gotFilters []api.Filter

	mock := &testutil.MockClient{
		ProjectValue: "test",
		GetMeFn: func() (*api.User, error) {
			return &api.User{ID: 123, Name: "Me"}, nil
		},
		GetFn: func(path string, result interface{}) error {
			if path != "/types" {
				return fmt.Errorf("unexpected Get path: %s", path)
			}
			return json.Unmarshal([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}},{"id":8,"name":"Task","_links":{"self":{"href":"/api/v3/types/8"}}}]}}`), result)
		},
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = append([]api.Filter(nil), filters...)
			return &api.WPCollection{
				Total: 0,
				Embedded: struct {
					Elements []api.WorkPackage `json:"elements"`
				}{},
			}, nil
		},
	}
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sprint", "", "")
	cmd.Flags().Bool("no-sprint", false, "")
	cmd.Flags().Bool("author", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("component", "", "")
	cmd.Flags().StringSlice("type", nil, "")
	cmd.Flags().Bool("by-sprint", false, "")

	_ = cmd.Flags().Set("no-sprint", "true")
	_ = cmd.Flags().Set("type", "bug,task")

	_ = testutil.CaptureStdout(func() {
		err := runMy(cmd, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !hasFilter(gotFilters, "type", "=", "7", "8") {
		t.Errorf("expected multi-type filter in filters: %#v", gotFilters)
	}
}
