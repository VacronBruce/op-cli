package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// stubSetupEnv points the oprc/shell-rc seams at a temp dir, seeds viper, and
// restores everything afterwards.
func stubSetupEnv(t *testing.T, oprcContent string, vals map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	oprc := filepath.Join(dir, ".oprc")
	if oprcContent != "" {
		if err := os.WriteFile(oprc, []byte(oprcContent), 0600); err != nil {
			t.Fatal(err)
		}
	}

	origOprc, origRC := oprcPath, shellRCPath
	t.Cleanup(func() { oprcPath, shellRCPath = origOprc, origRC; viper.Reset() })
	oprcPath = func() string { return oprc }
	shellRCPath = func() (string, string) { return "zsh", filepath.Join(dir, ".zshrc") }

	viper.Reset()
	for k, v := range vals {
		viper.Set(k, v)
	}
	return dir
}

func newSetupCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().String("url", "", "")
	c.Flags().String("api-key", "", "")
	c.Flags().String("project", "", "")
	c.Flags().String("sprint", "", "")
	return c
}

func setupMock() *testutil.MockClient {
	return &testutil.MockClient{
		GetMeFn: func() (*api.User, error) { return &api.User{Name: "Bruce Chen"}, nil },
		GetProjectFn: func(identifier string) (*api.Project, error) {
			if identifier != "app" {
				return nil, os.ErrNotExist
			}
			return &api.Project{Identifier: "app"}, nil
		},
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			if name != "Sprint 1" {
				return nil, os.ErrNotExist
			}
			return &api.Version{ID: 1, Name: name}, nil
		},
	}
}

// A fully configured environment must read all-[ok] and exit clean — this is
// the post-install "everything works" signal install.sh points users at.
func TestSetup_AllGood(t *testing.T) {
	dir := stubSetupEnv(t, "url: https://op.example.com\napi_key: k\n", map[string]string{
		"url": "https://op.example.com", "api_key": "secret-key-12345",
		"project": "app", "sprint": "Sprint 1",
	})
	rc := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rc, []byte("source <(op completion zsh)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	SetClient(setupMock())

	var err error
	out := testutil.CaptureStdout(func() { err = runSetup(newSetupCmd(), nil) })
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "[--]") {
		t.Errorf("expected no failures, got: %s", out)
	}
	for _, want := range []string{"logged in as Bruce Chen", "project: app", `sprint: "Sprint 1"`, "All good."} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

// Each failing check must carry its fix command — the checklist exists so
// users never have to consult the README to repair their setup.
func TestSetup_MissingPiecesShowFixes(t *testing.T) {
	stubSetupEnv(t, "", map[string]string{}) // no .oprc, nothing set
	SetClient(&testutil.MockClient{})

	var err error
	out := testutil.CaptureStdout(func() { err = runSetup(newSetupCmd(), nil) })
	if err == nil || !strings.Contains(err.Error(), "setup issue") {
		t.Fatalf("expected setup issues error, got: %v", err)
	}
	for _, want := range []string{"[--] config:", "api_key: (not set)", "op setup --api-key", "fix:"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

// The placeholder key install.sh writes must NOT count as configured.
func TestSetup_PlaceholderKeyIsNotConfigured(t *testing.T) {
	stubSetupEnv(t, "api_key: YOUR_API_KEY_HERE\n", map[string]string{
		"url": "https://op.example.com", "api_key": "YOUR_API_KEY_HERE",
	})
	SetClient(&testutil.MockClient{})

	var err error
	out := testutil.CaptureStdout(func() { err = runSetup(newSetupCmd(), nil) })
	if err == nil {
		t.Fatal("expected setup issues, got nil")
	}
	if !strings.Contains(out, "placeholder") {
		t.Errorf("placeholder key must be called out, got: %s", out)
	}
}

// A stale sprint (the value rotates every two weeks) must be flagged with the
// command that lists current ones.
func TestSetup_StaleSprintFlagged(t *testing.T) {
	stubSetupEnv(t, "x: y\n", map[string]string{
		"url": "https://op.example.com", "api_key": "secret-key-12345",
		"project": "app", "sprint": "Old Sprint",
	})
	SetClient(setupMock())

	var err error
	out := testutil.CaptureStdout(func() { err = runSetup(newSetupCmd(), nil) })
	if err == nil {
		t.Fatal("expected setup issues, got nil")
	}
	if !strings.Contains(out, `sprint: "Old Sprint" does not resolve`) || !strings.Contains(out, "op sprint list") {
		t.Errorf("stale sprint must be flagged with fix, got: %s", out)
	}
}

// --- updateOprc ---

// Setting a key must replace ONLY that line: comments, nested custom_fields
// blocks, and other keys stay byte-identical. This is what makes `op setup
// --sprint=...` safe to run on a hand-edited config.
func TestUpdateOprc_PreservesEverythingElse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".oprc")
	orig := `url: https://op.example.com
api_key: secret
# project: app
sprint: "Old Sprint"

custom_fields:
  component:
    field: customField12
templates:
  bug: |
    ## Steps
`
	if err := os.WriteFile(path, []byte(orig), 0600); err != nil {
		t.Fatal(err)
	}

	if err := updateOprc(path, map[string]string{"sprint": "App_06/15/2026", "project": "app"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	s := string(got)
	if !strings.Contains(s, `sprint: "App_06/15/2026"`) {
		t.Errorf("sprint not updated: %s", s)
	}
	// The commented-out project line becomes the live setting.
	if !strings.Contains(s, `project: "app"`) || strings.Contains(s, "# project") {
		t.Errorf("commented project line should be activated: %s", s)
	}
	for _, keep := range []string{"api_key: secret", "field: customField12", "## Steps"} {
		if !strings.Contains(s, keep) {
			t.Errorf("must preserve %q, got: %s", keep, s)
		}
	}
	// Nested keys must never be touched even when they share a name prefix.
	if !strings.Contains(s, "  component:") {
		t.Errorf("nested block damaged: %s", s)
	}
}

func TestUpdateOprc_CreatesFileWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".oprc")
	if err := updateOprc(path, map[string]string{"api_key": "k123"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `api_key: "k123"`) {
		t.Errorf("key not written: %s", got)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("config with credentials must be 0600, got %v", info.Mode().Perm())
	}
}
