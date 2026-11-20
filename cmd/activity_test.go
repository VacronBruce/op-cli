package cmd

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func TestActivity_PrintsHistoryWithResolvedNames(t *testing.T) {
	a := activityByUser(11, "/api/v3/users/36")
	a.Comment = nil
	a.Details = []api.Formattable{{Raw: "Status changed from To Do to In Progress"}}
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{a}
	mock := &testutil.MockClient{
		ListActivitiesFn: func(id int) (*api.ActivityCollection, error) {
			if id != 86156 {
				t.Errorf("id = %d", id)
			}
			return ac, nil
		},
		GetFn: func(path string, result interface{}) error {
			return json.Unmarshal([]byte(`{"name":"Ada"}`), result)
		},
	}
	SetClient(mock)
	var err error
	out := testutil.CaptureStdout(func() { err = runActivity(&cobra.Command{}, []string{"86156"}) })
	if err != nil {
		t.Fatal(err)
	}
	want := "[2026-11-06 09:30 UTC] Ada (#11)\n  - Status changed from To Do to In Progress\n"
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func TestActivity_AgentGuessesAreAliases(t *testing.T) {
	for _, name := range []string{"activity", "activities", "history"} {
		if c, _, err := rootCmd.Find([]string{name}); err != nil || c.Name() != "activity" {
			t.Errorf("%q must resolve to the activity command", name)
		}
	}
}

func TestActivity_WithoutDescriptionFlagNeverTouchesTheWeb(t *testing.T) {
	prev := newDiffSource
	t.Cleanup(func() { newDiffSource = prev })
	newDiffSource = func() (diffSource, error) {
		t.Error("web session opened for a plain activity listing")
		return nil, errors.New("unexpected")
	}
	a := activityByUser(11, "/api/v3/users/36")
	a.Comment = nil
	a.Details = []api.Formattable{{Raw: "Description changed (/journals/9/diff/description)"}}
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{a}
	SetClient(&testutil.MockClient{
		ListActivitiesFn: func(int) (*api.ActivityCollection, error) { return ac, nil },
		GetFn: func(path string, result interface{}) error {
			return json.Unmarshal([]byte(`{"name":"Ada"}`), result)
		},
	})
	cmd := &cobra.Command{}
	cmd.Flags().Bool("description", false, "")
	cmd.Flags().Bool("json", false, "")
	var err error
	out := testutil.CaptureStdout(func() { err = runActivity(cmd, []string{"1"}) })
	want := "[2026-11-06 09:30 UTC] Ada (#11)\n  - Description changed (/journals/9/diff/description)\n"
	if err != nil || out != want {
		t.Errorf("out=%q err=%v", out, err)
	}
}
