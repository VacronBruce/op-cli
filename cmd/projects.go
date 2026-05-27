package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all accessible projects",
	RunE:  runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	result, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	if result.Total == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tIDENTIFIER\tNAME\tACTIVE")
	fmt.Fprintln(w, "--\t----------\t----\t------")

	for _, p := range result.Embedded.Elements {
		active := "yes"
		if !p.Active {
			active = "no"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Identifier, p.Name, active)
	}
	w.Flush()
	return nil
}
