package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().StringP("status", "s", "", "")
	c.Flags().StringP("assignee", "a", "", "")
	c.Flags().String("priority", "", "")
	c.Flags().Int("points", 0, "")
	c.Flags().Int("done", -1, "")
	c.Flags().String("subject", "", "")
	c.Flags().StringP("description", "d", "", "")
	c.Flags().Bool("force", false, "")
	c.Flags().String("user-story", "", "")
	c.Flags().String("sprint", "", "")
	c.Flags().String("to-project", "", "")
	c.Flags().String("release", "", "")
	c.Flags().StringSlice("component", nil, "")
	c.Flags().StringP("epic", "e", "", "")
	c.Flags().String("parent", "", "")
	c.Flags().String("start", "", "")
	c.Flags().String("due", "", "")
	c.Flags().StringSlice("product", nil, "")
	c.Flags().StringSlice("label", nil, "")
	return c
}

// resolverCollections serves the /statuses, assignee and epic collections the
// update resolver fetches, so flag tests can exercise real resolution.
func resolverCollections(path string, result interface{}) error {
	var js string
	switch {
	case strings.HasPrefix(path, "/statuses"):
		js = `{"_embedded":{"elements":[
			{"id":1,"name":"New","_links":{"self":{"href":"/api/v3/statuses/1"}}},
			{"id":7,"name":"In progress","_links":{"self":{"href":"/api/v3/statuses/7"}}}]}}`
	case strings.Contains(path, "available_assignees"):
		js = `{"_embedded":{"elements":[
			{"id":5,"name":"Ken Peng","_links":{"self":{"href":"/api/v3/users/5"}}}]}}`
	case strings.Contains(path, "/work_packages?filters=") && strings.Contains(path, "%225%22"):
		// epics (type id 5) in the project
		js = `{"_embedded":{"elements":[
			{"id":100,"subject":"NTD+ Launch","_links":{"self":{"href":"/api/v3/work_packages/100"}}}]}}`
	default:
		return fmt.Errorf("unexpected GET %s", path)
	}
	return json.Unmarshal([]byte(js), result)
}

func updateMock(captured **api.UpdateWPRequest) *testutil.MockClient {
	var mu sync.Mutex
	return &testutil.MockClient{
		ProjectValue: "app",
		GetFn:        resolverCollections,
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			mu.Lock()
			if captured != nil {
				*captured = req
			}
			mu.Unlock()
			wp := &api.WorkPackage{ID: id, Subject: "updated"}
			wp.Links.Type = api.Link{Title: "Task"}
			wp.Links.Status = api.Link{Title: "In progress"}
			wp.Links.Project = api.Link{Title: "App"}
			return wp, nil
		},
	}
}

func TestUpdate_StatusResolvesToHref(t *testing.T) {
	// --status takes a human name ("in-progress"); the PATCH must carry the
	// resolved href, not the raw string — OpenProject rejects names.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("status", "in-progress")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got == nil || got.Links["status"].(api.Link).Href != "/api/v3/statuses/7" {
		t.Errorf("expected resolved status href, got %+v", got)
	}
}

func TestUpdate_FieldsOnlySentWhenFlagged(t *testing.T) {
	// A partial update must not clobber fields the user didn't mention:
	// points/done/subject stay nil/empty unless their flag was given.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("points", "8")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.StoryPoints == nil || *got.StoryPoints != 8 {
		t.Errorf("expected StoryPoints=8, got %v", got.StoryPoints)
	}
	if got.PercentageDone != nil || got.Subject != "" || got.Description != nil {
		t.Errorf("unflagged fields must stay unset, got %+v", got)
	}
}

func TestUpdate_UserStoryUsesCustomField36(t *testing.T) {
	// The User Story field is a formattable custom field (customField36), sent as
	// a top-level property like description — NOT a link. The refine skill relies
	// on this to write back a corrected story, so the raw markdown must land there.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("user-story", "As a reader, I want offline mode so that I can read on the subway")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.UserStory == nil {
		t.Fatalf("expected UserStory set on the request, got %+v", got)
	}
	if got.UserStory.Format != "markdown" || !strings.Contains(got.UserStory.Raw, "offline mode") {
		t.Errorf("unexpected user story payload: %+v", got.UserStory)
	}
	// It must not leak into the description field.
	if got.Description != nil {
		t.Errorf("user-story must not touch description, got %+v", got.Description)
	}
}

