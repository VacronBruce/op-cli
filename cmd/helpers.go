package cmd

import (
	"fmt"
	"strconv"
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
