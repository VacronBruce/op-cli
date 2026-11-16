package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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

func showMock(t *testing.T, wps map[int]*api.WorkPackage) *testutil.MockClient {
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
	SetClient(showMock(t, map[int]*api.WorkPackage{1: briefWP(1, "First", "Open", "", "", ""), 2: briefWP(2, "Second", "Done", "", "", "")}))
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
	SetClient(showMock(t, map[int]*api.WorkPackage{1: briefWP(1, "First", "Open", "", "", ""), 3: briefWP(3, "Third", "Open", "", "", "")}))
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
	mock := showMock(t, map[int]*api.WorkPackage{
		1: briefWP(1, "Paywall crash", "In Progress", "Ada", "App_11/17/2026", "[Android][EET] 3.2.1"),
		2: briefWP(2, "Bare ticket", "To Do", "", "", ""),
	})
	mock.ListAttachmentsFn = func(int) (*api.AttachmentCollection, error) { t.Error("brief must not list attachments"); return nil, nil }
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

func rawResponder(t *testing.T, bodies map[string]string, hits *[]string) func(method, href string) (*http.Response, error) {
	return func(method, href string) (*http.Response, error) {
		*hits = append(*hits, method+" "+href)
		b, ok := bodies[href]
		if !ok {
			return nil, errors.New("HTTP 404")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(b))}, nil
	}
}

func TestShow_JSONSingleIDIsPrettyObjectAndNothingElse(t *testing.T) {
	var hits []string
	mock := &testutil.MockClient{DoRawFn: rawResponder(t, map[string]string{"/api/v3/work_packages/5": `{"id":5,"subject":"x"}`}, &hits)}
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
	if fmt.Sprint(hits) != "[GET /api/v3/work_packages/5]" {
		t.Errorf("only the work package may be requested, got %v", hits)
	}
}

func TestShow_JSONSeveralIDsIsArray(t *testing.T) {
	var hits []string
	mock := &testutil.MockClient{DoRawFn: rawResponder(t, map[string]string{
		"/api/v3/work_packages/5": `{"id":5}`, "/api/v3/work_packages/6": `{"id":6}`}, &hits)}
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

func TestShow_URLWithSeveralIDsPrintsOnePerLine(t *testing.T) {
	SetClient(&testutil.MockClient{BaseURLValue: "https://op.test"})
	c := newShowCmd()
	_ = c.Flags().Set("url", "true")
	out := testutil.CaptureStdout(func() { _ = runShow(c, []string{"1", "2"}) })
	if out != "https://op.test/work_packages/1\nhttps://op.test/work_packages/2\n" {
		t.Errorf("got %q", out)
	}
}
