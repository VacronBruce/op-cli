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
	version, err := client.ResolveVersion(project, sprintName)
	if err != nil {
		return err
	}
	fmt.Printf("Sprint: %s\n", version.Name)

	filters := []api.Filter{
		api.NewFilter("version", "=", fmt.Sprintf("%d", version.ID)),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.Board(result.Embedded.Elements)
	return nil
}
