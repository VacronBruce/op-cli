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

// --- downloadViaGlab ---

// fakeGlab installs a shell script named "glab" as the only thing on PATH.
// The script body sees the real arguments, so $ASSET/$DEST (parsed from
// --asset-name= and -D) let tests simulate any glab outcome.
func fakeGlab(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
ASSET=""; DEST=""; PREV=""
for a in "$@"; do
  case "$a" in --asset-name=*) ASSET="${a#--asset-name=}" ;; esac
  if [ "$PREV" = "-D" ]; then DEST="$a"; fi
  PREV="$a"
done
` + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "glab"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestDownloadViaGlab_NotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: no glab anywhere
	_, err := downloadViaGlab("op-darwin-arm64")
	if err == nil || !strings.Contains(err.Error(), "glab not found") {
		t.Fatalf("expected glab-not-found error, got: %v", err)
	}
}

func TestDownloadViaGlab_DownloadsAsset(t *testing.T) {
	fakeGlab(t, `printf 'glab-bytes' > "$DEST/$ASSET"`)

	var path string
	var err error
	testutil.CaptureStdout(func() { path, err = downloadViaGlab("op-darwin-arm64") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("reading downloaded binary: %v", readErr)
	}
	if string(got) != "glab-bytes" {
		t.Errorf("downloaded content mismatch: %s", got)
	}
}

// glab's own stderr is the only clue to what went wrong (auth, network, repo),
// so a failed download must surface it in the error.
func TestDownloadViaGlab_FailureSurfacesGlabOutput(t *testing.T) {
	fakeGlab(t, `echo "401 unauthorized: run glab auth login" >&2; exit 1`)

	_, err := downloadViaGlab("op-darwin-arm64")
	if err == nil || !strings.Contains(err.Error(), "401 unauthorized") {
		t.Fatalf("expected glab's output in the error, got: %v", err)
	}
}

// glab exiting 0 without producing the asset (e.g. wrong asset name in a
// release) must be an error, not an empty binary installed over the real one.
func TestDownloadViaGlab_MissingAssetAfterDownload(t *testing.T) {
	fakeGlab(t, `exit 0`)

	_, err := downloadViaGlab("op-darwin-arm64")
	if err == nil || !strings.Contains(err.Error(), "asset not found") {
		t.Fatalf("expected asset-not-found error, got: %v", err)
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
