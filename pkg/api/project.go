package api

import "fmt"

// Project represents an OpenProject project.
type Project struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	Identifier  string       `json:"identifier"`
	Description *Formattable `json:"description,omitempty"`
	Active      bool         `json:"active"`
	Public      bool         `json:"public"`
	Links       struct {
		Self Link `json:"self"`
	} `json:"_links"`
}

// ProjectCollection is the response from listing projects.
type ProjectCollection struct {
	Total    int `json:"total"`
	Embedded struct {
		Elements []Project `json:"elements"`
	} `json:"_embedded"`
}

// ListProjects lists all visible projects.
func (c *Client) ListProjects() (*ProjectCollection, error) {
	var result ProjectCollection
	if err := c.Get("/projects?pageSize=100", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProject retrieves a project by identifier.
func (c *Client) GetProject(identifier string) (*Project, error) {
	var p Project
	if err := c.Get(fmt.Sprintf("/projects/%s", identifier), &p); err != nil {
		return nil, err
	}
	return &p, nil
}
