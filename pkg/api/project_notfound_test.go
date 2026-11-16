package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func projectsServer(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total": 3, "_embedded": {"elements": [
			{"id": 1, "identifier": "app", "name": "App"},
			{"id": 2, "identifier": "tdjy", "name": "Taiwan DJY"},
			{"id": 3, "identifier": "wp", "name": "WordPress"}]}}`))
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
	for bad, suggest := range map[string]string{"ntd-app": "did you mean app?", "djy-app": "did you mean app, tdjy?"} {
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
