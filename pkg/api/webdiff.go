package api

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DiffKind says whether a stretch of description text was kept, added or removed.
type DiffKind int

const (
	Same DiffKind = iota
	Inserted
	Deleted
)

// DiffSegment is one run of text in OpenProject's word-based description diff.
type DiffSegment struct {
	Kind DiffKind
	Text string
}

var (
	diffLabelRe  = regexp.MustCompile(`<label class="sr-only">[^<]*</label>`)
	diffPieceRe  = regexp.MustCompile(`(?s)<(ins|del) class="diffmod">(.*?)</(?:ins|del)>|([^<]+|<br />)`)
	inlineSpaces = regexp.MustCompile(`[ \t]+`)
)

// ParseTextDiff parses the HTML of /journals/<id>/diff/description. It returns
// an error when the page has no text-diff block, which means the request got a
// login page or something else.
func ParseTextDiff(page string) ([]DiffSegment, error) {
	const open = `<div class="text-diff">`
	s := strings.Index(page, open)
	if s < 0 {
		return nil, errors.New("page has no text-diff block")
	}
	body := page[s+len(open):]
	if e := strings.Index(body, "\n</div>"); e >= 0 {
		body = body[:e]
	}
	body = strings.ReplaceAll(diffLabelRe.ReplaceAllString(body, ""), "\n", "")

	var segs []DiffSegment
	for _, m := range diffPieceRe.FindAllStringSubmatch(body, -1) {
		kind, text := Same, m[3]
		switch m[1] {
		case "ins":
			kind, text = Inserted, m[2]
		case "del":
			kind, text = Deleted, m[2]
		}
		text = html.UnescapeString(strings.ReplaceAll(text, "<br />", "\n"))
		if kind == Deleted {
			// OpenProject escapes deleted text twice.
			text = html.UnescapeString(text)
		}
		segs = append(segs, DiffSegment{Kind: kind, Text: text})
	}
	return segs, nil
}

// Before returns the whitespace-normalised description before the edit.
func Before(segs []DiffSegment) string { return joinSegments(segs, Deleted) }

// After returns the whitespace-normalised description after the edit.
func After(segs []DiffSegment) string { return joinSegments(segs, Inserted) }

func joinSegments(segs []DiffSegment, also DiffKind) string {
	var b strings.Builder
	for _, s := range segs {
		if s.Kind == Same || s.Kind == also {
			b.WriteString(s.Text)
		}
	}
	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(inlineSpaces.ReplaceAllString(l, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var journalDiffRe = regexp.MustCompile(`/journals/(\d+)/diff/description`)

// DescriptionJournal is an activity that changed the description. JournalID
// comes from the diff link, not from the activity id.
type DescriptionJournal struct {
	JournalID int
	Activity  Activity
}

// DescriptionJournals returns the activities that edited the description, oldest first.
func DescriptionJournals(ac *ActivityCollection) []DescriptionJournal {
	var out []DescriptionJournal
	for _, a := range ac.Embedded.Elements {
		for _, d := range a.Details {
			if m := journalDiffRe.FindStringSubmatch(d.Raw); m != nil {
				id, _ := strconv.Atoi(m[1])
				out = append(out, DescriptionJournal{JournalID: id, Activity: a})
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Activity.CreatedAt < out[j].Activity.CreatedAt })
	return out
}

// WebSession is a logged-in OpenProject browser session. The REST API cannot
// return old description text, so the diff pages are read through the web UI.
type WebSession struct {
	base string
	http *http.Client
}

var authTokenRe = regexp.MustCompile(`name="authenticity_token" value="([^"]+)"`)

// LoginWeb signs in to the OpenProject web UI. The password is never put in an error.
func LoginWeb(baseURL, user, password string) (*WebSession, error) {
	base := strings.TrimRight(baseURL, "/")
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	resp, err := c.Get(base + "/login")
	if err != nil {
		return nil, fmt.Errorf("web login failed for %s: %w", user, err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	m := authTokenRe.FindSubmatch(page)
	if m == nil {
		return nil, fmt.Errorf("web login failed for %s: no authenticity_token on the login page", user)
	}
	resp, err = c.PostForm(base+"/login", url.Values{
		"authenticity_token": {string(m[1])},
		"username":           {user},
		"password":           {password},
		"back_url":           {base + "/"},
	})
	if err != nil {
		return nil, fmt.Errorf("web login failed for %s: %w", user, err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("web login failed for %s: HTTP %d (check the web password)", user, resp.StatusCode)
	}
	return &WebSession{base: base, http: c}, nil
}

// DescriptionDiff fetches and parses one journal's description diff.
func (w *WebSession) DescriptionDiff(journalID int) ([]DiffSegment, error) {
	resp, err := w.http.Get(fmt.Sprintf("%s/journals/%d/diff/description", w.base, journalID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.Request.URL.Path == "/login" {
		return nil, fmt.Errorf("journal %d: web session is not logged in (redirected to /login)", journalID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("journal %d: HTTP %d", journalID, resp.StatusCode)
	}
	page, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	segs, err := ParseTextDiff(string(page))
	if err != nil {
		return nil, fmt.Errorf("journal %d: %w (not logged in?)", journalID, err)
	}
	return segs, nil
}

// WordDiffLines renders the changed lines of a diff in `git diff --word-diff=plain`
// form, e.g. "See [-old-]{+new+}.". Lines with no change are left out.
func WordDiffLines(segs []DiffSegment) []string {
	var lines []string
	var cur strings.Builder
	changed := false
	flush := func() {
		if changed {
			lines = append(lines, strings.TrimSpace(inlineSpaces.ReplaceAllString(cur.String(), " ")))
		}
		cur.Reset()
		changed = false
	}
	for _, s := range segs {
		for i, piece := range strings.Split(s.Text, "\n") {
			if i > 0 {
				flush()
			}
			if s.Kind == Same || strings.TrimSpace(piece) == "" {
				cur.WriteString(piece)
				continue
			}
			core := strings.TrimSpace(piece)
			lead := piece[:strings.Index(piece, core)]
			trail := piece[len(lead)+len(core):]
			open, closing := "{+", "+}"
			if s.Kind == Deleted {
				open, closing = "[-", "-]"
			}
			cur.WriteString(lead + open + core + closing + trail)
			changed = true
		}
	}
	flush()
	return lines
}
