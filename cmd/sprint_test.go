package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// sprintWP builds a work package with the fields the sprint commands read.
func sprintWP(id int, subject, status string, points *int) api.WorkPackage {
	wp := api.WorkPackage{ID: id, Subject: subject, StoryPoints: points}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

func intPtr(n int) *int { return &n }

func activeSprintMock() *testutil.MockClient {
	return &testutil.MockClient{
		ProjectValue: "app",
		FindActiveSprintFn: func(project string) (*api.Version, error) {
			return &api.Version{ID: 11, Name: "Sprint 24", StartDate: "2026-06-01", EndDate: "2026-06-14"}, nil
		},
	}
}

func newSprintAddCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().Int("points", 0, "")
	c.Flags().String("sprint", "", "")
	return c
}

// --- sprint add ---

func TestSprintAdd_MovesItemsIntoResolvedSprint(t *testing.T) {
	// The whole point of `sprint add` is that every listed ID gets its version
	// link set to the SAME resolved sprint — not whatever sprint each item had.
	var updated []int
	mock := activeSprintMock()
	mock.ResolveVersionFn = func(project, name string) (*api.Version, error) {
		v := &api.Version{ID: 11, Name: "Sprint 24"}
		v.Links.Self = api.Link{Href: "/api/v3/versions/11"}
		return v, nil
	}
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		if href := req.Links["version"].(api.Link).Href; href != "/api/v3/versions/11" {
			t.Errorf("expected version href /api/v3/versions/11, got %s", href)
		}
		updated = append(updated, id)
		return &api.WorkPackage{ID: id, Subject: "wp"}, nil
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runSprintAdd(newSprintAddCmd(), []string{"101", "102"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if len(updated) != 2 || updated[0] != 101 || updated[1] != 102 {
		t.Errorf("expected updates for 101,102, got %v", updated)
	}
	if !strings.Contains(out, "Sprint 24") {
		t.Errorf("expected target sprint in output, got: %s", out)
	}
}

func TestSprintAdd_SetsPointsOnlyWhenFlagGiven(t *testing.T) {
	// --points piggybacks on the same update; when omitted the request must NOT
	// carry StoryPoints, or it would zero existing estimates.
	var gotPoints *int
	mock := activeSprintMock()
	mock.ResolveVersionFn = func(project, name string) (*api.Version, error) {
		v := &api.Version{ID: 11, Name: "Sprint 24"}
		v.Links.Self = api.Link{Href: "/api/v3/versions/11"}
		return v, nil
	}
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		gotPoints = req.StoryPoints
		return &api.WorkPackage{ID: id}, nil
	}
	SetClient(mock)

	cmd := newSprintAddCmd()
	testutil.CaptureStdout(func() { _ = runSprintAdd(cmd, []string{"101"}) })
	if gotPoints != nil {
		t.Errorf("expected no StoryPoints without --points, got %v", *gotPoints)
	}

	cmd = newSprintAddCmd()
	_ = cmd.Flags().Set("points", "5")
	testutil.CaptureStdout(func() { _ = runSprintAdd(cmd, []string{"101"}) })
	if gotPoints == nil || *gotPoints != 5 {
		t.Errorf("expected StoryPoints=5 with --points=5, got %v", gotPoints)
	}
}

func TestSprintAdd_AggregatesFailuresAndContinues(t *testing.T) {
	// A bad ID in the middle must not abort the batch: remaining items still
	// move, and the error reports how many failed so the exit code is non-zero.
	var updated []int
	mock := activeSprintMock()
	mock.ResolveVersionFn = func(project, name string) (*api.Version, error) {
		v := &api.Version{ID: 11, Name: "Sprint 24"}
		v.Links.Self = api.Link{Href: "/api/v3/versions/11"}
		return v, nil
	}
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		if id == 102 {
			return nil, errors.New("locked")
		}
		updated = append(updated, id)
		return &api.WorkPackage{ID: id}, nil
	}
	SetClient(mock)

	var err error
	testutil.CaptureStdout(func() {
		err = runSprintAdd(newSprintAddCmd(), []string{"101", "abc", "102", "103"})
	})

	if err == nil || !strings.Contains(err.Error(), "2 of 4") {
		t.Fatalf("expected '2 of 4' aggregate error, got: %v", err)
	}
	if len(updated) != 2 || updated[0] != 101 || updated[1] != 103 {
		t.Errorf("expected 101 and 103 still updated, got %v", updated)
	}
}

// --- sprint progress ---

