package display

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// WorkPackageTable prints work packages as a formatted table.
func WorkPackageTable(wps []api.WorkPackage) {
	if len(wps) == 0 {
		fmt.Println("No work packages found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tPRIORITY\tASSIGNEE\tSUBJECT")
	fmt.Fprintln(w, "--\t----\t------\t--------\t--------\t-------")

	for _, wp := range wps {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			wp.ID,
			wp.Links.Type.Title,
			wp.Links.Status.Title,
			wp.Links.Priority.Title,
			assigneeName(wp),
			truncate(wp.Subject, 50),
		)
	}
	w.Flush()
}

// WorkPackageDetail prints a single work package with full details.
func WorkPackageDetail(wp *api.WorkPackage) {
	fmt.Printf("#%d %s\n", wp.ID, wp.Subject)
	fmt.Printf("  Type:       %s\n", wp.Links.Type.Title)
	fmt.Printf("  Status:     %s\n", wp.Links.Status.Title)
	fmt.Printf("  Priority:   %s\n", wp.Links.Priority.Title)
	fmt.Printf("  Assignee:   %s\n", assigneeName(*wp))
	if wp.StoryPoints != nil {
		fmt.Printf("  Points:     %d\n", *wp.StoryPoints)
	}
	fmt.Printf("  Progress:   %d%%\n", wp.PercentageDone)
	if wp.StartDate != "" {
		fmt.Printf("  Start:      %s\n", wp.StartDate)
	}
	if wp.DueDate != "" {
		fmt.Printf("  Due:        %s\n", wp.DueDate)
	}
	if wp.Links.Version.Title != "" {
		fmt.Printf("  Sprint:     %s\n", wp.Links.Version.Title)
	}
	if wp.Description != nil && wp.Description.Raw != "" {
		fmt.Printf("  Description:\n    %s\n", wp.Description.Raw)
	}
}

// GroupByAssignee groups work packages by assignee and prints them.
func GroupByAssignee(wps []api.WorkPackage) {
	if len(wps) == 0 {
		fmt.Println("No work packages found.")
		return
	}

	groups := make(map[string][]api.WorkPackage)
	var order []string

	for _, wp := range wps {
		name := assigneeName(wp)
		if _, seen := groups[name]; !seen {
			order = append(order, name)
		}
		groups[name] = append(groups[name], wp)
	}

	for _, name := range order {
		items := groups[name]
		fmt.Printf("\n%s (%d items)\n", name, len(items))
		fmt.Println(strings.Repeat("-", len(name)+15))
		for _, wp := range items {
			pts := ""
			if wp.StoryPoints != nil {
				pts = fmt.Sprintf(" [%dpt]", *wp.StoryPoints)
			}
			fmt.Printf("  #%-6d %-12s %-10s %s%s\n",
				wp.ID,
				wp.Links.Status.Title,
				wp.Links.Priority.Title,
				truncate(wp.Subject, 40),
				pts,
			)
		}
	}
}

func assigneeName(wp api.WorkPackage) string {
	if wp.Links.Assignee.Title != "" {
		return wp.Links.Assignee.Title
	}
	return "(unassigned)"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
