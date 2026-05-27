package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a work package",
	Long: `Update an existing work package.

Examples:
  op update 123 --status=in-progress
  op update 123 --assignee=@david --points=5
  op update 123 --done=80`,
	Args: cobra.ExactArgs(1),
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
	updateCmd.Flags().String("sprint", "", "Move to sprint/version")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
	}

	resolver := api.NewResolver(client)
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

	// Sprint
	if sprintName, _ := cmd.Flags().GetString("sprint"); sprintName != "" {
		project, err := client.RequireProject()
		if err != nil {
			return err
		}
		versions, err := client.ListVersions(project)
		if err != nil {
			return fmt.Errorf("listing versions: %w", err)
		}
		found := false
		for _, v := range versions.Embedded.Elements {
			if v.Name == sprintName {
				req.Links["version"] = api.Link{Href: v.Links.Self.Href}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("sprint %q not found", sprintName)
		}
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified (use --status, --assignee, --points, etc.)")
	}

	// Remove empty links map to avoid sending it
	if len(req.Links) == 0 {
		req.Links = nil
	}

	wp, err := client.UpdateWorkPackage(id, req)
	if err != nil {
		return fmt.Errorf("updating work package: %w", err)
	}

	fmt.Printf("Updated #%d\n", wp.ID)
	display.WorkPackageDetail(wp)
	return nil
}
