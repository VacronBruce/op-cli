package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate sprint report for stakeholders",
	Long: `Generate a text summary of the current sprint for sharing with stakeholders.

Examples:
  op report`,
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprint, err := client.FindActiveSprint(project)
	if err != nil {
		return err
	}

	filters := []api.Filter{
		api.NewFilter("version", "=", fmt.Sprintf("%d", sprint.ID)),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return err
	}

	wps := result.Embedded.Elements

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

	fmt.Printf("# Sprint Report: %s\n", sprint.Name)
	if sprint.StartDate != "" && sprint.EndDate != "" {
		fmt.Printf("Period: %s to %s\n", sprint.StartDate, sprint.EndDate)
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

	printSection("Completed", done)
	printSection("In Progress", inProgress)
	printSection("Blocked", blocked)
	printSection("Not Started", notStarted)

	return nil
}

func printSection(title string, wps []api.WorkPackage) {
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
