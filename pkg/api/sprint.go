package api

import "fmt"

// Version represents an OpenProject version (used as sprint in Scrum).
type Version struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Description *Formattable `json:"description,omitempty"`
	StartDate   string       `json:"startDate,omitempty"`
	EndDate     string       `json:"endDate,omitempty"`
	Status      string       `json:"status"`
	Links       struct {
		Self Link `json:"self"`
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
	for _, v := range versions.Embedded.Elements {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("sprint %q not found", name)
}
