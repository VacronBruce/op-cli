package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func newAttachCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().String("desc", "", "")
	c.Flags().Bool("list", false, "")
	c.Flags().Int("remove", 0, "")
	return c
}

func attachmentsFixture() *api.AttachmentCollection {
	col := &api.AttachmentCollection{Total: 2}
	col.Embedded.Elements = []api.Attachment{
		{ID: 318, FileName: "screen.png", ContentType: "image/png", FileSize: 1024},
		{ID: 319, FileName: "crash.log", ContentType: "text/plain", FileSize: 99},
	}
	return col
}

// --list is how users discover what to --remove: each line must carry the
// attachment ID alongside the filename.
func TestAttach_ListShowsAttachmentIDs(t *testing.T) {
	mock := &testutil.MockClient{
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			if wpID != 81317 {
				t.Errorf("expected wpID 81317, got %d", wpID)
			}
			return attachmentsFixture(), nil
		},
	}
	SetClient(mock)

	cmd := newAttachCmd()
	_ = cmd.Flags().Set("list", "true")
	var err error
	out := testutil.CaptureStdout(func() { err = runAttach(cmd, []string{"81317"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "#318") || !strings.Contains(out, "screen.png") {
		t.Errorf("expected ID + filename lines, got: %s", out)
	}
	if !strings.Contains(out, "#319") || !strings.Contains(out, "crash.log") {
		t.Errorf("expected second attachment line, got: %s", out)
	}
}

func TestAttach_ListEmpty(t *testing.T) {
	mock := &testutil.MockClient{
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			return &api.AttachmentCollection{}, nil
		},
	}
	SetClient(mock)

	cmd := newAttachCmd()
	_ = cmd.Flags().Set("list", "true")
	var err error
	out := testutil.CaptureStdout(func() { err = runAttach(cmd, []string{"81317"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No attachments") {
		t.Errorf("expected no-attachments message, got: %s", out)
	}
}

// --remove only deletes after confirming the attachment belongs to THIS work
// package — a bare attachment ID must not be able to delete a file from a
// different ticket via typo.
func TestAttach_RemoveDeletesOnlyOwnAttachment(t *testing.T) {
	var deleted int
	mock := &testutil.MockClient{
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			return attachmentsFixture(), nil
		},
		DeleteAttachmentFn: func(attID int) error {
			deleted = attID
			return nil
		},
	}
	SetClient(mock)

	cmd := newAttachCmd()
	_ = cmd.Flags().Set("remove", "318")
	var err error
	out := testutil.CaptureStdout(func() { err = runAttach(cmd, []string{"81317"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 318 {
		t.Errorf("expected attachment 318 deleted, got %d", deleted)
	}
	if !strings.Contains(out, "Removed attachment #318") || !strings.Contains(out, "screen.png") {
		t.Errorf("confirmation must name the ID and file, got: %s", out)
	}
}

func TestAttach_RemoveNotOnTicketFailsLoudListingExisting(t *testing.T) {
	deleted := false
	mock := &testutil.MockClient{
		ListAttachmentsFn: func(wpID int) (*api.AttachmentCollection, error) {
			return attachmentsFixture(), nil
		},
		DeleteAttachmentFn: func(attID int) error {
			deleted = true
			return nil
		},
	}
	SetClient(mock)

	cmd := newAttachCmd()
	_ = cmd.Flags().Set("remove", "999")
	err := runAttach(cmd, []string{"81317"})
	if err == nil || !strings.Contains(err.Error(), "no attachment #999") {
		t.Fatalf("expected no-match error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "#318 screen.png") {
		t.Errorf("error should list existing attachments, got: %v", err)
	}
	if deleted {
		t.Error("nothing must be deleted on no-match")
	}
}

func TestAttach_RemoveNegativeIDRejected(t *testing.T) {
	SetClient(&testutil.MockClient{})
	cmd := newAttachCmd()
	_ = cmd.Flags().Set("remove", "-1")
	err := runAttach(cmd, []string{"81317"})
	if err == nil || !strings.Contains(err.Error(), "positive attachment ID") {
		t.Fatalf("expected positive-ID error, got: %v", err)
	}
}

// A bare `op attach <id>` (no files, no mode flag) must explain the three
// modes instead of silently doing nothing.
func TestAttach_NoFilesNoFlagsIsAnError(t *testing.T) {
	SetClient(&testutil.MockClient{})
	err := runAttach(newAttachCmd(), []string{"81317"})
	if err == nil || !strings.Contains(err.Error(), "provide files to upload") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}
