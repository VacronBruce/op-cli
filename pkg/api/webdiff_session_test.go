package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOP mimics the parts of OpenProject's web UI that LoginWeb and
// DescriptionDiff touch: a CSRF-protected login form and a diff page that
// needs the session cookie.
func fakeOP(t *testing.T, password string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte(fixture(t, "login_page.html")))
			return
		}
		r.ParseForm()
		if r.Form.Get("authenticity_token") != "tok123" || r.Form.Get("username") != "ada" || r.Form.Get("password") != password {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("bad login"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sess", Value: "ok", Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.HandleFunc("/journals/42/diff/description", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sess"); err != nil || c.Value != "ok" {
			http.Redirect(w, r, "/login?back_url=/journals/42/diff/description", http.StatusFound)
			return
		}
		w.Write([]byte(fixture(t, "diff_first.html")))
	})
	mux.HandleFunc("/journals/43/diff/description", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>no diff here</html>"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("home")) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestWebSession_FetchesDescriptionDiffAfterLogin(t *testing.T) {
	srv := fakeOP(t, "s3cret")
	ws, err := LoginWeb(srv.URL, "ada", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	segs, err := ws.DescriptionDiff(42)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(After(segs), "Draft:") {
		t.Errorf("after = %q", After(segs))
	}
}

func TestLoginWeb_BadPasswordNamesUserNotPassword(t *testing.T) {
	srv := fakeOP(t, "s3cret")
	_, err := LoginWeb(srv.URL, "ada", "wrong-pw")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "web login failed") || !strings.Contains(msg, "ada") {
		t.Errorf("unclear error: %q", msg)
	}
	if strings.Contains(msg, "wrong-pw") {
		t.Errorf("error leaks the password: %q", msg)
	}
}

func TestWebSession_UnauthenticatedRedirectToLoginIsAnError(t *testing.T) {
	srv := fakeOP(t, "s3cret")
	ws := &WebSession{base: srv.URL, http: &http.Client{}}
	if _, err := ws.DescriptionDiff(42); err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("err = %v", err)
	}
}

func TestWebSession_PageWithoutDiffIsAnError(t *testing.T) {
	srv := fakeOP(t, "s3cret")
	ws, _ := LoginWeb(srv.URL, "ada", "s3cret")
	if _, err := ws.DescriptionDiff(43); err == nil {
		t.Fatal("expected error")
	}
}
