package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type seen struct {
	method, uri, contentType, user, pass, body string
	hits                                       int
}

func requestServer(t *testing.T, status int, reply string) (*httptest.Server, *seen) {
	t.Helper()
	got := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.hits++
		got.method, got.uri, got.contentType = r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type")
		got.user, got.pass, _ = r.BasicAuth()
		b, _ := io.ReadAll(r.Body)
		got.body = string(b)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestRequest_AcceptsEveryPathSpelling(t *testing.T) {
	srv, got := requestServer(t, 200, `{}`)
	c := NewClient(srv.URL, "secret", "")
	for _, p := range []string{
		"work_packages/1?pageSize=5",
		"/work_packages/1?pageSize=5",
		"/api/v3/work_packages/1?pageSize=5",
		"api/v3/work_packages/1?pageSize=5",
		srv.URL + "/api/v3/work_packages/1?pageSize=5",
	} {
		status, _, err := c.Request("GET", p, nil)
		if err != nil || status != 200 {
			t.Fatalf("%s: status %d err %v", p, status, err)
		}
		if got.uri != "/api/v3/work_packages/1?pageSize=5" {
			t.Errorf("%s reached %q", p, got.uri)
		}
		if got.user != "apikey" || got.pass != "secret" {
			t.Errorf("auth not sent: %q %q", got.user, got.pass)
		}
	}
}

func TestRequest_RefusesOtherHostsSoTheKeyNeverLeaves(t *testing.T) {
	srv, got := requestServer(t, 200, `{}`)
	c := NewClient(srv.URL, "secret", "")
	for _, p := range []string{
		"https://evil.example.com/api/v3/work_packages/1",
		"http://127.0.0.1:1/api/v3/x",
		strings.Replace(srv.URL, "http://", "https://", 1) + "/api/v3/x",
		"http://user@evil.example.com/",
	} {
		if _, _, err := c.Request("GET", p, nil); err == nil {
			t.Errorf("%s must be refused", p)
		}
	}
	if got.hits != 0 {
		t.Errorf("no request may be sent, got %d", got.hits)
	}
}

func TestRequest_BodyAndMethod(t *testing.T) {
	srv, got := requestServer(t, 201, `{"ok":true}`)
	c := NewClient(srv.URL, "k", "")
	status, body, err := c.Request("POST", "work_packages/1/activities", []byte(`{"a":1}`))
	if err != nil || status != 201 || string(body) != `{"ok":true}` {
		t.Fatalf("status %d body %s err %v", status, body, err)
	}
	if got.method != "POST" || got.body != `{"a":1}` || got.contentType != "application/json" {
		t.Errorf("sent %+v", got)
	}
}

func TestRequest_NonSuccessIsReturnedNotErrored(t *testing.T) {
	srv, _ := requestServer(t, 404, `{"message":"nope"}`)
	status, body, err := NewClient(srv.URL, "k", "").Request("GET", "work_packages/9", nil)
	if err != nil || status != 404 || string(body) != `{"message":"nope"}` {
		t.Errorf("status %d body %s err %v", status, body, err)
	}
}

func TestRedirectOffTheBaseHostDropsTheAPIKey(t *testing.T) {
	other, got := requestServer(t, 200, `{}`)
	base := httptest.NewServer(http.RedirectHandler(other.URL+"/api/v3/statuses", http.StatusFound))
	t.Cleanup(base.Close)
	c := NewClient(base.URL, "secret", "")

	if _, _, err := c.Request("GET", "statuses", nil); err != nil {
		t.Fatalf("Request: %v", err)
	}
	var v map[string]interface{}
	if err := c.Get("/statuses", &v); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.hits != 2 {
		t.Fatalf("expected both redirects to be followed, got %d hits", got.hits)
	}
	if got.user != "" || got.pass != "" {
		t.Errorf("the API key reached %s: user=%q pass=%q", other.URL, got.user, got.pass)
	}
}

func TestRedirectWithinTheBaseHostKeepsTheAPIKey(t *testing.T) {
	var user string
	mux := http.NewServeMux()
	mux.Handle("/api/v3/old", http.RedirectHandler("/api/v3/new", http.StatusFound))
	mux.HandleFunc("/api/v3/new", func(w http.ResponseWriter, r *http.Request) {
		user, _, _ = r.BasicAuth()
		_, _ = w.Write([]byte(`{}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if _, _, err := NewClient(srv.URL, "secret", "").Request("GET", "old", nil); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if user != "apikey" {
		t.Errorf("same-host redirect lost the API key, user=%q", user)
	}
}
