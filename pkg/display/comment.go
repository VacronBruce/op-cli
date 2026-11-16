package display

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// inlineImgRe matches a full inline image reference in a comment body — either an
// HTML <img ... /attachments/<id>/content ...> tag or a markdown
// ![alt](/api/v3/attachments/<id>/content) link — so the whole thing can be
// replaced with a readable marker.
var inlineImgRe = regexp.MustCompile(`<img[^>]*?/attachments/\d+/content[^>]*>|!\[[^\]]*\]\([^)]*?/attachments/\d+/content[^)]*\)`)

// attIDRe pulls the numeric attachment ID out of a matched inline image reference.
var attIDRe = regexp.MustCompile(`/attachments/(\d+)/content`)

// Activities prints activity comments for a work package. Inline image tags in
// the comment body are replaced with readable "[image #ID: filename]" markers;
// names maps attachment ID to filename (the filename is omitted when unknown).
func Activities(ac *api.ActivityCollection, names map[int]string) {
	// Filter to only activities that have a comment
	var comments []api.Activity
	for _, a := range ac.Embedded.Elements {
		if a.Comment != nil && a.Comment.Raw != "" {
			comments = append(comments, a)
		}
	}

	if len(comments) == 0 {
		fmt.Println("No comments.")
		return
	}

	fmt.Printf("Comments (%d):\n\n", len(comments))
	for _, a := range comments {
		date := ""
		if len(a.CreatedAt) >= 10 {
			date = a.CreatedAt[:10]
		}
		fmt.Printf("  [%s] %s (#%d):\n", date, a.Links.User.Title, a.ID)
		fmt.Printf("    %s\n\n", renderCommentBody(a.Comment.Raw, names))
	}
}

// renderCommentBody replaces inline image references with readable markers.
func renderCommentBody(raw string, names map[int]string) string {
	return inlineImgRe.ReplaceAllStringFunc(raw, func(match string) string {
		m := attIDRe.FindStringSubmatch(match)
		if m == nil {
			return match
		}
		id, _ := strconv.Atoi(m[1])
		if name := names[id]; name != "" {
			return fmt.Sprintf("[image #%d: %s]", id, name)
		}
		return fmt.Sprintf("[image #%d]", id)
	})
}

// ActivityLog prints every activity oldest first: a header with time, author
// and activity ID, each field change as a "- " line, then the comment body.
func ActivityLog(ac *api.ActivityCollection, names map[int]string) {
	els := append([]api.Activity(nil), ac.Embedded.Elements...)
	if len(els) == 0 {
		fmt.Println("No activity.")
		return
	}
	sort.SliceStable(els, func(i, j int) bool { return els[i].CreatedAt < els[j].CreatedAt })

	for i, a := range els {
		if i > 0 {
			fmt.Println()
		}
		when := a.CreatedAt
		if t, err := time.Parse(time.RFC3339, a.CreatedAt); err == nil {
			when = t.UTC().Format("2006-01-02 15:04 UTC")
		}
		fmt.Printf("[%s] %s (#%d)\n", when, a.Links.User.Title, a.ID)
		for _, d := range a.Details {
			fmt.Printf("  - %s\n", d.Raw)
		}
		if a.Comment != nil && a.Comment.Raw != "" {
			for _, line := range strings.Split(renderCommentBody(a.Comment.Raw, names), "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
	}
}
