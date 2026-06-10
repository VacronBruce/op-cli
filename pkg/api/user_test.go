package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetMe(t *testing.T) {
	// GetMe backs `op my` and `op start` (assign-to-me): the returned ID is fed
	// straight into assignee filters, so the decode must yield the numeric ID.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/users/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(User{ID: 7, Name: "Bruce Chen", Login: "bruce"})
	})
	defer ts.Close()

	u, err := c.GetMe()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 7 || u.Login != "bruce" {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	// An invalid API key must surface as an error, not a zero-value user that
	// would silently produce assignee=0 filters downstream.
	ts, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"_type": "Error", "message": "invalid key"})
	})
	defer ts.Close()

	if _, err := c.GetMe(); err == nil {
		t.Fatal("expected error on 401, got nil")
	}
}
