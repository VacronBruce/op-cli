package display

import (
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

func owp(project, sprint, status, updated string) api.WorkPackage {
	wp := api.WorkPackage{UpdatedAt: updated}
	wp.Links.Project = api.Link{Title: project}
	wp.Links.Version = api.Link{Title: sprint}
	wp.Links.Status = api.Link{Title: status}
	return wp
}

// Projects rank by most-recent activity, sprints likewise; counts (open/blocked)
// aggregate per group; caps surface as MoreProjects/MoreSprints rather than
// silently dropping work.
func TestBuildOverview_GroupsRanksAndCaps(t *testing.T) {
	wps := []api.WorkPackage{
		owp("app", "S1", "Blocked", "2026-06-03"),
		owp("app", "S1", "In progress", "2026-06-01"),
		owp("app", "S0", "New", "2026-05-01"),
		owp("web", "W1", "New", "2026-06-02"),
		owp("old", "O1", "New", "2026-01-01"),
	}

	m := BuildOverview(wps, 2 /*projects*/, 1 /*sprints*/)

	if m.TotalProjects != 3 {
		t.Fatalf("expected 3 total projects, got %d", m.TotalProjects)
	}
	if m.MoreProjects != 1 {
		t.Errorf("expected MoreProjects=1 (old dropped), got %d", m.MoreProjects)
	}
	if len(m.Projects) != 2 {
		t.Fatalf("expected 2 shown projects, got %d", len(m.Projects))
	}

	// app is most recently active → first; web second; old dropped.
	app := m.Projects[0]
	if app.Name != "app" {
		t.Errorf("expected 'app' first, got %s", app.Name)
	}
	if app.Open != 3 || app.Blocked != 1 {
		t.Errorf("app: expected 3 open / 1 blocked, got %d/%d", app.Open, app.Blocked)
	}
	if app.MoreSprints != 1 {
		t.Errorf("app: expected MoreSprints=1 (S0 dropped), got %d", app.MoreSprints)
	}
	if len(app.Sprints) != 1 || app.Sprints[0].Name != "S1" {
		t.Fatalf("app: expected top sprint S1, got %+v", app.Sprints)
	}
	if app.Sprints[0].Open != 2 || app.Sprints[0].Blocked != 1 {
		t.Errorf("S1: expected 2 open / 1 blocked, got %d/%d", app.Sprints[0].Open, app.Sprints[0].Blocked)
	}
	if m.Projects[1].Name != "web" {
		t.Errorf("expected 'web' second, got %s", m.Projects[1].Name)
	}
}

// Items with no version land in a single "(no sprint)" bucket rather than being
// dropped or scattered.
func TestBuildOverview_NoSprintBucket(t *testing.T) {
	wps := []api.WorkPackage{
		owp("app", "", "New", "2026-06-03"),
		owp("app", "", "Blocked", "2026-06-02"),
	}
	m := BuildOverview(wps, 5, 5)
	if len(m.Projects) != 1 || len(m.Projects[0].Sprints) != 1 {
		t.Fatalf("expected one project with one sprint bucket, got %+v", m.Projects)
	}
	s := m.Projects[0].Sprints[0]
	if s.Name != noSprintLabel {
		t.Errorf("expected %q bucket, got %q", noSprintLabel, s.Name)
	}
	if s.Open != 2 || s.Blocked != 1 {
		t.Errorf("expected 2 open / 1 blocked in no-sprint bucket, got %d/%d", s.Open, s.Blocked)
	}
}

func TestBuildOverview_Empty(t *testing.T) {
	m := BuildOverview(nil, 5, 3)
	if len(m.Projects) != 0 || m.TotalProjects != 0 {
		t.Errorf("expected empty model, got %+v", m)
	}
}