func TestUpdate_ToProjectMovesViaResolvedHref(t *testing.T) {
	// --to-project resolves the identifier through GetProject so a typo fails
	// loudly instead of PATCHing a bogus project link.
	var got *api.UpdateWPRequest
	mock := updateMock(&got)
	mock.GetProjectFn = func(identifier string) (*api.Project, error) {
		if identifier != "wp" {
			t.Errorf("expected identifier wp, got %q", identifier)
		}
		p := &api.Project{ID: 9, Identifier: "wp"}
		p.Links.Self = api.Link{Href: "/api/v3/projects/9"}
		return p, nil
	}
	SetClient(mock)

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("to-project", "wp")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.Links["project"].(api.Link).Href != "/api/v3/projects/9" {
		t.Errorf("expected project href /api/v3/projects/9, got %+v", got.Links["project"])
	}
}

func TestUpdate_ReleaseUsesCustomField50(t *testing.T) {
	// Releases are NOT the version link: they live on customField50. Mixing
	// them up would silently move the ticket between sprints.
	var got *api.UpdateWPRequest
	mock := updateMock(&got)
	mock.ResolveReleaseFn = func(project, name string) (*api.Version, error) {
		v := &api.Version{ID: 60, Name: name, Kind: "release"}
		v.Links.Self = api.Link{Href: "/api/v3/versions/60"}
		return v, nil
	}
	SetClient(mock)

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("release", "[iOS] 1.0.9")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.Links["customField50"].(api.Link).Href != "/api/v3/versions/60" {
		t.Errorf("expected release on customField50, got %+v", got.Links)
	}
	if _, hasVersion := got.Links["version"]; hasVersion {
		t.Error("release must not touch the sprint version link")
	}
}

// --- description image preservation (inline attachment guard) ---

// descMock serves an existing description containing an inline screenshot, so
// the guard can compare it against the incoming --description.
func descMock(captured **api.UpdateWPRequest, oldDesc string) *testutil.MockClient {
	mock := updateMock(captured)
	mock.GetWorkPackageFn = func(id int) (*api.WorkPackage, error) {
		return &api.WorkPackage{
			ID:          id,
			Description: &api.Formattable{Format: "markdown", Raw: oldDesc},
		}, nil
	}
	return mock
}

func TestUpdate_DescriptionDroppingInlineImageIsRefused(t *testing.T) {
	// A refine that rewrites the description without carrying over an embedded
	// screenshot silently orphans it in the rendered ticket. Fail loud, no PATCH.
	var got *api.UpdateWPRequest
	SetClient(descMock(&got, "Repro steps\n\n![](/api/v3/attachments/999/content)"))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("description", "Rewritten steps with no image")
	err := runUpdate(cmd, []string{"123"})
	if err == nil || !strings.Contains(err.Error(), "999") {
		t.Fatalf("expected refusal naming attachment 999, got: %v", err)
	}
	if got != nil {
		t.Error("PATCH must not happen when an inline image would be dropped")
	}
}

func TestUpdate_DescriptionKeepingInlineImageSucceeds(t *testing.T) {
	// Preserving the image markdown is the correct refine: the update proceeds.
	var got *api.UpdateWPRequest
	SetClient(descMock(&got, "old\n\n![](/api/v3/attachments/999/content)"))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("description", "New AC\n\n![](/api/v3/attachments/999/content)")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if got == nil || got.Description == nil || !strings.Contains(got.Description.Raw, "999") {
		t.Errorf("expected description PATCH preserving the image, got %+v", got)
	}
}

func TestUpdate_ForceAllowsDroppingInlineImage(t *testing.T) {
	// --force is the explicit escape hatch: the user accepts losing the image.
	var got *api.UpdateWPRequest
	SetClient(descMock(&got, "old\n\n![](/api/v3/attachments/999/content)"))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("description", "Deliberately image-free")
	_ = cmd.Flags().Set("force", "true")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if got == nil || got.Description == nil {
		t.Errorf("expected description PATCH under --force, got %+v", got)
	}
}

