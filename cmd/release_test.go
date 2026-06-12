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
