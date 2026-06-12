package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// opKeyRe matches OP/Jira-style identifiers like AR-178 or WEB-1462.
var opKeyRe = regexp.MustCompile(`^[A-Z]+-\d+$`)

var searchCmd = &cobra.Command{
	Use:   "search <value>",
	Short: "Map a JIRA ID (or other custom field) to its OpenProject work package",
	Long: `Find the OpenProject work package(s) whose custom field matches the given value.

Searches across all projects. Defaults to the jira-id field. Use --field to
search a different custom field (must be configured in ~/.oprc).

Use --scan to search through activity journals when a ticket key only appears
in historical activity entries (e.g. a Jira key recorded as a journal note).

Examples:
  op search WP-23
  op search BUG-655
  op search AR-178 --field key
  op search AR-178 --scan --project app`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringP("field", "f", "jira-id", "custom field to search; jira-id prefers exact matches, other fields substring-match (configured in ~/.oprc)")
	searchCmd.Flags().Bool("scan", false, "scan work package activity journals for the value")
	searchCmd.Flags().String("project", "", "project identifier to scan (required with --scan)")
	searchCmd.Flags().Int("limit", 200, "max work packages to scan (with --scan)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.TrimSpace(args[0])
	searchField, _ := cmd.Flags().GetString("field")
	searchScan, _ := cmd.Flags().GetBool("scan")
	searchProject, _ := cmd.Flags().GetString("project")
	searchScanLimit, _ := cmd.Flags().GetInt("limit")

	if searchScan {
		if searchProject == "" {
			return fmt.Errorf("--scan requires --project <identifier>")
		}
		return runScanSearch(query, searchProject, searchScanLimit)
	}

	if searchField == "jira-id" {
		result, err := client.SearchByJiraID(query)
		if err != nil {
			return fmt.Errorf("searching by JIRA ID: %w", err)
		}
		matches := selectJiraMatches(result.Embedded.Elements, query)
		if len(matches) > 0 {
			for _, wp := range matches {
				fmt.Printf("%s -> #%d  %s  [%s]\n", wp.JiraID, wp.ID, wp.Subject, wp.Links.Status.Title)
			}
			return nil
		}
		// Fallback: if the query looks like an OP identifier (AR-178), try the
		// OP `identifier` filter, which resolves the current project-scoped key
		// even when the work package has since moved to a different project.
		if opKeyRe.MatchString(query) {
			idFilters := []api.Filter{api.NewFilter("identifier", "=", query)}
			if r2, err2 := client.ListAllWorkPackages(idFilters, "", 5); err2 == nil && len(r2.Embedded.Elements) > 0 {
				for _, wp := range r2.Embedded.Elements {
					fmt.Printf("%s -> #%d  %s  [%s]\n", wp.JiraID, wp.ID, wp.Subject, wp.Links.Status.Title)
				}
				return nil
			}
		}
		return fmt.Errorf("no work package found with JIRA ID %q", query)
	}

	cf, err := api.CustomFieldByName(searchField)
	if err != nil {
		return err
	}
	filters := []api.Filter{api.NewFilter(cf.Field, "~", query)}
	result, err := client.ListAllWorkPackages(filters, "", 20)
	if err != nil {
		return fmt.Errorf("searching by %s: %w", searchField, err)
	}
	if len(result.Embedded.Elements) == 0 {
		return fmt.Errorf("no work package found with %s %q", searchField, query)
	}
	for _, wp := range result.Embedded.Elements {
		fmt.Printf("%s -> #%d  %s  [%s]\n", wp.JiraID, wp.ID, wp.Subject, wp.Links.Status.Title)
	}
	return nil
}

// runScanSearch lists work packages in a project (most recently updated first,
// up to limit) and scans each one's activity journal for the search term. The
// raw JSON response is searched so it catches both user comments and structured
// field-change entries (e.g. "Key: BR-136 -> AR-178").
func runScanSearch(query, project string, limit int) error {
	wps, err := client.ListWorkPackages(project, nil, "updatedAt:desc", limit)
	if err != nil {
		return fmt.Errorf("listing %s work packages: %w", project, err)
	}
	fmt.Fprintf(os.Stderr, "scanning %d work packages in %s for %q...\n", len(wps.Embedded.Elements), project, query)

	needle := bytes.ToLower([]byte(query))
	found := 0
	for _, wp := range wps.Embedded.Elements {
		path := fmt.Sprintf("/api/v3/work_packages/%d/activities", wp.ID)
		resp, err := client.DoRaw("GET", path)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		if bytes.Contains(bytes.ToLower(body), needle) {
			fmt.Printf("%s -> #%d  %s  [%s]\n", wp.JiraID, wp.ID, wp.Subject, wp.Links.Status.Title)
			found++
		}
	}

	if found == 0 {
		return fmt.Errorf("no work package found with %q in activities (scanned %d in %s)", query, len(wps.Embedded.Elements), project)
	}
	return nil
}

// selectJiraMatches prefers an exact (case-insensitive) JIRA ID match; if none
// match exactly it returns all candidates so the caller can disambiguate.
func selectJiraMatches(els []api.WorkPackage, jiraID string) []api.WorkPackage {
	var exact []api.WorkPackage
	for _, wp := range els {
		if strings.EqualFold(strings.TrimSpace(wp.JiraID), jiraID) {
			exact = append(exact, wp)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return els
}
