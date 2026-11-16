package api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// dropFirst closes the first n connections before any response, the way a
// dropped network or a refused dial looks to the client.
type dropFirst struct {
	net.Listener
	mu      sync.Mutex
	n, seen int
}

func (l *dropFirst) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		l.mu.Lock()
		l.seen++
		drop := l.seen <= l.n
		l.mu.Unlock()
		if !drop {
			return conn, nil
		}
		conn.Close()
	}
}

func (l *dropFirst) connections() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.seen
}

func flakyServer(t *testing.T, drop, status int) (*Client, *dropFirst, *int) {
	t.Helper()
	saved := readRetryWaits
	readRetryWaits = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() { readRetryWaits = saved })

	hits := 0
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
	}))
	l := &dropFirst{Listener: srv.Listener, n: drop}
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "secret", ""), l, &hits
}

func TestARead_SurvivesTwoDroppedConnections(t *testing.T) {
	c, _, hits := flakyServer(t, 2, 200)
	var v map[string]interface{}
	if err := c.Get("/work_packages/1", &v); err != nil {
		t.Fatalf("a read failed on a network blip the next try would have survived: %v", err)
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}

func TestAReadThatNeverConnects_SaysHowManyTimesItTried(t *testing.T) {
	c, l, _ := flakyServer(t, 100, 200)
	var v map[string]interface{}
	err := c.Get("/work_packages/1", &v)
	if err == nil || !strings.Contains(err.Error(), "after 3 tries") {
		t.Fatalf("err = %v, want it to say it tried 3 times", err)
	}
	if l.connections() != 3 {
		t.Errorf("connections = %d, want 3", l.connections())
	}
}

// A write may have reached the server before the connection broke; sending it
// again could post a comment or create a ticket twice.
func TestAWrite_IsNeverResent(t *testing.T) {
	c, l, _ := flakyServer(t, 1, 200)
	if err := c.Post("/work_packages/1/activities", map[string]string{"comment": "x"}, nil); err == nil {
		t.Fatal("expected the dropped POST to fail")
	}
	if l.connections() != 1 {
		t.Errorf("the POST was sent %d times, want 1", l.connections())
	}
}

// Any HTTP answer, even a 503, came from the server; retrying it is the
// caller's call, not the transport's.
func TestAnHTTPError_IsNotRetried(t *testing.T) {
	c, _, hits := flakyServer(t, 0, 503)
	var v map[string]interface{}
	if err := c.Get("/work_packages/1", &v); err == nil {
		t.Fatal("expected the 503 to surface")
	}
	if *hits != 1 {
		t.Errorf("hits = %d, want 1", *hits)
	}
}
