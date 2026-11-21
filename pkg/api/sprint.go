package api

import (
	"fmt"
	"math"
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

// UpdateVersion patches an existing version (release/sprint) by ID. Only the
// fields set in req are sent, so an empty field leaves the current value intact.
// Versions carry no lockVersion, so unlike UpdateWorkPackage there is nothing
// to fetch first.
func (c *Client) UpdateVersion(id int, req *UpdateVersionRequest) (*Version, error) {
	var v Version
	if err := c.Patch(fmt.Sprintf("/versions/%d", id), req, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateVersionRequest is the partial PATCH body for updating a version.
// omitempty on every field means a zero value is dropped from the payload
// rather than clearing the server-side value.
type UpdateVersionRequest struct {
	Name        string       `json:"name,omitempty"`
	Status      string       `json:"status,omitempty"`
	StartDate   string       `json:"startDate,omitempty"`
	EndDate     string       `json:"endDate,omitempty"`
	Description *Formattable `json:"description,omitempty"`
}

// FindActiveSprint finds the currently active (open) version for a project.
// When several versions are open, it prefers the shortest one whose date range
// contains today, so neither a future sprint left open nor a long-running
// bucket version shadows the real current one. If none contains today (or
// dates are missing) it falls back to the first open version.
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

// selectActiveSprint returns the open version with the shortest
// [StartDate, EndDate] range containing today (YYYY-MM-DD, inclusive; the first
// listed wins a tie), or the first open version if none contains today, or nil
// if no version is open. Containment compares the ISO-8601 strings
// lexicographically; a version is only considered "current" when both of its
// dates are set.
func selectActiveSprint(versions []Version, today string) *Version {
	var firstOpen, best *Version
	var bestDays int
	for i := range versions {
		v := &versions[i]
		if v.Status != "open" {
			continue
		}
		if firstOpen == nil {
			firstOpen = v
		}
		if v.StartDate == "" || v.EndDate == "" || today < v.StartDate || v.EndDate < today {
			continue
		}
		if days := spanDays(v); best == nil || days < bestDays {
			best, bestDays = v, days
		}
	}
	if best != nil {
		return best
	}
	return firstOpen
}

// spanDays is the length of a version's date range; unparseable dates rank
// last so a well-formed sprint always wins.
func spanDays(v *Version) int {
	start, err1 := time.Parse("2006-01-02", v.StartDate)
	end, err2 := time.Parse("2006-01-02", v.EndDate)
	if err1 != nil || err2 != nil {
		return math.MaxInt
	}
	return int(end.Sub(start).Hours() / 24)
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
