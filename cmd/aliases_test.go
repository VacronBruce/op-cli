package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

// Names agents typed in the last 21 days that op didn't know: view (3 times),
// project (3) and statuses (2).
func TestTheNamesAgentsGuess_ReachTheRightCommand(t *testing.T) {
	for name, want := range map[string]string{"view": "show", "project": "projects"} {
		got, _, err := rootCmd.Find([]string{name})
		if err != nil || got.Name() != want {
			t.Errorf("op %s ran %v (err %v), want op %s", name, got.Name(), err, want)
		}
	}
}

func TestStatuses_PointsAtFields(t *testing.T) {
	rootCmd.SetArgs([]string{"statuses"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Did you mean this?\n\tfields") {
		t.Errorf("err = %v, want a suggestion of op fields", err)
	}
	overview := testutil.CaptureStdout(func() { _ = runFields(fieldsCmd, nil) })
	if !strings.Contains(overview, "op fields status") {
		t.Errorf("op fields, where op statuses points, never mentions statuses:\n%s", overview)
	}
}
