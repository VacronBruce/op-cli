package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>...",
	Short: "Show work package details and attachments",
	Long: `Show full details of one or more work packages including attachments.

Several IDs print each block in order; an ID that fails is reported on stderr,
the rest still print, and the exit status is non-zero.

Examples:
  op show 81321
  op show 81321 81322 --brief Status, assignee, sprint, release on one line each
  op show 81321 --json        Raw API JSON (an array for several IDs)
  op show 81321 --url         Print only the browser URL
  op show 81321 --download    Download all attachments to current directory
  op show 81321 --download --out=/tmp`,
	Args: cobra.MinimumNArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().Bool("url", false, "Print only the browser URL (one per ID) and exit")
	showCmd.Flags().Bool("brief", false, "One line per ID: #id [status] subject | assignee | sprint | release")
	showCmd.Flags().Bool("json", false, "Print the raw work package JSON and nothing else")
	showCmd.Flags().BoolP("download", "d", false, "Download attachments")
	showCmd.Flags().StringP("out", "o", ".", "Download directory")
	showCmd.MarkFlagsMutuallyExclusive("url", "brief", "json", "download")
}

func runShow(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	urlOnly, _ := cmd.Flags().GetBool("url")
	brief, _ := cmd.Flags().GetBool("brief")
	asJSON, _ := cmd.Flags().GetBool("json")

	raws := []json.RawMessage{}
	failed := 0
	for i, arg := range args {
		id, err := parseWorkPackageID(arg)
		if err == nil {
			switch {
			case urlOnly:
				fmt.Println(client.WorkPackageURL(id))
			case asJSON:
				var raw json.RawMessage
				if raw, err = fetchWorkPackageJSON(id); err == nil {
					raws = append(raws, raw)
				}
			case brief:
				var wp *api.WorkPackage
				if wp, err = client.GetWorkPackage(id); err != nil {
					err = fmt.Errorf("getting work package: %w", err)
				} else {
					fmt.Println(display.WorkPackageBrief(wp))
				}
			default:
				if i > 0 {
					fmt.Println()
				}
				err = showFull(cmd, id)
			}
		}
		if err != nil {
			if len(args) == 1 {
				return err
			}
			failed++
			fmt.Fprintf(os.Stderr, "Error: %s: %s\n", arg, err)
		}
	}

	if asJSON {
		if err := printJSON(raws, len(args) == 1); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d work packages failed", failed, len(args))
	}
	return nil
}

func fetchWorkPackageJSON(id int) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := client.Get(fmt.Sprintf("/work_packages/%d", id), &raw); err != nil {
		return nil, fmt.Errorf("getting work package: %w", err)
	}
	return raw, nil
}

func printJSON(raws []json.RawMessage, single bool) error {
	var v interface{} = raws
	if single {
		v = raws[0]
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func showFull(cmd *cobra.Command, id int) error {
	wp, err := client.GetWorkPackage(id)
	if err != nil {
		return fmt.Errorf("getting work package: %w", err)
	}

	display.WorkPackageDetail(wp)

	// List attachments
	attachments, err := client.ListAttachments(id)
	if err != nil {
		return fmt.Errorf("listing attachments: %w", err)
	}

	if attachments.Total > 0 {
		fmt.Printf("\n  Attachments (%d):\n", attachments.Total)
		for _, att := range attachments.Embedded.Elements {
			fmt.Printf("    - #%d %s (%s, %d bytes)\n", att.ID, att.FileName, att.ContentType, att.FileSize)
			fmt.Printf("      %s\n", att.Links.DownloadLocation.Href)
		}
	}

	// Download if requested
	download, _ := cmd.Flags().GetBool("download")
	if download && attachments.Total > 0 {
		outDir, _ := cmd.Flags().GetString("out")
		fmt.Println()

		for _, att := range attachments.Embedded.Elements {
			outPath := filepath.Join(outDir, safeFileName(att.FileName, fmt.Sprintf("attachment-%d", att.ID)))
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
					name := fmt.Sprintf("%d-%s", iid, safeFileName(att.FileName, "attachment"))
					outPath := filepath.Join(outDir, name)
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

// safeFileName reduces a server-supplied filename to its base name so a
// hostile value like "../../x" cannot climb out of the download directory,
// falling back when nothing usable remains.
func safeFileName(name, fallback string) string {
	base := filepath.Base(name)
	if base == "." || base == ".." || base == string(filepath.Separator) || base == "" {
		return fallback
	}
	return base
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
