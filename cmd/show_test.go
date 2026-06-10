package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("download", "d", false, "")
	cmd.Flags().StringP("out", "o", ".", "")
	return cmd
}

func TestShow_ValidID(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			if id != 81321 {
				t.Errorf("expected id 81321, got %d", id)
			}
			wp := &api.WorkPackage{
				ID:      81321,
				Subject: "Test Task",
			}
			// Add basic links to prevent nil pointer panics in WorkPackageDetail
			wp.Links.Type = api.Link{Title: "Task"}
			wp.Links.Status = api.Link{Title: "Open"}
			wp.Links.Project = api.Link{Title: "App"}
			return wp, nil
		},
		GetFn: func(path string, result interface{}) error {
			if strings.Contains(path, "attachments") {
				// return empty attachments
				return nil
			}
			return fmt.Errorf("unexpected Get path: %s", path)
		},
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return &api.ActivityCollection{}, nil
		},
	}
	SetClient(mock)

	out := captureStdout(func() {
		if err := runShow(newShowCmd(), []string{"81321"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "Test Task") {
		t.Errorf("expected task subject in output, got: %s", out)
	}
	if !strings.Contains(out, "Type:") || !strings.Contains(out, "Task") {
		t.Errorf("expected Type label in output, got: %s", out)
	}
	if !strings.Contains(out, "Status:") || !strings.Contains(out, "Open") {
		t.Errorf("expected Status label in output, got: %s", out)
	}
}

func TestShow_InvalidID(t *testing.T) {
	mock := &testutil.MockClient{}
	SetClient(mock)

	captureStdout(func() {
		err := runShow(newShowCmd(), []string{"abc"})
		if err == nil {
			t.Fatal("expected error for invalid ID, got nil")
		}
		if !strings.Contains(err.Error(), "invalid work package ID") {
			t.Errorf("expected invalid id error, got: %v", err)
		}
	})
}

func TestShow_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return nil, errors.New("not found")
		},
	}
	SetClient(mock)

	captureStdout(func() {
		err := runShow(newShowCmd(), []string{"81321"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "getting work package") {
			t.Errorf("expected error to mention 'getting work package', got: %v", err)
		}
	})
}

func TestShow_DownloadAttachments(t *testing.T) {
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			wp := &api.WorkPackage{ID: 81321, Subject: "Task with attachment"}
			wp.Links.Type = api.Link{Title: "Task"}
			wp.Links.Status = api.Link{Title: "Open"}
			wp.Links.Project = api.Link{Title: "App"}
			return wp, nil
		},
		GetFn: func(path string, result interface{}) error {
			if strings.Contains(path, "attachments") {
				// mock returning an attachment collection
				attCol, ok := result.(*attachmentCollection)
				if ok {
					attCol.Total = 1
					attCol.Embedded.Elements = []api.Attachment{
						{
							FileName:    "test.png",
							ContentType: "image/png",
							FileSize:    1024,
							Links: struct {
								Self             api.Link `json:"self"`
								DownloadLocation api.Link `json:"downloadLocation"`
							}{
								Self:             api.Link{},
								DownloadLocation: api.Link{Href: "http://example.com/download/1"},
							},
						},
					}
				}
				return nil
			}
			return fmt.Errorf("unexpected path: %s", path)
		},
		ListActivitiesFn: func(wpID int) (*api.ActivityCollection, error) {
			return &api.ActivityCollection{}, nil
		},
		DoRawFn: func(method, href string) (*http.Response, error) {
			if method != "GET" || href != "http://example.com/download/1" {
				return nil, fmt.Errorf("unexpected DoRaw call")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte("dummy image content"))),
			}, nil
		},
	}
	SetClient(mock)

	tempDir := t.TempDir()

	cmd := newShowCmd()
	_ = cmd.Flags().Set("download", "true")
	_ = cmd.Flags().Set("out", tempDir)

	out := captureStdout(func() {
		if err := runShow(cmd, []string{"81321"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "test.png") {
		t.Errorf("expected 'test.png' in output, got: %s", out)
	}
	if !strings.Contains(out, "Downloaded:") {
		t.Errorf("expected 'Downloaded:' in output, got: %s", out)
	}

	// Verify the file was created
	fileInfo, err := os.Stat(filepath.Join(tempDir, "test.png"))
	if err != nil {
		t.Fatalf("expected downloaded file, got error: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Errorf("expected non-empty downloaded file")
	}
}
