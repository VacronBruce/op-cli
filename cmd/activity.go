package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:     "activity <id>",
	Aliases: []string{"activities", "history"},
	Short:   "Show a work package's full history: status changes, field edits and comments",
	Long: `Show every activity on a work package, oldest first: who changed what and
when (status, assignee, version, ...) plus the comments.

Use "op comment <id>" to see only the comments.

With --description, show how the description changed: one block per edit
with the journal id, time and author, then the changed lines as a word diff
([-deleted-]{+inserted+}). --json prints before and after for every edit.
OpenProject's API never returns old description text, so this reads its web
diff pages with a web login (web_user in ~/.oprc, password in the macOS
Keychain item "op-cli-web" or OP_WEB_PASSWORD). The diff is word-based, so
the text is whitespace-normalised: runs of spaces collapse and lines are trimmed.

Examples:
  op activity 86156
  op history 86156
  op activity 86156 --description
  op activity 86156 --description --json`,
	Args: cobra.ExactArgs(1),
	RunE: runActivity,
}

func init() {
	activityCmd.Flags().Bool("description", false, "Show how the description changed, before and after each edit (whitespace-normalised)")
	activityCmd.Flags().Bool("json", false, "With --description: print before/after for every edit as JSON")
	rootCmd.AddCommand(activityCmd)
}

func runActivity(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}
	activities, err := client.ListActivities(id)
	if err != nil {
		return fmt.Errorf("listing activities: %w", err)
	}
	resolveActivityAuthors(activities)
	if showDescription, _ := cmd.Flags().GetBool("description"); showDescription {
		asJSON, _ := cmd.Flags().GetBool("json")
		return printDescriptionHistory(activities, asJSON)
	}
	names := inlineAttachmentNames(fetchInlineAttachments(api.CommentInlineAttachmentIDs(activities)))
	display.ActivityLog(activities, names)
	return nil
}
