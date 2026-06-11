package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id> [<file>...]",
	Short: "Upload, list, or remove attachments on a work package",
	Long: `Upload one or more files to an existing work package, list its
attachments (with their IDs), or remove one by attachment ID.

Examples:
  op attach 81317 screenshot.png
  op attach 81317 bug.png crash.log --desc="CC button screenshot"
  op attach 81317 --list               List attachments with their IDs
  op attach 81317 --remove=318         Remove attachment #318`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().String("desc", "", "Description for the attachment(s)")
	attachCmd.Flags().Bool("list", false, "List the work package's attachments")
	attachCmd.Flags().Int("remove", 0, "Remove an attachment by its ID (see --list)")
}

func runAttach(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	list, _ := cmd.Flags().GetBool("list")
	removeID, _ := cmd.Flags().GetInt("remove")
	if removeID < 0 {
		return fmt.Errorf("--remove value must be a positive attachment ID")
	}

	if list {
		return listAttachments(id)
	}
	if removeID > 0 {
		return removeAttachment(id, removeID)
	}

	desc, _ := cmd.Flags().GetString("desc")

	files := args[1:]
	if len(files) == 0 {
		return fmt.Errorf("provide files to upload, or use --list / --remove=<attachment-id>")
	}
	failures := 0
	for _, filePath := range files {
		att, err := client.UploadAttachment(id, filePath, desc)
		if err != nil {
			fmt.Printf("Error attaching %s: %s\n", filePath, err)
			failures++
			continue
		}
		fmt.Printf("Attached to #%d: %s (%d bytes)\n", id, att.FileName, att.FileSize)
	}

	if failures > 0 {
		return fmt.Errorf("%d of %d file(s) failed to attach", failures, len(files))
	}
	return nil
}

// listAttachments prints a work package's attachments with the IDs that
// --remove consumes.
func listAttachments(id int) error {
	col, err := client.ListAttachments(id)
	if err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}
	if len(col.Embedded.Elements) == 0 {
		fmt.Printf("No attachments on #%d\n", id)
		return nil
	}
	fmt.Printf("Attachments on #%d (%d):\n", id, len(col.Embedded.Elements))
	for _, att := range col.Embedded.Elements {
		fmt.Printf("  #%-6d %s (%s, %d bytes)\n", att.ID, att.FileName, att.ContentType, att.FileSize)
	}
	return nil
}

// removeAttachment deletes an attachment only after confirming it belongs to
// the given work package — a bare attachment ID could otherwise delete a file
// from a different ticket via typo.
func removeAttachment(id, attID int) error {
	col, err := client.ListAttachments(id)
	if err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}

	for _, att := range col.Embedded.Elements {
		if att.ID != attID {
			continue
		}
		if err := client.DeleteAttachment(attID); err != nil {
			return fmt.Errorf("removing attachment: %w", err)
		}
		fmt.Printf("Removed attachment #%d (%s) from #%d\n", attID, att.FileName, id)
		return nil
	}

	// Fail loud with what IS attached so the user can correct the command.
	var existing []string
	for _, att := range col.Embedded.Elements {
		existing = append(existing, fmt.Sprintf("#%d %s", att.ID, att.FileName))
	}
	if len(existing) == 0 {
		return fmt.Errorf("no attachment #%d on #%d (no attachments at all)", attID, id)
	}
	return fmt.Errorf("no attachment #%d on #%d; existing: %s", attID, id, strings.Join(existing, ", "))
}
