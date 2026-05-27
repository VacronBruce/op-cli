package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Sprint management commands",
}

var sprintPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show backlog items available for sprint planning",
	RunE:  runSprintPlan,
}

var sprintAddCmd = &cobra.Command{
	Use:   "add <id> [<id>...]",
	Short: "Add work packages to the current sprint",
	Long: `Move work packages into the active sprint.

Examples:
  op sprint add 101 102 103
  op sprint add 101 --points=5
  op sprint add 101 --sprint="Sprint 25"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSprintAdd,
}

var sprintProgressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show sprint progress summary",
	RunE:  runSprintProgress,
}

var sprintCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Show sprint summary for closing",
	RunE:  runSprintClose,
}

func init() {
	rootCmd.AddCommand(sprintCmd)
	sprintCmd.AddCommand(sprintPlanCmd)
	sprintCmd.AddCommand(sprintAddCmd)
	sprintCmd.AddCommand(sprintProgressCmd)
	sprintCmd.AddCommand(sprintCloseCmd)

	sprintAddCmd.Flags().Int("points", 0, "Set story points when adding")
	sprintAddCmd.Flags().String("sprint", "", "Target sprint name (defaults to active)")
}

func runSprintPlan(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	// Items NOT in any version, open status
	filters := []api.Filter{
		api.NewFilter("version", "!*", ""),
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters,
		`[["priority","asc"],["createdAt","desc"]]`, 50)
	if err != nil {
		return fmt.Errorf("listing backlog: %w", err)
	}

	fmt.Printf("Backlog items ready for sprint (%d):\n", result.Total)
	display.WorkPackageTable(result.Embedded.Elements)
	return nil
}

func runSprintAdd(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	// Find target sprint
	sprintName, _ := cmd.Flags().GetString("sprint")
	targetVersion, err := client.ResolveVersion(project, sprintName)
	if err != nil {
		return err
	}

	points, _ := cmd.Flags().GetInt("points")

	for _, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Printf("Skipping invalid ID: %s\n", arg)
			continue
		}

		req := &api.UpdateWPRequest{
			Links: map[string]api.LinkValue{
				"version": api.Link{Href: targetVersion.Links.Self.Href},
			},
		}
		if points > 0 {
			req.StoryPoints = &points
		}

		wp, err := client.UpdateWorkPackage(id, req)
		if err != nil {
			fmt.Printf("Error adding #%d: %s\n", id, err)
			continue
		}
		fmt.Printf("Added #%d %q to %s\n", wp.ID, wp.Subject, targetVersion.Name)
	}

	return nil
}

func runSprintProgress(cmd *cobra.Command, args []string) error {
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
	total := len(wps)
	totalPoints := 0
	doneCount := 0
	donePoints := 0
	inProgressCount := 0

	for _, wp := range wps {
		pts := 0
		if wp.StoryPoints != nil {
			pts = *wp.StoryPoints
		}
		totalPoints += pts

		status := strings.ToLower(wp.Links.Status.Title)
		switch {
		case status == "closed" || status == "resolved" || status == "done":
			doneCount++
			donePoints += pts
		case status != "new":
			inProgressCount++
		}
	}

	fmt.Printf("Sprint: %s", sprint.Name)
	if sprint.StartDate != "" && sprint.EndDate != "" {
		fmt.Printf(" (%s to %s)", sprint.StartDate, sprint.EndDate)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Items:    %d/%d done, %d in progress\n", doneCount, total, inProgressCount)
	if totalPoints > 0 {
		pct := float64(donePoints) / float64(totalPoints) * 100
		fmt.Printf("Points:   %d/%d (%.0f%%)\n", donePoints, totalPoints, pct)
	}

	return nil
}

func runSprintClose(cmd *cobra.Command, args []string) error {
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

	var done, notDone []api.WorkPackage
	for _, wp := range result.Embedded.Elements {
		status := strings.ToLower(wp.Links.Status.Title)
		if status == "closed" || status == "resolved" || status == "done" {
			done = append(done, wp)
		} else {
			notDone = append(notDone, wp)
		}
	}

	fmt.Printf("Sprint Close: %s\n", sprint.Name)
	fmt.Println(strings.Repeat("=", 40))

	fmt.Printf("\nCompleted (%d):\n", len(done))
	for _, wp := range done {
		pts := ""
		if wp.StoryPoints != nil {
			pts = fmt.Sprintf(" [%dpt]", *wp.StoryPoints)
		}
		fmt.Printf("  #%-6d %s%s\n", wp.ID, wp.Subject, pts)
	}

	if len(notDone) > 0 {
		fmt.Printf("\nIncomplete - carry over (%d):\n", len(notDone))
		for _, wp := range notDone {
			pts := ""
			if wp.StoryPoints != nil {
				pts = fmt.Sprintf(" [%dpt]", *wp.StoryPoints)
			}
			fmt.Printf("  #%-6d %-12s %s%s\n", wp.ID, wp.Links.Status.Title, wp.Subject, pts)
		}
		fmt.Printf("\nUse 'op sprint add <ids> --sprint=\"<next>\"' to carry over items.\n")
	}

	return nil
}
