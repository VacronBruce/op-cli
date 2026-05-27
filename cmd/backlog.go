package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Show backlog items (not in any sprint)",
	Long: `List all open work packages not assigned to any sprint.

Examples:
  op backlog
  op backlog groom`,
	RunE: runBacklog,
}

var backlogGroomCmd = &cobra.Command{
	Use:   "groom",
	Short: "Show items needing grooming (unestimated, stale)",
	RunE:  runBacklogGroom,
}

func init() {
	rootCmd.AddCommand(backlogCmd)
	backlogCmd.AddCommand(backlogGroomCmd)
}

func runBacklog(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	filters := []api.Filter{
		api.NewFilter("version", "!*", ""),
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters,
		`[["priority","asc"],["createdAt","desc"]]`, 100)
	if err != nil {
		return fmt.Errorf("listing backlog: %w", err)
	}

	fmt.Printf("Backlog (%d items):\n", result.Total)
	display.WorkPackageTable(result.Embedded.Elements)
	return nil
}

func runBacklogGroom(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	filters := []api.Filter{
		api.NewFilter("version", "!*", ""),
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters, "", 100)
	if err != nil {
		return fmt.Errorf("listing backlog: %w", err)
	}

	var unestimated []api.WorkPackage
	for _, wp := range result.Embedded.Elements {
		if wp.StoryPoints == nil || *wp.StoryPoints == 0 {
			unestimated = append(unestimated, wp)
		}
	}

	if len(unestimated) > 0 {
		fmt.Printf("Unestimated items (%d):\n", len(unestimated))
		display.WorkPackageTable(unestimated)
	} else {
		fmt.Println("All backlog items have estimates!")
	}

	return nil
}
