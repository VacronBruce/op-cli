package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
	Use:   "my",
	Short: "Show my assigned work packages",
	Long: `List work packages assigned to you.

Examples:
  op my                              (current sprint)
  op my --sprint="App_05/19/2026"    (specific sprint)
  op my --all                        (include closed items)
  op my --no-sprint                  (all items, no sprint filter)`,
	RunE: runMy,
}

var myTeamCmd = &cobra.Command{
	Use:   "my-team",
	Short: "Show all team work packages grouped by person",
	Long: `List all work packages in the current sprint, grouped by assignee.

Examples:
  op my-team
  op my-team --sprint="Sprint 24"`,
	RunE: runMyTeam,
}

func init() {
	rootCmd.AddCommand(myCmd)
	rootCmd.AddCommand(myTeamCmd)
	myCmd.Flags().Bool("all", false, "Include closed items")
	myCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
	myCmd.Flags().Bool("no-sprint", false, "Show all items without sprint filter")
	myTeamCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
}

func runMy(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	me, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	filters := []api.Filter{
		api.NewFilter("assignee", "=", fmt.Sprintf("%d", me.ID)),
	}

	showAll, _ := cmd.Flags().GetBool("all")
	if !showAll {
		filters = append(filters, api.NewFilter("status", "o", ""))
	}

	// Sprint filter
	noSprint, _ := cmd.Flags().GetBool("no-sprint")
	if !noSprint {
		sprintName, _ := cmd.Flags().GetString("sprint")
		version, err := client.ResolveVersion(project, sprintName)
		if err != nil {
			return err
		}
		filters = append(filters, api.NewFilter("version", "=", fmt.Sprintf("%d", version.ID)))
		fmt.Printf("Sprint: %s\n", version.Name)
	}

	result, err := client.ListWorkPackages(project, filters,
		`[["priority","asc"],["updatedAt","desc"]]`, 100)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	fmt.Printf("My items (%d):\n", result.Total)
	display.WorkPackageTable(result.Embedded.Elements)
	return nil
}

func runMyTeam(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprintName, _ := cmd.Flags().GetString("sprint")
	version, err := client.ResolveVersion(project, sprintName)
	if err != nil {
		return err
	}
	fmt.Printf("Sprint: %s\n", version.Name)

	filters := []api.Filter{
		api.NewFilter("version", "=", fmt.Sprintf("%d", version.ID)),
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.GroupByAssignee(result.Embedded.Elements)
	return nil
}
