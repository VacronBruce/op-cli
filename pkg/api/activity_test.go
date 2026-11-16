package api

import (
	"encoding/json"
	"testing"
)

func TestActivity_DecodesLiveDetailsAndUserID(t *testing.T) {
	raw := `{"_type":"Activity","id":9001,"comment":{"format":"markdown","raw":"","html":""},
	"details":[{"format":"custom","raw":"Status changed from To Do to In Progress","html":"<strong>Status</strong>"}],
	"createdAt":"2026-11-06T09:30:00.000Z","_links":{"user":{"href":"/api/v3/users/36"}}}`

	var a Activity
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Details) != 1 || a.Details[0].Raw != "Status changed from To Do to In Progress" {
		t.Errorf("details must expose the readable raw line, got %+v", a.Details)
	}
	if a.UserID() != 36 {
		t.Errorf("UserID must be parsed from _links.user.href, got %d", a.UserID())
	}
}

func TestActivity_UserIDZeroWhenHrefMissing(t *testing.T) {
	var a Activity
	if a.UserID() != 0 {
		t.Errorf("expected 0, got %d", a.UserID())
	}
}
