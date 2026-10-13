package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release management commands",
}

var releaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all releases for the project",
	RunE:  runReleaseList,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new release version for the project",
	Long: `Create a new release version in OpenProject.

Examples:
  op release create "[iOS][ETV] 1.0.9"
  op release create "[iOS][EET] 3.2.0" --status=locked
  op release create "v2.0" --start=2026-06-10 --end=2026-06-30`,
	Args: cobra.ExactArgs(1),
	RunE: runReleaseCreate,
}

var releaseShowCmd = &cobra.Command{
	Use:   "show <name|id>",
	Short: "Show a release and every ticket assigned to it",
	Long: `Show a release's details and all of its tickets, in any status (Done included),
so you can see what shipped and draft release notes.

Examples:
  op release show "[Android][EET] 3.3.1"
  op release show 2054`,
	Args:              cobra.ExactArgs(1),
	RunE:              runReleaseShow,
	ValidArgsFunction: completeRelease(),
}

var releaseUpdateCmd = &cobra.Command{
	Use:   "update <name|id>",
	Short: "Set a release's description (release notes)",
	Long: `Replace a release's description, e.g. with its "What's New" notes.

OpenProject stores release descriptions as plain text: Markdown such as
"## Heading" or "**bold**" shows literally, so write plain lines.

Examples:
  op release update "[Android][EET] 3.3.1" --description="Category pages load faster."
  op release update 2054 --description="$(cat whats-new.txt)"`,
	Args:              cobra.ExactArgs(1),
	RunE:              runReleaseUpdate,
	ValidArgsFunction: completeRelease(),
}

// releaseShowPageSize is the most tickets one release listing fetches; a
// larger release prints a warning instead of being silently cut short.
const releaseShowPageSize = 500

func init() {
	rootCmd.AddCommand(releaseCmd)
	releaseCmd.AddCommand(releaseListCmd)
	releaseCmd.AddCommand(releaseCreateCmd)
	releaseCmd.AddCommand(releaseShowCmd)
	releaseCmd.AddCommand(releaseUpdateCmd)

	releaseCreateCmd.Flags().String("status", "open", "Release status: open, locked, or closed")
	releaseCreateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	releaseCreateCmd.Flags().String("end", "", "End date (YYYY-MM-DD)")

	releaseUpdateCmd.Flags().StringP("description", "d", "", "New description (plain text; replaces the current one)")
}

func runReleaseShow(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	release, err := client.ResolveRelease(project, args[0])
	if err != nil {
		return err
	}

	filters := []api.Filter{
		// A ticket's release is customField50, not the built-in version field
		// (that one holds the sprint).
		api.NewFilter("customField50", "=", strconv.Itoa(release.ID)),
		// Any status: the shipped (Done) tickets are the point of a release.
		{"status": api.FilterSpec{Operator: "*", Values: []string{}}},
	}
	result, err := client.ListWorkPackages(project, filters, "", releaseShowPageSize)
	if err != nil {
		return fmt.Errorf("listing tickets in release %q: %w", release.Name, err)
	}
	wps := result.Embedded.Elements

	fmt.Printf("#%d %s (%s)\n", release.ID, release.Name, release.Status)
	if release.StartDate != "" || release.EndDate != "" {
		fmt.Printf("Dates: %s → %s\n", dateOrDash(release.StartDate), dateOrDash(release.EndDate))
	}
	if release.Description != nil && strings.TrimSpace(release.Description.Raw) != "" {
		fmt.Printf("Description:\n%s\n", indent(release.Description.Raw, "  "))
	} else {
		fmt.Println("Description: (none)")
	}
	fmt.Println()

	display.WorkPackageTable(wps)
	if len(wps) > 0 {
		fmt.Printf("\n%d tickets: %s\n", len(wps), statusCounts(wps))
	}
	if result.Total > len(wps) {
		fmt.Printf("WARNING: showing %d of %d tickets in this release\n", len(wps), result.Total)
	}
	return nil
}

func runReleaseUpdate(cmd *cobra.Command, args []string) error {
	desc, _ := cmd.Flags().GetString("description")
	if strings.TrimSpace(desc) == "" {
		return fmt.Errorf("nothing to update: pass --description")
	}

	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	release, err := client.ResolveRelease(project, args[0])
	if err != nil {
		return err
	}

	req := &api.UpdateVersionRequest{
		Description: &api.Formattable{Format: "plain", Raw: desc},
	}
	if _, err := client.UpdateVersion(release.ID, req); err != nil {
		return fmt.Errorf("updating release %q: %w", release.Name, err)
	}

	fmt.Printf("Updated release #%d %q description\n", release.ID, release.Name)
	if release.Description != nil && strings.TrimSpace(release.Description.Raw) != "" {
		fmt.Println("(replaced the previous description)")
	}
	return nil
}

// statusCounts summarizes tickets per status, most common first, e.g.
// "23 Done, 1 Ready for Release, 1 To Do".
func statusCounts(wps []api.WorkPackage) string {
	counts := map[string]int{}
	for _, wp := range wps {
		counts[wp.Links.Status.Title]++
	}
	statuses := make([]string, 0, len(counts))
	for s := range counts {
		statuses = append(statuses, s)
	}
	sort.Slice(statuses, func(i, j int) bool {
		if counts[statuses[i]] != counts[statuses[j]] {
			return counts[statuses[i]] > counts[statuses[j]]
		}
		return statuses[i] < statuses[j]
	})
	parts := make([]string, len(statuses))
	for i, s := range statuses {
		parts[i] = fmt.Sprintf("%d %s", counts[s], s)
	}
	return strings.Join(parts, ", ")
}

func dateOrDash(d string) string {
	if d == "" {
		return "-"
	}
	return d
}

func indent(text, prefix string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func runReleaseList(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	versions, err := client.ListVersions(project)
	if err != nil {
		return fmt.Errorf("listing releases: %w", err)
	}

	var releases []api.Version
	for _, v := range versions.Embedded.Elements {
		if v.Kind == "release" {
			releases = append(releases, v)
		}
	}
	display.VersionTable(releases)
	return nil
}

func runReleaseCreate(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	status, _ := cmd.Flags().GetString("status")
	switch status {
	case "open", "locked", "closed":
	default:
		return fmt.Errorf("invalid status %q: must be open, locked, or closed", status)
	}

	start, _ := cmd.Flags().GetString("start")
	if start != "" {
		if _, err := time.Parse("2006-01-02", start); err != nil {
			return fmt.Errorf("invalid start date %q: use YYYY-MM-DD", start)
		}
	}

	end, _ := cmd.Flags().GetString("end")
	if end != "" {
		if _, err := time.Parse("2006-01-02", end); err != nil {
			return fmt.Errorf("invalid end date %q: use YYYY-MM-DD", end)
		}
	}

	req := &api.CreateVersionRequest{
		Name:      args[0],
		Status:    status,
		StartDate: start,
		EndDate:   end,
		Kind:      "release",
		Links: map[string]api.Link{
			"definingProject": {Href: fmt.Sprintf("/api/v3/projects/%s", project)},
		},
	}

	v, err := client.CreateVersion(req)
	if err != nil {
		return err
	}

	fmt.Printf("Created release #%d %q\n", v.ID, v.Name)
	return nil
}
