package cmd

import (
	"encoding/json"
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
