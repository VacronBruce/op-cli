package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id> <file> [<file>...]",
	Short: "Upload attachments to a work package",
	Long: `Upload one or more files to an existing work package.

Examples:
  op attach 81317 screenshot.png
  op attach 81317 bug.png crash.log --desc="CC button screenshot"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().String("desc", "", "Description for the attachment(s)")
}

func runAttach(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
	}

	desc, _ := cmd.Flags().GetString("desc")

	for _, filePath := range args[1:] {
		att, err := client.UploadAttachment(id, filePath, desc)
		if err != nil {
			fmt.Printf("Error attaching %s: %s\n", filePath, err)
			continue
		}
		fmt.Printf("Attached to #%d: %s (%d bytes)\n", id, att.FileName, att.FileSize)
	}

	return nil
}
