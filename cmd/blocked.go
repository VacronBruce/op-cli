package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show blocked work packages in current sprint",
	Long: `List open work packages in the current sprint whose status is "Blocked".

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

	sprint, vf, err := activeSprintFilter(project)
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

	// Filter for items whose status is "Blocked" (case-insensitive).
	var blocked []api.WorkPackage
	for _, wp := range result.Embedded.Elements {
		if strings.EqualFold(wp.Links.Status.Title, "blocked") {
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
