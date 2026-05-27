package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <type> <subject>",
	Short: "Create a work package (task, bug, feature, etc.)",
	Long: `Create a new work package in the project.

Examples:
  op create task "Fix login page"
  op create bug "Crash on save" --assignee=@david --priority=high
  op create feature "Dark mode" --points=8 --sprint="Sprint 24"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("assignee", "a", "", "Assignee (user name or @login)")
	createCmd.Flags().String("priority", "Normal", "Priority (Low, Normal, High, Immediate)")
	createCmd.Flags().StringP("description", "d", "", "Description (markdown)")
	createCmd.Flags().Int("points", 0, "Story points")
	createCmd.Flags().String("sprint", "", "Sprint/version name")
	createCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	createCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	createCmd.Flags().String("parent", "", "Parent work package ID")
}

func runCreate(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	typeName := args[0]
	subject := args[1]

	resolver := api.NewResolver(client)

	// Resolve type
	wpType, err := resolver.ResolveType(typeName)
	if err != nil {
		return fmt.Errorf("resolving type: %w", err)
	}

	// Resolve priority
	priorityName, _ := cmd.Flags().GetString("priority")
	priority, err := resolver.ResolvePriority(priorityName)
	if err != nil {
		return fmt.Errorf("resolving priority: %w", err)
	}

	// Build request
	req := &api.CreateWPRequest{
		Subject: subject,
		Links: map[string]api.Link{
			"type":     {Href: wpType.Href},
			"priority": {Href: priority.Href},
		},
	}

	// Optional: description
	if desc, _ := cmd.Flags().GetString("description"); desc != "" {
		req.Description = &api.Formattable{Format: "markdown", Raw: desc}
	}

	// Optional: story points
	if pts, _ := cmd.Flags().GetInt("points"); pts > 0 {
		req.StoryPoints = &pts
	}

	// Optional: dates
	if start, _ := cmd.Flags().GetString("start"); start != "" {
		req.StartDate = start
	}
	if due, _ := cmd.Flags().GetString("due"); due != "" {
		req.DueDate = due
	}

	// Optional: assignee
	if assignee, _ := cmd.Flags().GetString("assignee"); assignee != "" {
		user, err := resolver.ResolveUser(assignee)
		if err != nil {
			return fmt.Errorf("resolving assignee: %w", err)
		}
		req.Links["assignee"] = api.Link{Href: user.Href}
	}

	// Optional: sprint/version
	if sprintName, _ := cmd.Flags().GetString("sprint"); sprintName != "" {
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
	}

	// Create
	wp, err := client.CreateWorkPackage(project, req)
	if err != nil {
		return fmt.Errorf("creating work package: %w", err)
	}

	fmt.Printf("Created #%d\n", wp.ID)
	display.WorkPackageDetail(wp)
	return nil
}
