package api

import (
	"fmt"
	"regexp"
	"strings"
)

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

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// unknownProjectMessage explains a 404 under /projects/<identifier> when that
// project doesn't exist, naming identifiers that share a word with it. It
// returns "" when the path names no project or the project exists, so the
// server's own message stands.
func (c *Client) unknownProjectMessage(path string) string {
	rest, ok := strings.CutPrefix(path, "/projects/")
	if !ok {
		return ""
	}
	identifier, _, _ := strings.Cut(rest, "/")
	identifier, _, _ = strings.Cut(identifier, "?")
	if identifier == "" {
		return ""
	}
	projects, err := c.ListProjects()
	if err != nil {
		return ""
	}
	var words []string
	for _, w := range nonAlnum.Split(strings.ToLower(identifier), -1) {
		if len(w) >= 3 {
			words = append(words, w)
		}
	}
	var similar []string
	for _, p := range projects.Embedded.Elements {
		if strings.EqualFold(p.Identifier, identifier) || fmt.Sprint(p.ID) == identifier {
			return ""
		}
		for _, w := range words {
			if strings.Contains(p.Identifier, w) || strings.Contains(strings.ToLower(p.Name), w) {
				similar = append(similar, p.Identifier)
				break
			}
		}
	}
	msg := fmt.Sprintf("project %q not found", identifier)
	if len(similar) > 0 {
		msg += fmt.Sprintf("; did you mean %s?", strings.Join(similar[:min(len(similar), 5)], ", "))
	}
	return msg + " (op projects lists every identifier)"
}
