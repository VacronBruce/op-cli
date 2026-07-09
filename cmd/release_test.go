package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func runReleaseCreateWith(t *testing.T, mock *testutil.MockClient, flagMap map[string]string, args []string) (string, error) {
	t.Helper()
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().String("status", "open", "")
	c.Flags().String("start", "", "")
	c.Flags().String("end", "", "")

	for k, v := range flagMap {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("setting flag --%s=%s: %v", k, v, err)
		}
	}

	var err error
	out := testutil.CaptureStdout(func() {
		err = runReleaseCreate(c, args)
	})
	return out, err
}

func runReleaseUpdateWith(t *testing.T, mock *testutil.MockClient, flagMap map[string]string, args []string) (string, error) {
	t.Helper()
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().String("name", "", "")
	c.Flags().String("status", "", "")
	c.Flags().String("start", "", "")
	c.Flags().String("end", "", "")

	// Only flags present in flagMap are marked Changed, matching how the CLI
	// distinguishes "leave as-is" from "set to this value".
	for k, v := range flagMap {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("setting flag --%s=%s: %v", k, v, err)
		}
	}

	var err error
	out := testutil.CaptureStdout(func() {
		err = runReleaseUpdate(c, args)
	})
	return out, err
}

// Renaming + rescheduling a release must PATCH only the fields the user passed,
// against the version ID resolved from the ID-or-name argument.
func TestReleaseUpdate_RenameAndEndDate(t *testing.T) {
	var capturedID int
	var captured *api.UpdateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		ResolveReleaseFn: func(project, name string) (*api.Version, error) {
			if name != "1821" {
				t.Errorf("expected release arg 1821, got %q", name)
			}
			return &api.Version{ID: 1821, Name: "[iOS][EET] 3.1.27"}, nil
		},
		UpdateVersionFn: func(id int, req *api.UpdateVersionRequest) (*api.Version, error) {
			capturedID = id
			captured = req
			return &api.Version{ID: id, Name: req.Name}, nil
		},
	}

	flags := map[string]string{"name": "[iOS][EET] 3.2.1", "end": "2026-06-25"}
	out, err := runReleaseUpdateWith(t, mock, flags, []string{"1821"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedID != 1821 {
		t.Errorf("expected PATCH against version 1821, got %d", capturedID)
	}
	if captured.Name != "[iOS][EET] 3.2.1" {
		t.Errorf("expected name '[iOS][EET] 3.2.1', got %q", captured.Name)
	}
	if captured.EndDate != "2026-06-25" {
		t.Errorf("expected endDate 2026-06-25, got %q", captured.EndDate)
	}
	// Untouched fields must stay empty so they're omitted from the PATCH body.
	if captured.Status != "" || captured.StartDate != "" {
		t.Errorf("expected status/start untouched, got status=%q start=%q", captured.Status, captured.StartDate)
	}
	if !strings.Contains(out, "#1821") || !strings.Contains(out, "[iOS][EET] 3.2.1") {
		t.Errorf("expected updated ID and name in output, got: %s", out)
	}
}

// No flags means there's nothing to change; the CLI must refuse rather than
// send an empty PATCH that silently does nothing.
func TestReleaseUpdate_NoFlags(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		UpdateVersionFn: func(int, *api.UpdateVersionRequest) (*api.Version, error) {
			t.Error("UpdateVersion should not be called when no fields change")
			return nil, nil
		},
	}

	_, err := runReleaseUpdateWith(t, mock, nil, []string{"1821"})
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Errorf("expected 'nothing to update' error, got: %v", err)
	}
}

func TestReleaseUpdate_InvalidStatus(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		UpdateVersionFn: func(int, *api.UpdateVersionRequest) (*api.Version, error) {
			t.Error("UpdateVersion should not be called for invalid status")
			return nil, nil
		},
	}

	_, err := runReleaseUpdateWith(t, mock, map[string]string{"status": "draft"}, []string{"1821"})
	if err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected 'invalid status' error, got: %v", err)
	}
}

func TestReleaseUpdate_InvalidEndDate(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		UpdateVersionFn: func(int, *api.UpdateVersionRequest) (*api.Version, error) {
			t.Error("UpdateVersion should not be called for invalid date")
			return nil, nil
		},
	}

	_, err := runReleaseUpdateWith(t, mock, map[string]string{"end": "6/25"}, []string{"1821"})
	if err == nil || !strings.Contains(err.Error(), "invalid end date") {
		t.Errorf("expected 'invalid end date' error, got: %v", err)
	}
}

