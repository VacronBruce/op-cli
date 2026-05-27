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
  op my
  op my --all   (include closed items)`,
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
		// Exclude closed statuses
		filters = append(filters, api.NewFilter("status", "o", ""))
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

	var versionID string
	if sprintName != "" {
		versions, err := client.ListVersions(project)
		if err != nil {
			return fmt.Errorf("listing versions: %w", err)
		}
		for _, v := range versions.Embedded.Elements {
			if v.Name == sprintName {
				versionID = fmt.Sprintf("%d", v.ID)
				break
			}
		}
		if versionID == "" {
			return fmt.Errorf("sprint %q not found", sprintName)
		}
	} else {
		sprint, err := client.FindActiveSprint(project)
		if err != nil {
			return err
		}
		versionID = fmt.Sprintf("%d", sprint.ID)
		fmt.Printf("Sprint: %s\n", sprint.Name)
	}

	filters := []api.Filter{
		api.NewFilter("version", "=", versionID),
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.GroupByAssignee(result.Embedded.Elements)
	return nil
}
