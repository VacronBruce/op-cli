package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	createCmd.Flags().String("sprint", "", "Sprint/version name (default from config)")
	_ = viper.BindPFlag("sprint", createCmd.Flags().Lookup("sprint"))
	createCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	createCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	createCmd.Flags().String("parent", "", "Parent work package ID")
	createCmd.Flags().StringP("epic", "e", "", "Epic name (partial match)")
	createCmd.Flags().StringSlice("component", nil, "Component (android, ios, ott, engineering, analytics)")
	createCmd.Flags().StringSlice("product", nil, "Product (eet, entd, djy, cntd, competition, others)")
	createCmd.Flags().String("tech-area", "", "Tech area (web, app, adtech, video, infra, portal, seo)")
	createCmd.Flags().StringSlice("label", nil, "Label (team#appios, team#appandroid, team#appall, team#web, ntd, seo, roku)")
	createCmd.Flags().StringSlice("attach", nil, "File path(s) to attach (images, PDFs, etc.)")

	_ = createCmd.RegisterFlagCompletionFunc("component", completeCustomField("component"))
	_ = createCmd.RegisterFlagCompletionFunc("product", completeCustomField("product"))
	_ = createCmd.RegisterFlagCompletionFunc("tech-area", completeCustomField("tech-area"))
	_ = createCmd.RegisterFlagCompletionFunc("label", completeCustomField("label"))
}

func runCreate(cmd *cobra.Command, args []string) error {
	typeName := args[0]
	subject := args[1]

	// Resolve the type before routing so `op create bug` routes by the canonical
	// type ("Bug") even when abbreviated (`op create b`). The /types collection is
	// project-independent, so a bare resolver suffices here.
	wpType, err := api.NewResolver(client, "").ResolveType(typeName)
	if err != nil {
		return fmt.Errorf("resolving type: %w", err)
	}

	project, routed, err := createProject(cmd, wpType.Name)
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client, project)

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

	// Description: explicit flag > template from config > none
	desc, _ := cmd.Flags().GetString("description")
	if desc == "" {
		desc = viper.GetString("templates." + strings.ToLower(wpType.Name))
	}
	if desc != "" {
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

	// Optional: sprint/version (flag overrides config). A config/env sprint belongs
	// to the ambient project — don't carry it onto a routed board (e.g. a bug filed
	// on the bug board); an explicitly typed --sprint still applies.
	if sprintName := viper.GetString("sprint"); sprintName != "" &&
		(cmd.Flags().Changed("sprint") || !routed) {
		version, err := client.ResolveVersion(project, sprintName)
		if err != nil {
			return fmt.Errorf("resolving sprint: %w", err)
		}
		req.SetLink("version", api.Link{Href: version.Links.Self.Href})
	}

	// Optional: parent
	if parentStr, _ := cmd.Flags().GetString("parent"); parentStr != "" {
		parentInt, err := strconv.Atoi(parentStr)
		if err != nil {
			return fmt.Errorf("invalid parent ID: %s", parentStr)
		}
		req.SetLink("parent", api.Link{Href: fmt.Sprintf("/api/v3/work_packages/%d", parentInt)})
	}

	// Optional: epic
	if epicName, _ := cmd.Flags().GetString("epic"); epicName != "" {
		epic, err := resolver.ResolveEpic(epicName)
		if err != nil {
			return fmt.Errorf("resolving epic: %w", err)
		}
		req.SetLink("epic", api.Link{Href: epic.Href})
	}

	// Optional: multi-value custom fields (component / product / label). Field
	// keys and options come from the registry (overridable via ~/.oprc).
	if components, _ := cmd.Flags().GetStringSlice("component"); len(components) > 0 {
		field, links, err := customFieldLinks("component", components)
		if err != nil {
			return err
		}
		req.SetMultiLink(field, links)
	}

	if products, _ := cmd.Flags().GetStringSlice("product"); len(products) > 0 {
		field, links, err := customFieldLinks("product", products)
		if err != nil {
			return err
		}
		req.SetMultiLink(field, links)
	}

	// tech-area is single-valued at the flag level but a multi-value field.
	if techArea, _ := cmd.Flags().GetString("tech-area"); techArea != "" {
		field, links, err := customFieldLinks("tech-area", []string{techArea})
		if err != nil {
			return err
		}
		req.SetMultiLink(field, links)
	}

	if labels, _ := cmd.Flags().GetStringSlice("label"); len(labels) > 0 {
		field, links, err := customFieldLinks("label", labels)
		if err != nil {
			return err
		}
		req.SetMultiLink(field, links)
	}

	// When a type was auto-routed to its dedicated board (e.g. a bug to the bug
	// board), tell the user before the write and how to override. Only fires on
	// auto-routing: an explicit -p is a deliberate choice that needs no notice.
	if routed {
		fmt.Printf("Filing this %s on the %q board; pass -p <board> to create it elsewhere.\n",
			strings.ToLower(wpType.Name), project)
	}

	// Create
	wp, err := client.CreateWorkPackage(project, req)
	if err != nil {
		return fmt.Errorf("creating work package: %w", err)
	}

	fmt.Printf("Created #%d\n", wp.ID)
	fmt.Println(client.WorkPackageURL(wp.ID))
	display.WorkPackageDetail(wp)

	// Upload attachments. The work package already exists at this point, so a
	// failed upload is reported but the create output above still stands; we
	// return a non-zero error so callers/scripts can detect the partial failure.
	attachFailures := 0
	if attachments, _ := cmd.Flags().GetStringSlice("attach"); len(attachments) > 0 {
		for _, filePath := range attachments {
			att, err := client.UploadAttachment(wp.ID, filePath, "")
			if err != nil {
				fmt.Printf("  Warning: failed to attach %s: %s\n", filePath, err)
				attachFailures++
				continue
			}
			fmt.Printf("  Attached: %s (%d bytes)\n", att.FileName, att.FileSize)
		}
	}

	if attachFailures > 0 {
		return fmt.Errorf("#%d created, but %d attachment(s) failed", wp.ID, attachFailures)
	}
	return nil
}

// createProject decides which project a new work package lands in. An explicitly
// typed -p always wins; otherwise bugs default to the bug board (so `op create
// bug` never lands on the App board by accident even when an ambient
// OP_PROJECT/.oprc points elsewhere); other types use the ambient project.
func createProject(cmd *cobra.Command, typeName string) (project string, routed bool, err error) {
	if f := cmd.Flag("project"); f != nil && f.Changed {
		p, err := client.RequireProject()
		return p, false, err
	}
	if proj := typeProjectFor(typeName); proj != "" {
		return proj, true, nil
	}
	p, err := client.RequireProject()
	return p, false, err
}

// typeProjectFor returns the board a work-package type routes to, or "" for none.
func typeProjectFor(typeName string) string {
	if strings.EqualFold(typeName, "bug") {
		return "bug"
	}
	return ""
}
