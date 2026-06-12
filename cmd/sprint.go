package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Sprint management commands",
}

var sprintPlanCmd = &cobra.Command{
	Use:        "plan",
	Short:      "Show backlog items available for sprint planning",
	Hidden:     true,
	Deprecated: "use 'op backlog' instead",
	RunE:       runSprintPlan,
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

var sprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sprints (versions) for the project",
	RunE:  runSprintList,
}

var sprintCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new sprint for the project",
	Long: `Create a new sprint (version) in OpenProject.

Examples:
  op sprint create "Sprint 2026-07-07" --start=2026-07-07
  op sprint create "Sprint 2026-07-07" --start=2026-07-07 --end=2026-07-20`,
	Args: cobra.ExactArgs(1),
	RunE: runSprintCreate,
}

func init() {
	rootCmd.AddCommand(sprintCmd)
	sprintCmd.AddCommand(sprintPlanCmd)
	sprintCmd.AddCommand(sprintAddCmd)
	sprintCmd.AddCommand(sprintProgressCmd)
	sprintCmd.AddCommand(sprintCloseCmd)
	sprintCmd.AddCommand(sprintListCmd)
	sprintCmd.AddCommand(sprintCreateCmd)

	sprintAddCmd.Flags().Int("points", 0, "Set story points when adding")
	sprintAddCmd.Flags().String("sprint", "", "Target sprint name (defaults to active)")
	sprintProgressCmd.Flags().BoolP("verbose", "v", false, "Show full report with item details")
	sprintCreateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD, required)")
	sprintCreateCmd.Flags().String("end", "", "End date (YYYY-MM-DD, defaults to start + 13 days)")
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
	warnTruncated(result.Total, len(result.Embedded.Elements))
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

	failures := 0
	for _, arg := range args {
		id, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Printf("Skipping invalid ID: %s\n", arg)
			failures++
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
			failures++
			continue
		}
		fmt.Printf("Added #%d %q to %s\n", wp.ID, wp.Subject, targetVersion.Name)
	}

	if failures > 0 {
		return fmt.Errorf("%d of %d item(s) could not be added", failures, len(args))
	}
	return nil
}

func runSprintProgress(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprint, vf, err := activeSprintFilter(project)
	if err != nil {
		return err
	}

	filters := []api.Filter{
		vf,
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return err
	}

	wps := result.Embedded.Elements

	// Verbose mode: full report with item details
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		display.SprintReport(wps, sprint.Name, sprint.StartDate, sprint.EndDate)
		return nil
	}

	// Compact summary
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

		switch {
		case display.IsCompleted(wp):
			doneCount++
			donePoints += pts
		case !strings.EqualFold(wp.Links.Status.Title, "new"):
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

func runSprintList(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	versions, err := client.ListVersions(project)
	if err != nil {
		return fmt.Errorf("listing versions: %w", err)
	}

	display.VersionTable(versions.Embedded.Elements)
	return nil
}

func runSprintCreate(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	start, _ := cmd.Flags().GetString("start")
	if start == "" {
		return fmt.Errorf("--start is required (YYYY-MM-DD)")
	}
	startTime, err := time.Parse("2006-01-02", start)
	if err != nil {
		return fmt.Errorf("invalid start date %q: use YYYY-MM-DD", start)
	}

	end, _ := cmd.Flags().GetString("end")
	if end == "" {
		end = startTime.AddDate(0, 0, 13).Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", end); err != nil {
		return fmt.Errorf("invalid end date %q: use YYYY-MM-DD", end)
	}

	req := &api.CreateVersionRequest{
		Name:      args[0],
		Status:    "open",
		StartDate: start,
		EndDate:   end,
		Links: map[string]api.Link{
			"definingProject": {Href: fmt.Sprintf("/api/v3/projects/%s", project)},
		},
	}

	v, err := client.CreateVersion(req)
	if err != nil {
		return err
	}

	fmt.Printf("Created sprint #%d %q (%s to %s)\n", v.ID, v.Name, v.StartDate, v.EndDate)
	return nil
}

func runSprintClose(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprint, vf, err := activeSprintFilter(project)
	if err != nil {
		return err
	}

	filters := []api.Filter{
		vf,
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return err
	}

	var done, notDone []api.WorkPackage
	for _, wp := range result.Embedded.Elements {
		if display.IsCompleted(wp) {
			done = append(done, wp)
		} else {
			notDone = append(notDone, wp)
		}
	}

	fmt.Printf("Sprint Close: %s\n", sprint.Name)
	fmt.Println(strings.Repeat("=", 40))

	fmt.Printf("\nCompleted (%d):\n", len(done))
	for _, wp := range done {
		fmt.Printf("  #%-6d %s%s\n", wp.ID, wp.Subject, display.FormatPoints(wp))
	}

	if len(notDone) > 0 {
		fmt.Printf("\nIncomplete - carry over (%d):\n", len(notDone))
		for _, wp := range notDone {
			fmt.Printf("  #%-6d %-12s %s%s\n", wp.ID, wp.Links.Status.Title, wp.Subject, display.FormatPoints(wp))
		}
		fmt.Printf("\nUse 'op sprint add <ids> --sprint=\"<next>\"' to carry over items.\n")
	}

	return nil
}
