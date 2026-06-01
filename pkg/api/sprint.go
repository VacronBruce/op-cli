package api

import (
	"fmt"
	"strings"
)

// Version represents an OpenProject version (used as sprint in Scrum).
type Version struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description *Formattable `json:"description,omitempty"`
	StartDate   string       `json:"startDate,omitempty"`
	EndDate     string       `json:"endDate,omitempty"`
	Status      string       `json:"status"`
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

// ListVersions lists all versions for a project.
func (c *Client) ListVersions(project string) (*VersionCollection, error) {
	var result VersionCollection
	if err := c.Get(fmt.Sprintf("/projects/%s/versions", project), &result); err != nil {
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
	Name        string `json:"name"`
	Description *Formattable `json:"description,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	EndDate     string `json:"endDate,omitempty"`
	Status      string `json:"status,omitempty"`
	Links       map[string]Link `json:"_links"`
}

// FindActiveSprint finds the currently active (open) version for a project.
func (c *Client) FindActiveSprint(project string) (*Version, error) {
	versions, err := c.ListVersions(project)
	if err != nil {
		return nil, err
	}

	for _, v := range versions.Embedded.Elements {
		if v.Status == "open" {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no active sprint found in project %q", project)
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

	// Try exact match first, then case-insensitive match.
	var caseMatch *Version
	for _, v := range versions.Embedded.Elements {
		if v.Name == name {
			return &v, nil
		}
		if caseMatch == nil && strings.EqualFold(v.Name, name) {
			v := v
			caseMatch = &v
		}
	}
	if caseMatch != nil {
		return caseMatch, nil
	}
	return nil, fmt.Errorf("sprint %q not found", name)
}

// VersionFilter creates a version filter for work package queries.
// It validates the version belongs to the given project. Shared versions
// from other projects are visible in the versions list but may be rejected
// by the work package filter API.
func VersionFilter(v *Version, project string) (Filter, error) {
	if v.ID <= 0 {
		return nil, fmt.Errorf("sprint %q has invalid ID %d", v.Name, v.ID)
	}
	// Check if this is a shared version from another project.
	if href := v.Links.DefiningProject.Href; href != "" {
		if !strings.HasSuffix(href, "/"+project) {
			return nil, fmt.Errorf(
				"sprint %q (ID %d) belongs to another project and cannot be used to filter work packages here.\n"+
					"Use 'op sprint list' to see sprints defined in this project",
				v.Name, v.ID)
		}
	}
	return NewFilter("version", "=", fmt.Sprintf("%d", v.ID)), nil
}
