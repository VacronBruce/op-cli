package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// relationFlag maps one CLI flag to its OpenProject relation type.
type relationFlag struct {
	flag    string // CLI flag name, e.g. "blocked-by"
	relType string // API relation type sent on create, e.g. "blocked"
	reverse string // counterpart type OpenProject may store instead, for unlink matching
	verb    string // human phrasing for confirmation output
}

// relationFlags lists every relation type the OpenProject API accepts.
// OpenProject canonicalizes reversed types on creation (e.g. "blocked" is
// stored as "blocks" from the other side), so unlink matches a relation by
// either relType or reverse between the two work packages.
var relationFlags = []relationFlag{
	{"relates-to", "relates", "relates", "relates to"},
	{"blocks", "blocks", "blocked", "blocks"},
	{"blocked-by", "blocked", "blocks", "is blocked by"},
	{"duplicates", "duplicates", "duplicated", "duplicates"},
	{"duplicated-by", "duplicated", "duplicates", "is duplicated by"},
	{"precedes", "precedes", "follows", "precedes"},
	{"follows", "follows", "precedes", "follows"},
	{"includes", "includes", "partof", "includes"},
	{"part-of", "partof", "includes", "is part of"},
	{"requires", "requires", "required", "requires"},
	{"required-by", "required", "requires", "is required by"},
}

var linkCmd = &cobra.Command{
	Use:   "link <id>",
	Short: "Set parent or create relations between work packages",
	Long: `Link work packages via parent-child or relation types.

Relation flags: --relates-to, --blocks, --blocked-by, --duplicates,
--duplicated-by, --precedes, --follows, --includes, --part-of,
--requires, --required-by.

Examples:
  op link 81482 --parent=81477         Set parent
  op link 81482 --no-parent            Remove parent
  op link 81482 --relates-to=81483     Create "relates" relation
  op link 81482 --blocks=81485         Create "blocks" relation
  op link 81482 --blocked-by=81485     Reverse of --blocks
  op link 81482 --follows=81480        Schedule after #81480
  op link 81482 --list                 List existing relations`,
	Args: cobra.ExactArgs(1),
	RunE: runLink,
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <id>",
	Short: "Remove a relation between work packages",
	Long: `Remove a relation by type and target — relation IDs are resolved for you.

Takes the same relation flags as op link; a relation matches in either
direction (e.g. --blocked-by also removes the stored "blocks" relation).

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
	linkCmd.Flags().Bool("list", false, "List the work package's relations")

	rootCmd.AddCommand(unlinkCmd)
	for _, rf := range relationFlags {
		linkCmd.Flags().String(rf.flag, "", fmt.Sprintf("Create '%s' relation to target ID", rf.relType))
		unlinkCmd.Flags().String(rf.flag, "", fmt.Sprintf("Remove '%s' relation to target ID", rf.relType))
	}
}

func runLink(cmd *cobra.Command, args []string) error {
	id, err := parseWorkPackageID(args[0])
	if err != nil {
		return err
	}

	parentStr, _ := cmd.Flags().GetString("parent")
	noParent, _ := cmd.Flags().GetBool("no-parent")
	list, _ := cmd.Flags().GetBool("list")

	relTargets := map[string]string{} // flag name -> target ID string
	for _, rf := range relationFlags {
		if val, _ := cmd.Flags().GetString(rf.flag); val != "" {
			relTargets[rf.flag] = val
		}
	}

	if list {
		return listRelations(id)
	}

	hasAction := parentStr != "" || noParent || len(relTargets) > 0
	if !hasAction {
		return fmt.Errorf("specify at least one: --parent, --no-parent, --list, or a relation flag (--relates-to, --blocks, --blocked-by, ...)")
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
	for _, rf := range relationFlags {
		val, ok := relTargets[rf.flag]
		if !ok {
			continue
		}
		toID, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid %s ID: %s", rf.flag, val)
		}
		if err := client.CreateRelation(id, rf.relType, toID); err != nil {
			return fmt.Errorf("creating relation: %w", err)
		}
		fmt.Printf("#%d %s #%d\n", id, rf.verb, toID)
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

	var chosen *relationFlag
	var targetStr string
	for i, rf := range relationFlags {
		val, _ := cmd.Flags().GetString(rf.flag)
		if val == "" {
			continue
		}
		if chosen != nil {
			return fmt.Errorf("specify exactly one relation flag (got --%s and --%s)", chosen.flag, rf.flag)
		}
		chosen = &relationFlags[i]
		targetStr = val
	}
	if chosen == nil {
		return fmt.Errorf("specify a relation flag (--relates-to, --blocks, --blocked-by, ...) with the target work package ID")
	}
	relType := chosen.relType
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
		// OpenProject may have stored the relation under the counterpart type
		// (e.g. --blocked-by created from this side is stored as "blocks" from
		// the other side), so match either form between the two work packages.
		if (rel.Type == relType || rel.Type == chosen.reverse) && wpIDFromHref(other.Href) == targetID {
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
