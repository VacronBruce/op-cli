package display

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/chenhuijun/op-cli/pkg/check"
)

// CheckReport prints a single work package check report to stdout.
func CheckReport(report *check.Report) {
	fmt.Printf("#%d %s\n", report.WPID, report.Subject)
	fmt.Printf("  Type: %s | Score: %s\n\n", report.Type, report.Score())

	for _, r := range report.Results {
		label := levelLabel(r.Level)
		if r.Message != "" {
			fmt.Printf("  %s  %s (%s)\n", label, r.Name, r.Message)
		} else {
			fmt.Printf("  %s  %s\n", label, r.Name)
		}
	}
	fmt.Println()
}

// CheckSummary prints a batch summary table for multiple reports.
func CheckSummary(reports []check.Report, sprintName string) {
	if len(reports) == 0 {
		fmt.Println("No work packages to check.")
		return
	}

	if sprintName != "" {
		fmt.Printf("Sprint Readiness: %s\n", sprintName)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSCORE\tSUBJECT")
	fmt.Fprintln(w, "--\t----\t-----\t-------")

	totalPassed := 0
	totalChecks := 0
	fullyPassed := 0

	for _, r := range reports {
		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\n",
			r.WPID,
			r.Type,
			r.Score(),
			truncate(r.Subject, 50),
		)
		totalPassed += r.Passed()
		totalChecks += r.Total()
		if !r.HasFailures() {
			fullyPassed++
		}
	}
	w.Flush()

	avg := 0
	if totalChecks > 0 {
		avg = totalPassed * 100 / totalChecks
	}
	fmt.Printf("\nSummary: %d/%d fully pass | Average: %d%%\n", fullyPassed, len(reports), avg)
}

// CheckReportMarkdown returns the check report formatted as markdown,
// suitable for posting as a comment on a work package.
func CheckReportMarkdown(report *check.Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Readiness Check — %s passed\n\n", report.Score())
	fmt.Fprintln(&b, "| Status | Check |")
	fmt.Fprintln(&b, "|--------|-------|")

	for _, r := range report.Results {
		icon := levelIcon(r.Level)
		detail := r.Name
		if r.Message != "" {
			detail += " — " + r.Message
		}
		fmt.Fprintf(&b, "| %s | %s |\n", icon, detail)
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "*Run by `op check`*")
	return b.String()
}

func levelLabel(l check.Level) string {
	switch l {
	case check.Pass:
		return "PASS"
	case check.Warn:
		return "WARN"
	case check.Fail:
		return "FAIL"
	default:
		return "????"
	}
}

func levelIcon(l check.Level) string {
	switch l {
	case check.Pass:
		return "ok"
	case check.Warn:
		return "warn"
	case check.Fail:
		return "fail"
	default:
		return "?"
	}
}
