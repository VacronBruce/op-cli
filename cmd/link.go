package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <id>",
	Short: "Set parent or create relations between work packages",
	Long: `Link work packages via parent-child or relation types.

Examples:
  op link 81482 --parent=81477         Set parent
  op link 81482 --no-parent            Remove parent
  op link 81482 --relates-to=81483     Create "relates" relation
  op link 81482 --blocks=81485         Create "blocks" relation`,
	Args: cobra.ExactArgs(1),
	RunE: runLink,
}

func init() {
	rootCmd.AddCommand(linkCmd)
	linkCmd.Flags().String("parent", "", "Set parent work package ID")
	linkCmd.Flags().Bool("no-parent", false, "Remove parent link")
	linkCmd.Flags().String("relates-to", "", "Create 'relates' relation to target ID")
	linkCmd.Flags().String("blocks", "", "Create 'blocks' relation to target ID")
}

func runLink(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	parentStr, _ := cmd.Flags().GetString("parent")
	noParent, _ := cmd.Flags().GetBool("no-parent")
	relatesTo, _ := cmd.Flags().GetString("relates-to")
	blocksStr, _ := cmd.Flags().GetString("blocks")

	hasAction := parentStr != "" || noParent || relatesTo != "" || blocksStr != ""
	if !hasAction {
		return fmt.Errorf("specify at least one: --parent, --no-parent, --relates-to, --blocks")
	}

	if parentStr != "" && noParent {
		return fmt.Errorf("--parent and --no-parent are mutually exclusive")
	}

	// Handle parent linking
	if parentStr != "" || noParent {
		var parentHref string
		if noParent {
			parentHref = ""
		} else {
			parentID, err := strconv.Atoi(parentStr)
			if err != nil {
				return fmt.Errorf("invalid parent ID: %s", parentStr)
			}
			parentHref = fmt.Sprintf("/api/v3/work_packages/%d", parentID)
		}

		req := &api.UpdateWPRequest{
			Links: map[string]api.LinkValue{
				"parent": api.Link{Href: parentHref},
			},
		}

		wp, err := client.UpdateWorkPackage(id, req)
		if err != nil {
			return fmt.Errorf("setting parent: %w", err)
		}

		if noParent {
			fmt.Printf("#%d %q parent removed\n", wp.ID, wp.Subject)
		} else {
			fmt.Printf("#%d %q parent set to #%s\n", wp.ID, wp.Subject, parentStr)
		}
	}

	// Handle relations
	if relatesTo != "" {
		toID, err := strconv.Atoi(relatesTo)
		if err != nil {
			return fmt.Errorf("invalid relates-to ID: %s", relatesTo)
		}
		if err := client.CreateRelation(id, "relates", toID); err != nil {
			return fmt.Errorf("creating relation: %w", err)
		}
		fmt.Printf("#%d relates to #%d\n", id, toID)
	}

	if blocksStr != "" {
		toID, err := strconv.Atoi(blocksStr)
		if err != nil {
			return fmt.Errorf("invalid blocks ID: %s", blocksStr)
		}
		if err := client.CreateRelation(id, "blocks", toID); err != nil {
			return fmt.Errorf("creating relation: %w", err)
		}
		fmt.Printf("#%d blocks #%d\n", id, toID)
	}

	return nil
}
