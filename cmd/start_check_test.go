package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	dorFailsWithoutDescription = `{"types":{"":["description"]}}`
	dorWarnsWithoutPoints      = `{"types":{"":["story_points"]}}`
)

func useDoR(t *testing.T, config string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dor.json")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OP_DOR_CONFIG", path)
}

func useStartHook(t *testing.T, hook string) {
	t.Helper()
	viper.Set("start_hook", hook)
	t.Cleanup(func() { viper.Set("start_hook", "") })
}

func startCmdWithFlags(requireReady bool) *cobra.Command {
	c := &cobra.Command{}
	c.Flags().Bool("require-ready", requireReady, "")
	return c
}

type startRun struct {
	stdout, stderr string
	err            error
	updates        int
}

func runStartWith(t *testing.T, requireReady bool) startRun {
	t.Helper()
	var r startRun
	mock := startMock(nil)
	inner := mock.UpdateWorkPackageFn
	mock.UpdateWorkPackageFn = func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
		r.updates++
		return inner(id, req)
	}
	SetClient(mock)
	r.stderr = testutil.CaptureStderr(func() {
		r.stdout = testutil.CaptureStdout(func() {
			r.err = runStart(startCmdWithFlags(requireReady), []string{"42"})
		})
	})
	return r
}

func TestStart_PrintsReadinessReport(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorFailsWithoutDescription)

	r := runStartWith(t, false)

	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	for _, want := range []string{"#42 Crash on save", "Score:", "NEEDS WORK", "FAIL"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("report missing %q in: %s", want, r.stdout)
		}
	}
	if r.updates != 1 || currentBranch(t) != "app-42-crash-on-save" {
		t.Errorf("without --require-ready a failing ticket must still start")
	}
}

func TestStart_RequireReadyRefusesFailingTicketWithoutWrites(t *testing.T) {
	initTestRepo(t)
	before := currentBranch(t)
	useDoR(t, dorFailsWithoutDescription)
	useStartHook(t, "touch "+filepath.Join(t.TempDir(), "hook-ran"))

	r := runStartWith(t, true)

	if r.err == nil || !strings.Contains(r.err.Error(), "failing readiness checks") {
		t.Fatalf("expected refusal, got: %v", r.err)
	}
	t.Logf("refusal: %v", r.err)
	if r.updates != 0 {
		t.Errorf("ticket was written %d times", r.updates)
	}
	if b := currentBranch(t); b != before {
		t.Errorf("branch changed from %q to %q", before, b)
	}
	if branchExists("app-42-crash-on-save") {
		t.Error("ticket branch was created")
	}
	if !strings.Contains(r.stdout, "FAIL") {
		t.Errorf("report should explain the refusal, got: %s", r.stdout)
	}
}

func TestStart_RequireReadyAllowsWarningsOnly(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorWarnsWithoutPoints)

	r := runStartWith(t, true)

	if r.err != nil {
		t.Fatalf("WARN must not block: %v", r.err)
	}
	if !strings.Contains(r.stdout, "WARN") || r.updates != 1 {
		t.Errorf("expected WARN in report and a started ticket; updates=%d out=%s", r.updates, r.stdout)
	}
}

func TestStart_CheckErrorWarnsAndContinues(t *testing.T) {
	initTestRepo(t)
	t.Setenv("OP_DOR_CONFIG", filepath.Join(t.TempDir(), "missing.json"))

	r := runStartWith(t, false)

	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if !strings.Contains(r.stderr, "readiness check could not run") {
		t.Errorf("expected stderr warning, got: %q", r.stderr)
	}
	if r.updates != 1 {
		t.Error("start should continue when the check cannot run")
	}
}

func TestStart_RequireReadyRefusesWhenCheckCannotRun(t *testing.T) {
	initTestRepo(t)
	t.Setenv("OP_DOR_CONFIG", filepath.Join(t.TempDir(), "missing.json"))

	r := runStartWith(t, true)

	if r.err == nil || !strings.Contains(r.err.Error(), "could not run") {
		t.Fatalf("expected refusal, got: %v", r.err)
	}
	if r.updates != 0 || branchExists("app-42-crash-on-save") {
		t.Error("refusal must not write anything")
	}
}

