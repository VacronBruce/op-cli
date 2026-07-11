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
	if wp.Links.Author.Title != "" {
		fmt.Printf("  Author:     %s\n", wp.Links.Author.Title)
	}
	if len(wp.CreatedAt) >= 10 {
		fmt.Printf("  Created:    %s\n", wp.CreatedAt[:10])
	}
	fmt.Printf("  Assignee:   %s\n", assigneeName(*wp))
	if wp.JiraID != "" {
		fmt.Printf("  JIRA ID:    %s\n", wp.JiraID)
	}
	if wp.StoryPoints != nil {
		fmt.Printf("  Points:     %d\n", *wp.StoryPoints)
	}
	if work := api.FormatEstimate(wp.EstimatedTime); work != "" {
		fmt.Printf("  Work:       %s\n", work)
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
	// Release lives on customField50 as an array holding at most one release.
	if len(wp.Links.Release) > 0 && wp.Links.Release[0].Title != "" {
		fmt.Printf("  Release:    %s\n", wp.Links.Release[0].Title)
	}
	if wp.UserStory != nil && wp.UserStory.Raw != "" {
		fmt.Printf("  User Story:\n    %s\n", wp.UserStory.Raw)
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
			pts := FormatPoints(wp)
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

// GroupBySprint groups work packages by sprint/version and prints them.
func GroupBySprint(wps []api.WorkPackage) {
	if len(wps) == 0 {
		fmt.Println("No work packages found.")
		return
	}

	groups := make(map[string][]api.WorkPackage)
	var order []string

	for _, wp := range wps {
		sprint := wp.Links.Version.Title
		if sprint == "" {
			sprint = "(backlog)"
		}
		if _, seen := groups[sprint]; !seen {
			order = append(order, sprint)
		}
		groups[sprint] = append(groups[sprint], wp)
	}

	total := 0
	for _, sprint := range order {
		items := groups[sprint]
		total += len(items)
		fmt.Printf("\n%s (%d items)\n", sprint, len(items))
		fmt.Println(strings.Repeat("-", len(sprint)+15))
		for _, wp := range items {
			fmt.Printf("  #%-6d %-12s %-10s %-14s %-16s %s\n",
				wp.ID,
				wp.Links.Status.Title,
				wp.Links.Priority.Title,
				assigneeName(wp),
				truncate(wp.Links.Project.Title, 16),
				truncate(wp.Subject, 45),
			)
		}
	}
	fmt.Printf("\nTotal: %d items across %d sprints\n", total, len(order))
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

// FormatPoints renders a work package's story points as " [Npt]", or "" when
// unestimated, for inline list rendering.
func FormatPoints(wp api.WorkPackage) string {
	if wp.StoryPoints == nil {
		return ""
	}
	return fmt.Sprintf(" [%dpt]", *wp.StoryPoints)
}

// IsCompleted reports whether a work package's status counts as done for
// sprint accounting (closed, resolved, or done — case-insensitive).
func IsCompleted(wp api.WorkPackage) bool {
	status := strings.ToLower(wp.Links.Status.Title)
	return status == "closed" || status == "resolved" || status == "done"
}

// VersionTable prints versions (sprints or releases) as the shared
// ID/STATUS/START/END/NAME table, with "-" for missing dates.
func VersionTable(versions []api.Version) {
	fmt.Printf("%-6s  %-8s  %-12s  %-12s  %s\n", "ID", "STATUS", "START", "END", "NAME")
	fmt.Printf("%-6s  %-8s  %-12s  %-12s  %s\n", "--", "------", "-----", "---", "----")
	for _, v := range versions {
		start := v.StartDate
		if start == "" {
			start = "-"
		}
		end := v.EndDate
		if end == "" {
			end = "-"
		}
		fmt.Printf("%-6d  %-8s  %-12s  %-12s  %s\n", v.ID, v.Status, start, end, v.Name)
	}
}
