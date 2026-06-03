package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

// autoDetectSample is how many of my most-recent open items to inspect when
// guessing which project+sprint to show.
const autoDetectSample = 100

// myBucket is one (project, sprint) group of my open items.
type myBucket struct {
	ProjectTitle string
	ProjectID    string // numeric id parsed from the project href, for -p
	SprintTitle  string // "(no sprint)" when the items have no version
	HasSprint    bool
	Items        []api.WorkPackage
	lastUpdated  string
}

// runMyAutoDetect handles `op my` when no project is set. It samples my recent
// open work across all projects, shows the project+sprint holding the most of
// it, and points the user at the commands for broader or specific views.
func runMyAutoDetect(cmd *cobra.Command) error {
	me, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	byAuthor, _ := cmd.Flags().GetBool("author")
	whose := "assignee"
	mine := "assigned to you"
	if byAuthor {
		whose = "author"
		mine = "created by you"
	}

	filters := []api.Filter{
		api.NewFilter(whose, "=", fmt.Sprintf("%d", me.ID)),
		api.NewFilter("status", "o", ""), // open = not in a closed/done state
	}
	result, err := client.ListAllWorkPackages(filters, `[["updatedAt","desc"]]`, autoDetectSample)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	items := result.Embedded.Elements
	if len(items) == 0 {
		fmt.Printf("No open work %s in any project.\n", mine)
		fmt.Println("Set a project with -p <id> or OP_PROJECT in ~/.oprc, or try 'op overview'.")
		return nil
	}

	top := pickTopBucket(items)

	fmt.Printf("No project set — showing the sprint with most of your open work (%s).\n\n", mine)
	sprintLabel := top.SprintTitle
	fmt.Printf("%s / %s — %d of %d sampled open item(s)\n", top.ProjectTitle, sprintLabel, len(top.Items), len(items))
	if result.Total > len(items) {
		fmt.Printf("(sampled your %d most recently updated; %d open in total)\n", len(items), result.Total)
	}
	fmt.Println()

	display.WorkPackageTable(top.Items)

	fmt.Println("\nSee more:")
	fmt.Println("  op overview                          all your open work across projects")
	if top.ProjectID != "" {
		if top.HasSprint {
			fmt.Printf("  op my -p %s --sprint %q   just this sprint\n", top.ProjectID, top.SprintTitle)
		}
		fmt.Printf("  op my -p %s --no-sprint              everything in this project\n", top.ProjectID)
	}
	fmt.Println("Tip: set `project:` in ~/.oprc to make a default project stick.")
	return nil
}

// pickTopBucket groups work packages by (project, sprint) and returns the group
// with the most items, tie-broken by most-recent activity then first-seen
// order. wps must be non-empty.
func pickTopBucket(wps []api.WorkPackage) myBucket {
	buckets := map[string]*myBucket{}
	var order []string

	for i := range wps {
		wp := &wps[i]
		ptitle := firstNonEmpty(wp.Links.Project.Title, wp.Links.Project.Href, "(unknown project)")
		stitle := wp.Links.Version.Title
		hasSprint := stitle != ""
		if !hasSprint {
			stitle = "(no sprint)"
		}

		key := ptitle + "\x00" + stitle
		b := buckets[key]
		if b == nil {
			b = &myBucket{
				ProjectTitle: ptitle,
				ProjectID:    lastPathSegment(wp.Links.Project.Href),
				SprintTitle:  stitle,
				HasSprint:    hasSprint,
			}
			buckets[key] = b
			order = append(order, key)
		}
		b.Items = append(b.Items, *wp)
		if wp.UpdatedAt > b.lastUpdated {
			b.lastUpdated = wp.UpdatedAt
		}
	}

	var best *myBucket
	for _, k := range order {
		b := buckets[k]
		switch {
		case best == nil:
			best = b
		case len(b.Items) > len(best.Items):
			best = b
		case len(b.Items) == len(best.Items) && b.lastUpdated > best.lastUpdated:
			best = b
		}
	}
	return *best
}

// lastPathSegment returns the trailing segment of a path/href (e.g. the numeric
// id from "/api/v3/projects/382"), or "" when there is none.
func lastPathSegment(href string) string {
	href = strings.TrimRight(href, "/")
	if href == "" {
		return ""
	}
	if i := strings.LastIndex(href, "/"); i >= 0 {
		return href[i+1:]
	}
	return href
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
