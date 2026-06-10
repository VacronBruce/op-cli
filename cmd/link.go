package cmd

import (
	"fmt"
	"strconv"
	"strings"

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
  op link 81482 --blocks=81485         Create "blocks" relation
  op link 81482 --list                 List existing relations`,
	Args: cobra.ExactArgs(1),
	RunE: runLink,
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <id>",
	Short: "Remove a relation between work packages",
	Long: `Remove a relation by type and target — relation IDs are resolved for you.

Examples:
  op unlink 81482 --relates-to=81483   Remove the "relates" relation to #81483
  op unlink 81482 --blocks=81485       Remove the "blocks" relation to #81485`,
	Args: cobra.ExactArgs(1),
	RunE: runUnlink,
}

func init() {
	rootCmd.AddCommand(linkCmd)
	linkCmd.Flags().String("parent", "", "Set parent work package ID")
	linkCmd.Flags().Bool("no-parent", false, "Remove parent link")
	linkCmd.Flags().String("relates-to", "", "Create 'relates' relation to target ID")
	linkCmd.Flags().String("blocks", "", "Create 'blocks' relation to target ID")
	linkCmd.Flags().Bool("list", false, "List the work package's relations")

	rootCmd.AddCommand(unlinkCmd)
	unlinkCmd.Flags().String("relates-to", "", "Remove 'relates' relation to target ID")
	unlinkCmd.Flags().String("blocks", "", "Remove 'blocks' relation to target ID")
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
	list, _ := cmd.Flags().GetBool("list")

	if list {
		return listRelations(id)
	}

	hasAction := parentStr != "" || noParent || relatesTo != "" || blocksStr != ""
	if !hasAction {
		return fmt.Errorf("specify at least one: --parent, --no-parent, --relates-to, --blocks, --list")
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

// listRelations prints a work package's relations with the other end of each,
// in the shape `op unlink` consumes.
func listRelations(id int) error {
	col, err := client.ListRelations(id)
	if err != nil {
		return fmt.Errorf("listing relations: %w", err)
	}
	if len(col.Embedded.Elements) == 0 {
		fmt.Printf("No relations on #%d\n", id)
		return nil
	}

	fmt.Printf("Relations of #%d (%d):\n", id, len(col.Embedded.Elements))
	for _, rel := range col.Embedded.Elements {
		other, direction := relationOtherEnd(rel, id)
		fmt.Printf("  %-10s %s #%d  %s\n", rel.Type, direction, wpIDFromHref(other.Href), other.Title)
	}
	return nil
}

func runUnlink(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	relatesTo, _ := cmd.Flags().GetString("relates-to")
	blocksStr, _ := cmd.Flags().GetString("blocks")

	relType, targetStr := "relates", relatesTo
	if blocksStr != "" {
		relType, targetStr = "blocks", blocksStr
	}
	if targetStr == "" {
		return fmt.Errorf("specify --relates-to or --blocks with the target work package ID")
	}
	targetID, err := strconv.Atoi(targetStr)
	if err != nil {
		return fmt.Errorf("invalid target ID: %s", targetStr)
	}

	col, err := client.ListRelations(id)
	if err != nil {
		return fmt.Errorf("listing relations: %w", err)
	}

	for _, rel := range col.Embedded.Elements {
		other, _ := relationOtherEnd(rel, id)
		if rel.Type == relType && wpIDFromHref(other.Href) == targetID {
			if err := client.DeleteRelation(rel.ID); err != nil {
				return fmt.Errorf("removing relation: %w", err)
			}
			fmt.Printf("Removed %s relation between #%d and #%d\n", relType, id, targetID)
			return nil
		}
	}

	// Fail loud with what IS linked so the user can correct the command.
	var existing []string
	for _, rel := range col.Embedded.Elements {
		other, _ := relationOtherEnd(rel, id)
		existing = append(existing, fmt.Sprintf("%s #%d", rel.Type, wpIDFromHref(other.Href)))
	}
	if len(existing) == 0 {
		return fmt.Errorf("no %s relation to #%d on #%d (no relations at all)", relType, targetID, id)
	}
	return fmt.Errorf("no %s relation to #%d on #%d; existing: %s",
		relType, targetID, id, strings.Join(existing, ", "))
}

// relationOtherEnd returns the link for the work package on the other side of
// a relation, plus an arrow showing the direction relative to id.
func relationOtherEnd(rel api.Relation, id int) (api.Link, string) {
	if wpIDFromHref(rel.Links.From.Href) == id {
		return rel.Links.To, "->"
	}
	return rel.Links.From, "<-"
}

// wpIDFromHref extracts the numeric work-package id from an href like
// /api/v3/work_packages/123; returns 0 when it cannot.
func wpIDFromHref(href string) int {
	idx := strings.LastIndex(href, "/")
	if idx < 0 {
		return 0
	}
	n, _ := strconv.Atoi(href[idx+1:])
	return n
}
