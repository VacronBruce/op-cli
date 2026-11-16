package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

// Agents fetched single attachments, mostly screenshots pasted into comments,
// through 32 raw API calls in 21 days. --get downloads one by its ID, and the
// file name starts with that ID because pasted screenshots are all image.png.
func TestAttach_GetDownloadsOneAttachmentByID(t *testing.T) {
	dir := t.TempDir()
	var fetched string
	SetClient(&testutil.MockClient{
		GetFn: func(path string, result interface{}) error {
			if path != "/attachments/46813" {
				t.Errorf("metadata path = %s", path)
			}
			return json.Unmarshal([]byte(`{"id": 46813, "fileName": "image.png",
				"_links": {"downloadLocation": {"href": "/api/v3/attachments/46813/content"}}}`), result)
		},
		DoRawFn: func(method, href string) (*http.Response, error) {
			fetched = href
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("PNGDATA"))}, nil
		},
	})
	cmd := newAttachCmd()
	_ = cmd.Flags().Set("get", "46813")
	_ = cmd.Flags().Set("out", dir)

	out := testutil.CaptureStdout(func() {
		if err := runAttach(cmd, []string{"86074"}); err != nil {
			t.Fatal(err)
		}
	})

	want := filepath.Join(dir, "46813-image.png")
	got, err := os.ReadFile(want)
	if err != nil || string(got) != "PNGDATA" {
		t.Fatalf("%s holds %q (%v), want the attachment's bytes", want, got, err)
	}
	if fetched != "/api/v3/attachments/46813/content" {
		t.Errorf("downloaded %s, want the attachment's downloadLocation", fetched)
	}
	if !strings.Contains(out, "Downloaded: "+want) {
		t.Errorf("output %q does not name the file", out)
	}
}
