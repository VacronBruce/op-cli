package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func newAPICmd(method, input string) *cobra.Command {
	c := &cobra.Command{}
	c.Flags().StringP("method", "X", "GET", "")
	c.Flags().String("input", "", "")
	_ = c.Flags().Set("method", method)
	if input != "" {
		_ = c.Flags().Set("input", input)
	}
	return c
}

func runAPIWith(mock *testutil.MockClient, c *cobra.Command, args []string) (stdout, stderr string, err error) {
	SetClient(mock)
	stderr = testutil.CaptureStderr(func() {
		stdout = testutil.CaptureStdout(func() { err = runAPI(c, args) })
	})
	return
}

func TestAPI_GETPrintsPrettyJSON(t *testing.T) {
	mock := &testutil.MockClient{RequestFn: func(method, path string, body []byte) (int, []byte, error) {
		if method != "GET" || path != "statuses" || body != nil {
			t.Errorf("sent %s %s %q", method, path, body)
		}
		return 200, []byte(`{"total":2}`), nil
	}}
	out, _, err := runAPIWith(mock, newAPICmd("GET", ""), []string{"statuses"})
	if err != nil || out != "{\n  \"total\": 2\n}\n" {
		t.Errorf("out %q err %v", out, err)
	}
}

func TestAPI_NonJSONBodyIsPrintedUntouched(t *testing.T) {
	mock := &testutil.MockClient{RequestFn: func(string, string, []byte) (int, []byte, error) { return 200, []byte("plain text"), nil }}
	out, _, _ := runAPIWith(mock, newAPICmd("GET", ""), []string{"x"})
	if out != "plain text" {
		t.Errorf("got %q", out)
	}
}

func TestAPI_NonSuccessSendsBodyToStderrAndFails(t *testing.T) {
	mock := &testutil.MockClient{RequestFn: func(string, string, []byte) (int, []byte, error) {
		return 404, []byte(`{"message":"gone"}`), nil
	}}
	out, errOut, err := runAPIWith(mock, newAPICmd("GET", ""), []string{"work_packages/9"})
	if err == nil || err.Error() != "HTTP 404" {
		t.Errorf("err %v", err)
	}
	if out != "" || !strings.Contains(errOut, "gone") {
		t.Errorf("stdout %q stderr %q", out, errOut)
	}
}

func TestAPI_InputFileIsSentAsBodyWithMethod(t *testing.T) {
	f := filepath.Join(t.TempDir(), "b.json")
	_ = os.WriteFile(f, []byte(`{"a":1}`), 0o600)
	var gotMethod, gotBody string
	mock := &testutil.MockClient{RequestFn: func(method, path string, body []byte) (int, []byte, error) {
		gotMethod, gotBody = method, string(body)
		return 200, []byte(`{}`), nil
	}}
	_, _, err := runAPIWith(mock, newAPICmd("patch", f), []string{"work_packages/1"})
	if err != nil || gotMethod != "PATCH" || gotBody != `{"a":1}` {
		t.Errorf("method %q body %q err %v", gotMethod, gotBody, err)
	}
}

func TestAPI_InputFromStdin(t *testing.T) {
	var gotBody string
	mock := &testutil.MockClient{RequestFn: func(_, _ string, body []byte) (int, []byte, error) {
		gotBody = string(body)
		return 200, []byte(`{}`), nil
	}}
	c := newAPICmd("POST", "-")
	c.SetIn(strings.NewReader(`{"s":1}`))
	if _, _, err := runAPIWith(mock, c, []string{"x"}); err != nil || gotBody != `{"s":1}` {
		t.Errorf("body %q err %v", gotBody, err)
	}
}

func TestAPI_InputWithGETIsRejectedBeforeAnyRequest(t *testing.T) {
	mock := &testutil.MockClient{RequestFn: func(string, string, []byte) (int, []byte, error) {
		t.Error("must not send")
		return 0, nil, nil
	}}
	_, _, err := runAPIWith(mock, newAPICmd("GET", "-"), []string{"x"})
	if err == nil || !strings.Contains(err.Error(), "--input") {
		t.Errorf("err %v", err)
	}
}

func TestAPI_InvalidJSONInputIsRejected(t *testing.T) {
	f := filepath.Join(t.TempDir(), "b.json")
	_ = os.WriteFile(f, []byte(`{nope`), 0o600)
	mock := &testutil.MockClient{RequestFn: func(string, string, []byte) (int, []byte, error) {
		t.Error("must not send")
		return 0, nil, nil
	}}
	if _, _, err := runAPIWith(mock, newAPICmd("POST", f), []string{"x"}); err == nil {
		t.Error("expected error")
	}
}
