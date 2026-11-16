package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
)

func briefWP(id int, subject, status, assignee, sprint, release string) *api.WorkPackage {
	wp := &api.WorkPackage{ID: id, Subject: subject}
	wp.Links.Type = api.Link{Title: "Bug"}
	wp.Links.Status = api.Link{Title: status}
	if assignee != "" {
		wp.Links.Assignee = api.Link{Title: assignee}
	}
	if sprint != "" {
		wp.Links.Version = api.Link{Title: sprint}
	}
	if release != "" {
		wp.Links.Release = []api.Link{{Title: release}}
	}
	return wp
}

func showMock(wps map[int]*api.WorkPackage) *testutil.MockClient {
	return &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			if wp, ok := wps[id]; ok {
				return wp, nil
			}
			return nil, errors.New("not found")
		},
		ListAttachmentsFn: func(int) (*api.AttachmentCollection, error) { return &api.AttachmentCollection{}, nil },
	}
}

func TestShow_SeveralIDsPrintEachBlockSeparatedByBlankLine(t *testing.T) {
	SetClient(showMock(map[int]*api.WorkPackage{1: briefWP(1, "First", "Open", "", "", ""), 2: briefWP(2, "Second", "Done", "", "", "")}))
	var err error
	out := testutil.CaptureStdout(func() { err = runShow(newShowCmd(), []string{"1", "2"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "#1 First") || !strings.Contains(out, "0%\n\n#2 Second") {
		t.Errorf("blocks missing or not blank-line separated:\n%s", out)
	}
}

func TestShow_FailingIDReportsToStderrOthersStillPrintAndExitNonZero(t *testing.T) {
	SetClient(showMock(map[int]*api.WorkPackage{1: briefWP(1, "First", "Open", "", "", ""), 3: briefWP(3, "Third", "Open", "", "", "")}))
	var err error
	var out string
	stderr := testutil.CaptureStderr(func() {
		out = testutil.CaptureStdout(func() { err = runShow(newShowCmd(), []string{"1", "2", "3"}) })
	})
	if err == nil {
		t.Fatal("expected non-zero exit (error) when an ID fails")
	}
	if !strings.Contains(out, "#1 First") || !strings.Contains(out, "#3 Third") {
		t.Errorf("remaining IDs must still print:\n%s", out)
	}
	if !strings.Contains(stderr, "2") || !strings.Contains(stderr, "not found") {
		t.Errorf("failure must be on stderr, got %q", stderr)
	}
}

func TestShow_BriefIsOneLinePerIDWithNoExtraRequests(t *testing.T) {
	mock := showMock(map[int]*api.WorkPackage{
		1: briefWP(1, "Paywall crash", "In Progress", "Ada", "App_11/17/2026", "[Android][EET] 3.2.1"),
		2: briefWP(2, "Bare ticket", "To Do", "", "", ""),
	})
	mock.ListAttachmentsFn = func(int) (*api.AttachmentCollection, error) {
		t.Error("brief must not list attachments")
		return nil, nil
	}
	mock.ListActivitiesFn = func(int) (*api.ActivityCollection, error) { t.Error("brief must not list activities"); return nil, nil }
	SetClient(mock)
	c := newShowCmd()
	_ = c.Flags().Set("brief", "true")
	var err error
	out := testutil.CaptureStdout(func() { err = runShow(c, []string{"1", "2"}) })
	if err != nil {
		t.Fatal(err)
	}
	want := "#1 [In Progress] Paywall crash | Ada | App_11/17/2026 | [Android][EET] 3.2.1\n" +
		"#2 [To Do] Bare ticket | - | - | -\n"
	if out != want {
		t.Errorf("got %q want %q", out, want)
	}
}

func jsonResponder(bodies map[string]string, hits *[]string) func(path string, result interface{}) error {
	return func(path string, result interface{}) error {
		*hits = append(*hits, path)
		b, ok := bodies[path]
		if !ok {
			return &api.APIError{StatusCode: 404, Message: "The work package you are looking for cannot be found or has been deleted."}
		}
		return json.Unmarshal([]byte(b), result)
	}
}

func TestShow_JSONSingleIDIsPrettyObjectAndNothingElse(t *testing.T) {
	var hits []string
	mock := &testutil.MockClient{GetFn: jsonResponder(map[string]string{"/work_packages/5": `{"id":5,"subject":"x"}`}, &hits)}
	SetClient(mock)
	c := newShowCmd()
	_ = c.Flags().Set("json", "true")
	var err error
	out := testutil.CaptureStdout(func() { err = runShow(c, []string{"5"}) })
	if err != nil {
		t.Fatal(err)
	}
	if out != "{\n  \"id\": 5,\n  \"subject\": \"x\"\n}\n" {
		t.Errorf("got %q", out)
	}
	if fmt.Sprint(hits) != "[/work_packages/5]" {
		t.Errorf("only the work package may be requested, got %v", hits)
	}
}

func TestShow_JSONSeveralIDsIsArray(t *testing.T) {
	var hits []string
	mock := &testutil.MockClient{GetFn: jsonResponder(map[string]string{
		"/work_packages/5": `{"id":5}`, "/work_packages/6": `{"id":6}`}, &hits)}
	SetClient(mock)
	c := newShowCmd()
	_ = c.Flags().Set("json", "true")
	var err error
	out := testutil.CaptureStdout(func() { err = runShow(c, []string{"5", "6"}) })
	if err != nil {
		t.Fatal(err)
	}
	if out != "[\n  {\n    \"id\": 5\n  },\n  {\n    \"id\": 6\n  }\n]\n" {
		t.Errorf("got %q", out)
	}
}

func TestShow_JSONWhenEveryIDFailsIsAnEmptyArrayAndKeepsTheServerMessage(t *testing.T) {
	var hits []string
	SetClient(&testutil.MockClient{GetFn: jsonResponder(nil, &hits)})
	c := newShowCmd()
	_ = c.Flags().Set("json", "true")
	var err error
	var out string
	stderr := testutil.CaptureStderr(func() {
		out = testutil.CaptureStdout(func() { err = runShow(c, []string{"8", "9"}) })
	})
	if out != "[]\n" {
		t.Errorf("stdout must stay valid JSON, got %q", out)
	}
	if err == nil {
		t.Error("expected a non-zero exit")
	}
	if !strings.Contains(stderr, "cannot be found or has been deleted") {
		t.Errorf("the server's message must reach stderr, got %q", stderr)
	}
}

func TestShow_URLWithSeveralIDsPrintsOnePerLine(t *testing.T) {
	SetClient(&testutil.MockClient{BaseURLValue: "https://op.test"})
	c := newShowCmd()
	_ = c.Flags().Set("url", "true")
	out := testutil.CaptureStdout(func() { _ = runShow(c, []string{"1", "2"}) })
	if out != "https://op.test/work_packages/1\nhttps://op.test/work_packages/2\n" {
		t.Errorf("got %q", out)
	}
}
