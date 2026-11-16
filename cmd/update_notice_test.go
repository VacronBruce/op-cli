package cmd

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

// The op plugin's skills update by themselves and the binary doesn't, so a
// teammate's agent can read about a command its op lacks. The notice tells it
// why the command failed and how to fix it.

func cacheWith(t *testing.T, tag string, age time.Duration) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "op-cli", "latest-release")
	if tag == "" {
		return file
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(tag+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(file, when, when); err != nil {
		t.Fatal(err)
	}
	return file
}

func fetcher(tag string, err error, calls *int) func() (string, error) {
	return func() (string, error) {
		*calls++
		return tag, err
	}
}

func TestNewerRelease_UsesTodaysAnswerWithoutAskingAgain(t *testing.T) {
	calls := 0
	got := newerRelease("0.25.0", cacheWith(t, "v0.27.0", time.Hour), fetcher("v0.28.0", nil, &calls))
	if got != "v0.27.0" || calls != 0 {
		t.Errorf("got %q after %d fetches, want v0.27.0 from the cache", got, calls)
	}
}

func TestNewerRelease_AsksGitHubOnceTheAnswerIsADayOld(t *testing.T) {
	calls := 0
	file := cacheWith(t, "v0.27.0", 25*time.Hour)
	if got := newerRelease("0.27.0", file, fetcher("v0.28.0", nil, &calls)); got != "v0.28.0" || calls != 1 {
		t.Errorf("got %q after %d fetches, want v0.28.0 after one", got, calls)
	}
	if got := newerRelease("0.27.0", file, fetcher("v0.29.0", nil, &calls)); got != "v0.28.0" || calls != 1 {
		t.Errorf("second run got %q after %d fetches, want the remembered v0.28.0", got, calls)
	}
}

// Offline, every failing command would otherwise wait on GitHub again.
func TestNewerRelease_WaitsADayAfterGitHubFails(t *testing.T) {
	calls := 0
	file := cacheWith(t, "", 0)
	newerRelease("0.27.0", file, fetcher("", errors.New("offline"), &calls))
	newerRelease("0.27.0", file, fetcher("", errors.New("offline"), &calls))
	if calls != 1 {
		t.Errorf("asked GitHub %d times, want once a day", calls)
	}
}

func TestNewerRelease_StaysQuiet(t *testing.T) {
	calls := 0
	for name, version := range map[string]string{"up to date": "0.27.0", "ahead": "0.28.0-rc", "dev build": "dev"} {
		if got := newerRelease(version, cacheWith(t, "v0.27.0", time.Hour), fetcher("", nil, &calls)); got != "" {
			t.Errorf("%s (%s): got %q, want no notice", name, version, got)
		}
	}
	t.Setenv("OP_NO_UPDATE_CHECK", "1")
	if got := newerRelease("0.25.0", cacheWith(t, "v0.27.0", time.Hour), fetcher("", nil, &calls)); got != "" {
		t.Errorf("OP_NO_UPDATE_CHECK=1: got %q", got)
	}
}

func TestFetchLatestTag_ReadsTheTagFromGitHubsRedirect(t *testing.T) {
	srv := httptest.NewServer(http.RedirectHandler("/VacronBruce/op-cli/releases/tag/v0.28.0", http.StatusFound))
	t.Cleanup(srv.Close)
	saved := latestReleaseURL
	latestReleaseURL = srv.URL + "/VacronBruce/op-cli/releases/latest"
	t.Cleanup(func() { latestReleaseURL = saved })

	if tag, err := fetchLatestTag(); err != nil || tag != "v0.28.0" {
		t.Errorf("tag = %q, err = %v", tag, err)
	}
}

// The notice goes out only when a command failed or the version was asked
// for, so it never lands in the output of a command that worked.
func TestExecute_PrintsTheNoticeOnlyAfterAFailure(t *testing.T) {
	savedVersion, savedFile := Version, updateCacheFile
	Version = "0.25.0"
	file := cacheWith(t, "v0.27.0", time.Hour)
	updateCacheFile = func() string { return file }
	t.Cleanup(func() { Version, updateCacheFile = savedVersion, savedFile; rootCmd.SetArgs(nil) })

	rootCmd.SetArgs([]string{"activty", "1"})
	failed := testutil.CaptureStderr(func() { _ = Execute() })
	if !strings.Contains(failed, "op-cli v0.27.0 is out (this is 0.25.0); update with: op upgrade") {
		t.Errorf("after a failure stderr was %q", failed)
	}

	rootCmd.SetArgs([]string{"completion", "bash"})
	worked := testutil.CaptureStderr(func() {
		testutil.CaptureStdout(func() { _ = Execute() })
	})
	if strings.Contains(worked, "is out") {
		t.Errorf("a command that worked printed %q", worked)
	}
}
