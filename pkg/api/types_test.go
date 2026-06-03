package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestResolve_ExactMatch(t *testing.T) {
	resources := []NamedResource{
		{ID: 1, Name: "Bug", Href: "/types/1"},
		{ID: 2, Name: "Task", Href: "/types/2"},
		{ID: 3, Name: "Feature", Href: "/types/3"},
	}

	r, err := resolve(resources, "Task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID != 2 || r.Name != "Task" {
		t.Errorf("expected Task (ID=2), got %s (ID=%d)", r.Name, r.ID)
	}
}

func TestResolve_CaseInsensitive(t *testing.T) {
	resources := []NamedResource{
		{ID: 1, Name: "Bug", Href: "/types/1"},
	}

	r, err := resolve(resources, "bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "Bug" {
		t.Errorf("expected Bug, got %s", r.Name)
	}
}

func TestResolve_PrefixMatch(t *testing.T) {
	resources := []NamedResource{
		{ID: 1, Name: "In Progress", Href: "/statuses/1"},
		{ID: 2, Name: "In Review", Href: "/statuses/2"},
	}

	r, err := resolve(resources, "in p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "In Progress" {
		t.Errorf("expected 'In Progress', got %s", r.Name)
	}
}

func TestResolve_NotFound(t *testing.T) {
	resources := []NamedResource{
		{ID: 1, Name: "Bug", Href: "/types/1"},
	}

	_, err := resolve(resources, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown resource")
	}
}

func TestResolveCustomOption_ExactMatch(t *testing.T) {
	href, err := ResolveCustomOption(ComponentOptions, "android")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if href != "/api/v3/custom_options/42" {
		t.Errorf("unexpected href: %s", href)
	}
}

func TestResolveCustomOption_PrefixMatch(t *testing.T) {
	href, err := ResolveCustomOption(ComponentOptions, "and")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if href != "/api/v3/custom_options/42" {
		t.Errorf("unexpected href: %s", href)
	}
}

func TestResolveCustomOption_NotFound(t *testing.T) {
	_, err := ResolveCustomOption(ComponentOptions, "windows")
	if err == nil {
		t.Error("expected error for unknown option")
	}
}

// OptionID must accept the same unique-prefix abbreviations that
// ResolveCustomOption does, so a value like "eng" works identically whether the
// command filters (OptionID) or sets a link (ResolveCustomOption).
func TestOptionID_UniquePrefix(t *testing.T) {
	id, err := OptionID(ComponentOptions, "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "45" {
		t.Errorf("expected engineering id 45, got %s", id)
	}
}

// An ambiguous prefix must be rejected deterministically instead of silently
// picking one option via random map iteration.
func TestMatchOption_AmbiguousPrefixRejected(t *testing.T) {
	_, err := ResolveCustomOption(LabelOptions, "team#app")
	if err == nil {
		t.Fatal("expected ambiguity error for 'team#app', got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
	for _, want := range []string{"team#appall", "team#appandroid", "team#appios"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected candidate %q listed, got: %v", want, err)
		}
	}
}

// The "available" list in an unknown-value error must be sorted, so the message
// is stable across runs (map iteration order is otherwise random).
func TestMatchOption_UnknownListsSortedOptions(t *testing.T) {
	_, err := OptionID(ComponentOptions, "windows")
	if err == nil {
		t.Fatal("expected error for unknown option")
	}
	want := "analytics, android, engineering, ios, ott"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected sorted options %q, got: %v", want, err)
	}
}

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"in-progress": "in progress",
		"In_Progress": "in progress",
		"  Blocked  ": "blocked",
	}
	for in, want := range cases {
		if got := NormalizeName(in); got != want {
			t.Errorf("NormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolver_Types(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(collectionResponse{
			Embedded: struct {
				Elements []json.RawMessage `json:"elements"`
			}{
				Elements: []json.RawMessage{
					json.RawMessage(`{"id":1,"name":"Bug","_links":{"self":{"href":"/api/v3/types/1"}}}`),
					json.RawMessage(`{"id":2,"name":"Task","_links":{"self":{"href":"/api/v3/types/2"}}}`),
				},
			},
		})
	})
	defer ts.Close()

	resolver := NewResolver(c, "proj")
	types, err := resolver.Types()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
	if types[0].Name != "Bug" {
		t.Errorf("expected Bug, got %s", types[0].Name)
	}

	// Test cache — second call should not hit server
	types2, _ := resolver.Types()
	if len(types2) != 2 {
		t.Error("cache miss")
	}
}

func TestResolver_ResolveType(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(collectionResponse{
			Embedded: struct {
				Elements []json.RawMessage `json:"elements"`
			}{
				Elements: []json.RawMessage{
					json.RawMessage(`{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}}`),
				},
			},
		})
	})
	defer ts.Close()

	resolver := NewResolver(c, "proj")
	r, err := resolver.ResolveType("bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Href != "/api/v3/types/7" {
		t.Errorf("expected href /api/v3/types/7, got %s", r.Href)
	}
}

// A hyphenated CLI value ("in-progress") must resolve to a space-separated
// status label ("In progress"); neither exact nor prefix matching handled this.
func TestResolver_ResolveStatus_Normalized(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(collectionResponse{
			Embedded: struct {
				Elements []json.RawMessage `json:"elements"`
			}{
				Elements: []json.RawMessage{
					json.RawMessage(`{"id":3,"name":"In progress","_links":{"self":{"href":"/api/v3/statuses/3"}}}`),
				},
			},
		})
	})
	defer ts.Close()

	resolver := NewResolver(c, "proj")
	r, err := resolver.ResolveStatus("in-progress")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Href != "/api/v3/statuses/3" {
		t.Errorf("expected href /api/v3/statuses/3, got %s", r.Href)
	}
}

func TestResolver_ResolveUser(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(collectionResponse{
			Embedded: struct {
				Elements []json.RawMessage `json:"elements"`
			}{
				Elements: []json.RawMessage{
					json.RawMessage(`{"id":36,"name":"Bruce Chen","_links":{"self":{"href":"/api/v3/users/36"}}}`),
					json.RawMessage(`{"id":59,"name":"Chiayou Yen","_links":{"self":{"href":"/api/v3/users/59"}}}`),
				},
			},
		})
	})
	defer ts.Close()

	resolver := NewResolver(c, "proj")

	// Exact match
	r, err := resolver.ResolveUser("Bruce Chen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID != 36 {
		t.Errorf("expected ID=36, got %d", r.ID)
	}

	// Partial match
	r2, err := resolver.ResolveUser("chiayou")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r2.ID != 59 {
		t.Errorf("expected ID=59, got %d", r2.ID)
	}

	// @ prefix stripped
	r3, err := resolver.ResolveUser("@Bruce Chen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r3.ID != 36 {
		t.Errorf("expected ID=36, got %d", r3.ID)
	}
}
