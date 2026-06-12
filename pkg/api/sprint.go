package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Version represents an OpenProject version (used as sprint in Scrum).
type Version struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description *Formattable `json:"description,omitempty"`
	StartDate   string       `json:"startDate,omitempty"`
	EndDate     string       `json:"endDate,omitempty"`
	Status      string       `json:"status"`
	Kind        string       `json:"kind,omitempty"`
	Links       struct {
		Self            Link `json:"self"`
		DefiningProject Link `json:"definingProject"`
	} `json:"_links"`
}

// VersionCollection is the response from listing versions.
type VersionCollection struct {
	Total    int `json:"total"`
	Embedded struct {
		Elements []Version `json:"elements"`
	} `json:"_embedded"`
}

// ListVersions lists all versions for a project. The page size is explicit:
// OpenProject's default is 25, which silently truncates once a project
// accumulates more sprints+releases than that, breaking name resolution.
func (c *Client) ListVersions(project string) (*VersionCollection, error) {
	var result VersionCollection
	if err := c.Get(fmt.Sprintf("/projects/%s/versions?pageSize=500", project), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateVersion creates a new version (sprint) in a project.
func (c *Client) CreateVersion(req *CreateVersionRequest) (*Version, error) {
	var v Version
	if err := c.Post("/versions", req, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateVersionRequest is the request body for creating a version.
type CreateVersionRequest struct {
	Name        string          `json:"name"`
	Description *Formattable    `json:"description,omitempty"`
	StartDate   string          `json:"startDate,omitempty"`
	EndDate     string          `json:"endDate,omitempty"`
	Status      string          `json:"status,omitempty"`
	Kind        string          `json:"kind,omitempty"`
	Links       map[string]Link `json:"_links"`
}

// FindActiveSprint finds the currently active (open) version for a project.
// When several versions are open, it prefers the one whose date range contains
// today, so a future sprint that was left open doesn't shadow the real current
// one. If none contains today (or dates are missing) it falls back to the first
// open version.
func (c *Client) FindActiveSprint(project string) (*Version, error) {
	versions, err := c.ListVersions(project)
	if err != nil {
		return nil, err
	}

	if v := selectActiveSprint(versions.Embedded.Elements, time.Now().Format("2006-01-02")); v != nil {
		return v, nil
	}
	return nil, fmt.Errorf("no active sprint found in project %q", project)
}

// selectActiveSprint returns the open version whose [StartDate, EndDate] range
// contains today (YYYY-MM-DD, inclusive), or the first open version if none
// does, or nil if no version is open. Date comparison is lexicographic, which
// is correct for ISO-8601 dates; a version is only considered "current" when
// both of its dates are set.
func selectActiveSprint(versions []Version, today string) *Version {
	var firstOpen *Version
	for i := range versions {
		v := &versions[i]
		if v.Status != "open" {
			continue
		}
		if firstOpen == nil {
			firstOpen = v
		}
		if v.StartDate != "" && v.EndDate != "" && v.StartDate <= today && today <= v.EndDate {
			return v
		}
	}
	return firstOpen
}

// ResolveVersion finds a version by name, or returns the active sprint if name is empty.
func (c *Client) ResolveVersion(project, name string) (*Version, error) {
	if name == "" {
		return c.FindActiveSprint(project)
	}
	versions, err := c.ListVersions(project)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}

	// Prefer a name match (exact, then case-insensitive).
	var caseMatch *Version
	for i := range versions.Embedded.Elements {
		v := &versions.Embedded.Elements[i]
		if v.Name == name {
			return v, nil
		}
		if caseMatch == nil && strings.EqualFold(v.Name, name) {
			caseMatch = v
		}
	}
	if caseMatch != nil {
		return caseMatch, nil
	}

	// Fall back to numeric ID lookup (IDs are shown in 'op sprint list').
	if id, err := strconv.Atoi(name); err == nil {
		for i := range versions.Embedded.Elements {
			if versions.Embedded.Elements[i].ID == id {
				return &versions.Embedded.Elements[i], nil
			}
		}
	}
	return nil, fmt.Errorf("sprint %q not found", name)
}

// ResolveRelease finds a release (kind=release) by name or numeric ID.
// On failure it lists available release names to help the caller correct the input.
func (c *Client) ResolveRelease(project, name string) (*Version, error) {
	versions, err := c.ListVersions(project)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}

	var releases []Version
	for _, v := range versions.Embedded.Elements {
		if v.Kind == "release" {
			releases = append(releases, v)
		}
	}

	for i := range releases {
		if releases[i].Name == name {
			return &releases[i], nil
		}
	}
	for i := range releases {
		if strings.EqualFold(releases[i].Name, name) {
			return &releases[i], nil
		}
	}
	if id, err := strconv.Atoi(name); err == nil {
		for i := range releases {
			if releases[i].ID == id {
				return &releases[i], nil
			}
		}
	}

	names := make([]string, len(releases))
	for i, r := range releases {
		names[i] = r.Name
	}
	return nil, fmt.Errorf("release %q not found; available: %s", name, strings.Join(names, ", "))
}

// VersionFilter creates a version filter for work package queries.
//
// It does not compare the version's definingProject against the project
// argument: that href uses the numeric project ID (e.g. "/api/v3/projects/382")
// while callers pass the project identifier (e.g. "app"), so the comparison
// produced false positives for versions genuinely owned by the project.
// OpenProject validates the version server-side and accepts filtering by both
// owned and shared versions, so we only guard against an invalid ID here.
func VersionFilter(v *Version, project string) (Filter, error) {
	if v.ID <= 0 {
		return nil, fmt.Errorf("sprint %q has invalid ID %d", v.Name, v.ID)
	}
	return NewFilter("version", "=", fmt.Sprintf("%d", v.ID)), nil
}
