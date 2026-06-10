package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show work package details and attachments",
	Long: `Show full details of a work package including attachments.

Examples:
  op show 81321
  op show 81321 --download    Download all attachments to current directory
  op show 81321 --download --out=/tmp`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolP("download", "d", false, "Download attachments")
	showCmd.Flags().StringP("out", "o", ".", "Download directory")
}

type attachmentCollection struct {
	Total    int `json:"total"`
	Embedded struct {
		Elements []api.Attachment `json:"elements"`
	} `json:"_embedded"`
}

func runShow(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	wp, err := client.GetWorkPackage(id)
	if err != nil {
		return fmt.Errorf("getting work package: %w", err)
	}

	display.WorkPackageDetail(wp)

	// List attachments
	var attachments attachmentCollection
	if err := client.Get(fmt.Sprintf("/work_packages/%d/attachments", id), &attachments); err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}

	if attachments.Total > 0 {
		fmt.Printf("\n  Attachments (%d):\n", attachments.Total)
		for _, att := range attachments.Embedded.Elements {
			fmt.Printf("    - %s (%s, %d bytes)\n", att.FileName, att.ContentType, att.FileSize)
			fmt.Printf("      %s\n", att.Links.DownloadLocation.Href)
		}
	}

	// Download if requested
	download, _ := cmd.Flags().GetBool("download")
	if download && attachments.Total > 0 {
		outDir, _ := cmd.Flags().GetString("out")
		fmt.Println()

		for _, att := range attachments.Embedded.Elements {
			outPath := filepath.Join(outDir, att.FileName)
			if err := downloadAttachment(att.Links.DownloadLocation.Href, outPath); err != nil {
				fmt.Printf("  Error downloading %s: %s\n", att.FileName, err)
				continue
			}
			fmt.Printf("  Downloaded: %s\n", outPath)
		}
	}

	// Inline images embedded in comments live in Activity::Comment containers, so
	// they are absent from the work package attachments endpoint above. Surface
	// them separately so they are visible and downloadable.
	if activities, err := client.ListActivities(id); err == nil {
		inlineIDs := api.CommentInlineAttachmentIDs(activities)
		if len(inlineIDs) > 0 {
			inline := fetchInlineAttachments(inlineIDs)
			fmt.Printf("\n  Inline images in comments (%d):\n", len(inlineIDs))
			for _, iid := range inlineIDs {
				att := inline[iid]
				if att == nil {
					fmt.Printf("    - #%d (unavailable)\n", iid)
					continue
				}
				fmt.Printf("    - #%d %s (%s, %d bytes)\n", iid, att.FileName, att.ContentType, att.FileSize)
			}

			if download {
				outDir, _ := cmd.Flags().GetString("out")
				fmt.Println()
				for _, iid := range inlineIDs {
					att := inline[iid]
					if att == nil {
						continue
					}
					// Prefix with the attachment ID: inline screenshots are often all
					// named "image.png" and would otherwise overwrite each other.
					outPath := filepath.Join(outDir, fmt.Sprintf("%d-%s", iid, att.FileName))
					if err := downloadAttachment(att.Links.DownloadLocation.Href, outPath); err != nil {
						fmt.Printf("  Error downloading #%d: %s\n", iid, err)
						continue
					}
					fmt.Printf("  Downloaded: %s\n", outPath)
				}
			}
		}
	}

	return nil
}

func downloadAttachment(href, outPath string) error {
	httpResp, err := client.DoRaw("GET", href)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, httpResp.Body)
	return err
}
