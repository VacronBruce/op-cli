package display

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// SprintReport prints a verbose sprint report with progress bar and items by status.
func SprintReport(wps []api.WorkPackage, sprintName, startDate, endDate string) {
	var done, inProgress, notStarted, blocked []api.WorkPackage
	totalPoints := 0
	donePoints := 0

	for _, wp := range wps {
		pts := 0
		if wp.StoryPoints != nil {
			pts = *wp.StoryPoints
		}
		totalPoints += pts

		status := strings.ToLower(wp.Links.Status.Title)
		switch {
		case status == "closed" || status == "resolved" || status == "done":
			done = append(done, wp)
			donePoints += pts
		case status == "blocked":
			blocked = append(blocked, wp)
		case status == "new":
			notStarted = append(notStarted, wp)
		default:
			inProgress = append(inProgress, wp)
		}
	}

	fmt.Printf("# Sprint Report: %s\n", sprintName)
	if startDate != "" && endDate != "" {
		fmt.Printf("Period: %s to %s\n", startDate, endDate)
	}
	fmt.Println()

	// Progress bar
	if totalPoints > 0 {
		pct := float64(donePoints) / float64(totalPoints) * 100
		barLen := 30
		filled := int(pct / 100 * float64(barLen))
		bar := strings.Repeat("#", filled) + strings.Repeat("-", barLen-filled)
		fmt.Printf("Progress: [%s] %.0f%% (%d/%d points)\n\n", bar, pct, donePoints, totalPoints)
	}

	reportSection("Completed", done)
	reportSection("In Progress", inProgress)
	reportSection("Blocked", blocked)
	reportSection("Not Started", notStarted)
}

func reportSection(title string, wps []api.WorkPackage) {
	if len(wps) == 0 {
		return
	}
	fmt.Printf("## %s (%d)\n", title, len(wps))
	for _, wp := range wps {
		assignee := wp.Links.Assignee.Title
		if assignee == "" {
			assignee = "unassigned"
		}
		pts := ""
		if wp.StoryPoints != nil {
			pts = fmt.Sprintf(" [%dpt]", *wp.StoryPoints)
		}
		fmt.Printf("- #%d %s%s (@%s)\n", wp.ID, wp.Subject, pts, assignee)
	}
	fmt.Println()
}
