package display

import (
	"fmt"
	"regexp"
	"strconv"

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
