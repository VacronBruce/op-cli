package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetWorkPackage(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/work_packages/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(WorkPackage{
			ID:      123,
			Subject: "Test WP",
		})
	})
	defer ts.Close()

	wp, err := c.GetWorkPackage(123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wp.ID != 123 {
		t.Errorf("expected ID=123, got %d", wp.ID)
	}
	if wp.Subject != "Test WP" {
		t.Errorf("expected subject='Test WP', got %s", wp.Subject)
	}
}

func TestGetWorkPackage_NotFound(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{
			"_type":   "Error",
			"message": "Work package not found",
		})
	})
	defer ts.Close()

	_, err := c.GetWorkPackage(999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestListWorkPackages_WithFilters(t *testing.T) {
	var receivedPath string
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		json.NewEncoder(w).Encode(WPCollection{
			Total: 1,
			Embedded: struct {
				Elements []WorkPackage `json:"elements"`
			}{
				Elements: []WorkPackage{{ID: 1, Subject: "Item 1"}},
			},
		})
	})
	defer ts.Close()

	filters := []Filter{
		NewFilter("status", "o", ""),
	}
	result, err := c.ListWorkPackages("myproject", filters, "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Embedded.Elements) != 1 {
		t.Errorf("expected 1 element, got %d", len(result.Embedded.Elements))
	}
	// Verify path contains project and filters
	if receivedPath == "" {
		t.Error("no request received")
	}
}

func TestCreateWorkPackage(t *testing.T) {
	var receivedBody map[string]interface{}
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(WorkPackage{ID: 42, Subject: "New WP"})
	})
	defer ts.Close()

	req := &CreateWPRequest{
		Subject: "New WP",
		Links:   make(map[string]LinkValue),
	}
	req.SetLink("type", Link{Href: "/api/v3/types/1"})

	wp, err := c.CreateWorkPackage("myproject", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wp.ID != 42 {
		t.Errorf("expected ID=42, got %d", wp.ID)
	}
	if receivedBody["subject"] != "New WP" {
		t.Errorf("expected subject='New WP' in body")
	}
}

func TestUpdateWorkPackage_FetchesLockVersion(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			// First call: GET to fetch lockVersion
			json.NewEncoder(w).Encode(WorkPackage{
				ID:          1,
				LockVersion: 5,
				Subject:     "Original",
			})
		} else if r.Method == "PATCH" {
			// Second call: PATCH with lockVersion
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if lv, ok := body["lockVersion"]; !ok || lv != float64(5) {
				t.Errorf("expected lockVersion=5 in PATCH body, got %v", lv)
			}
			json.NewEncoder(w).Encode(WorkPackage{ID: 1, Subject: "Updated"})
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL, "key", "proj")
	req := &UpdateWPRequest{Subject: "Updated"}
	wp, err := c.UpdateWorkPackage(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wp.Subject != "Updated" {
		t.Errorf("expected subject='Updated', got %s", wp.Subject)
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls (GET + PATCH), got %d", callCount)
	}
}

func TestNewFilter(t *testing.T) {
	f := NewFilter("status", "=", "open")
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var result map[string]map[string]interface{}
	json.Unmarshal(data, &result)

	spec, ok := result["status"]
	if !ok {
		t.Fatal("expected 'status' key in filter")
	}
	if spec["operator"] != "=" {
		t.Errorf("expected operator='=', got %v", spec["operator"])
	}
}

func TestSearchByJiraID_FilterAndPageSize(t *testing.T) {
	// `op search` relies on this building a contains-filter on the configured
	// jira-id custom field (default customField3) across ALL projects; a wrong
	// field key or a project-scoped path would silently return nothing.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/work_packages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		filters := r.URL.Query().Get("filters")
		if !strings.Contains(filters, "customField3") || !strings.Contains(filters, `"~"`) {
			t.Errorf("expected contains-filter on customField3, got %s", filters)
		}
		if r.URL.Query().Get("pageSize") != "20" {
			t.Errorf("expected pageSize=20, got %q", r.URL.Query().Get("pageSize"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total": 1,
			"_embedded": map[string]interface{}{
				"elements": []map[string]interface{}{{"id": 42, "subject": "Found"}},
			},
		})
	})
	defer ts.Close()

	got, err := c.SearchByJiraID("AR-178")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Embedded.Elements) != 1 || got.Embedded.Elements[0].ID != 42 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListAllWorkPackages_DefaultsPageSizeTo200(t *testing.T) {
	// Cross-project listings (overview, search fallback) must not inherit the
	// API's tiny default page size, or results silently truncate.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageSize") != "200" {
			t.Errorf("expected default pageSize=200, got %q", r.URL.Query().Get("pageSize"))
		}
		json.NewEncoder(w).Encode(WPCollection{})
	})
	defer ts.Close()

	if _, err := c.ListAllWorkPackages(nil, "", 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetMultiLink_MarshalsAsArray(t *testing.T) {
	// Multi-value custom fields (components, labels) require a JSON array in
	// _links; a single object would make OpenProject reject the create with 422.
	req := &CreateWPRequest{Links: map[string]LinkValue{}}
	req.SetMultiLink("customField12", []Link{{Href: "/a"}, {Href: "/b"}})

	data, err := json.Marshal(req.Links)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string][]map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("multi-link field did not marshal as an array: %s", data)
	}
	if len(decoded["customField12"]) != 2 {
		t.Errorf("expected 2 links, got %s", data)
	}
}

func TestCreateRelation(t *testing.T) {
	// `op link --blocks/--relates-to` posts to the FROM work package's relations
	// collection with the TO end as an href — direction matters for "blocks".
	var gotPath string
	var gotBody map[string]interface{}
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]string{"_type": "Relation"})
	})
	defer ts.Close()

	if err := c.CreateRelation(12, "blocks", 34); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v3/work_packages/12/relations" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if gotBody["type"] != "blocks" {
		t.Errorf("expected type=blocks, got %v", gotBody["type"])
	}
	links := gotBody["_links"].(map[string]interface{})
	to := links["to"].(map[string]interface{})
	if to["href"] != "/api/v3/work_packages/34" {
		t.Errorf("expected to-href for #34, got %v", to["href"])
	}
}