func TestStart_HookReceivesEventOnStdin(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorWarnsWithoutPoints)
	out := filepath.Join(t.TempDir(), "event.json")
	useStartHook(t, "cat > "+out)

	r := runStartWith(t, false)
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("hook did not run: %v", err)
	}
	var ev map[string]any
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("hook stdin is not JSON: %v: %s", err, raw)
	}
	want := map[string]any{
		"event": "ticket_start", "id": float64(42), "subject": "Crash on save",
		"project": "app", "branch": "app-42-crash-on-save",
	}
	for k, v := range want {
		if ev[k] != v {
			t.Errorf("%s = %v, want %v", k, ev[k], v)
		}
	}
	if _, err := time.Parse(time.RFC3339, ev["at"].(string)); err != nil {
		t.Errorf("at is not RFC3339: %v", err)
	}
	chk, ok := ev["check"].(map[string]any)
	if !ok {
		t.Fatalf("check missing: %s", raw)
	}
	if chk["readiness"] != "READY" || chk["score"] != float64(0) || chk["total"] != float64(1) || chk["percent"] != float64(50) {
		t.Errorf("unexpected check summary: %v", chk)
	}
	res := chk["results"].([]any)[0].(map[string]any)
	if res["level"] != "WARN" || res["name"] == "" || res["message"] == "" {
		t.Errorf("unexpected result: %v", res)
	}
	if _, has := ev["check_error"]; has {
		t.Error("check_error must be absent when the check ran")
	}
}

func TestStart_HookEventWhenCheckCouldNotRun(t *testing.T) {
	initTestRepo(t)
	t.Setenv("OP_DOR_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	out := filepath.Join(t.TempDir(), "event.json")
	useStartHook(t, "cat > "+out)

	runStartWith(t, false)

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("hook did not run: %v", err)
	}
	var ev map[string]any
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if v, has := ev["check"]; !has || v != nil {
		t.Errorf("check should be null, got %v (present=%v)", v, has)
	}
	if s, _ := ev["check_error"].(string); s == "" {
		t.Errorf("check_error should say why: %s", raw)
	}
}

func TestStart_FailingHookWarnsAndKeepsSuccess(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorWarnsWithoutPoints)
	useStartHook(t, "echo hook-said-boom >&2; exit 3")

	r := runStartWith(t, false)

	if r.err != nil {
		t.Fatalf("a failing hook must not fail op start: %v", r.err)
	}
	if !strings.Contains(r.stderr, "start_hook") || !strings.Contains(r.stderr, "hook-said-boom") {
		t.Errorf("expected warning and hook output on stderr, got: %q", r.stderr)
	}
	if r.updates != 1 {
		t.Error("ticket should have started")
	}
}

func TestStart_TimedOutHookWarnsAndKeepsSuccess(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorWarnsWithoutPoints)
	useStartHook(t, "sleep 5")
	old := startHookTimeout
	startHookTimeout = 100 * time.Millisecond
	t.Cleanup(func() { startHookTimeout = old })

	begin := time.Now()
	r := runStartWith(t, false)

	if r.err != nil {
		t.Fatalf("a timed-out hook must not fail op start: %v", r.err)
	}
	if !strings.Contains(r.stderr, "start_hook") || time.Since(begin) > 3*time.Second {
		t.Errorf("expected prompt timeout warning, got %q after %v", r.stderr, time.Since(begin))
	}
}

func TestStart_NoHookConfiguredRunsNothing(t *testing.T) {
	initTestRepo(t)
	useDoR(t, dorWarnsWithoutPoints)
	useStartHook(t, "")

	r := runStartWith(t, false)

	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if strings.Contains(r.stderr, "start_hook") {
		t.Errorf("no hook configured, but stderr mentions one: %q", r.stderr)
	}
}
