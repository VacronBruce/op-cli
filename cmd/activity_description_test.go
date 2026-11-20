package cmd

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type fakeDiffs map[int][]api.DiffSegment

func (f fakeDiffs) DescriptionDiff(id int) ([]api.DiffSegment, error) {
	s, ok := f[id]
	if !ok {
		return nil, errors.New("no such journal")
	}
	return s, nil
}

func descActivity(id int, created, raw string) api.Activity {
	a := activityByUser(id, "/api/v3/users/36")
	a.Comment = nil
	a.CreatedAt = created
	a.Details = []api.Formattable{{Raw: raw}}
	return a
}

func setupDescriptionHistory(t *testing.T) {
	t.Helper()
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		descActivity(20, "2026-11-07T10:00:00Z", "Description changed (/journals/702/diff/description)"),
		descActivity(10, "2026-11-06T09:30:00Z", "Description set (/journals/701/diff/description)"),
		descActivity(15, "2026-11-06T09:45:00Z", "Status changed from To Do to In Progress"),
	}
	SetClient(&testutil.MockClient{
		ListActivitiesFn: func(int) (*api.ActivityCollection, error) { return ac, nil },
		GetFn: func(path string, result interface{}) error {
			return json.Unmarshal([]byte(`{"name":"Ada"}`), result)
		},
	})
	prev := newDiffSource
	t.Cleanup(func() { newDiffSource = prev })
	newDiffSource = func() (diffSource, error) {
		return fakeDiffs{
			701: {{Kind: api.Inserted, Text: "Fix the login bug.\n\nSee notes.\n\nUnrelated line."}},
			702: {
				{Kind: api.Same, Text: "Fix the login bug.\n\nSee "},
				{Kind: api.Deleted, Text: "notes"},
				{Kind: api.Inserted, Text: "the spec"},
				{Kind: api.Same, Text: ".\n\nUnrelated line."},
			},
		}, nil
	}
}

func runDescription(t *testing.T, asJSON bool) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().Bool("description", true, "")
	cmd.Flags().Bool("json", asJSON, "")
	var err error
	out := testutil.CaptureStdout(func() { err = runActivity(cmd, []string{"83049"}) })
	return out, err
}

func TestActivityDescription_PrintsWordDiffPerChange(t *testing.T) {
	setupDescriptionHistory(t)
	out, err := runDescription(t, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "[2026-11-06 09:30 UTC] Ada (journal #701)\n" +
		"  {+Fix the login bug.+}\n" +
		"  {+See notes.+}\n" +
		"  {+Unrelated line.+}\n" +
		"\n" +
		"[2026-11-07 10:00 UTC] Ada (journal #702)\n" +
		"  See [-notes-]{+the spec+}.\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestActivityDescription_JSONChainsBeforeToPreviousAfter(t *testing.T) {
	setupDescriptionHistory(t)
	out, err := runDescription(t, true)
	if err != nil {
		t.Fatal(err)
	}
	var got []struct {
		JournalID int    `json:"journal_id"`
		CreatedAt string `json:"created_at"`
		Author    string `json:"author"`
		Before    string `json:"before"`
		After     string `json:"after"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(got) != 2 || got[0].JournalID != 701 || got[1].Author != "Ada" || got[0].CreatedAt != "2026-11-06T09:30:00Z" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Before != "" || got[1].Before != got[0].After {
		t.Errorf("chain broken: %+v", got)
	}
	if got[1].After != "Fix the login bug.\n\nSee the spec.\n\nUnrelated line." {
		t.Errorf("after = %q", got[1].After)
	}
}

func TestActivityDescription_NoChanges(t *testing.T) {
	setupDescriptionHistory(t)
	SetClient(&testutil.MockClient{ListActivitiesFn: func(int) (*api.ActivityCollection, error) {
		return &api.ActivityCollection{}, nil
	}})
	out, err := runDescription(t, false)
	if err != nil || out != "No description changes.\n" {
		t.Errorf("out=%q err=%v", out, err)
	}
}

func TestWebCredentials(t *testing.T) {
	prevKC := keychainPassword
	t.Cleanup(func() { keychainPassword = prevKC })
	keychainPassword = func(user string) (string, error) {
		if user != "ada@example.com" {
			t.Errorf("keychain user = %q", user)
		}
		return "kc-pass", nil
	}
	set := func(k, v string) {
		prev := viper.Get(k)
		viper.Set(k, v)
		t.Cleanup(func() { viper.Set(k, prev) })
	}

	set("web_user", "")
	if _, _, err := webCredentials(); err == nil || !strings.Contains(err.Error(), "web_user") {
		t.Errorf("missing user error = %v", err)
	}

	set("web_user", "ada@example.com")
	set("web_password", "")
	u, p, err := webCredentials()
	if err != nil || u != "ada@example.com" || p != "kc-pass" {
		t.Errorf("keychain fallback: %q %q %v", u, p, err)
	}

	set("web_password", "env-pass")
	if _, p, _ := webCredentials(); p != "env-pass" {
		t.Errorf("config password should win, got %q", p)
	}

	set("web_password", "")
	keychainPassword = func(string) (string, error) { return "", errors.New("not found") }
	_, _, err = webCredentials()
	if err == nil || !strings.Contains(err.Error(), "security add-generic-password") || !strings.Contains(err.Error(), "OP_WEB_PASSWORD") {
		t.Errorf("missing password error = %v", err)
	}
}
