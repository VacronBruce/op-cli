package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment <id> [message]",
	Short: "List or post comments on a work package",
	Long: `List comments on a work package, or post a new comment.

Examples:
  op comment 81321              List all comments
  op comment 81321 "LGTM"      Post a comment`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runComment,
}

func init() {
	rootCmd.AddCommand(commentCmd)
}

func runComment(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
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
		return nil
	}

	// List mode: op comment <id>
	activities, err := client.ListActivities(id)
	if err != nil {
		return fmt.Errorf("listing activities: %w", err)
	}

	display.Activities(activities)
	return nil
}
