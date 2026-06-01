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
  op backlog --unestimated`,
	RunE: runBacklog,
}

func init() {
	rootCmd.AddCommand(backlogCmd)
	backlogCmd.Flags().Bool("unestimated", false, "Show only unestimated items (no story points)")
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

	items := result.Embedded.Elements

	unestimated, _ := cmd.Flags().GetBool("unestimated")
	if unestimated {
		var filtered []api.WorkPackage
		for _, wp := range items {
			if wp.StoryPoints == nil || *wp.StoryPoints == 0 {
				filtered = append(filtered, wp)
			}
		}
		items = filtered
		if len(items) == 0 {
			fmt.Println("All backlog items have estimates!")
			return nil
		}
		fmt.Printf("Unestimated backlog items (%d):\n", len(items))
	} else {
		fmt.Printf("Backlog (%d items):\n", len(items))
	}

	display.WorkPackageTable(items)
	return nil
}
