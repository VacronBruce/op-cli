package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

// stubUpgradeSeams swaps the upgrade seams for the test and restores them.
func stubUpgradeSeams(t *testing.T, execPath func() (string, error), download func(string) (string, error)) {
	t.Helper()
	origExec, origDownload := upgradeExecPath, upgradeDownload
	t.Cleanup(func() { upgradeExecPath, upgradeDownload = origExec, origDownload })
	if execPath != nil {
		upgradeExecPath = execPath
	}
	if download != nil {
		upgradeDownload = download
	}
}

// The whole point of `op upgrade` is replacing the running binary in place:
// the downloaded file must land at the exec path, executable, and the old
// content must be gone.
func TestUpgrade_ReplacesBinaryAtExecPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "op")
	if err := os.WriteFile(target, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	stubUpgradeSeams(t,
		func() (string, error) { return target, nil },
		func(assetName string) (string, error) {
			if !strings.HasPrefix(assetName, "op-") {
				t.Errorf("asset name must be os/arch qualified, got %q", assetName)
			}
			tmp := filepath.Join(dir, "downloaded")
			if err := os.WriteFile(tmp, []byte("new-binary"), 0600); err != nil {
				return "", err
			}
			return tmp, nil
		})

	var err error
	out := testutil.CaptureStdout(func() { err = runUpgrade(upgradeCmd, nil) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Errorf("binary not replaced, content: %s", got)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("replaced binary must be executable, mode: %v", info.Mode())
	}
	if !strings.Contains(out, "Upgraded: "+target) {
		t.Errorf("expected upgrade confirmation with path, got: %s", out)
	}
}

// A failed download must abort BEFORE touching the installed binary.
func TestUpgrade_DownloadFailureLeavesBinaryUntouched(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "op")
	if err := os.WriteFile(target, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	stubUpgradeSeams(t,
		func() (string, error) { return target, nil },
		func(string) (string, error) { return "", fmt.Errorf("network down") })

	var err error
	testutil.CaptureStdout(func() { err = runUpgrade(upgradeCmd, nil) })
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected download error, got: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old-binary" {
		t.Errorf("installed binary must be untouched on failure, content: %s", got)
	}
}

// --- downloadViaCurl ---

// Without a token the error must TEACH the fix (both auth options), because
// this is the path users hit on a fresh machine.
func TestDownloadViaCurl_NoTokenExplainsBothAuthOptions(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	_, err := downloadViaCurl("op-darwin-arm64")
	if err == nil {
		t.Fatal("expected error without token")
	}
	for _, hint := range []string{"glab auth login", "GITLAB_TOKEN"} {
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("error must mention %q, got: %v", hint, err)
		}
	}
}

func TestDownloadViaCurl_DownloadsAssetWithToken(t *testing.T) {
	var gotToken, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		gotPath = r.URL.Path
		w.Write([]byte("binary-bytes"))
	}))
	defer ts.Close()

	orig := pkgBaseURL
	pkgBaseURL = ts.URL
	t.Cleanup(func() { pkgBaseURL = orig })
	t.Setenv("GITLAB_TOKEN", "tok-123")

	var path string
	var err error
	testutil.CaptureStdout(func() { path, err = downloadViaCurl("op-darwin-arm64") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	if gotToken != "tok-123" {
		t.Errorf("token header not sent, got %q", gotToken)
	}
	if gotPath != "/op-darwin-arm64" {
		t.Errorf("expected asset path, got %q", gotPath)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "binary-bytes" {
		t.Errorf("downloaded content mismatch: %s", got)
	}
}

func TestDownloadViaCurl_HTTPErrorMentionsStatusAndToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	orig := pkgBaseURL
	pkgBaseURL = ts.URL
	t.Cleanup(func() { pkgBaseURL = orig })
	t.Setenv("GITLAB_TOKEN", "bad-token")

	_, err := downloadViaCurl("op-darwin-arm64")
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") || !strings.Contains(err.Error(), "GITLAB_TOKEN") {
		t.Fatalf("expected HTTP-status error pointing at the token, got: %v", err)
	}
}
