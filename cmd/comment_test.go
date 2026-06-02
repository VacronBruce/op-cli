package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// captureStdout captures everything written to os.Stdout by fn.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// runCommentWith injects mock, runs runComment, and returns stdout + error.
func runCommentWith(t *testing.T, mock *testutil.MockClient, args []string) (string, error) {
	t.Helper()
	SetClient(mock)
	var err error
	out := captureStdout(func() {
		err = runComment(&cobra.Command{}, args)
	})
	return out, err
}

// --- List mode tests ---

func TestComment_List_WithComments(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Total = 2
	ac.Embedded.Elements = []api.Activity{
		{
			ID:        1,
			Comment:   &api.Formattable{Format: "markdown", Raw: "First comment"},
			CreatedAt: "2024-01-15T10:00:00Z",
			Links: struct {
				User api.Link `json:"user"`
			}{User: api.Link{Title: "Alice"}},
		},
		{
			ID:        2,
			Comment:   &api.Formattable{Format: "markdown", Raw: "Second comment"},
			CreatedAt: "2024-01-16T11:30:00Z",
			Links: struct {
				User api.Link `json:"user"`
			}{User: api.Link{Title: "Bob"}},
		},
	}

	mock := &testutil.MockClient{
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			if wpID != 42 {
				t.Errorf("expected wpID=42, got %d", wpID)
			}
			return ac, nil
		},
	}

	out, err := runCommentWith(t, mock, []string{"42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Comments (2)") {
		t.Errorf("expected 'Comments (2)' in output, got: %s", out)
	}
	if !strings.Contains(out, "First comment") {
		t.Errorf("expected 'First comment' in output, got: %s", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %s", out)
	}
}

func TestComment_List_NoComments(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Total = 0

	mock := &testutil.MockClient{
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return ac, nil
		},
	}

	out, err := runCommentWith(t, mock, []string{"99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No comments.") {
		t.Errorf("expected 'No comments.' in output, got: %s", out)
	}
}

func TestComment_List_ActivitiesWithoutCommentFiltered(t *testing.T) {
	// Activity with nil comment should be filtered out; only the one with a comment shows.
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		{
			ID:      1,
			Comment: nil, // journal entry, no comment
		},
		{
			ID:        2,
			Comment:   &api.Formattable{Format: "markdown", Raw: "Visible comment"},
			CreatedAt: "2024-03-01T08:00:00Z",
			Links: struct {
				User api.Link `json:"user"`
			}{User: api.Link{Title: "Carol"}},
		},
	}

	mock := &testutil.MockClient{
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return ac, nil
		},
	}

	out, err := runCommentWith(t, mock, []string{"10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Comments (1)") {
		t.Errorf("expected 'Comments (1)', got: %s", out)
	}
	if !strings.Contains(out, "Visible comment") {
		t.Errorf("expected 'Visible comment', got: %s", out)
	}
}

func TestComment_List_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return nil, errors.New("network timeout")
		},
	}

	_, err := runCommentWith(t, mock, []string{"5"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "listing activities") {
		t.Errorf("expected error to mention 'listing activities', got: %v", err)
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("expected original error in message, got: %v", err)
	}
}

// --- Post mode tests ---

func TestComment_Post_Success(t *testing.T) {
	var capturedID int
	var capturedMsg string

	mock := &testutil.MockClient{
		PostCommentFn: func(wpID int, markdown string) error {
			capturedID = wpID
			capturedMsg = markdown
			return nil
		},
	}

	out, err := runCommentWith(t, mock, []string{"81321", "LGTM"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != 81321 {
		t.Errorf("expected wpID=81321, got %d", capturedID)
	}
	if capturedMsg != "LGTM" {
		t.Errorf("expected msg='LGTM', got %q", capturedMsg)
	}
	if !strings.Contains(out, "Comment posted on #81321") {
		t.Errorf("expected confirmation message, got: %s", out)
	}
}

func TestComment_Post_TrimsWhitespace(t *testing.T) {
	var capturedMsg string
	mock := &testutil.MockClient{
		PostCommentFn: func(wpID int, markdown string) error {
			capturedMsg = markdown
			return nil
		},
	}

	_, err := runCommentWith(t, mock, []string{"1", "  hello world  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMsg != "hello world" {
		t.Errorf("expected trimmed message, got %q", capturedMsg)
	}
}

func TestComment_Post_EmptyMessage(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runCommentWith(t, mock, []string{"42", "   "})
	if err == nil {
		t.Fatal("expected error for empty message, got nil")
	}
	if !strings.Contains(err.Error(), "comment message cannot be empty") {
		t.Errorf("expected empty-message error, got: %v", err)
	}
}

// --- Edit mode tests ---

// runCommentEditWith runs runComment with the --edit flag set to editID.
func runCommentEditWith(t *testing.T, mock *testutil.MockClient, editID int, args []string) (string, error) {
	t.Helper()
	SetClient(mock)
	c := &cobra.Command{}
	c.Flags().Int("edit", editID, "")
	var err error
	out := captureStdout(func() {
		err = runComment(c, args)
	})
	return out, err
}

func TestComment_Edit_Success(t *testing.T) {
	var capturedID int
	var capturedMsg string
	mock := &testutil.MockClient{
		EditCommentFn: func(activityID int, markdown string) error {
			capturedID = activityID
			capturedMsg = markdown
			return nil
		},
		PostCommentFn: func(int, string) error {
			t.Error("PostComment should not be called in edit mode")
			return nil
		},
	}

	out, err := runCommentEditWith(t, mock, 1234, []string{"81321", "  fixed typo  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != 1234 {
		t.Errorf("expected activityID=1234, got %d", capturedID)
	}
	if capturedMsg != "fixed typo" {
		t.Errorf("expected trimmed msg 'fixed typo', got %q", capturedMsg)
	}
	if !strings.Contains(out, "Comment #1234 updated") {
		t.Errorf("expected edit confirmation, got: %s", out)
	}
}

func TestComment_Edit_RequiresMessage(t *testing.T) {
	mock := &testutil.MockClient{
		EditCommentFn: func(int, string) error {
			t.Error("EditComment should not be called without a message")
			return nil
		},
	}
	_, err := runCommentEditWith(t, mock, 1234, []string{"81321"})
	if err == nil || !strings.Contains(err.Error(), "--edit requires the new comment text") {
		t.Fatalf("expected missing-text error, got: %v", err)
	}
}

func TestComment_Post_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		PostCommentFn: func(wpID int, markdown string) error {
			return errors.New("forbidden")
		},
	}

	_, err := runCommentWith(t, mock, []string{"7", "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "posting comment") {
		t.Errorf("expected error to mention 'posting comment', got: %v", err)
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("expected original error in message, got: %v", err)
	}
}

// --- ID validation tests ---

func TestComment_InvalidID_NotNumeric(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runCommentWith(t, mock, []string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid work package ID") {
		t.Errorf("expected invalid ID error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Errorf("expected the bad ID in error message, got: %v", err)
	}
}

func TestComment_InvalidID_Float(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runCommentWith(t, mock, []string{"3.14"})
	if err == nil {
		t.Fatal("expected error for float ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid work package ID") {
		t.Errorf("expected invalid ID error, got: %v", err)
	}
}

func TestComment_InvalidID_Empty(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runCommentWith(t, mock, []string{""})
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid work package ID") {
		t.Errorf("expected invalid ID error, got: %v", err)
	}
}
