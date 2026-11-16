package api

import (
	"fmt"
	"net/http"
	"time"
)

// readRetryWaits are the pauses before each extra try of a read. A refused or
// dropped connection gets three tries in about 7s. A connection that times out
// takes 30s per try, so the client's 60s timeout ends it after the second.
var readRetryWaits = []time.Duration{2 * time.Second, 5 * time.Second}

// retryReads sends a GET or HEAD again when no response arrived at all: a dial
// timeout, a refused or dropped connection. Writes are sent once, because the
// server may have acted on them before the connection broke, and any HTTP
// response, even a 5xx, goes back to the caller as it is.
type retryReads struct {
	next  http.RoundTripper
	waits []time.Duration
}

func newTransport() http.RoundTripper {
	return retryReads{next: http.DefaultTransport, waits: readRetryWaits}
}

func (r retryReads) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.next.RoundTrip(req)
	if err == nil || (req.Method != http.MethodGet && req.Method != http.MethodHead) {
		return resp, err
	}
	tries := 1
	for _, wait := range r.waits {
		select {
		case <-time.After(wait):
		case <-req.Context().Done():
			return nil, err
		}
		tries++
		if resp, err = r.next.RoundTrip(req); err == nil {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("no response from %s after %d tries: %w", req.URL.Host, tries, err)
}
