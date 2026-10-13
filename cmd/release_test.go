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

func releaseWP(id int, status, subject string) api.WorkPackage {
	wp := api.WorkPackage{ID: id, Subject: subject}
	wp.Links.Type = api.Link{Title: "Task"}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

func eet331(desc string) *api.Version {
	v := &api.Version{ID: 2054, Name: "[Android][EET] 3.3.1", Status: "open"}
	if desc != "" {
		v.Description = &api.Formattable{Format: "plain", Raw: desc}
	}
	return v
}

func runReleaseCmd(t *testing.T, mock *testutil.MockClient, run func(*cobra.Command, []string) error, flagMap map[string]string, args []string) (string, error) {
	t.Helper()
	SetClient(mock)
	c := &cobra.Command{}
	c.Flags().StringP("description", "d", "", "")
	for k, v := range flagMap {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("setting flag --%s=%s: %v", k, v, err)
		}
	}
	var err error
	out := testutil.CaptureStdout(func() { err = run(c, args) })
	return out, err
}

func TestReleaseShow_ListsShippedTicketsByReleaseField(t *testing.T) {
	// A ticket's release lives in customField50; the built-in version field
	// holds the sprint, so filtering on it returns nothing. And release notes
	// are about what shipped, so Done tickets must not be filtered out the way
	// the open-only board and backlog queries do.
	var gotFilters []api.Filter
	mock := &testutil.MockClient{
		ProjectValue: "app",
		ResolveReleaseFn: func(_, name string) (*api.Version, error) {
			return eet331(""), nil
		},
		ListWorkPackagesFn: func(_ string, filters []api.Filter, _ string, _ int) (*api.WPCollection, error) {
			gotFilters = filters
			col := &api.WPCollection{Total: 3}
			col.Embedded.Elements = []api.WorkPackage{
				releaseWP(84923, "Done", "Inbox video plays with sound"),
				releaseWP(84926, "Done", "Tablet landscape on EpochTV detail"),
				releaseWP(85140, "To Do", "Test release candidate 3.3.1"),
			}
			return col, nil
		},
	}

	out, err := runReleaseCmd(t, mock, runReleaseShow, nil, []string{"[Android][EET] 3.3.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var byRelease, anyStatus bool
	for _, f := range gotFilters {
		if spec, ok := f["customField50"]; ok && spec.Operator == "=" && len(spec.Values) == 1 && spec.Values[0] == "2054" {
			byRelease = true
		}
		if spec, ok := f["status"]; ok && spec.Operator == "*" {
			anyStatus = true
		}
		if _, ok := f["version"]; ok {
			t.Errorf("must not filter on the sprint's version field: %v", f)
		}
	}
	if !byRelease {
		t.Errorf("expected filter customField50 = 2054, got %v", gotFilters)
	}
	if !anyStatus {
		t.Errorf("expected an any-status filter so Done tickets are listed, got %v", gotFilters)
	}
	for _, want := range []string{"#2054 [Android][EET] 3.3.1 (open)", "Inbox video plays with sound", "3 tickets: 2 Done, 1 To Do", "Description: (none)"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "WARNING") {
		t.Errorf("no truncation warning expected when every ticket was fetched:\n%s", out)
	}
}

func TestReleaseShow_WarnsWhenTicketsAreCutOff(t *testing.T) {
	// Release notes written from a partial list silently omit changes, so a
	// truncated listing must say so.
	mock := &testutil.MockClient{
		ProjectValue:     "app",
		ResolveReleaseFn: func(_, _ string) (*api.Version, error) { return eet331(""), nil },
		ListWorkPackagesFn: func(_ string, _ []api.Filter, _ string, _ int) (*api.WPCollection, error) {
			col := &api.WPCollection{Total: 600}
			col.Embedded.Elements = []api.WorkPackage{releaseWP(1, "Done", "a")}
			return col, nil
		},
	}

	out, err := runReleaseCmd(t, mock, runReleaseShow, nil, []string{"2054"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "WARNING: showing 1 of 600 tickets") {
		t.Errorf("expected truncation warning, got:\n%s", out)
	}
}

func TestReleaseShow_PrintsExistingNotes(t *testing.T) {
	// Showing the current description lets you see what is already written
	// before replacing it.
	mock := &testutil.MockClient{
		ProjectValue:     "app",
		ResolveReleaseFn: func(_, _ string) (*api.Version, error) { return eet331("Line one\nLine two"), nil },
	}

	out, err := runReleaseCmd(t, mock, runReleaseShow, nil, []string{"2054"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Description:\n  Line one\n  Line two") {
		t.Errorf("expected indented description, got:\n%s", out)
	}
}

func TestReleaseUpdate_WritesPlainTextNotesToResolvedRelease(t *testing.T) {
	// The name resolves to an ID, and the PATCH must go to that release with
	// the text untouched. Format is plain: OpenProject renders release
	// descriptions as plain text, so Markdown would show literally.
	var gotID int
	var gotReq *api.UpdateVersionRequest
	notes := "What's New\n- Videos from the notification inbox now play with sound."
	mock := &testutil.MockClient{
		ProjectValue:     "app",
		ResolveReleaseFn: func(_, _ string) (*api.Version, error) { return eet331(""), nil },
		UpdateVersionFn: func(id int, req *api.UpdateVersionRequest) (*api.Version, error) {
			gotID, gotReq = id, req
			return eet331(notes), nil
		},
	}

	out, err := runReleaseCmd(t, mock, runReleaseUpdate, map[string]string{"description": notes}, []string{"[Android][EET] 3.3.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != 2054 {
		t.Errorf("expected PATCH to release 2054, got %d", gotID)
	}
	if gotReq == nil || gotReq.Description == nil || gotReq.Description.Raw != notes || gotReq.Description.Format != "plain" {
		t.Errorf("expected plain description %q, got %+v", notes, gotReq)
	}
	if !strings.Contains(out, "Updated release #2054") {
		t.Errorf("expected confirmation, got:\n%s", out)
	}
	if strings.Contains(out, "replaced") {
		t.Errorf("an empty description was not replaced, got:\n%s", out)
	}
}

func TestReleaseUpdate_SaysWhenItReplacesNotes(t *testing.T) {
	// The update overwrites the whole description; say so, so nobody loses
	// existing notes without noticing.
	mock := &testutil.MockClient{
		ProjectValue:     "app",
		ResolveReleaseFn: func(_, _ string) (*api.Version, error) { return eet331("old notes"), nil },
		UpdateVersionFn:  func(_ int, _ *api.UpdateVersionRequest) (*api.Version, error) { return eet331("new"), nil },
	}

	out, err := runReleaseCmd(t, mock, runReleaseUpdate, map[string]string{"description": "new"}, []string{"2054"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "(replaced the previous description)") {
		t.Errorf("expected replace notice, got:\n%s", out)
	}
}

func TestReleaseUpdate_RequiresDescription(t *testing.T) {
	// Without --description there is nothing to write; it must not send an
	// empty PATCH that would wipe the existing notes.
	called := false
	mock := &testutil.MockClient{
		ProjectValue:     "app",
		ResolveReleaseFn: func(_, _ string) (*api.Version, error) { return eet331("keep me"), nil },
		UpdateVersionFn: func(_ int, _ *api.UpdateVersionRequest) (*api.Version, error) {
			called = true
			return nil, nil
		},
	}

	_, err := runReleaseCmd(t, mock, runReleaseUpdate, map[string]string{"description": "   "}, []string{"2054"})
	if err == nil || !strings.Contains(err.Error(), "--description") {
		t.Errorf("expected an error asking for --description, got %v", err)
	}
	if called {
		t.Error("UpdateVersion must not be called without a description")
	}
}
