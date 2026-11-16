package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The usage block is for a command typed wrong. After a server error it buries
// the message an agent needs under forty lines of flags.
func TestARuntimeError_PrintsNoUsageBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "boom"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OP_URL", srv.URL)
	t.Setenv("OP_API_KEY", "key")

	var stderr bytes.Buffer
	rootCmd.SetArgs([]string{"projects"})
	rootCmd.SetErr(&stderr)
	rootCmd.SetOut(&stderr)
	t.Cleanup(func() { rootCmd.SetArgs(nil); rootCmd.SetErr(nil); rootCmd.SetOut(nil) })

	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want the server error", err)
	}
	if strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("a server error printed the usage block:\n%s", stderr.String())
	}
}
