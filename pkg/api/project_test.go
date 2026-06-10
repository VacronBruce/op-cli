package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListProjects(t *testing.T) {
	// `op projects` renders Name + Identifier for every visible project; if the
	// path or the embedded element decoding breaks, the command shows nothing.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pageSize") != "100" {
			t.Errorf("expected pageSize=100, got %q", r.URL.Query().Get("pageSize"))
		}
		json.NewEncoder(w).Encode(ProjectCollection{
			Total: 2,
			Embedded: struct {
				Elements []Project `json:"elements"`
			}{Elements: []Project{
				{ID: 1, Name: "App", Identifier: "app", Active: true},
				{ID: 2, Name: "Bug Backlog", Identifier: "bug", Active: true},
			}},
		})
	})
	defer ts.Close()

	got, err := c.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Total != 2 || len(got.Embedded.Elements) != 2 {
		t.Fatalf("expected 2 projects, got total=%d len=%d", got.Total, len(got.Embedded.Elements))
	}
	if got.Embedded.Elements[1].Identifier != "bug" {
		t.Errorf("expected identifier 'bug', got %q", got.Embedded.Elements[1].Identifier)
	}
}

func TestListProjects_Error(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"_type": "Error", "message": "unauthorized"})
	})
	defer ts.Close()

	if _, err := c.ListProjects(); err == nil {
		t.Fatal("expected error on 401, got nil")
	}
}

func TestGetProject(t *testing.T) {
	// GetProject is addressed by identifier, not numeric ID — the identifier is
	// what users type with -p, so the path must carry it through verbatim.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/projects/app" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Project{ID: 1, Name: "App", Identifier: "app"})
	})
	defer ts.Close()

	p, err := c.GetProject("app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != 1 || p.Identifier != "app" {
		t.Errorf("unexpected project: %+v", p)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"_type": "Error", "message": "not found"})
	})
	defer ts.Close()

	if _, err := c.GetProject("nope"); err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}
