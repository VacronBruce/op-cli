package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// APIClient defines the interface for OpenProject API operations.
// *Client satisfies this interface. Tests can inject a mock implementation.
type APIClient interface {
	RequireProject() (string, error)
	Get(path string, result interface{}) error
	Post(path string, body interface{}, result interface{}) error
	Patch(path string, body interface{}, result interface{}) error
	DoRaw(method, href string) (*http.Response, error)

	// Work packages
	GetWorkPackage(id int) (*WorkPackage, error)
	ListWorkPackages(project string, filters []Filter, sortBy string, pageSize int) (*WPCollection, error)
	ListAllWorkPackages(filters []Filter, sortBy string, pageSize int) (*WPCollection, error)
	SearchByJiraID(jiraID string) (*WPCollection, error)
	CreateWorkPackage(project string, req *CreateWPRequest) (*WorkPackage, error)
	UpdateWorkPackage(id int, req *UpdateWPRequest) (*WorkPackage, error)

	// Versions/sprints
	ListVersions(project string) (*VersionCollection, error)
	CreateVersion(req *CreateVersionRequest) (*Version, error)
	FindActiveSprint(project string) (*Version, error)
	ResolveVersion(project, name string) (*Version, error)
	ResolveRelease(project, name string) (*Version, error)

	// Projects
	ListProjects() (*ProjectCollection, error)
	GetProject(identifier string) (*Project, error)

	// Users
	GetMe() (*User, error)

	// Attachments
	UploadAttachment(wpID int, filePath string, description string) (*Attachment, error)

	// Activities/comments
	ListActivities(wpID int) (*ActivityCollection, error)
	PostComment(wpID int, markdown string) error
	EditComment(activityID int, markdown string) error

	// Relations
	CreateRelation(fromID int, relType string, toID int) error
}

// Client is the OpenProject API v3 client.
type Client struct {
	BaseURL    string
	APIKey     string
	Project    string
	HTTPClient *http.Client
}

// APIError represents an error response from the OpenProject API.
type APIError struct {
	Type       string `json:"_type"`
	ErrorID    string `json:"errorIdentifier"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	// Embedded.Errors holds per-field details. When several constraints are
	// violated, the top-level Message is generic ("Multiple field constraints
	// have been violated.") and the specifics live here.
	Embedded struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"_embedded"`
}

func (e *APIError) Error() string {
	msg := e.Message
	if len(e.Embedded.Errors) > 0 {
		details := make([]string, 0, len(e.Embedded.Errors))
		for _, sub := range e.Embedded.Errors {
			if sub.Message != "" && sub.Message != e.Message {
				details = append(details, sub.Message)
			}
		}
		if len(details) > 0 {
			msg = fmt.Sprintf("%s (%s)", msg, strings.Join(details, "; "))
		}
	}
	return fmt.Sprintf("OpenProject API error (%d): %s", e.StatusCode, msg)
}

// Collection represents a paginated API response.
type Collection struct {
	Type   string            `json:"_type"`
	Total  int               `json:"total"`
	Count  int               `json:"count"`
	Offset int               `json:"offset"`
	Items  []json.RawMessage `json:"_embedded"`
}

// NewClient creates a new OpenProject API client.
func NewClient(baseURL, apiKey, project string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Project: project,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// do executes an HTTP request with auth headers.
func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + "/api/v3" + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth("apikey", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if json.Unmarshal(data, apiErr) != nil {
			apiErr.Message = string(data)
		}
		return nil, apiErr
	}

	return resp, nil
}

// Get performs a GET request and decodes the response.
func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

