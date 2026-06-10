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

func newUpdateCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().StringP("status", "s", "", "")
	c.Flags().StringP("assignee", "a", "", "")
	c.Flags().String("priority", "", "")
	c.Flags().Int("points", 0, "")
	c.Flags().Int("done", -1, "")
	c.Flags().String("subject", "", "")
	c.Flags().StringP("description", "d", "", "")
	c.Flags().String("sprint", "", "")
	c.Flags().String("to-project", "", "")
	c.Flags().String("release", "", "")
	c.Flags().StringSlice("component", nil, "")
	return c
}

// resolverCollections serves the /statuses and project assignee collections the
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
	default:
		return fmt.Errorf("unexpected GET %s", path)
	}
	return json.Unmarshal([]byte(js), result)
}

func updateMock(captured **api.UpdateWPRequest) *testutil.MockClient {
	return &testutil.MockClient{
		ProjectValue: "app",
		GetFn:        resolverCollections,
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			*captured = req
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

// --- assign (deprecated, kept until removed) ---

func TestAssign_ResolvesUserAndPatchesAssignee(t *testing.T) {
	var got *api.UpdateWPRequest
	SetClient(updateMock(&got))

	out := testutil.CaptureStdout(func() {
		if err := runAssign(&cobra.Command{}, []string{"42", "Ken"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if got.Links["assignee"].(api.Link).Href != "/api/v3/users/5" {
		t.Errorf("expected resolved assignee href, got %+v", got.Links)
	}
	if !strings.Contains(out, "assigned to Ken Peng") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}
