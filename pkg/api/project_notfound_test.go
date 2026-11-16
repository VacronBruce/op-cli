package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func projectsServer(t *testing.T) *Client {
	return projectsServerOfTotal(t, 3)
}

// projectsServerOfTotal lists three projects but reports total of them, as a
// server with more projects than one page holds does.
func projectsServerOfTotal(t *testing.T, total int) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"total": %d, "_embedded": {"elements": [
			{"id": 1, "identifier": "app", "name": "App"},
			{"id": 2, "identifier": "tdjy", "name": "Taiwan DJY"},
			{"id": 3, "identifier": "wp", "name": "WordPress"}]}}`, total)
	})
	mux.HandleFunc("/api/v3/projects/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"_type": "Error", "message": "The requested resource could not be found."}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "secret", "")
}

// Agents guessed project identifiers like ntd-app and got a bare 404 they
// could not act on.
func TestAWrongProjectIdentifier_IsNamedWithTheClosestOnes(t *testing.T) {
	c := projectsServer(t)
	for bad, suggest := range map[string]string{"ntd-app": "did you mean app?", "djy-app": "did you mean app, tdjy?", "apps": "did you mean app?"} {
		_, err := c.ListVersions(bad)
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			t.Fatalf("%s: err = %v, want a 404 APIError", bad, err)
		}
		msg := err.Error()
		if !strings.Contains(msg, `project "`+bad+`" not found`) || !strings.Contains(msg, suggest) || !strings.Contains(msg, "op projects") {
			t.Errorf("%s: %s", bad, msg)
		}
	}
}

func TestA404InsideAnExistingProject_KeepsTheServerMessage(t *testing.T) {
	_, err := projectsServer(t).ListVersions("app")
	if err == nil || !strings.Contains(err.Error(), "The requested resource could not be found.") {
		t.Errorf("err = %v", err)
	}
}

// The list holds one page. A project missing from it may still exist, so the
// 404 isn't blamed on the project.
func TestAProjectBeyondTheFirstPage_KeepsTheServerMessage(t *testing.T) {
	_, err := projectsServerOfTotal(t, 150).ListVersions("zz-later")
	if err == nil || !strings.Contains(err.Error(), "The requested resource could not be found.") {
		t.Errorf("err = %v", err)
	}
}
