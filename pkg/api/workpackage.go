package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// WorkPackage represents an OpenProject work package.
type WorkPackage struct {
	ID             int          `json:"id"`
	LockVersion    int          `json:"lockVersion"`
	Subject        string       `json:"subject"`
	Description    *Formattable `json:"description,omitempty"`
	StoryPoints    *int         `json:"storyPoints,omitempty"`
	PercentageDone int          `json:"percentageDone"`
	StartDate      string       `json:"startDate,omitempty"`
	DueDate        string       `json:"dueDate,omitempty"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	JiraID         string       // populated by UnmarshalJSON from configured jira-id field
	UserStory      *Formattable `json:"customField36,omitempty"`
	Links          WPLinks      `json:"_links"`
}

// wpWire is used internally for JSON decoding with the default customField3 tag.
type wpWire struct {
	ID             int          `json:"id"`
	LockVersion    int          `json:"lockVersion"`
	Subject        string       `json:"subject"`
	Description    *Formattable `json:"description,omitempty"`
	StoryPoints    *int         `json:"storyPoints,omitempty"`
	PercentageDone int          `json:"percentageDone"`
	StartDate      string       `json:"startDate,omitempty"`
	DueDate        string       `json:"dueDate,omitempty"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	JiraID         string       `json:"customField3,omitempty"`
	UserStory      *Formattable `json:"customField36,omitempty"`
	Links          WPLinks      `json:"_links"`
}

// UnmarshalJSON populates JiraID from whichever custom field is configured for
// "jira-id" (default: customField3). If the instance uses a different field
// number, set it in ~/.oprc under custom_fields.jira-id.field.
func (wp *WorkPackage) UnmarshalJSON(data []byte) error {
	var w wpWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*wp = WorkPackage(w)
	// If the configured field differs from the default, read from the actual field.
	if fieldKey := jiraIDFieldKey(); fieldKey != "customField3" {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err == nil {
			if v, ok := raw[fieldKey]; ok {
				var s string
				if err := json.Unmarshal(v, &s); err == nil {
					wp.JiraID = s
				}
			}
		}
	}
	return nil
}

// jiraIDFieldKey returns the custom field key for JIRA ID from config, defaulting to customField3.
func jiraIDFieldKey() string {
	if cf, ok := customFields["jira-id"]; ok && cf.Field != "" {
		return cf.Field
	}
	return "customField3"
}

// Formattable represents a formattable text field.
type Formattable struct {
	Format string `json:"format"`
	Raw    string `json:"raw"`
	HTML   string `json:"html,omitempty"`
}

// WPLinks contains the linked resources of a work package.
type WPLinks struct {
	Self        Link `json:"self"`
	Type        Link `json:"type"`
	Status      Link `json:"status"`
	Priority    Link `json:"priority"`
	Author      Link `json:"author"`
	Assignee    Link `json:"assignee"`
	Project     Link `json:"project"`
	Version     Link `json:"version"`
	Parent      Link `json:"parent"`
	Responsible Link `json:"responsible"`

	// Multi-value custom fields (epochbase.com instance): component, product,
	// and label. Each is an array of links; absent/empty fields decode to nil.
	Component []Link `json:"customField12"`
	Product   []Link `json:"customField4"`
	Label     []Link `json:"customField13"`

	// Release is a version link scoped to kind=release (customField50). Separate
	// from Version, which holds the sprint; a work package can carry both. The
	// API serializes it as an array even though it holds at most one release.
	Release []Link `json:"customField50"`
}

// Link represents an HAL link.
type Link struct {
	Href  string `json:"href"`
	Title string `json:"title"`
}

// WPCollection is the response from listing work packages.
type WPCollection struct {
	Total    int `json:"total"`
	Count    int `json:"count"`
	Embedded struct {
		Elements []WorkPackage `json:"elements"`
	} `json:"_embedded"`
}

// LinkValue can be a single Link or a []Link for multi-value custom fields.
// Both serialize correctly to JSON.
type LinkValue interface{}

