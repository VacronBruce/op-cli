package cmd

import (
	"fmt"
	"strconv"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// parseWorkPackageID parses a work-package ID argument with the shared
// "invalid work package ID" error every command reports.
func parseWorkPackageID(arg string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid work package ID: %s", arg)
	}
	return id, nil
}

// activeSprintFilter resolves the project's active sprint and returns it with
// its work-package version filter.
func activeSprintFilter(project string) (*api.Version, api.Filter, error) {
	version, err := client.FindActiveSprint(project)
	return sprintVersionFilter(version, project, err)
}

// namedSprintFilter resolves a sprint by name (active sprint when empty) and
// returns it with its work-package version filter.
func namedSprintFilter(project, name string) (*api.Version, api.Filter, error) {
	version, err := client.ResolveVersion(project, name)
	return sprintVersionFilter(version, project, err)
}

func sprintVersionFilter(version *api.Version, project string, err error) (*api.Version, api.Filter, error) {
	if err != nil {
		return nil, nil, err
	}
	vf, err := api.VersionFilter(version, project)
	if err != nil {
		return nil, nil, err
	}
	return version, vf, nil
}
