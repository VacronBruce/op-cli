package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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
}

func (e *APIError) Error() string {
	return fmt.Sprintf("OpenProject API error (%d): %s", e.StatusCode, e.Message)
}

// Collection represents a paginated API response.
type Collection struct {
	Type    string            `json:"_type"`
	Total   int               `json:"total"`
	Count   int               `json:"count"`
	Offset  int               `json:"offset"`
	Items   []json.RawMessage `json:"_embedded"`
}

// NewClient creates a new OpenProject API client.
func NewClient(baseURL, apiKey, project string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Project: project,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
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

// RequireProject returns the project identifier or an error if not set.
func (c *Client) RequireProject() (string, error) {
	if c.Project == "" {
		return "", fmt.Errorf("no project specified: use -p flag or set OP_PROJECT")
	}
	return c.Project, nil
}
