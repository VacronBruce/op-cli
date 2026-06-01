package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestFindActiveSprint_Found(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 2,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 1, Name: "Sprint 1", Status: "closed"},
					{ID: 2, Name: "Sprint 2", Status: "open"},
				},
			},
		})
	})
	defer ts.Close()

	v, err := c.FindActiveSprint("myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "Sprint 2" {
		t.Errorf("expected 'Sprint 2', got %s", v.Name)
	}
	if v.ID != 2 {
		t.Errorf("expected ID=2, got %d", v.ID)
	}
}

func TestFindActiveSprint_NoneOpen(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 1,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 1, Name: "Sprint 1", Status: "closed"},
				},
			},
		})
	})
	defer ts.Close()

	_, err := c.FindActiveSprint("myproject")
	if err == nil {
		t.Error("expected error when no open sprint")
	}
}

func TestResolveVersion_ByName(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 2,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 1, Name: "Sprint 1", Status: "closed", Links: struct {
						Self            Link `json:"self"`
						DefiningProject Link `json:"definingProject"`
					}{Self: Link{Href: "/api/v3/versions/1"}}},
					{ID: 2, Name: "Sprint 2", Status: "open", Links: struct {
						Self            Link `json:"self"`
						DefiningProject Link `json:"definingProject"`
					}{Self: Link{Href: "/api/v3/versions/2"}}},
				},
			},
		})
	})
	defer ts.Close()

	v, err := c.ResolveVersion("myproject", "Sprint 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "Sprint 1" {
		t.Errorf("expected 'Sprint 1', got %s", v.Name)
	}
}

func TestResolveVersion_ActiveFallback(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 1,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 5, Name: "Active Sprint", Status: "open"},
				},
			},
		})
	})
	defer ts.Close()

	v, err := c.ResolveVersion("myproject", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "Active Sprint" {
		t.Errorf("expected 'Active Sprint', got %s", v.Name)
	}
}

func TestResolveVersion_NotFound(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 1,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 1, Name: "Sprint 1", Status: "open"},
				},
			},
		})
	})
	defer ts.Close()

	_, err := c.ResolveVersion("myproject", "Nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent sprint")
	}
}

func TestListVersions(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(VersionCollection{Total: 0})
	})
	defer ts.Close()

	result, err := c.ListVersions("proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestResolveVersion_CaseInsensitive(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionCollection{
			Total: 1,
			Embedded: struct {
				Elements []Version `json:"elements"`
			}{
				Elements: []Version{
					{ID: 3, Name: "App_EET_Test 123", Status: "open"},
				},
			},
		})
	})
	defer ts.Close()

	v, err := c.ResolveVersion("myproject", "app_eet_test 123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "App_EET_Test 123" {
		t.Errorf("expected 'App_EET_Test 123', got %s", v.Name)
	}
}

func TestVersionFilter_Valid(t *testing.T) {
	v := &Version{ID: 42, Name: "Sprint 1"}
	v.Links.DefiningProject = Link{Href: "/api/v3/projects/myproject"}

	f, err := VersionFilter(v, "myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected filter, got nil")
	}
}

func TestVersionFilter_ZeroID(t *testing.T) {
	v := &Version{ID: 0, Name: "Bad Sprint"}

	_, err := VersionFilter(v, "myproject")
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestVersionFilter_SharedVersion(t *testing.T) {
	v := &Version{ID: 42, Name: "Shared Sprint"}
	v.Links.DefiningProject = Link{Href: "/api/v3/projects/other-project"}

	_, err := VersionFilter(v, "myproject")
	if err == nil {
		t.Error("expected error for shared version from different project")
	}
}

func TestVersionFilter_NoDefiningProject(t *testing.T) {
	// When definingProject link is empty (older API or missing field),
	// the filter should still work.
	v := &Version{ID: 42, Name: "Sprint 1"}

	f, err := VersionFilter(v, "myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected filter, got nil")
	}
}
