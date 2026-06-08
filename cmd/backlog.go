package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Show backlog items (not in any sprint)",
	Long: `List all open work packages not assigned to any sprint.

Examples:
  op backlog
  op backlog --unestimated
  op backlog --priority p0,p1
  op backlog --type bug --priority sev1,sev2`,
	RunE: runBacklog,
}

func init() {
	rootCmd.AddCommand(backlogCmd)
	backlogCmd.Flags().Bool("unestimated", false, "Show only unestimated items (no story points)")
	backlogCmd.Flags().StringSlice("priority", nil, "Filter by priority values (e.g. p0,p1,sev1,sev2)")
	backlogCmd.Flags().StringSlice("type", nil, "Filter by work package type (e.g. bug,task)")
}

func runBacklog(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	filters := []api.Filter{
		api.NewFilter("version", "!*", ""),
		api.NewFilter("status", "o", ""),
	}

	resolver := api.NewResolver(client, project)

	// Optional: filter by priority (resolves names to numeric IDs).
	priorities, _ := cmd.Flags().GetStringSlice("priority")
	if len(priorities) > 0 {
		ids := make([]string, 0, len(priorities))
		for _, name := range priorities {
			p, err := resolver.ResolvePriority(name)
			if err != nil {
				return fmt.Errorf("resolving priority %q: %w", name, err)
			}
			ids = append(ids, strconv.Itoa(p.ID))
		}
		filters = append(filters, api.NewFilter("priority", "=", ids...))
	}

	// Optional: filter by work package type (resolves names to numeric IDs).
	types, _ := cmd.Flags().GetStringSlice("type")
	if len(types) > 0 {
		ids := make([]string, 0, len(types))
		for _, name := range types {
			t, err := resolver.ResolveType(name)
			if err != nil {
				return fmt.Errorf("resolving type %q: %w", name, err)
			}
			ids = append(ids, strconv.Itoa(t.ID))
		}
		filters = append(filters, api.NewFilter("type", "=", ids...))
	}

	result, err := client.ListWorkPackages(project, filters,
		`[["priority","asc"],["createdAt","desc"]]`, 100)
	if err != nil {
		return fmt.Errorf("listing backlog: %w", err)
	}

	items := result.Embedded.Elements

	unestimated, _ := cmd.Flags().GetBool("unestimated")
	if unestimated {
		var filtered []api.WorkPackage
		for _, wp := range items {
			if wp.StoryPoints == nil || *wp.StoryPoints == 0 {
				filtered = append(filtered, wp)
			}
		}
		items = filtered
		if len(items) == 0 {
			fmt.Println("All backlog items have estimates!")
			return nil
		}
		fmt.Printf("Unestimated backlog items (%d):\n", len(items))
	} else {
		fmt.Printf("Backlog (%d items):\n", len(items))
	}

	display.WorkPackageTable(items)
	return nil
}
