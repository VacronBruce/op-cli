package cmd

import (
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

func TestRootVersionFlag_PrintsSameTextAsVersionCommand(t *testing.T) {
	want := testutil.CaptureStdout(func() { versionCmd.Run(versionCmd, nil) })

	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		_ = rootCmd.Flags().Set("version", "false")
	})
	var err error
	got := testutil.CaptureStdout(func() { err = rootCmd.Execute() })
	if err != nil {
		t.Fatal(err)
	}
	if got != want || got == "" {
		t.Errorf("--version = %q, version = %q", got, want)
	}
}
