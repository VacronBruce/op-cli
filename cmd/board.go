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
  op board --sprint="Sprint 24"`,
	RunE: runBoard,
}

func init() {
	rootCmd.AddCommand(boardCmd)
	boardCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
}

func runBoard(cmd *cobra.Command, args []string) error {
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
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.Board(result.Embedded.Elements)
	return nil
}
