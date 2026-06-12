package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
	Use:   "my",
	Short: "Show my assigned work packages",
	Long: `List work packages assigned to you, or created by you with --author.

By default op my shows all your open work across every sprint. Pass --sprint
to scope to one sprint.

With no project set (no -p flag and no OP_PROJECT), op my auto-detects the
project + sprint where most of your recent open work lives, and points you at
'op overview' for the cross-project view.

Examples:
  op my                              (all my open work, any sprint)
  op my --sprint="App_05/19/2026"    (scope to one sprint)
  op my --by-sprint                  (all my open work, grouped by sprint)
  op my --all                        (include closed items)
  op my --author                     (created by me)
  op my --author --since=2w          (created by me in last 2 weeks)
  op my --author --since=30d         (created by me in last 30 days)`,
	RunE: runMy,
}

var myTeamCmd = &cobra.Command{
	Use:   "team",
	Short: "Show all team work packages grouped by person",
	Long: `List all work packages in the current sprint, grouped by assignee.

Examples:
  op my team
  op my team --sprint="Sprint 24"`,
	RunE: runMyTeam,
}

var myTeamAliasCmd = &cobra.Command{
	Use:        "my-team",
	Short:      "Show all team work packages grouped by person",
	Hidden:     true,
	Deprecated: "use 'op my team' instead",
	RunE:       runMyTeam,
}

func init() {
	rootCmd.AddCommand(myCmd)
	myCmd.AddCommand(myTeamCmd)
	rootCmd.AddCommand(myTeamAliasCmd)
	myCmd.Flags().Bool("all", false, "Include closed items")
	myCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
	myCmd.Flags().Bool("no-sprint", false, "Deprecated: no-sprint is now the default; this flag is a no-op")
	_ = myCmd.Flags().MarkHidden("no-sprint")
	myCmd.Flags().Bool("author", false, "Filter by author (created by me) instead of assignee")
	myCmd.Flags().String("since", "", "Filter by creation date (e.g. 2w, 30d, 3m)")
	myCmd.Flags().String("component", "", "Filter by component (android, ios, ott, engineering, analytics)")
	myCmd.Flags().StringSlice("type", nil, "Filter by type (e.g. bug,task)")
	myCmd.Flags().Bool("by-sprint", false, "Group results by sprint")
	registerCustomFieldCompletions(myCmd, "component")
	myTeamCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
	myTeamAliasCmd.Flags().String("sprint", "", "Sprint name (defaults to active sprint)")
}

func runMy(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		// No project set (no -p flag, no OP_PROJECT): instead of erroring,
		// infer the project+sprint where most of my open work lives.
		return runMyAutoDetect(cmd)
	}

	me, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	byAuthor, _ := cmd.Flags().GetBool("author")

	var filters []api.Filter
	if byAuthor {
		filters = append(filters, api.NewFilter("author", "=", fmt.Sprintf("%d", me.ID)))
	} else {
		filters = append(filters, api.NewFilter("assignee", "=", fmt.Sprintf("%d", me.ID)))
	}

	showAll, _ := cmd.Flags().GetBool("all")
	if !showAll {
		filters = append(filters, api.NewFilter("status", "o", ""))
	}

	// --since flag: date range filter on createdAt
	since, _ := cmd.Flags().GetString("since")
	if since != "" {
		days, err := parseDuration(since)
		if err != nil {
			return fmt.Errorf("invalid --since value %q: %w", since, err)
		}
		start := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
		tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		filters = append(filters, api.NewFilter("createdAt", "<>d", start, tomorrow))
	}

	// Component filter
	if component, _ := cmd.Flags().GetString("component"); component != "" {
		field, value, err := customFieldFilter("component", component)
		if err != nil {
			return err
		}
		filters = append(filters, api.NewFilter(field, "=", value))
	}

	if types, _ := cmd.Flags().GetStringSlice("type"); len(types) > 0 {
		resolver := api.NewResolver(client, project)
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

	// Sprint filter is opt-in. By default `op my` shows all your open work
	// across every sprint; pass --sprint to scope to one.
	if sprintName, _ := cmd.Flags().GetString("sprint"); sprintName != "" {
		version, vf, err := namedSprintFilter(project, sprintName)
		if err != nil {
			return err
		}
		filters = append(filters, vf)
		fmt.Printf("Sprint: %s\n", version.Name)
	}

	sortBy := `[["priority","asc"],["updatedAt","desc"]]`
	if byAuthor {
		sortBy = `[["createdAt","desc"]]`
	}

	result, err := client.ListWorkPackages(project, filters, sortBy, 100)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	label := "My items"
	if byAuthor {
		label = "Created by me"
	}
	fmt.Printf("%s (%d):\n", label, result.Total)
	warnTruncated(result.Total, len(result.Embedded.Elements))

	bySprint, _ := cmd.Flags().GetBool("by-sprint")
	if bySprint {
		display.GroupBySprint(result.Embedded.Elements)
	} else {
		display.WorkPackageTable(result.Embedded.Elements)
	}
	return nil
}

// parseDuration parses a human duration like "2w", "30d", "3m" into days.
func parseDuration(s string) (int, error) {
	re := regexp.MustCompile(`^(\d+)([dwm])$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("expected format: <number><d|w|m> (e.g. 2w, 30d, 3m)")
	}
	n, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "d":
		return n, nil
	case "w":
		return n * 7, nil
	case "m":
		return n * 30, nil
	}
	return 0, fmt.Errorf("unknown unit: %s", m[2])
}

func runMyTeam(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	sprintName, _ := cmd.Flags().GetString("sprint")
	version, vf, err := namedSprintFilter(project, sprintName)
	if err != nil {
		return err
	}
	fmt.Printf("Sprint: %s\n", version.Name)

	filters := []api.Filter{
		vf,
		api.NewFilter("status", "o", ""),
	}

	result, err := client.ListWorkPackages(project, filters, "", 200)
	if err != nil {
		return fmt.Errorf("listing work packages: %w", err)
	}

	display.GroupByAssignee(result.Embedded.Elements)
	return nil
}