func TestUpdate_NoFlagsIsAnError(t *testing.T) {
	// `op update 123` with nothing to change must fail before any API write —
	// an empty PATCH would still bump lockVersion and updatedAt.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	err := runUpdate(newUpdateCmd(), []string{"123"})
	if err == nil || !strings.Contains(err.Error(), "no changes specified") {
		t.Fatalf("expected no-changes error, got: %v", err)
	}
	if got != nil {
		t.Error("UpdateWorkPackage must not be called with no changes")
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	SetClient(&testutil.MockClient{})
	err := runUpdate(newUpdateCmd(), []string{"abc"})
	if err == nil || !strings.Contains(err.Error(), "invalid work package ID") {
		t.Fatalf("expected invalid-ID error, got: %v", err)
	}
}

// --- create-parity flags (#81742) ---

func TestUpdate_EpicResolvesToLink(t *testing.T) {
	// --epic mirrors create: resolve the epic by name within the project and
	// link its href, so tickets can be re-parented to an epic after creation.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("epic", "ntd+")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.Links["epic"].(api.Link).Href != "/api/v3/work_packages/100" {
		t.Errorf("expected resolved epic href, got %+v", got.Links)
	}
}

func TestUpdate_ParentSetsHrefAndRejectsGarbage(t *testing.T) {
	// --parent takes a numeric WP id; a typo must fail before the PATCH.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("parent", "456")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if got.Links["parent"].(api.Link).Href != "/api/v3/work_packages/456" {
		t.Errorf("expected parent href, got %+v", got.Links)
	}

	got = nil
	cmd = newUpdateCmd()
	_ = cmd.Flags().Set("parent", "abc")
	err := runUpdate(cmd, []string{"123"})
	if err == nil || !strings.Contains(err.Error(), "invalid parent ID") {
		t.Fatalf("expected invalid parent error, got: %v", err)
	}
	if got != nil {
		t.Error("PATCH must not happen on invalid parent")
	}
}

func TestUpdate_StartAndDueDates(t *testing.T) {
	// Dates could only be set at create time; update must carry them in the
	// PATCH body (startDate/dueDate), and only when flagged.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("start", "2026-07-01")
	_ = cmd.Flags().Set("due", "2026-07-15")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.StartDate != "2026-07-01" || got.DueDate != "2026-07-15" {
		t.Errorf("expected dates in request, got start=%q due=%q", got.StartDate, got.DueDate)
	}
}

func TestUpdate_ProductAndLabelMultiLinks(t *testing.T) {
	// product/label are multi-value custom fields resolved from the local
	// registry — same as create — so classification can be fixed post-creation.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("product", "entd")
	_ = cmd.Flags().Set("label", "team#appandroid")
	testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	prod, ok := got.Links["customField4"].([]api.Link)
	if !ok || len(prod) != 1 {
		t.Fatalf("expected one product link on customField4, got %+v", got.Links)
	}
	label, ok := got.Links["customField13"].([]api.Link)
	if !ok || len(label) != 1 {
		t.Fatalf("expected one label link on customField13, got %+v", got.Links)
	}
}

// --- bulk update (#81744) ---

func TestUpdate_MultiID_AppliesSameChangeToAll(t *testing.T) {
	// Bulk update exists so sprint moves/status sweeps don't need shell loops:
	// every listed ID gets the same change. Each PATCH must arrive with
	// LockVersion reset to 0 — reusing ticket A's lockVersion on ticket B
	// would 409 (or worse, silently overwrite a newer revision).
	var mu sync.Mutex
	ids := map[int]bool{}
	mock := updateMock(nil)
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		if req.LockVersion != 0 {
			t.Errorf("PATCH for #%d carried stale LockVersion %d", id, req.LockVersion)
		}
		mu.Lock()
		ids[id] = true
		mu.Unlock()
		return &api.WorkPackage{ID: id, Subject: "x"}, nil
	}
	SetClient(mock)

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("status", "in-progress")
	out := testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"101", "102", "103"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if len(ids) != 3 || !ids[101] || !ids[102] || !ids[103] {
		t.Errorf("expected updates for 101,102,103, got %v", ids)
	}
	if !strings.Contains(out, "Updated 3 work package(s)") {
		t.Errorf("expected bulk summary, got: %s", out)
	}
}

