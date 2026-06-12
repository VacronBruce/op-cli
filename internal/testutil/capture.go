package testutil

import (
	"bytes"
	"io"
	"os"
)

// CaptureStdout captures everything written to os.Stdout by fn.
// It is not safe for parallel tests: it swaps the process-global os.Stdout.
func CaptureStdout(fn func()) string {
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

// CaptureStderr captures everything written to os.Stderr by fn.
// It is not safe for parallel tests: it swaps the process-global os.Stderr.
func CaptureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
