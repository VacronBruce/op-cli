package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// isolateSprintConfig detaches a test from the process-global viper "sprint" key.
// The real create command binds --sprint to viper and cobra.OnInitialize(initConfig)
// loads the host's ~/.oprc + OP_SPRINT into that same global viper, so without this a
// developer's configured sprint leaks into these tests and is resolved against the
// routed board (the reported `resolving sprint: …` failure). Sets sprint to value for
// the test and restores the prior state on cleanup.
func isolateSprintConfig(t *testing.T, value string) {
	t.Helper()
	t.Setenv("OP_SPRINT", "")
	prev := viper.Get("sprint")
	viper.Set("sprint", value)
	t.Cleanup(func() { viper.Set("sprint", prev) })
}

// newCreateRoot wires create the way the real CLI does: --project is a PERSISTENT
// flag on the root command (not on create), so createProject's cmd.Flag("project")
// exercises the genuine inherited-flag lookup and .Changed is set only by an
// actually-typed -p (mirroring production). Returns a root ready for SetArgs.
func newCreateRoot() *cobra.Command {
	root := &cobra.Command{Use: "op", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringP("project", "p", "", "")

	create := &cobra.Command{Use: "create", Args: cobra.MinimumNArgs(2), RunE: runCreate, SilenceUsage: true, SilenceErrors: true}
	create.Flags().StringP("assignee", "a", "", "")
	create.Flags().String("priority", "Normal", "")
	create.Flags().StringP("description", "d", "", "")
	create.Flags().Int("points", 0, "")
	create.Flags().String("sprint", "", "")
	create.Flags().String("start", "", "")
	create.Flags().String("due", "", "")
	create.Flags().String("parent", "", "")
	create.Flags().StringP("epic", "e", "", "")
	create.Flags().StringSlice("component", nil, "")
	create.Flags().StringSlice("product", nil, "")
	create.Flags().String("tech-area", "", "")
	create.Flags().StringSlice("label", nil, "")
	create.Flags().StringSlice("attach", nil, "")
	// Mirror the production wiring (create.go init): --sprint is read via viper, so
	// bind it here too. Without this, viper.GetString("sprint") never sees an
	// explicitly typed --sprint in tests and the explicit-sprint path goes untested.
	_ = viper.BindPFlag("sprint", create.Flags().Lookup("sprint"))
	root.AddCommand(create)
	return root
}

// resolverGetFn serves the /types and /priorities collections the create
// resolver needs, so tests can focus on routing behavior.
func resolverGetFn() func(string, interface{}) error {
	return func(path string, result interface{}) error {
		var js string
		switch {
		case strings.HasPrefix(path, "/types"):
			js = `{"_embedded":{"elements":[
				{"id":1,"name":"Task","_links":{"self":{"href":"/api/v3/types/1"}}},
				{"id":7,"name":"Bug","_links":{"self":{"href":"/api/v3/types/7"}}},
				{"id":5,"name":"Feature","_links":{"self":{"href":"/api/v3/types/5"}}}]}}`
		case strings.HasPrefix(path, "/priorities"):
			js = `{"_embedded":{"elements":[
				{"id":8,"name":"Normal","_links":{"self":{"href":"/api/v3/priorities/8"}}}]}}`
		default:
			return fmt.Errorf("unexpected GET %s", path)
		}
		return json.Unmarshal([]byte(js), result)
	}
}

// runCreateForType records the project a work package would be created in, runs
// `op create [-p <explicitP>] <typeName> <subject>` through the real cobra root,
// and returns the captured project.
func runCreateForType(t *testing.T, typeName, ambientProject, explicitP string) (string, error) {
	t.Helper()
	isolateSprintConfig(t, "") // routing tests must not depend on the host's sprint config
	var createdProject string
	mock := &testutil.MockClient{
		ProjectValue: ambientProject,
		GetFn:        resolverGetFn(),
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			return nil, fmt.Errorf("ResolveVersion must not run in a routing test (project=%q name=%q)", project, name)
		},
		CreateWorkPackageFn: func(project string, req *api.CreateWPRequest) (*api.WorkPackage, error) {
			createdProject = project
			return &api.WorkPackage{ID: 123, Subject: req.Subject}, nil
		},
	}
	SetClient(mock)

	root := newCreateRoot()
	args := []string{"create"}
	if explicitP != "" {
		args = append(args, "-p", explicitP) // a genuinely typed -p sets .Changed
	}
	args = append(args, typeName, "a subject")
	root.SetArgs(args)

	var err error
	testutil.CaptureStdout(func() { err = root.Execute() })
	return createdProject, err
}