// CreateWPRequest is the request body for creating a work package.
type CreateWPRequest struct {
	Subject     string               `json:"subject"`
	Description *Formattable         `json:"description,omitempty"`
	StoryPoints *int                 `json:"storyPoints,omitempty"`
	StartDate   string               `json:"startDate,omitempty"`
	DueDate     string               `json:"dueDate,omitempty"`
	Links       map[string]LinkValue `json:"_links"`
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
	Subject        string               `json:"subject,omitempty"`
	Description    *Formattable         `json:"description,omitempty"`
	StoryPoints    *int                 `json:"storyPoints,omitempty"`
	PercentageDone *int                 `json:"percentageDone,omitempty"`
	StartDate      string               `json:"startDate,omitempty"`
	DueDate        string               `json:"dueDate,omitempty"`
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

// ListAllWorkPackages lists work packages across all projects via the global
// endpoint (no project scope). Used by the cross-project overview.
func (c *Client) ListAllWorkPackages(filters []Filter, sortBy string, pageSize int) (*WPCollection, error) {
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
		params.Set("pageSize", "200")
	}

	var result WPCollection
	if err := c.Get("/work_packages?"+params.Encode(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchByJiraID finds work packages whose JIRA ID custom field matches the
// given value. The field used is configured via jira-id in ~/.oprc
// (default: customField3). Searches across all projects.
func (c *Client) SearchByJiraID(jiraID string) (*WPCollection, error) {
	filters := []Filter{NewFilter(jiraIDFieldKey(), "~", jiraID)}
	filterJSON, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("marshaling filters: %w", err)
	}
	params := url.Values{}
	params.Set("filters", string(filterJSON))
	params.Set("pageSize", "20")

	var result WPCollection
	if err := c.Get("/work_packages?"+params.Encode(), &result); err != nil {
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

// createRelationRequest is the body for creating a relation.
type createRelationRequest struct {
	Type  string               `json:"type"`
	Links map[string]LinkValue `json:"_links"`
}

// CreateRelation creates a relation between two work packages.
func (c *Client) CreateRelation(fromID int, relType string, toID int) error {
	body := createRelationRequest{
		Type: relType,
		Links: map[string]LinkValue{
			"to": Link{Href: fmt.Sprintf("/api/v3/work_packages/%d", toID)},
		},
	}
	path := fmt.Sprintf("/work_packages/%d/relations", fromID)
	return c.Post(path, body, nil)
}

// UpdateWorkPackage updates an existing work package.
// Automatically fetches lockVersion to avoid conflicts.
func (c *Client) UpdateWorkPackage(id int, req *UpdateWPRequest) (*WorkPackage, error) {
	// Work on a copy — writing the fetched lockVersion into the caller's
	// request would poison reuse: a second call (retry, bulk loop) would send
	// the first ticket's stale version and 409.
	r := *req

	// Fetch current lockVersion if not set
	if r.LockVersion == 0 {
		current, err := c.GetWorkPackage(id)
		if err != nil {
			return nil, fmt.Errorf("fetching lockVersion: %w", err)
		}
		r.LockVersion = current.LockVersion
	}

	var wp WorkPackage
	path := fmt.Sprintf("/work_packages/%d", id)
	if err := c.Patch(path, r, &wp); err != nil {
		return nil, err
	}
	return &wp, nil
}

// Relation is a typed link between two work packages.
type Relation struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Links struct {
		From Link `json:"from"`
		To   Link `json:"to"`
	} `json:"_links"`
}

// RelationCollection is the response from listing a work package's relations.
type RelationCollection struct {
	Total    int `json:"total"`
	Embedded struct {
		Elements []Relation `json:"elements"`
	} `json:"_embedded"`
}

// ListRelations lists the relations of a work package.
func (c *Client) ListRelations(wpID int) (*RelationCollection, error) {
	var result RelationCollection
	if err := c.Get(fmt.Sprintf("/work_packages/%d/relations", wpID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteRelation removes a relation by its relation ID (not a work-package ID).
func (c *Client) DeleteRelation(relID int) error {
	resp, err := c.DoRaw("DELETE", fmt.Sprintf("/api/v3/relations/%d", relID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("deleting relation %d: HTTP %d", relID, resp.StatusCode)
	}
	return nil
}
