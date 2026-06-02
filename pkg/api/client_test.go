package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	c := NewClient(ts.URL, "test-key", "test-project")
	return ts, c
}

func TestClient_Get_Success(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	})
	defer ts.Close()

	var result map[string]string
	if err := c.Get("/test", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected name=test, got %s", result["name"])
	}
}

func TestClient_Get_APIError(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{
			"_type":   "Error",
			"message": "Not found",
		})
	})
	defer ts.Close()

	var result map[string]string
	err := c.Get("/missing", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Not found" {
		t.Errorf("expected 'Not found', got %s", apiErr.Message)
	}
}

func TestClient_Post_Body(t *testing.T) {
	var receivedBody map[string]string
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"id": 1})
	})
	defer ts.Close()

	var result map[string]int
	body := map[string]string{"subject": "test item"}
	if err := c.Post("/items", body, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["subject"] != "test item" {
		t.Errorf("expected subject='test item', got %s", receivedBody["subject"])
	}
	if result["id"] != 1 {
		t.Errorf("expected id=1, got %d", result["id"])
	}
}

func TestClient_AuthHeader(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "apikey" {
			t.Errorf("expected user=apikey, got %s", user)
		}
		if pass != "test-key" {
			t.Errorf("expected pass=test-key, got %s", pass)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})
	defer ts.Close()

	var result map[string]interface{}
	c.Get("/auth-test", &result)
}

func TestClient_Patch(t *testing.T) {
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	})
	defer ts.Close()

	var result map[string]string
	if err := c.Patch("/items/1", map[string]string{"subject": "new"}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "updated" {
		t.Errorf("expected status=updated, got %s", result["status"])
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"photo.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"animation.gif", "image/gif"},
		{"image.webp", "image/webp"},
		{"icon.svg", "image/svg+xml"},
		{"document.pdf", "application/pdf"},
		{"video.mp4", "video/mp4"},
		{"clip.mov", "video/quicktime"},
		{"data.bin", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := detectContentType(tt.name)
		if got != tt.expected {
			t.Errorf("detectContentType(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestRequireProject(t *testing.T) {
	c := NewClient("http://localhost", "key", "myproject")
	p, err := c.RequireProject()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "myproject" {
		t.Errorf("expected myproject, got %s", p)
	}

	c2 := NewClient("http://localhost", "key", "")
	_, err = c2.RequireProject()
	if err == nil {
		t.Error("expected error for empty project")
	}
}

func TestEditComment_SendsPlainStringBody(t *testing.T) {
	// Regression: the activity-update endpoint expects `comment` as a plain
	// string, not the {format, raw} Formattable object the create endpoint uses.
	// Sending the object form returns 400 "comment is invalid".
	var gotMethod, gotPath string
	var gotBody map[string]json.RawMessage
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"_type": "Activity", "id": 42})
	})
	defer ts.Close()

	if err := c.EditComment(42, "updated text"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != "PATCH" {
		t.Errorf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v3/activities/42" {
		t.Errorf("expected /api/v3/activities/42, got %s", gotPath)
	}
	// comment must be a JSON string, not an object.
	raw, ok := gotBody["comment"]
	if !ok {
		t.Fatal("body missing `comment` field")
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err != nil {
		t.Fatalf("expected `comment` to be a JSON string, got %s", raw)
	}
	if asString != "updated text" {
		t.Errorf("expected 'updated text', got %q", asString)
	}
}
