package display

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// Board prints work packages as a kanban board grouped by status.
func Board(wps []api.WorkPackage) {
	if len(wps) == 0 {
		fmt.Println("No work packages in current sprint.")
		return
	}

	// Group by status
	groups := make(map[string][]api.WorkPackage)
	for _, wp := range wps {
		status := wp.Links.Status.Title
		groups[status] = append(groups[status], wp)
	}

	// Common status order
	statusOrder := []string{
		"New", "In progress", "In review",
		"Resolved", "Closed", "Rejected",
	}

	// Add any statuses not in our predefined order
	seen := make(map[string]bool)
	for _, s := range statusOrder {
		seen[s] = true
	}
	for status := range groups {
		if !seen[status] {
			statusOrder = append(statusOrder, status)
		}
	}

	// Print each column
	for _, status := range statusOrder {
		items, ok := groups[status]
		if !ok {
			continue
		}

		header := fmt.Sprintf(" %s (%d) ", status, len(items))
		fmt.Println()
		fmt.Println(strings.Repeat("=", len(header)))
		fmt.Println(header)
		fmt.Println(strings.Repeat("=", len(header)))

		for _, wp := range items {
			assignee := wp.Links.Assignee.Title
			if assignee == "" {
				assignee = "unassigned"
			}
			fmt.Printf("  #%-5d %s%s\n", wp.ID, truncate(wp.Subject, 45), FormatPoints(wp))
			fmt.Printf("         @%s  %s\n", assignee, wp.Links.Priority.Title)
		}
	}

	// Summary
	total := len(wps)
	totalPoints := 0
	donePoints := 0
	doneCount := 0
	for _, wp := range wps {
		pts := 0
		if wp.StoryPoints != nil {
			pts = *wp.StoryPoints
		}
		totalPoints += pts
		if IsCompleted(wp) {
			doneCount++
			donePoints += pts
		}
	}

	fmt.Printf("\n--- %d/%d items done", doneCount, total)
	if totalPoints > 0 {
		fmt.Printf(" | %d/%d points", donePoints, totalPoints)
	}
	fmt.Println(" ---")
}