// Post performs a POST request with a JSON body and decodes the response.
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling body: %w", err)
	}

	resp, err := c.do("POST", path, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Patch performs a PATCH request with a JSON body and decodes the response.
func (c *Client) Patch(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling body: %w", err)
	}

	resp, err := c.do("PATCH", path, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// DoRaw performs an authenticated GET request and returns the raw response.
// The href can be a full URL or a path relative to the API base.
func (c *Client) DoRaw(method, href string) (*http.Response, error) {
	url := href
	if !strings.HasPrefix(href, "http") {
		url = c.BaseURL + href
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth("apikey", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	return resp, nil
}

// RequireProject returns the project identifier or an error if not set.
func (c *Client) RequireProject() (string, error) {
	if c.Project == "" {
		return "", fmt.Errorf("no project specified: use -p flag or set OP_PROJECT")
	}
	return c.Project, nil
}

// Attachment represents an uploaded attachment.
type Attachment struct {
	ID          int    `json:"id"`
	FileName    string `json:"fileName"`
	FileSize    int    `json:"fileSize"`
	ContentType string `json:"contentType"`
	Links       struct {
		Self             Link `json:"self"`
		DownloadLocation Link `json:"downloadLocation"`
	} `json:"_links"`
}

// UploadAttachment uploads a file to a work package.
func (c *Client) UploadAttachment(wpID int, filePath string, description string) (*Attachment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	fileName := filepath.Base(filePath)

	// Build multipart body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Part 1: metadata (application/json)
	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Disposition", `form-data; name="metadata"`)
	metaHeader.Set("Content-Type", "application/json")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return nil, fmt.Errorf("creating metadata part: %w", err)
	}

	meta := map[string]string{"fileName": fileName}
	if description != "" {
		meta["description"] = description
	}
	if err := json.NewEncoder(metaPart).Encode(meta); err != nil {
		return nil, fmt.Errorf("writing metadata: %w", err)
	}

	// Part 2: file
	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	fileHeader.Set("Content-Type", detectContentType(fileName))
	filePart, err := writer.CreatePart(fileHeader)
	if err != nil {
		return nil, fmt.Errorf("creating file part: %w", err)
	}
	if _, err := io.Copy(filePart, f); err != nil {
		return nil, fmt.Errorf("writing file: %w", err)
	}

	writer.Close()

	// Send request
	url := fmt.Sprintf("%s/api/v3/work_packages/%d/attachments", c.BaseURL, wpID)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth("apikey", c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uploading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if json.Unmarshal(data, apiErr) != nil {
			apiErr.Message = string(data)
		}
		return nil, apiErr
	}

	var att Attachment
	if err := json.NewDecoder(resp.Body).Decode(&att); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &att, nil
}

// Activity represents a single activity (comment or change) on a work package.
type Activity struct {
	ID        int          `json:"id"`
	Comment   *Formattable `json:"comment"`
	CreatedAt string       `json:"createdAt"`
	UpdatedAt string       `json:"updatedAt"`
	Links     struct {
		User Link `json:"user"`
	} `json:"_links"`
}

// ActivityCollection is the response from listing activities.
type ActivityCollection struct {
	Total    int `json:"total"`
	Embedded struct {
		Elements []Activity `json:"elements"`
	} `json:"_embedded"`
}

// ListActivities lists all activities (comments and changes) for a work package.
func (c *Client) ListActivities(wpID int) (*ActivityCollection, error) {
	var result ActivityCollection
	path := fmt.Sprintf("/work_packages/%d/activities", wpID)
	if err := c.Get(path, &result); err != nil {
		return nil, fmt.Errorf("listing activities for work package %d: %w", wpID, err)
	}
	return &result, nil
}

// commentRequest is the body for posting a comment on a work package.
type commentRequest struct {
	Comment *Formattable `json:"comment"`
}

// PostComment posts a markdown comment on a work package as an activity.
func (c *Client) PostComment(wpID int, markdown string) error {
	body := commentRequest{
		Comment: &Formattable{Format: "markdown", Raw: markdown},
	}
	path := fmt.Sprintf("/work_packages/%d/activities", wpID)
	return c.Post(path, body, nil)
}

// editCommentRequest is the body for editing an existing activity. Unlike the
// create endpoint (which takes a Formattable {format, raw} object), the activity
// update endpoint PATCH /activities/{id} expects `comment` as a plain string.
type editCommentRequest struct {
	Comment string `json:"comment"`
}

// EditComment updates the text of an existing comment (activity) by its ID.
func (c *Client) EditComment(activityID int, markdown string) error {
	body := editCommentRequest{Comment: markdown}
	path := fmt.Sprintf("/activities/%d", activityID)
	return c.Patch(path, body, nil)
}

func detectContentType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}
