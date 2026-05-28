package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// WorkPackage represents an OpenProject work package.
type WorkPackage struct {
	ID             int    `json:"id"`
	LockVersion    int    `json:"lockVersion"`
	Subject        string `json:"subject"`
	Description    *Formattable `json:"description,omitempty"`
	StoryPoints    *int   `json:"storyPoints,omitempty"`
	PercentageDone int    `json:"percentageDone"`
	StartDate      string `json:"startDate,omitempty"`
	DueDate        string `json:"dueDate,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	Links          WPLinks `json:"_links"`
}

// Formattable represents a formattable text field.
type Formattable struct {
	Format string `json:"format"`
	Raw    string `json:"raw"`
	HTML   string `json:"html,omitempty"`
}

// WPLinks contains the linked resources of a work package.
type WPLinks struct {
	Self       Link `json:"self"`
	Type       Link `json:"type"`
	Status     Link `json:"status"`
	Priority   Link `json:"priority"`
	Author     Link `json:"author"`
	Assignee   Link `json:"assignee"`
	Project    Link `json:"project"`
	Version    Link `json:"version"`
	Parent     Link `json:"parent"`
	Responsible Link `json:"responsible"`
}

// Link represents an HAL link.
type Link struct {
	Href  string `json:"href"`
	Title string `json:"title"`
}

// WPCollection is the response from listing work packages.
type WPCollection struct {
	Total    int           `json:"total"`
	Count    int           `json:"count"`
	Embedded struct {
		Elements []WorkPackage `json:"elements"`
	} `json:"_embedded"`
}

// LinkValue can be a single Link or a []Link for multi-value custom fields.
// Both serialize correctly to JSON.
type LinkValue interface{}

// CreateWPRequest is the request body for creating a work package.
type CreateWPRequest struct {
	Subject     string                `json:"subject"`
	Description *Formattable          `json:"description,omitempty"`
	StoryPoints *int                  `json:"storyPoints,omitempty"`
	StartDate   string                `json:"startDate,omitempty"`
	DueDate     string                `json:"dueDate,omitempty"`
	Links       map[string]LinkValue  `json:"_links"`
}

// SetLink sets a single-value link field.
func (r *CreateWPRequest) SetLink(field string, link Link) {
	r.Links[field] = link
}

// SetMultiLink sets a multi-value link field (for custom fields like components, labels).
func (r *CreateWPRequest) SetMultiLink(field string, links []Link) {
	r.Links[field] = links
}

// UpdateWPRequest is the request body for updating a work package.
type UpdateWPRequest struct {
	LockVersion    int                  `json:"lockVersion"`
	Subject        string              `json:"subject,omitempty"`
	Description    *Formattable        `json:"description,omitempty"`
	StoryPoints    *int                `json:"storyPoints,omitempty"`
	PercentageDone *int                `json:"percentageDone,omitempty"`
	Links          map[string]LinkValue `json:"_links,omitempty"`
}

// ListWorkPackages lists work packages with optional filters.
func (c *Client) ListWorkPackages(project string, filters []Filter, sortBy string, pageSize int) (*WPCollection, error) {
	path := fmt.Sprintf("/projects/%s/work_packages", project)
	params := url.Values{}

	if len(filters) > 0 {
		filterJSON, err := json.Marshal(filters)
		if err != nil {
			return nil, fmt.Errorf("marshaling filters: %w", err)
		}
		params.Set("filters", string(filterJSON))
	}

	if sortBy != "" {
		params.Set("sortBy", sortBy)
	}

	if pageSize > 0 {
		params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	} else {
		params.Set("pageSize", "100")
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result WPCollection
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Filter represents an OpenProject API filter.
type Filter map[string]FilterSpec

// FilterSpec specifies the operator and values for a filter.
type FilterSpec struct {
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// NewFilter creates a new filter.
func NewFilter(field, operator string, values ...string) Filter {
	return Filter{
		field: {Operator: operator, Values: values},
	}
}

// GetWorkPackage retrieves a single work package by ID.
func (c *Client) GetWorkPackage(id int) (*WorkPackage, error) {
	var wp WorkPackage
	if err := c.Get(fmt.Sprintf("/work_packages/%d", id), &wp); err != nil {
		return nil, err
	}
	return &wp, nil
}

// CreateWorkPackage creates a new work package in the given project.
func (c *Client) CreateWorkPackage(project string, req *CreateWPRequest) (*WorkPackage, error) {
	var wp WorkPackage
	path := fmt.Sprintf("/projects/%s/work_packages", project)
	if err := c.Post(path, req, &wp); err != nil {
		return nil, err
	}
	return &wp, nil
}

// UpdateWorkPackage updates an existing work package.
// Automatically fetches lockVersion to avoid conflicts.
func (c *Client) UpdateWorkPackage(id int, req *UpdateWPRequest) (*WorkPackage, error) {
	// Fetch current lockVersion if not set
	if req.LockVersion == 0 {
		current, err := c.GetWorkPackage(id)
		if err != nil {
			return nil, fmt.Errorf("fetching lockVersion: %w", err)
		}
		req.LockVersion = current.LockVersion
	}

	var wp WorkPackage
	path := fmt.Sprintf("/work_packages/%d", id)
	if err := c.Patch(path, req, &wp); err != nil {
		return nil, err
	}
	return &wp, nil
}
