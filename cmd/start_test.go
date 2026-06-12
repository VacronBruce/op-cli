package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func TestBranchName(t *testing.T) {
	tests := []struct {
		name    string
		project string
		id      int
		subject string
		want    string
	}{
		{"simple", "app", 12345, "Crash on save", "app-12345-crash-on-save"},
		{"punctuation collapses", "web", 81477, "Document new CLI tool!", "web-81477-document-new-cli-tool"},
		{"mixed case and symbols", "app", 42, "Fix: API 500 @ /login", "app-42-fix-api-500-login"},
		{"leading/trailing junk trimmed", "app", 7, "  ...Hello...  ", "app-7-hello"},
		// A project name (vs identifier) is slugified the same way, so spaces and
		// case in the prefix are normalized too.
		{"project name slugified", "NTD App", 5, "Hi", "ntd-app-5-hi"},
		// Empty project identifier falls back to "wp" so the branch stays valid.
		{"empty project falls back to wp", "", 5, "Hello world", "wp-5-hello-world"},
		// Subject with no usable characters falls back to <project>-<id>.
		{"empty slug falls back to project+id", "app", 99, "！！！", "app-99"},
		// Long subjects are capped to 50 slug chars; this subject is built so the
		// cut lands exactly on a dash, proving the trailing dash gets trimmed.
		{"long subject capped, trailing dash trimmed", "app", 1,
			"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk",
			"app-1-aaaa-bbbb-cccc-dddd-eeee-ffff-gggg-hhhh-iiii-jjjj"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchName(tt.project, tt.id, tt.subject)
			if got != tt.want {
				t.Errorf("branchName(%q, %d, %q) = %q, want %q", tt.project, tt.id, tt.subject, got, tt.want)
			}
		})
	}
}

// initTestRepo creates a temp git repository with one commit (an unborn branch
// has no refs, so branchExists could never report true without it) and makes it
// the working directory for the rest of the test.
func initTestRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=test@test", "-c", "user.name=test", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// startMock serves everything runStart asks for: the ticket (in project 14),
// the project identifier for the branch prefix, status resolution, and the
// current user. The captured update request lets tests assert the write.
func startMock(captured **api.UpdateWPRequest) *testutil.MockClient {
	return &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			wp := &api.WorkPackage{ID: id, Subject: "Crash on save"}
			wp.Links.Project = api.Link{Href: "/api/v3/projects/14", Title: "NTD App"}
			return wp, nil
		},
		GetProjectFn: func(identifier string) (*api.Project, error) {
			return &api.Project{ID: 14, Identifier: "app"}, nil
		},
		GetFn: resolverCollections, // serves /statuses for ResolveStatus
		GetMeFn: func() (*api.User, error) {
			me := &api.User{ID: 5, Name: "Ken Peng"}
			me.Links.Self = api.Link{Href: "/api/v3/users/5"}
			return me, nil
		},
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			if captured != nil {
				*captured = req
			}
			return &api.WorkPackage{ID: id}, nil
		},
	}
}

func currentBranch(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// The whole point of op start: one command leaves you on a ticket-named branch
// with the ticket In Progress and assigned to you. Branch prefix must come from
// the ticket's own project identifier ("app"), not its display name ("NTD App").
func TestStart_CreatesBranchAndUpdatesTicket(t *testing.T) {
	initTestRepo(t)
	var got *api.UpdateWPRequest
	SetClient(startMock(&got))

	var err error
	out := testutil.CaptureStdout(func() { err = runStart(&cobra.Command{}, []string{"42"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b := currentBranch(t); b != "app-42-crash-on-save" {
		t.Errorf("expected branch app-42-crash-on-save, got %q", b)
	}
	if !strings.Contains(out, "Created and switched to branch app-42-crash-on-save") {
		t.Errorf("expected branch-created message, got: %q", out)
	}
	if got == nil {
		t.Fatal("ticket was never updated")
	}
	if got.Links["status"].(api.Link).Href != "/api/v3/statuses/7" {
		t.Errorf("expected In progress status link, got %+v", got.Links)
	}
	if got.Links["assignee"].(api.Link).Href != "/api/v3/users/5" {
		t.Errorf("expected self-assignment, got %+v", got.Links)
	}
}

// Re-running op start on the same ticket must switch to the existing branch,
// not fail with "branch already exists" — resuming work is the common case.
func TestStart_SwitchesToExistingBranch(t *testing.T) {
	initTestRepo(t)
	SetClient(startMock(nil))

	testutil.CaptureStdout(func() {
		if err := runStart(&cobra.Command{}, []string{"42"}); err != nil {
			t.Fatalf("first run: %v", err)
		}
	})
	// Commit so the branch has a ref, then leave it — the second run must come back.
	for _, args := range [][]string{
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "wip"},
		{"checkout", "-q", "-"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	var err error
	out := testutil.CaptureStdout(func() { err = runStart(&cobra.Command{}, []string{"42"}) })
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if !strings.Contains(out, "Switched to existing branch app-42-crash-on-save") {
		t.Errorf("expected switch-to-existing message, got: %q", out)
	}
	if b := currentBranch(t); b != "app-42-crash-on-save" {
		t.Errorf("expected to be back on the ticket branch, got %q", b)
	}
}

// Outside a git repo the command must refuse before touching the ticket —
// otherwise it would move it In Progress with no branch to show for it.
func TestStart_RequiresGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	var got *api.UpdateWPRequest
	SetClient(startMock(&got))

	err := runStart(&cobra.Command{}, []string{"42"})
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("expected not-a-git-repository error, got: %v", err)
	}
	if got != nil {
		t.Error("ticket must not be updated outside a git repo")
	}
}