func TestSprintProgress_CompactSummaryCountsDoneAndPoints(t *testing.T) {
	// The summary drives standups: done/in-progress counts and the done-points
	// percentage must reflect status buckets (closed/resolved/done vs new).
	mock := activeSprintMock()
	mock.ListWorkPackagesFn = func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
		return &api.WPCollection{
			Total: 3,
			Embedded: struct {
				Elements []api.WorkPackage `json:"elements"`
			}{Elements: []api.WorkPackage{
				sprintWP(1, "a", "Closed", intPtr(3)),
				sprintWP(2, "b", "In progress", intPtr(5)),
				sprintWP(3, "c", "New", intPtr(2)),
			}},
		}, nil
	}
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().BoolP("verbose", "v", false, "")
	out := testutil.CaptureStdout(func() {
		if err := runSprintProgress(c, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Items:    1/3 done, 1 in progress") {
		t.Errorf("expected item counts, got: %s", out)
	}
	if !strings.Contains(out, "Points:   3/10 (30%)") {
		t.Errorf("expected points percentage, got: %s", out)
	}
}

// --- sprint close ---

func TestSprintClose_SplitsDoneFromCarryOver(t *testing.T) {
	// Close summary must bucket case-insensitively on status and point users at
	// the carry-over command for whatever is incomplete.
	mock := activeSprintMock()
	mock.ListWorkPackagesFn = func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
		return &api.WPCollection{
			Total: 2,
			Embedded: struct {
				Elements []api.WorkPackage `json:"elements"`
			}{Elements: []api.WorkPackage{
				sprintWP(1, "shipped thing", "RESOLVED", intPtr(3)),
				sprintWP(2, "leftover thing", "In progress", nil),
			}},
		}, nil
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runSprintClose(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Completed (1):") || !strings.Contains(out, "shipped thing [3pt]") {
		t.Errorf("expected completed bucket with points, got: %s", out)
	}
	if !strings.Contains(out, "Incomplete - carry over (1):") || !strings.Contains(out, "leftover thing") {
		t.Errorf("expected carry-over bucket, got: %s", out)
	}
	if !strings.Contains(out, "op sprint add") {
		t.Errorf("expected carry-over hint, got: %s", out)
	}
}

// --- sprint list ---

func TestSprintList_RendersDashForMissingDates(t *testing.T) {
	// Versions without dates are common (backlog buckets); the table must not
	// render empty cells that break column alignment.
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ListVersionsFn: func(project string) (*api.VersionCollection, error) {
			return &api.VersionCollection{
				Total: 2,
				Embedded: struct {
					Elements []api.Version `json:"elements"`
				}{Elements: []api.Version{
					{ID: 1, Name: "Sprint 24", Status: "open", StartDate: "2026-06-01", EndDate: "2026-06-14"},
					{ID: 2, Name: "Backlog", Status: "open"},
				}},
			}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runSprintList(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Sprint 24") || !strings.Contains(out, "2026-06-01") {
		t.Errorf("expected dated sprint row, got: %s", out)
	}
	if !strings.Contains(out, "-             -             Backlog") {
		t.Errorf("expected dash placeholders for missing dates, got: %s", out)
	}
}

func TestSprintList_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ListVersionsFn: func(project string) (*api.VersionCollection, error) {
			return nil, errors.New("boom")
		},
	}
	SetClient(mock)

	err := runSprintList(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "listing versions") {
		t.Fatalf("expected wrapped listing error, got: %v", err)
	}
}

// --- sprint plan (deprecated alias of backlog) ---

func TestSprintPlan_ListsUnscheduledOpenItems(t *testing.T) {
	// Deprecated but still wired: it must keep filtering to version-less open
	// items until removed, or users following old docs see scheduled work.
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ListWorkPackagesFn: func(project string, filters []api.Filter, sortBy string, pageSize int) (*api.WPCollection, error) {
			gotFilters = filters
			return &api.WPCollection{Total: 0}, nil
		},
	}
	SetClient(mock)

	out := testutil.CaptureStdout(func() {
		if err := runSprintPlan(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var hasNoVersion, hasOpen bool
	for _, f := range gotFilters {
		if spec, ok := f["version"]; ok && spec.Operator == "!*" {
			hasNoVersion = true
		}
		if spec, ok := f["status"]; ok && spec.Operator == "o" {
			hasOpen = true
		}
	}
	if !hasNoVersion || !hasOpen {
		t.Errorf("expected version!* and status=o filters, got %v", gotFilters)
	}
	if !strings.Contains(out, "Backlog items ready for sprint (0):") {
		t.Errorf("expected backlog header, got: %s", out)
	}
}
