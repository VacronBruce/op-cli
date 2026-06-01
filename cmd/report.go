package cmd

import (
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:        "report",
	Short:      "Generate sprint report for stakeholders",
	Hidden:     true,
	Deprecated: "use 'op sprint progress --verbose' instead",
	Long: `Generate a text summary of the current sprint for sharing with stakeholders.

Examples:
  op report`,
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
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

	filters := []api.Filter{
		vf,
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return err
	}

	display.SprintReport(result.Embedded.Elements, sprint.Name, sprint.StartDate, sprint.EndDate)
	return nil
}