// `op create bug` with no project at all must route to the bug board — the goal:
// it should never require -p and never land on the ambient board.
func TestCreate_BugRoutesToBugBoardByDefault(t *testing.T) {
	got, err := runCreateForType(t, "bug", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "bug" {
		t.Errorf("bug must route to 'bug' board, got %q", got)
	}
}

// An abbreviated type arg (`op create b`) resolves to Bug and must route the same
// way — routing keys off the resolved canonical type, not the raw arg. Keyed off
// the raw arg, typeProjectFor("b") would be "" and this would fall to ambient.
func TestCreate_AbbreviatedBugRoutesToBugBoard(t *testing.T) {
	got, err := runCreateForType(t, "b", "app", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "bug" {
		t.Errorf("abbreviated 'b' must resolve to Bug and route to 'bug', got %q", got)
	}
}

// An ambient project (OP_PROJECT/.oprc) must NOT override bug routing — otherwise
// a session set to -p app silently mis-files bugs onto the App board.
func TestCreate_AmbientProjectDoesNotOverrideBugRouting(t *testing.T) {
	got, err := runCreateForType(t, "bug", "app", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "bug" {
		t.Errorf("ambient 'app' must not override bug routing, got %q", got)
	}
}

// An explicitly typed -p is the documented override and must win over routing.
// This exercises the real persistent-flag .Changed path via cobra parsing.
func TestCreate_ExplicitProjectOverridesBugRouting(t *testing.T) {
	got, err := runCreateForType(t, "bug", "app", "app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "app" {
		t.Errorf("explicit -p app must override bug routing, got %q", got)
	}
}

// A non-routed type falls back to the ambient project, unchanged from before.
func TestCreate_NonBugUsesAmbientProject(t *testing.T) {
	got, err := runCreateForType(t, "task", "app", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "app" {
		t.Errorf("task must use ambient 'app', got %q", got)
	}
}

// Creating a bug that routes to the bug board must announce the destination
// before the write; an explicit -p override must NOT print that notice (it isn't
// going to the bug board).
func TestCreate_RoutedBugAnnouncesDestination(t *testing.T) {
	isolateSprintConfig(t, "")
	mock := &testutil.MockClient{
		ProjectValue: "",
		GetFn:        resolverGetFn(),
		CreateWorkPackageFn: func(project string, req *api.CreateWPRequest) (*api.WorkPackage, error) {
			return &api.WorkPackage{ID: 1, Subject: req.Subject}, nil
		},
	}
	SetClient(mock)

	root := newCreateRoot()
	root.SetArgs([]string{"create", "bug", "a subject"})
	out := testutil.CaptureStdout(func() { _ = root.Execute() })
	if !strings.Contains(out, `Filing this bug on the "bug" board`) {
		t.Errorf("routed bug must announce the bug board, got: %s", out)
	}

	root = newCreateRoot()
	root.SetArgs([]string{"create", "-p", "app", "bug", "a subject"})
	out = testutil.CaptureStdout(func() { _ = root.Execute() })
	if strings.Contains(out, "Filing this bug on") {
		t.Errorf("explicit -p override must not announce the bug board, got: %s", out)
	}
}

// A sprint configured in .oprc/OP_SPRINT belongs to the ambient project. When a bug
// routes to the bug board, that ambient sprint must NOT be resolved against (it lives
// on a different board, so resolution fails) nor attached. Pins the reported crash.
func TestCreate_RoutedBugIgnoresAmbientSprint(t *testing.T) {
	isolateSprintConfig(t, "App_05/19/2026") // an App-board sprint set in config
	resolveCalls := 0
	var gotReq *api.CreateWPRequest
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn:        resolverGetFn(),
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			resolveCalls++
			return nil, fmt.Errorf("ResolveVersion should not run for a routed bug (project=%q name=%q)", project, name)
		},
		CreateWorkPackageFn: func(project string, req *api.CreateWPRequest) (*api.WorkPackage, error) {
			gotReq = req
			return &api.WorkPackage{ID: 123, Subject: req.Subject}, nil
		},
	}
	SetClient(mock)

	root := newCreateRoot()
	root.SetArgs([]string{"create", "bug", "a subject"})
	var err error
	testutil.CaptureStdout(func() { err = root.Execute() })

	if err != nil {
		t.Fatalf("routed bug with an ambient sprint must succeed, got: %v", err)
	}
	if resolveCalls != 0 {
		t.Errorf("ambient sprint must not be resolved for a routed bug, ResolveVersion called %d time(s)", resolveCalls)
	}
	if gotReq == nil {
		t.Fatal("create request was never captured")
	}
	if _, ok := gotReq.Links["version"]; ok {
		t.Error("routed bug must not carry a version link inherited from ambient config")
	}
}

// An explicitly typed --sprint is honored even on a routed bug: it resolves against the
// bug board (and fails loud there if it doesn't exist), unlike an ambient config sprint.
func TestCreate_ExplicitSprintResolvesAgainstBugBoard(t *testing.T) {
	// Drive the value through the real --sprint -> viper binding (newCreateRoot
	// mirrors the production BindPFlag), NOT a viper.Set seam. A Set override would
	// outrank the flag, so start from a clean viper — clearing any override a prior
	// test left — and let the bound, explicitly-typed flag be the source. An explicit
	// flag also outranks env/.oprc, so no extra masking is needed.
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("OP_SPRINT", "")
	var resolveProject, resolveName string
	resolveCalls := 0
	mock := &testutil.MockClient{
		ProjectValue: "app",
		GetFn:        resolverGetFn(),
		ResolveVersionFn: func(project, name string) (*api.Version, error) {
			resolveCalls++
			resolveProject, resolveName = project, name
			v := &api.Version{ID: 9, Name: name}
			v.Links.Self = api.Link{Href: "/api/v3/versions/9"}
			return v, nil
		},
		CreateWorkPackageFn: func(project string, req *api.CreateWPRequest) (*api.WorkPackage, error) {
			return &api.WorkPackage{ID: 123, Subject: req.Subject}, nil
		},
	}
	SetClient(mock)

	root := newCreateRoot()
	root.SetArgs([]string{"create", "bug", "a subject", "--sprint", "BugSprint1"})
	var err error
	testutil.CaptureStdout(func() { err = root.Execute() })

	if err != nil {
		t.Fatalf("explicit --sprint on a routed bug must succeed, got: %v", err)
	}
	if resolveCalls != 1 {
		t.Fatalf("explicit --sprint must resolve exactly once, got %d", resolveCalls)
	}
	if resolveProject != "bug" {
		t.Errorf("explicit --sprint must resolve against the bug board, got project %q", resolveProject)
	}
	if resolveName != "BugSprint1" {
		t.Errorf("ResolveVersion name = %q, want %q", resolveName, "BugSprint1")
	}
}
