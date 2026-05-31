package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/check"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [id]",
	Short: "Check work package readiness",
	Long: `Run quality checks against a work package and report readiness.

Examples:
  op check 81321
  op check 81321 --strict
  op check --sprint
  op check --sprint --component=android
  op check 81321 --comment`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("sprint", false, "Check all tickets in current sprint")
	checkCmd.Flags().Bool("strict", false, "Treat WARN as FAIL")
	checkCmd.Flags().Bool("comment", false, "Post results as comment on ticket")
	checkCmd.Flags().String("component", "", "Filter by component (for --sprint)")
}

func runCheck(cmd *cobra.Command, args []string) error {
	sprint, _ := cmd.Flags().GetBool("sprint")
	strict, _ := cmd.Flags().GetBool("strict")
	comment, _ := cmd.Flags().GetBool("comment")

	runner := &check.Runner{Client: client}

	if sprint {
		return runCheckSprint(cmd, runner, strict, comment)
	}

	if len(args) == 0 {
		return fmt.Errorf("provide a work package ID or use --sprint")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
	}

	report, err := runner.Run(id)
	if err != nil {
		return fmt.Errorf("checking work package: %w", err)
	}

	if strict {
		promoteWarnings(report)
	}

	display.CheckReport(report)

	if comment {
		md := display.CheckReportMarkdown(report)
		if err := client.PostComment(report.WPID, md); err != nil {
			return fmt.Errorf("posting comment: %w", err)
		}
		fmt.Printf("Posted check results as comment on #%d\n", report.WPID)
	}

	return nil
}

func runCheckSprint(cmd *cobra.Command, runner *check.Runner, strict, comment bool) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	activeSprint, err := client.FindActiveSprint(project)
	if err != nil {
		return fmt.Errorf("finding active sprint: %w", err)
	}

	filters := []api.Filter{
		api.NewFilter("version", "=", fmt.Sprintf("%d", activeSprint.ID)),
	}

	// Component filter
	if component, _ := cmd.Flags().GetString("component"); component != "" {
		optionID, err := api.OptionID(api.ComponentOptions, component)
		if err != nil {
			return fmt.Errorf("resolving component: %w", err)
		}
		filters = append(filters, api.NewFilter("customField12", "=", optionID))
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	wps := result.Embedded.Elements
	reports, err := runner.RunBatch(wps)
	if err != nil {
		return err
	}

	if strict {
		for i := range reports {
			promoteWarnings(&reports[i])
		}
	}

	display.CheckSummary(reports, activeSprint.Name)

	if comment {
		for _, r := range reports {
			md := display.CheckReportMarkdown(&r)
			if err := client.PostComment(r.WPID, md); err != nil {
				fmt.Printf("  Warning: failed to post comment on #%d: %s\n", r.WPID, err)
				continue
			}
			fmt.Printf("  Posted check results on #%d\n", r.WPID)
		}
	}

	return nil
}

// promoteWarnings changes all WARN results to FAIL in a report.
func promoteWarnings(report *check.Report) {
	for i := range report.Results {
		if report.Results[i].Level == check.Warn {
			report.Results[i].Level = check.Fail
		}
	}
}
