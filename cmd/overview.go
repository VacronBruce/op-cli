package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Cross-project dashboard of my open work",
	Long: `Summarize your open work across all projects: the most recently active
projects, each with their most recently active sprints and open/blocked counts.

Unlike 'op my', this is not scoped to one project, so it needs no -p. Use
'op my -p <project>' to drill into a single project's items.

Examples:
  op overview
  op overview --projects=8 --sprints=5`,
	Args: cobra.NoArgs,
	RunE: runOverview,
}

func init() {
	rootCmd.AddCommand(overviewCmd)
	overviewCmd.Flags().Int("projects", 5, "Max projects to show")
	overviewCmd.Flags().Int("sprints", 3, "Max sprints per project")
}

func runOverview(cmd *cobra.Command, args []string) error {
	projectsN, _ := cmd.Flags().GetInt("projects")
	sprintsN, _ := cmd.Flags().GetInt("sprints")
	if projectsN < 1 || sprintsN < 1 {
		return fmt.Errorf("--projects and --sprints must be >= 1")
	}

	me, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	// My open work across every project, freshest first — one global query.
	filters := []api.Filter{
		api.NewFilter("assignee", "=", fmt.Sprintf("%d", me.ID)),
		api.NewFilter("status", "o", ""),
	}
	const pageSize = 200
	result, err := client.ListAllWorkPackages(filters, `[["updatedAt","desc"]]`, pageSize)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.Overview(result.Embedded.Elements, projectsN, sprintsN, result.Total, len(result.Embedded.Elements))
	return nil
}
