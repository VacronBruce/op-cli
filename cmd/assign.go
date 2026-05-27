package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

var assignCmd = &cobra.Command{
	Use:   "assign <id> <@person>",
	Short: "Assign a work package to someone",
	Long: `Quick shorthand to reassign a work package.

Examples:
  op assign 123 @david
  op assign 123 david`,
	Args: cobra.ExactArgs(2),
	RunE: runAssign,
}

func init() {
	rootCmd.AddCommand(assignCmd)
}

func runAssign(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid work package ID: %s", args[0])
	}

	project, _ := client.RequireProject()
	resolver := api.NewResolver(client, project)
	user, err := resolver.ResolveUser(args[1])
	if err != nil {
		return fmt.Errorf("resolving user: %w", err)
	}

	req := &api.UpdateWPRequest{
		Links: map[string]api.LinkValue{
			"assignee": api.Link{Href: user.Href},
		},
	}

	wp, err := client.UpdateWorkPackage(id, req)
	if err != nil {
		return fmt.Errorf("updating work package: %w", err)
	}

	fmt.Printf("#%d %q assigned to %s\n", wp.ID, wp.Subject, user.Name)
	return nil
}
