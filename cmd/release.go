package cmd

import (
	"fmt"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/chenhuijun/op-cli/pkg/display"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release management commands",
}

var releaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all releases for the project",
	RunE:  runReleaseList,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new release version for the project",
	Long: `Create a new release version in OpenProject.

Examples:
  op release create "[iOS][ETV] 1.0.9"
  op release create "[iOS][EET] 3.2.0" --status=locked
  op release create "v2.0" --start=2026-06-10 --end=2026-06-30`,
	Args: cobra.ExactArgs(1),
	RunE: runReleaseCreate,
}

var releaseUpdateCmd = &cobra.Command{
	Use:   "update <id-or-name>",
	Short: "Update an existing release (rename, reschedule, or change status)",
	Long: `Update an existing release version in OpenProject. Identify the release by
its numeric ID (from 'op release list') or its exact name. Only the fields you
pass are changed; the rest are left as-is.

Examples:
  op release update 1821 --name="[iOS][EET] 3.2.1" --end=2026-06-25
  op release update "[iOS][EET] 3.2.1" --status=locked
  op release update 1821 --start=2026-06-10 --end=2026-06-25`,
	Args: cobra.ExactArgs(1),
	RunE: runReleaseUpdate,
}

func init() {
	rootCmd.AddCommand(releaseCmd)
	releaseCmd.AddCommand(releaseListCmd)
	releaseCmd.AddCommand(releaseCreateCmd)
	releaseCmd.AddCommand(releaseUpdateCmd)

	releaseCreateCmd.Flags().String("status", "open", "Release status: open, locked, or closed")
	releaseCreateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	releaseCreateCmd.Flags().String("end", "", "End date (YYYY-MM-DD)")

	releaseUpdateCmd.Flags().String("name", "", "New release name")
	releaseUpdateCmd.Flags().String("status", "", "Release status: open, locked, or closed")
	releaseUpdateCmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	releaseUpdateCmd.Flags().String("end", "", "End date (YYYY-MM-DD)")
}

func runReleaseList(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	versions, err := client.ListVersions(project)
	if err != nil {
		return fmt.Errorf("listing releases: %w", err)
	}

	var releases []api.Version
	for _, v := range versions.Embedded.Elements {
		if v.Kind == "release" {
			releases = append(releases, v)
		}
	}
	display.VersionTable(releases)
	return nil
}

func runReleaseCreate(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	status, _ := cmd.Flags().GetString("status")
	switch status {
	case "open", "locked", "closed":
	default:
		return fmt.Errorf("invalid status %q: must be open, locked, or closed", status)
	}

	start, _ := cmd.Flags().GetString("start")
	if start != "" {
		if _, err := time.Parse("2006-01-02", start); err != nil {
			return fmt.Errorf("invalid start date %q: use YYYY-MM-DD", start)
		}
	}

	end, _ := cmd.Flags().GetString("end")
	if end != "" {
		if _, err := time.Parse("2006-01-02", end); err != nil {
			return fmt.Errorf("invalid end date %q: use YYYY-MM-DD", end)
		}
	}

	req := &api.CreateVersionRequest{
		Name:      args[0],
		Status:    status,
		StartDate: start,
		EndDate:   end,
		Kind:      "release",
		Links: map[string]api.Link{
			"definingProject": {Href: fmt.Sprintf("/api/v3/projects/%s", project)},
		},
	}

	v, err := client.CreateVersion(req)
	if err != nil {
		return err
	}

	fmt.Printf("Created release #%d %q\n", v.ID, v.Name)
	return nil
}

func runReleaseUpdate(cmd *cobra.Command, args []string) error {
	project, err := client.RequireProject()
	if err != nil {
		return err
	}

	req := &api.UpdateVersionRequest{}
	changed := false

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("--name cannot be empty")
		}
		req.Name = name
		changed = true
	}

	if cmd.Flags().Changed("status") {
		status, _ := cmd.Flags().GetString("status")
		switch status {
		case "open", "locked", "closed":
		default:
			return fmt.Errorf("invalid status %q: must be open, locked, or closed", status)
		}
		req.Status = status
		changed = true
	}

	if cmd.Flags().Changed("start") {
		start, _ := cmd.Flags().GetString("start")
		if _, err := time.Parse("2006-01-02", start); err != nil {
			return fmt.Errorf("invalid start date %q: use YYYY-MM-DD", start)
		}
		req.StartDate = start
		changed = true
	}

	if cmd.Flags().Changed("end") {
		end, _ := cmd.Flags().GetString("end")
		if _, err := time.Parse("2006-01-02", end); err != nil {
			return fmt.Errorf("invalid end date %q: use YYYY-MM-DD", end)
		}
		req.EndDate = end
		changed = true
	}

	if !changed {
		return fmt.Errorf("nothing to update: pass at least one of --name, --status, --start, --end")
	}

	// Resolve the release by ID or name so we know which version to PATCH.
	rel, err := client.ResolveRelease(project, args[0])
	if err != nil {
		return err
	}

	v, err := client.UpdateVersion(rel.ID, req)
	if err != nil {
		return err
	}

	fmt.Printf("Updated release #%d %q\n", v.ID, v.Name)
	return nil
}