func TestUpdate_MultiID_ContinuesPastFailures(t *testing.T) {
	// One bad ticket (locked, missing, garbage id) must not strand the rest:
	// the loop continues, the summary names the count, and the exit is non-zero.
	var mu sync.Mutex
	ids := map[int]bool{}
	mock := updateMock(nil)
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		if id == 102 {
			return nil, errors.New("locked")
		}
		mu.Lock()
		ids[id] = true
		mu.Unlock()
		return &api.WorkPackage{ID: id, Subject: "x"}, nil
	}
	SetClient(mock)

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("points", "3")
	var err error
	out := testutil.CaptureStdout(func() {
		err = runUpdate(cmd, []string{"101", "102", "abc", "103"})
	})

	if err == nil || !strings.Contains(err.Error(), "2 of 4") {
		t.Fatalf("expected '2 of 4' aggregate error, got: %v", err)
	}
	if len(ids) != 2 || !ids[101] || !ids[103] {
		t.Errorf("expected 101 and 103 updated despite failures, got %v", ids)
	}
	// Updates run concurrently, but the report must read in ARGUMENT order —
	// users scan it against the list they typed.
	for _, pair := range [][2]string{
		{"Updated #101", "Error updating #102"},
		{"Error updating #102", "Skipping invalid ID: abc"},
		{"Skipping invalid ID: abc", "Updated #103"},
	} {
		if strings.Index(out, pair[0]) > strings.Index(out, pair[1]) {
			t.Errorf("output out of argument order: %q must precede %q in: %s", pair[0], pair[1], out)
		}
	}
}

// Bulk updates must actually overlap requests: two updates block until BOTH
// have started; a sequential implementation deadlocks here (caught by the
// timeout). The shared request is safe because UpdateWorkPackage never
// mutates it (contract-tested in pkg/api).
func TestUpdate_MultiID_RunsConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	mock := updateMock(nil)
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		started <- struct{}{}
		once.Do(func() {
			go func() {
				<-started
				<-started
				close(release)
			}()
		})
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			return nil, errors.New("timed out waiting for a second concurrent update — bulk update is sequential")
		}
		return &api.WorkPackage{ID: id, Subject: "x"}, nil
	}
	SetClient(mock)

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("points", "3")
	var err error
	testutil.CaptureStdout(func() {
		err = runUpdate(cmd, []string{"101", "102"})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdate_SingleID_KeepsDetailOutput(t *testing.T) {
	// The single-ID path is unchanged: full detail rendering, not the bulk
	// summary — existing muscle memory and scripts depend on it.
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("points", "3")
	out := testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Updated #123") {
		t.Errorf("expected single-ID detail output, got: %s", out)
	}
	if strings.Contains(out, "work package(s)") {
		t.Errorf("single ID must not print the bulk summary, got: %s", out)
	}
}

// --- URL output tests (#81722) ---

// Both update paths must surface the ticket's browser URL: the single-ID
// detail view on its own line, and each bulk line inline — so the user can
// open or share any updated ticket without hand-building the link.
func TestUpdate_PrintsWorkPackageURL(t *testing.T) {
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	cmd := newUpdateCmd()
	_ = cmd.Flags().Set("points", "3")
	out := testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"123"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "https://op.example.com/work_packages/123") {
		t.Errorf("single update must print the work package URL, got: %s", out)
	}

	SetClient(updateMock(&got))
	cmd = newUpdateCmd()
	_ = cmd.Flags().Set("points", "3")
	out = testutil.CaptureStdout(func() {
		if err := runUpdate(cmd, []string{"101", "102"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	for _, id := range []string{"101", "102"} {
		if !strings.Contains(out, "https://op.example.com/work_packages/"+id) {
			t.Errorf("bulk update must print each work package URL, got: %s", out)
		}
	}
}
