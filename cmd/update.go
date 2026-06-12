package cmd

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

// updateConcurrency bounds parallel bulk updates (each is a GET + PATCH pair)
// to keep load on the OpenProject server reasonable.
const updateConcurrency = 4

var updateCmd = &cobra.Command{
	Use:   "update <id> [<id>...]",
	Short: "Update one or more work packages",
	Long: `Update existing work packages. With several IDs the same change is
applied to each; failures are reported per ID and the command continues.

Examples:
  op update 123 --status=in-progress
  op update 123 --assignee=@david --points=5
  op update 123 --done=80
  op update 123 --to-project=wp        Move to another project
  op update 101 102 103 --status=done  Bulk status sweep`,
	Args: cobra.MinimumNArgs(1),
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("status", "s", "", "New status")
	updateCmd.Flags().StringP("assignee", "a", "", "New assignee")
	updateCmd.Flags().String("priority", "", "New priority")
	updateCmd.Flags().Int("points", 0, "Story points")
	updateCmd.Flags().Int("done", -1, "Percentage done (0-100)")
	updateCmd.Flags().String("subject", "", "New subject/title")
	updateCmd.Flags().StringP("description", "d", "", "New description (markdown)")
	updateCmd.Flags().String("sprint", "", "Move to sprint/version")
	updateCmd.Flags().String("to-project", "", "Move work package to another project (identifier)")
	updateCmd.Flags().String("release", "", "Set release (e.g. \"[iOS][ETV] 1.0.9\")")
	updateCmd.Flags().StringSlice("component", nil, "Component (android, ios, ott, engineering, analytics)")
	updateCmd.Flags().StringP("epic", "e", "", "Epic name (partial match)")
	updateCmd.Flags().String("parent", "", "Parent work package ID")
	updateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	updateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	updateCmd.Flags().StringSlice("product", nil, "Product (eet, entd, djy, cntd, gan_jing_world)")
	updateCmd.Flags().StringSlice("label", nil, "Label (team#appios, team#appandroid, ...)")

	registerCustomFieldCompletions(updateCmd, "component", "product", "label")
	_ = updateCmd.RegisterFlagCompletionFunc("release", completeRelease())
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Single-ID keeps its original contract: garbage fails fast with the
	// shared invalid-ID error before anything else.
	if len(args) == 1 {
		if _, err := parseWorkPackageID(args[0]); err != nil {
			return err
		}
	}

	project, err := client.RequireProject()
	if err != nil {
		return err
	}
	resolver := api.NewResolver(client, project)
	req := &api.UpdateWPRequest{
		Links: make(map[string]api.LinkValue),
	}

	hasChanges := false

	// Status
	if statusName, _ := cmd.Flags().GetString("status"); statusName != "" {
		status, err := resolver.ResolveStatus(statusName)
		if err != nil {
			return fmt.Errorf("resolving status: %w", err)
		}
		req.Links["status"] = api.Link{Href: status.Href}
		hasChanges = true
	}

	// Assignee
	if assignee, _ := cmd.Flags().GetString("assignee"); assignee != "" {
		user, err := resolver.ResolveUser(assignee)
		if err != nil {
			return fmt.Errorf("resolving assignee: %w", err)
		}
		req.Links["assignee"] = api.Link{Href: user.Href}
		hasChanges = true
	}

	// Priority
	if priorityName, _ := cmd.Flags().GetString("priority"); priorityName != "" {
		priority, err := resolver.ResolvePriority(priorityName)
		if err != nil {
			return fmt.Errorf("resolving priority: %w", err)
		}
		req.Links["priority"] = api.Link{Href: priority.Href}
		hasChanges = true
	}

	// Story points
	if pts, _ := cmd.Flags().GetInt("points"); pts > 0 {
		req.StoryPoints = &pts
		hasChanges = true
	}

	// Percentage done
	if done, _ := cmd.Flags().GetInt("done"); done >= 0 {
		req.PercentageDone = &done
		hasChanges = true
	}

	// Subject
	if subject, _ := cmd.Flags().GetString("subject"); subject != "" {
		req.Subject = subject
		hasChanges = true
	}

	// Description
	if desc, _ := cmd.Flags().GetString("description"); desc != "" {
		req.Description = &api.Formattable{Format: "markdown", Raw: desc}
		hasChanges = true
	}

	// Move to another project
	if toProject, _ := cmd.Flags().GetString("to-project"); toProject != "" {
		target, err := client.GetProject(toProject)
		if err != nil {
			return fmt.Errorf("resolving target project %q: %w", toProject, err)
		}
		req.Links["project"] = api.Link{Href: target.Links.Self.Href}
		hasChanges = true
	}

	// Sprint
	if sprintName, _ := cmd.Flags().GetString("sprint"); sprintName != "" {
		version, err := client.ResolveVersion(project, sprintName)
		if err != nil {
			return fmt.Errorf("resolving sprint: %w", err)
		}
		req.Links["version"] = api.Link{Href: version.Links.Self.Href}
		hasChanges = true
	}

	// Release (customField50 — version link scoped to kind=release)
	if releaseName, _ := cmd.Flags().GetString("release"); releaseName != "" {
		version, err := client.ResolveRelease(project, releaseName)
		if err != nil {
			return err
		}
		req.Links["customField50"] = api.Link{Href: version.Links.Self.Href}
		hasChanges = true
	}

	// Component (multi-value custom field; key/options from the registry)
	if components, _ := cmd.Flags().GetStringSlice("component"); len(components) > 0 {
		field, links, err := customFieldLinks("component", components)
		if err != nil {
			return err
		}
		req.Links[field] = links
		hasChanges = true
	}

	// Product / label (multi-value custom fields, same registry as create)
	for _, fieldName := range []string{"product", "label"} {
		if values, _ := cmd.Flags().GetStringSlice(fieldName); len(values) > 0 {
			field, links, err := customFieldLinks(fieldName, values)
			if err != nil {
				return err
			}
			req.Links[field] = links
			hasChanges = true
		}
	}

	// Epic (resolved by name within the project, same as create)
	if epicName, _ := cmd.Flags().GetString("epic"); epicName != "" {
		epic, err := resolver.ResolveEpic(epicName)
		if err != nil {
			return fmt.Errorf("resolving epic: %w", err)
		}
		req.Links["epic"] = api.Link{Href: epic.Href}
		hasChanges = true
	}

	// Parent
	if parentStr, _ := cmd.Flags().GetString("parent"); parentStr != "" {
		parentInt, err := strconv.Atoi(parentStr)
		if err != nil {
			return fmt.Errorf("invalid parent ID: %s", parentStr)
		}
		req.Links["parent"] = api.Link{Href: fmt.Sprintf("/api/v3/work_packages/%d", parentInt)}
		hasChanges = true
	}

	// Start / due dates
	if start, _ := cmd.Flags().GetString("start"); start != "" {
		req.StartDate = start
		hasChanges = true
	}
	if due, _ := cmd.Flags().GetString("due"); due != "" {
		req.DueDate = due
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified (use --status, --assignee, --points, --component, etc.)")
	}

	// Remove empty links map to avoid sending it
	if len(req.Links) == 0 {
		req.Links = nil
	}

	// Single-ID keeps its original contract: fail fast with the API error and
	// render the full detail view.
	if len(args) == 1 {
		wpID, _ := parseWorkPackageID(args[0]) // validated above
		wp, err := client.UpdateWorkPackage(wpID, req)
		if err != nil {
			return fmt.Errorf("updating work package: %w", err)
		}
		fmt.Printf("Updated #%d\n", wp.ID)
		fmt.Println(client.WorkPackageURL(wp.ID))
		display.WorkPackageDetail(wp)
		return nil
	}

	// Bulk: updates run concurrently (bounded) sharing one request —
	// UpdateWorkPackage never mutates it (contract-tested in pkg/api), and
	// each call fetches its own ticket's lockVersion. Results are collected
	// and printed in argument order so the report reads against the typed list.
	lines := make([]string, len(args))
	failed := make([]bool, len(args))
	sem := make(chan struct{}, updateConcurrency)
	var wg sync.WaitGroup
	for i, arg := range args {
		wpID, err := parseWorkPackageID(arg)
		if err != nil {
			lines[i] = fmt.Sprintf("Skipping invalid ID: %s", arg)
			failed[i] = true
			continue
		}
		wg.Add(1)
		go func(i, wpID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			wp, err := client.UpdateWorkPackage(wpID, req)
			if err != nil {
				lines[i] = fmt.Sprintf("Error updating #%d: %s", wpID, err)
				failed[i] = true
				return
			}
			lines[i] = fmt.Sprintf("Updated #%d %s  %s", wp.ID, wp.Subject, client.WorkPackageURL(wp.ID))
		}(i, wpID)
	}
	wg.Wait()

	failures := 0
	for i, line := range lines {
		fmt.Println(line)
		if failed[i] {
			failures++
		}
	}
	fmt.Printf("Updated %d work package(s)\n", len(args)-failures)
	if failures > 0 {
		return fmt.Errorf("%d of %d update(s) failed", failures, len(args))
	}
	return nil
}
