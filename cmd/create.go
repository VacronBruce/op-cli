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
  op create bug "[Android][NTD+] CC bug" --epic="NTD+" --component=android --product=entd --label=team#appandroid
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
	createCmd.Flags().StringP("epic", "e", "", "Epic name (partial match)")
	createCmd.Flags().StringSlice("component", nil, "Component (android, ios, ott, engineering, analytics)")
	createCmd.Flags().String("product", "", "Product (eet, entd, djy, cntd, others)")
	createCmd.Flags().String("tech-area", "", "Tech area (web, app, adtech, video, infra, seo)")
	createCmd.Flags().StringSlice("label", nil, "Label (team#appios, team#appandroid, team#appall, ntd, seo)")
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
		Links:   make(map[string]api.LinkValue),
	}
	req.SetLink("type", api.Link{Href: wpType.Href})
	req.SetLink("priority", api.Link{Href: priority.Href})

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
		req.SetLink("assignee", api.Link{Href: user.Href})
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
				req.SetLink("version", api.Link{Href: v.Links.Self.Href})
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("sprint %q not found", sprintName)
		}
	}

	// Optional: epic
	if epicName, _ := cmd.Flags().GetString("epic"); epicName != "" {
		epic, err := resolver.ResolveEpic(epicName)
		if err != nil {
			return fmt.Errorf("resolving epic: %w", err)
		}
		req.SetLink("epic", api.Link{Href: epic.Href})
	}

	// Optional: components (multi-value, customField12)
	if components, _ := cmd.Flags().GetStringSlice("component"); len(components) > 0 {
		var links []api.Link
		for _, c := range components {
			href, err := api.ResolveCustomOption(api.ComponentOptions, c)
			if err != nil {
				return fmt.Errorf("resolving component: %w", err)
			}
			links = append(links, api.Link{Href: href})
		}
		req.SetMultiLink("customField12", links)
	}

	// Optional: product (multi-value, customField4)
	if product, _ := cmd.Flags().GetString("product"); product != "" {
		href, err := api.ResolveCustomOption(api.ProductOptions, product)
		if err != nil {
			return fmt.Errorf("resolving product: %w", err)
		}
		req.SetMultiLink("customField4", []api.Link{{Href: href}})
	}

	// Optional: tech area (multi-value, customField6)
	if techArea, _ := cmd.Flags().GetString("tech-area"); techArea != "" {
		href, err := api.ResolveCustomOption(api.TechAreaOptions, techArea)
		if err != nil {
			return fmt.Errorf("resolving tech area: %w", err)
		}
		req.SetMultiLink("customField6", []api.Link{{Href: href}})
	}

	// Optional: labels (multi-value, customField13)
	if labels, _ := cmd.Flags().GetStringSlice("label"); len(labels) > 0 {
		var links []api.Link
		for _, l := range labels {
			href, err := api.ResolveCustomOption(api.LabelOptions, l)
			if err != nil {
				return fmt.Errorf("resolving label: %w", err)
			}
			links = append(links, api.Link{Href: href})
		}
		req.SetMultiLink("customField13", links)
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
