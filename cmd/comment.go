package cmd

import (
	"fmt"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment <id> [message]",
	Short: "List or post comments on a work package",
	Long: `List comments on a work package, or post or edit a comment.

Comment IDs are shown in the list output (e.g. "Alice (#1234)"); pass that
ID to --edit to replace an existing comment's text.

Examples:
  op comment 81321                       List all comments (with their IDs)
  op comment 81321 "LGTM"                Post a comment
  op comment 81321 "fixed typo" --edit=1234   Edit comment #1234`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runComment,
}

func init() {
	rootCmd.AddCommand(commentCmd)
	commentCmd.Flags().Int("edit", 0, "Edit an existing comment by its ID (replaces its text)")
}

func runComment(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	// Edit mode: op comment <id> "message" --edit=<comment-id>
	editID, _ := cmd.Flags().GetInt("edit")
	if editID < 0 {
		return fmt.Errorf("--edit value must be a positive comment ID")
	}
	if editID > 0 {
		if len(args) != 2 {
			return fmt.Errorf("--edit requires the new comment text as an argument")
		}
		msg := strings.TrimSpace(args[1])
		if msg == "" {
			return fmt.Errorf("comment message cannot be empty")
		}
		if err := client.EditComment(editID, msg); err != nil {
			return fmt.Errorf("editing comment: %w", err)
		}
		fmt.Printf("Comment #%d updated\n", editID)
		fmt.Println(client.WorkPackageURL(id))
		return nil
	}

	// Post mode: op comment <id> "message"
	if len(args) == 2 {
		msg := strings.TrimSpace(args[1])
		if msg == "" {
			return fmt.Errorf("comment message cannot be empty")
		}
		if err := client.PostComment(id, msg); err != nil {
			return fmt.Errorf("posting comment: %w", err)
		}
		fmt.Printf("Comment posted on #%d\n", id)
		fmt.Println(client.WorkPackageURL(id))
		return nil
	}

	// List mode: op comment <id>
	activities, err := client.ListActivities(id)
	if err != nil {
		return fmt.Errorf("listing activities: %w", err)
	}

	names := inlineAttachmentNames(fetchInlineAttachments(api.CommentInlineAttachmentIDs(activities)))
	display.Activities(activities, names)
	return nil
}
