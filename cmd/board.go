package cmd

import (
	"fmt"

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
		version, err := client.ResolveVersion(project, sprintName)
		if err != nil {
			return err
		}
		fmt.Printf("Sprint: %s\n", version.Name)
		filters = append(filters, api.NewFilter("version", "=", fmt.Sprintf("%d", version.ID)))
	}

	// Only show open items when querying across sprints
	if noSprint {
		filters = append(filters, api.NewFilter("status", "o", ""))
	}

	// Component filter (customField12)
	if component, _ := cmd.Flags().GetString("component"); component != "" {
		optionID, err := api.OptionID(api.ComponentOptions, component)
		if err != nil {
			return fmt.Errorf("resolving component: %w", err)
		}
		filters = append(filters, api.NewFilter("customField12", "=", optionID))
	}

	// Label filter (customField13)
	if label, _ := cmd.Flags().GetString("label"); label != "" {
		optionID, err := api.OptionID(api.LabelOptions, label)
		if err != nil {
			return fmt.Errorf("resolving label: %w", err)
		}
		filters = append(filters, api.NewFilter("customField13", "=", optionID))
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	if noSprint {
		display.GroupBySprint(result.Embedded.Elements)
	} else {
		display.Board(result.Embedded.Elements)
	}
	return nil
}
