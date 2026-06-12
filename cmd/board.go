package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Show current sprint as a kanban board",
	Long: `Display work packages from the current sprint grouped by status.

Examples:
  op board
  op board --sprint="Sprint 24"
  op board --component=android                  (current sprint, android only)
  op board --component=ios --no-sprint          (all sprints, ios only)
  op board --component=android --no-sprint      (all sprints, android only)`,
	RunE: runBoard,
}

func init() {
	rootCmd.AddCommand(boardCmd)
	boardCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
	boardCmd.Flags().Bool("no-sprint", false, "Show all items without sprint filter")
	boardCmd.Flags().String("component", "", "Filter by component (android, ios, ott, engineering, analytics)")
	boardCmd.Flags().String("label", "", "Filter by label (team#appios, team#appandroid, etc.)")
	boardCmd.Flags().String("status", "", "Filter by status (e.g. blocked, in-progress, new)")

	registerCustomFieldCompletions(boardCmd, "component", "label")
}

func runBoard(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	noSprint, _ := cmd.Flags().GetBool("no-sprint")

	var filters []api.Filter

	// Sprint filter
	if !noSprint {
		sprintName, _ := cmd.Flags().GetString("sprint")
		version, vf, err := namedSprintFilter(project, sprintName)
		if err != nil {
			return err
		}
		fmt.Printf("Sprint: %s\n", version.Name)
		filters = append(filters, vf)
	}

	// Status filter (server-side, resolved like `op update --status` so
	// "in-progress"/"in-prog" match "In progress"). Resolving up front also
	// means the filter sees ALL matching items, not just the fetched page.
	statusFilter, _ := cmd.Flags().GetString("status")
	if statusFilter != "" {
		status, err := api.NewResolver(client, project).ResolveStatus(statusFilter)
		if err != nil {
			return fmt.Errorf("resolving status: %w", err)
		}
		filters = append(filters, api.NewFilter("status", "=", strconv.Itoa(status.ID)))
	}

	// Only show open items when querying across sprints — unless the user
	// asked for a specific status, which would AND to nothing for closed ones.
	if noSprint && statusFilter == "" {
		filters = append(filters, api.NewFilter("status", "o", ""))
	}

	// Component filter
	if component, _ := cmd.Flags().GetString("component"); component != "" {
		field, value, err := customFieldFilter("component", component)
		if err != nil {
			return err
		}
		filters = append(filters, api.NewFilter(field, "=", value))
	}

	// Label filter
	if label, _ := cmd.Flags().GetString("label"); label != "" {
		field, value, err := customFieldFilter("label", label)
		if err != nil {
			return err
		}
		filters = append(filters, api.NewFilter(field, "=", value))
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	warnTruncated(result.Total, len(result.Embedded.Elements))

	if noSprint {
		display.GroupBySprint(result.Embedded.Elements)
	} else {
		display.Board(result.Embedded.Elements)
	}
	return nil
}
