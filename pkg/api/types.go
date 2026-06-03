package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// NamedResource represents a type, status, or priority with id, name, and href.
type NamedResource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Href string `json:"-"`
}

// Resolver caches types, statuses, priorities, and users for name→href lookup.
type Resolver struct {
	client     APIClient
	project    string
	mu         sync.Mutex
	types      []NamedResource
	statuses   []NamedResource
	priorities []NamedResource
	users      []NamedResource
}

// NewResolver creates a resolver attached to the given client.
func NewResolver(c APIClient, project string) *Resolver {
	return &Resolver{client: c, project: project}
}

type collectionResponse struct {
	Embedded struct {
		Elements []json.RawMessage `json:"elements"`
	} `json:"_embedded"`
	Total int `json:"total"`
}

type resourceWithLinks struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

// fetchCollection fetches a paginated collection and extracts NamedResources.
func (r *Resolver) fetchCollection(path string) ([]NamedResource, error) {
	var resp collectionResponse
	if err := r.client.Get(path, &resp); err != nil {
		return nil, err
	}

	var resources []NamedResource
	for _, raw := range resp.Embedded.Elements {
		var res resourceWithLinks
		if err := json.Unmarshal(raw, &res); err != nil {
			continue
		}
		resources = append(resources, NamedResource{
			ID:   res.ID,
			Name: res.Name,
			Href: res.Links.Self.Href,
		})
	}
	return resources, nil
}

// Types returns all work package types (cached).
func (r *Resolver) Types() ([]NamedResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.types != nil {
		return r.types, nil
	}

	types, err := r.fetchCollection("/types")
	if err != nil {
		return nil, fmt.Errorf("fetching types: %w", err)
	}
	r.types = types
	return r.types, nil
}

// Statuses returns all work package statuses (cached).
func (r *Resolver) Statuses() ([]NamedResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.statuses != nil {
		return r.statuses, nil
	}

	statuses, err := r.fetchCollection("/statuses")
	if err != nil {
		return nil, fmt.Errorf("fetching statuses: %w", err)
	}
	r.statuses = statuses
	return r.statuses, nil
}

// Priorities returns all work package priorities (cached).
func (r *Resolver) Priorities() ([]NamedResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.priorities != nil {
		return r.priorities, nil
	}

	priorities, err := r.fetchCollection("/priorities")
	if err != nil {
		return nil, fmt.Errorf("fetching priorities: %w", err)
	}
	r.priorities = priorities
	return r.priorities, nil
}

// Users returns available assignees for the project (cached).
// Falls back to global /users if no project is set.
func (r *Resolver) Users() ([]NamedResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.users != nil {
		return r.users, nil
	}

	path := "/users?pageSize=200"
	if r.project != "" {
		path = fmt.Sprintf("/projects/%s/available_assignees?pageSize=200", r.project)
	}

	users, err := r.fetchCollection(path)
	if err != nil {
		return nil, fmt.Errorf("fetching users: %w", err)
	}
	r.users = users
	return r.users, nil
}

// NormalizeName lowercases a name and treats '-' and '_' as spaces, so that
// CLI-friendly values like "in-progress" match human labels like "In progress".
func NormalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

// resolve finds a resource by case-insensitive name match.
func resolve(resources []NamedResource, name string) (NamedResource, error) {
	lower := strings.ToLower(name)
	for _, r := range resources {
		if strings.ToLower(r.Name) == lower {
			return r, nil
		}
	}

	// Normalized match: "in-progress" / "in_progress" → "in progress".
	norm := NormalizeName(name)
	for _, r := range resources {
		if NormalizeName(r.Name) == norm {
			return r, nil
		}
	}

	// Try prefix match
	for _, r := range resources {
		if strings.HasPrefix(strings.ToLower(r.Name), lower) {
			return r, nil
		}
	}

	names := make([]string, len(resources))
	for i, r := range resources {
		names[i] = r.Name
	}
	return NamedResource{}, fmt.Errorf("unknown %q, available: %s", name, strings.Join(names, ", "))
}

// ResolveType resolves a type name to its NamedResource.
func (r *Resolver) ResolveType(name string) (NamedResource, error) {
	types, err := r.Types()
	if err != nil {
		return NamedResource{}, err
	}
	return resolve(types, name)
}

// ResolveStatus resolves a status name to its NamedResource.
func (r *Resolver) ResolveStatus(name string) (NamedResource, error) {
	statuses, err := r.Statuses()
	if err != nil {
		return NamedResource{}, err
	}
	return resolve(statuses, name)
}

// ResolvePriority resolves a priority name to its NamedResource.
func (r *Resolver) ResolvePriority(name string) (NamedResource, error) {
	priorities, err := r.Priorities()
	if err != nil {
		return NamedResource{}, err
	}
	return resolve(priorities, name)
}

// ResolveUser resolves a user name (with or without @) to its NamedResource.
func (r *Resolver) ResolveUser(name string) (NamedResource, error) {
	name = strings.TrimPrefix(name, "@")
	users, err := r.Users()
	if err != nil {
		return NamedResource{}, err
	}

	lower := strings.ToLower(name)

	// Try login/name match
	for _, u := range users {
		if strings.ToLower(u.Name) == lower {
			return u, nil
		}
	}

	// Try partial match
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Name), lower) {
			return u, nil
		}
	}

	names := make([]string, len(users))
	for i, u := range users {
		names[i] = u.Name
	}
	return NamedResource{}, fmt.Errorf("unknown user %q, available: %s", name, strings.Join(names, ", "))
}

