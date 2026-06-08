package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func runSprintCreateWith(t *testing.T, mock *testutil.MockClient, flagMap map[string]string, args []string) (string, error) {
	t.Helper()
	SetClient(mock)

	c := &cobra.Command{}
	c.Flags().String("start", "", "")
	c.Flags().String("end", "", "")

	for k, v := range flagMap {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("setting flag --%s=%s: %v", k, v, err)
		}
	}

	var err error
	out := captureStdout(func() {
		err = runSprintCreate(c, args)
	})
	return out, err
}

func TestSprintCreate_Success_KindIsNotRelease(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 99, Name: "Sprint 2026-07-07", StartDate: "2026-07-07", EndDate: "2026-07-20"}, nil
		},
	}

	out, err := runSprintCreateWith(t, mock, map[string]string{"start": "2026-07-07"}, []string{"Sprint 2026-07-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sprint must not be created as a release — kind must be empty string.
	if captured.Kind != "" {
		t.Errorf("expected kind empty (sprint), got %q", captured.Kind)
	}
	if captured.Status != "open" {
		t.Errorf("expected status=open, got %q", captured.Status)
	}
	if !strings.Contains(out, "#99") || !strings.Contains(out, "Sprint 2026-07-07") {
		t.Errorf("expected ID and name in output, got: %s", out)
	}
}

func TestSprintCreate_EndDefaultsToStartPlus13Days(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 100, Name: "Sprint 2026-07-07", StartDate: req.StartDate, EndDate: req.EndDate}, nil
		},
	}

	_, err := runSprintCreateWith(t, mock, map[string]string{"start": "2026-07-07"}, []string{"Sprint 2026-07-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without --end, end date must default to start + 13 days (2-week sprint).
	if captured.EndDate != "2026-07-20" {
		t.Errorf("expected end=2026-07-20, got %q", captured.EndDate)
	}
}

func TestSprintCreate_ExplicitEndOverridesDefault(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 101, Name: "Sprint 2026-07-07", StartDate: req.StartDate, EndDate: req.EndDate}, nil
		},
	}

	flags := map[string]string{"start": "2026-07-07", "end": "2026-07-25"}
	_, err := runSprintCreateWith(t, mock, flags, []string{"Sprint 2026-07-07"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.EndDate != "2026-07-25" {
		t.Errorf("expected end=2026-07-25, got %q", captured.EndDate)
	}
}

func TestSprintCreate_MissingStart(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			t.Error("CreateVersion should not be called when --start is missing")
			return nil, nil
		},
	}

	_, err := runSprintCreateWith(t, mock, nil, []string{"Sprint 2026-07-07"})
	if err == nil || !strings.Contains(err.Error(), "--start is required") {
		t.Errorf("expected '--start is required' error, got: %v", err)
	}
}

func TestSprintCreate_InvalidStartDate(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			t.Error("CreateVersion should not be called for invalid date")
			return nil, nil
		},
	}

	_, err := runSprintCreateWith(t, mock, map[string]string{"start": "07/07/2026"}, []string{"Sprint 2026-07-07"})
	if err == nil || !strings.Contains(err.Error(), "invalid start date") {
		t.Errorf("expected 'invalid start date' error, got: %v", err)
	}
}

func TestSprintCreate_InvalidEndDate(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			t.Error("CreateVersion should not be called for invalid date")
			return nil, nil
		},
	}

	flags := map[string]string{"start": "2026-07-07", "end": "bad-date"}
	_, err := runSprintCreateWith(t, mock, flags, []string{"Sprint 2026-07-07"})
	if err == nil || !strings.Contains(err.Error(), "invalid end date") {
		t.Errorf("expected 'invalid end date' error, got: %v", err)
	}
}

func TestSprintCreate_APIErrorPropagates(t *testing.T) {
	mock := &testutil.MockClient{
		ProjectValue: "app",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			return nil, errors.New("sprint name already exists")
		},
	}

	_, err := runSprintCreateWith(t, mock, map[string]string{"start": "2026-07-07"}, []string{"Sprint 2026-07-07"})
	if err == nil || !strings.Contains(err.Error(), "sprint name already exists") {
		t.Errorf("expected API error to propagate, got: %v", err)
	}
}

func TestSprintCreate_DefiningProjectLinkContainsProject(t *testing.T) {
	var captured *api.CreateVersionRequest

	mock := &testutil.MockClient{
		ProjectValue: "myproject",
		CreateVersionFn: func(req *api.CreateVersionRequest) (*api.Version, error) {
			captured = req
			return &api.Version{ID: 102, Name: "Sprint X"}, nil
		},
	}

	_, err := runSprintCreateWith(t, mock, map[string]string{"start": "2026-07-07"}, []string{"Sprint X"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	link, ok := captured.Links["definingProject"]
	if !ok {
		t.Fatal("expected definingProject link in request")
	}
	if !strings.Contains(link.Href, "myproject") {
		t.Errorf("definingProject href should contain project identifier, got %q", link.Href)
	}
}
