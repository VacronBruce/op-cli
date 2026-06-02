package display

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// Activities prints activity comments for a work package.
func Activities(ac *api.ActivityCollection) {
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
		fmt.Printf("    %s\n\n", a.Comment.Raw)
	}
}