// TODO: make configurable via .oprc custom_fields section.
// Currently hardcoded for the epochbase.com OpenProject instance.
var (
	// Components (customField12)
	ComponentOptions = map[string]string{
		"android":     "/api/v3/custom_options/42",
		"ios":         "/api/v3/custom_options/43",
		"ott":         "/api/v3/custom_options/44",
		"engineering": "/api/v3/custom_options/45",
		"analytics":   "/api/v3/custom_options/46",
	}

	// Product (customField4)
	ProductOptions = map[string]string{
		"eet":         "/api/v3/custom_options/237",
		"entd":        "/api/v3/custom_options/238",
		"others":      "/api/v3/custom_options/239",
		"djy":         "/api/v3/custom_options/240",
		"cntd":        "/api/v3/custom_options/241",
		"competition": "/api/v3/custom_options/242",
	}

	// Tech Area (customField6)
	TechAreaOptions = map[string]string{
		"web":    "/api/v3/custom_options/255",
		"adtech": "/api/v3/custom_options/256",
		"app":    "/api/v3/custom_options/259",
		"video":  "/api/v3/custom_options/266",
		"infra":  "/api/v3/custom_options/268",
		"portal": "/api/v3/custom_options/271",
		"seo":    "/api/v3/custom_options/273",
	}

	// Labels (customField13)
	LabelOptions = map[string]string{
		"team#appios":     "/api/v3/custom_options/447",
		"team#appandroid": "/api/v3/custom_options/448",
		"team#appall":     "/api/v3/custom_options/452",
		"team#web":        "/api/v3/custom_options/453",
		"ntd":             "/api/v3/custom_options/449",
		"seo":             "/api/v3/custom_options/450",
		"roku":            "/api/v3/custom_options/451",
	}
)

// matchOption resolves name against an option map using an exact match first,
// then a unique case-insensitive prefix match. It returns the matched key.
//
// Matching is deterministic: an ambiguous prefix (e.g. "team#app" matching
// several labels) is rejected with the candidates listed, rather than silently
// picking one — map iteration order is random, so a prefix that hit multiple
// keys previously returned an arbitrary option. Unknown names list every valid
// option, sorted, so the same input always yields the same message.
func matchOption(options map[string]string, name string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if _, ok := options[lower]; ok {
		return lower, nil
	}

	var matches []string
	for k := range options {
		if lower != "" && strings.HasPrefix(k, lower) {
			matches = append(matches, k)
		}
	}
	sort.Strings(matches)

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("unknown %q, available: %s", name, sortedKeys(options))
	default:
		return "", fmt.Errorf("ambiguous %q, matches: %s", name, strings.Join(matches, ", "))
	}
}

// sortedKeys returns the option keys joined and sorted, for stable error text.
func sortedKeys(options map[string]string) string {
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// OptionID resolves a name to the numeric custom-option ID, for use as a filter
// value (e.g. customField12 = "42").
func OptionID(options map[string]string, name string) (string, error) {
	key, err := matchOption(options, name)
	if err != nil {
		return "", err
	}
	href := options[key]
	parts := strings.Split(href, "/")
	return parts[len(parts)-1], nil
}

// ResolveCustomOption resolves a name to an href from a custom field option map,
// for use as a link value when creating or updating a work package.
func ResolveCustomOption(options map[string]string, name string) (string, error) {
	key, err := matchOption(options, name)
	if err != nil {
		return "", err
	}
	return options[key], nil
}

// ResolveEpic finds an epic work package by name in the project.
func (r *Resolver) ResolveEpic(name string) (NamedResource, error) {
	if r.project == "" {
		return NamedResource{}, fmt.Errorf("project required to resolve epic")
	}

	// Fetch epics (type ID 5)
	filterJSON := url.QueryEscape(`[{"type":{"operator":"=","values":["5"]}}]`)
	path := fmt.Sprintf("/projects/%s/work_packages?filters=%s&pageSize=50",
		r.project, filterJSON)

	var result struct {
		Embedded struct {
			Elements []struct {
				ID      int    `json:"id"`
				Subject string `json:"subject"`
				Links   struct {
					Self Link `json:"self"`
				} `json:"_links"`
			} `json:"elements"`
		} `json:"_embedded"`
	}

	if err := r.client.Get(path, &result); err != nil {
		return NamedResource{}, fmt.Errorf("fetching epics: %w", err)
	}

	lower := strings.ToLower(name)
	for _, e := range result.Embedded.Elements {
		if strings.Contains(strings.ToLower(e.Subject), lower) {
			return NamedResource{
				ID:   e.ID,
				Name: e.Subject,
				Href: e.Links.Self.Href,
			}, nil
		}
	}

	var names []string
	for _, e := range result.Embedded.Elements {
		names = append(names, e.Subject)
	}
	return NamedResource{}, fmt.Errorf("unknown epic %q, available: %s", name, strings.Join(names, ", "))
}
