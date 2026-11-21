package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/check"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start work on a ticket: branch + In Progress + assign to you",
	Long: `Start work on a work package.

Creates (or switches to) a git branch named <project>-<id>-<slug> derived from
the ticket's project identifier and subject, moves the ticket to In Progress,
and assigns it to you. Must be run inside a git repository.

Before changing anything it runs the readiness check and prints the report.
With --require-ready, a ticket with any FAIL (or a check that cannot run) is
refused and nothing is written. WARNs never block.

If start_hook is set (~/.oprc or OP_START_HOOK), it is run with sh -c after a
successful start and receives a JSON ticket_start event on stdin. A failing
hook only prints a warning.

Examples:
  op start 12345                  # → branch app-12345-crash-on-save, ticket In Progress, assigned to you
  op start 12345 --require-ready  # refuse unless the readiness check has no FAIL`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().Bool("require-ready", false, "Refuse to start (no writes) if the readiness check has any FAIL or cannot run")
}

func runStart(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	// The command's purpose is to branch, so require a git repo up front —
	// before touching the ticket — so we never half-apply.
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not a git repository (run op start inside your repo)")
	}

	// A work-package id is global, so op start needs no default project — it
	// reads the ticket's own project from the fetched record.
	wp, err := client.GetWorkPackage(id)
	if err != nil {
		return fmt.Errorf("fetching work package: %w", err)
	}
	fmt.Printf("Starting #%d (%s): %s\n", id, wp.Links.Project.Title, wp.Subject)

	requireReady, _ := cmd.Flags().GetBool("require-ready")
	report, checkErr := startReadiness(id)
	if checkErr != nil {
		fmt.Fprintf(os.Stderr, "warning: readiness check could not run: %v\n", checkErr)
	} else {
		display.CheckReport(report)
	}
	if requireReady {
		if checkErr != nil {
			return fmt.Errorf("not starting #%d: readiness check could not run (--require-ready)", id)
		}
		if report.HasFailures() {
			return fmt.Errorf("not starting #%d: failing readiness checks (--require-ready); fix them or start without the flag", id)
		}
	}

	// Resolve the ticket's project identifier (e.g. "app") for the branch
	// prefix. Done before any git/ticket write so a failure aborts cleanly.
	proj, err := client.GetProject(lastPathSegment(wp.Links.Project.Href))
	if err != nil {
		return fmt.Errorf("resolving project for branch name: %w", err)
	}

	// Create + checkout the branch (or switch to it if it already exists).
	branch := branchName(proj.Identifier, id, wp.Subject)
	if branchExists(branch) {
		if out, err := runGit("checkout", branch); err != nil {
			return fmt.Errorf("switching to existing branch %s: %v: %s", branch, err, out)
		}
		fmt.Printf("Switched to existing branch %s\n", branch)
	} else {
		if out, err := runGit("checkout", "-b", branch); err != nil {
			return fmt.Errorf("creating branch %s: %v: %s", branch, err, out)
		}
		fmt.Printf("Created and switched to branch %s\n", branch)
	}

	// Move the ticket to In Progress and assign it to you. Status resolution
	// uses the global /statuses endpoint, so no project context is needed.
	resolver := api.NewResolver(client, "")
	req := &api.UpdateWPRequest{Links: make(map[string]api.LinkValue)}

	status, err := resolver.ResolveStatus("in-progress")
	if err != nil {
		return fmt.Errorf("resolving status: %w", err)
	}
	req.Links["status"] = api.Link{Href: status.Href}

	me, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("fetching current user: %w", err)
	}
	req.Links["assignee"] = api.Link{Href: me.Links.Self.Href}

	if _, err := client.UpdateWorkPackage(id, req); err != nil {
		return fmt.Errorf("updating work package: %w", err)
	}

	fmt.Printf("#%d → In Progress, assigned to %s\n", id, me.Name)

	runStartHook(newStartEvent(id, wp.Subject, wp.Links.Type.Title, proj.Identifier, branch, report, checkErr))
	return nil
}

func startReadiness(id int) (*check.Report, error) {
	dor, err := check.LoadDoR()
	if err != nil {
		return nil, err
	}
	return (&check.Runner{Client: client, Config: dor}).Run(id)
}

var branchSlugInvalid = regexp.MustCompile(`[^a-z0-9]+`)

// branchName derives a git branch of the form <project>-<id>-<slug> from a
// work package's project identifier, id, and subject. Both project and subject
// are lowercased with non-alphanumeric runs collapsed to single dashes; the
// subject slug is capped to keep branch names sane. If the project identifier
// is empty it falls back to "wp" so the branch is still valid.
func branchName(project string, id int, subject string) string {
	const maxSlug = 50
	slug := branchSlugInvalid.ReplaceAllString(strings.ToLower(subject), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > maxSlug {
		slug = strings.Trim(slug[:maxSlug], "-")
	}

	prefix := strings.Trim(branchSlugInvalid.ReplaceAllString(strings.ToLower(project), "-"), "-")
	if prefix == "" {
		prefix = "wp"
	}

	if slug == "" {
		return fmt.Sprintf("%s-%d", prefix, id)
	}
	return fmt.Sprintf("%s-%d-%s", prefix, id, slug)
}

func runGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func branchExists(branch string) bool {
	return exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}
