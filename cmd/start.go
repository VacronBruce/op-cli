package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start work on a ticket: branch + In Progress + assign to you",
	Long: `Start work on a work package.

Creates (or switches to) a git branch named wp-<id>-<slug> derived from the
ticket's subject, moves the ticket to In Progress, and assigns it to you.
Must be run inside a git repository.

Example:
  op start 12345   # → branch wp-12345-crash-on-save, ticket In Progress, assigned to you`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
	}

	// The command's purpose is to branch, so require a git repo up front —
	// before touching the ticket — so we never half-apply.
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not a git repository (run op start inside your repo)")
	}

	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	wp, err := client.GetWorkPackage(id)
	if err != nil {
		return fmt.Errorf("fetching work package: %w", err)
	}
	fmt.Printf("Starting #%d: %s\n", id, wp.Subject)

	// Create + checkout the branch (or switch to it if it already exists).
	branch := branchName(id, wp.Subject)
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

	// Move the ticket to In Progress and assign it to you.
	resolver := api.NewResolver(client, project)
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
	return nil
}

var branchSlugInvalid = regexp.MustCompile(`[^a-z0-9]+`)

// branchName derives a git branch of the form wp-<id>-<slug> from a work
// package id and subject. The subject is lowercased, non-alphanumeric runs
// collapse to single dashes, and the slug is capped to keep branch names sane.
func branchName(id int, subject string) string {
	const maxSlug = 50
	slug := branchSlugInvalid.ReplaceAllString(strings.ToLower(subject), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > maxSlug {
		slug = strings.Trim(slug[:maxSlug], "-")
	}
	if slug == "" {
		return fmt.Sprintf("wp-%d", id)
	}
	return fmt.Sprintf("wp-%d-%s", id, slug)
}

func runGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func branchExists(branch string) bool {
	return exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}
