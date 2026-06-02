package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <jira-id>",
	Short: "Map a JIRA ID to its OpenProject work package number",
	Long: `Find the OpenProject work package(s) whose JIRA ID custom field matches the given value.

Searches across all projects. When an exact JIRA ID match exists it is shown;
otherwise all partial matches are listed.

Examples:
  op search WP-23
  op search BUG-655`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	jiraID := strings.TrimSpace(args[0])

	result, err := client.SearchByJiraID(jiraID)
	if err != nil {
		return fmt.Errorf("searching by JIRA ID: %w", err)
	}

	matches := selectJiraMatches(result.Embedded.Elements, jiraID)
	if len(matches) == 0 {
		return fmt.Errorf("no work package found with JIRA ID %q", jiraID)
	}

	for _, wp := range matches {
		fmt.Printf("%s -> #%d  %s  [%s]\n", wp.JiraID, wp.ID, wp.Subject, wp.Links.Status.Title)
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
