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

Examples:
  op activity 86156
  op history 86156`,
	Args: cobra.ExactArgs(1),
	RunE: runActivity,
}

func init() {
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
	names := inlineAttachmentNames(fetchInlineAttachments(api.CommentInlineAttachmentIDs(activities)))
	display.ActivityLog(activities, names)
	return nil
}
