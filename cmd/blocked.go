package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show blocked work packages in current sprint",
	Long: `List work packages that have a "blocked" status or blocker relations.

Examples:
  op blocked`,
	RunE: runBlocked,
}

func init() {
	rootCmd.AddCommand(blockedCmd)
}

func runBlocked(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprint, err := client.FindActiveSprint(project)
	if err != nil {
		return err
	}

	vf, err := api.VersionFilter(sprint, project)
	if err != nil {
		return err
	}

	// Get all items in the sprint
	filters := []api.Filter{
		vf,
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	// Filter for blocked items (status contains "block" or has blocker relations)
	var blocked []api.WorkPackage
	for _, wp := range result.Embedded.Elements {
		status := wp.Links.Status.Title
		if status == "Blocked" || status == "blocked" {
			blocked = append(blocked, wp)
		}
	}

	if len(blocked) == 0 {
		fmt.Printf("No blocked items in %s\n", sprint.Name)
		return nil
	}

	fmt.Printf("Blocked items in %s (%d):\n", sprint.Name, len(blocked))
	display.WorkPackageTable(blocked)
	return nil
}