// An unknown release must surface ResolveRelease's error (which lists valid
// names) instead of attempting a PATCH against a bogus ID.
func TestReleaseUpdate_UnknownRelease(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ResolveReleaseFn: func(project, name string) (*api.Version, error) {
			return nil, errors.New("release \"nope\" not found; available: [iOS][EET] 3.1.27")
		},
		UpdateVersionFn: func(int, *api.UpdateVersionRequest) (*api.Version, error) {
			t.Error("UpdateVersion should not be called when release cannot be resolved")
			return nil, nil
		},
	}

	_, err := runReleaseUpdateWith(t, mock, map[string]string{"name": "x"}, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestReleaseCreate_Success_DefaultsStatusOpen(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 1828, Name: "[iOS][ETV] 1.0.9"}, nil
		},
	}

	out, err := runReleaseCreateWith(t, mock, nil, []string{"[iOS][ETV] 1.0.9"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Status must default to "open" so the release is immediately visible.
	if captured.Status != "open" {
		t.Errorf("expected status=open, got %q", captured.Status)
	}
	if captured.Name != "[iOS][ETV] 1.0.9" {
		t.Errorf("expected name='[iOS][ETV] 1.0.9', got %q", captured.Name)
	}
	// definingProject link must scope the release to the correct project.
	link, ok := captured.Links["definingProject"]
	if !ok {
		t.Fatal("expected definingProject link in request")
	}
	if !strings.Contains(link.Href, "app") {
		t.Errorf("definingProject href should contain project identifier, got %q", link.Href)
	}
	if !strings.Contains(out, "#1828") || !strings.Contains(out, "[iOS][ETV] 1.0.9") {
		t.Errorf("expected ID and name in output, got: %s", out)
	}
}

func TestReleaseCreate_WithDatesAndStatus(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 1829, Name: "[iOS][EET] 3.2.0"}, nil
		},
	}

	flags := map[string]string{"status": "locked", "start": "2026-06-10", "end": "2026-06-30"}
	_, err := runReleaseCreateWith(t, mock, flags, []string{"[iOS][EET] 3.2.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.Status != "locked" {
		t.Errorf("expected status=locked, got %q", captured.Status)
	}
	if captured.StartDate != "2026-06-10" {
		t.Errorf("expected startDate=2026-06-10, got %q", captured.StartDate)
	}
	if captured.EndDate != "2026-06-30" {
		t.Errorf("expected endDate=2026-06-30, got %q", captured.EndDate)
	}
}

func TestReleaseCreate_InvalidStatus(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			t.Error("CreateVersion should not be called for invalid status")
			return nil, nil
		},
	}

	_, err := runReleaseCreateWith(t, mock, map[string]string{"status": "draft"}, []string{"v1.0"})
	if err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected 'invalid status' error, got: %v", err)
	}
}

func TestReleaseCreate_InvalidStartDate(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			t.Error("CreateVersion should not be called for invalid date")
			return nil, nil
		},
	}

	_, err := runReleaseCreateWith(t, mock, map[string]string{"start": "06/10/2026"}, []string{"v1.0"})
	if err == nil || !strings.Contains(err.Error(), "invalid start date") {
		t.Errorf("expected 'invalid start date' error, got: %v", err)
	}
}

// `op release list` must show only kind=release versions — sprints share the
// same /versions collection, and listing them here would bury the releases.
func TestReleaseList_ShowsOnlyReleases(t *testing.T) {
	col := &api.VersionCollection{}
	col.Embedded.Elements = []api.Version{
		{ID: 1, Name: "App_06/02/2026", Kind: "sprint"},
		{ID: 2, Name: "[iOS][ETV] 1.0.9", Kind: "release", Status: "open"},
		{ID: 3, Name: "[Android][EET] 3.2.0", Kind: "release", Status: "closed"},
	}
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ListVersionsFn: func(project string) (*api.VersionCollection, error) {
			if project != "app" {
				t.Errorf("expected project app, got %q", project)
			}
			return col, nil
		},
	}
	SetClient(mock)

	var err error
	out := testutil.CaptureStdout(func() {
		err = runReleaseList(&cobra.Command{}, nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[iOS][ETV] 1.0.9") || !strings.Contains(out, "[Android][EET] 3.2.0") {
		t.Errorf("expected both releases listed, got: %q", out)
	}
	if strings.Contains(out, "App_06/02/2026") {
		t.Errorf("sprints must not appear in release list, got: %q", out)
	}
}

func TestReleaseList_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ListVersionsFn: func(string) (*api.VersionCollection, error) {
			return nil, errors.New("boom")
		},
	}
	SetClient(mock)

	err := runReleaseList(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "listing releases") {
		t.Errorf("expected wrapped listing error, got: %v", err)
	}
}

func TestReleaseCreate_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			return nil, errors.New("duplicate name")
		},
	}

	_, err := runReleaseCreateWith(t, mock, nil, []string{"[iOS][ETV] 1.0.9"})
	if err == nil || !strings.Contains(err.Error(), "duplicate name") {
		t.Errorf("expected API error to propagate, got: %v", err)
	}
}
