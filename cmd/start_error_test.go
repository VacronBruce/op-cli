package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// runStart's happy path creates a REAL git branch (os/exec), so only the
// pre-git error paths are unit-tested here; branchName's pure logic is covered
// in start_test.go and the exec surface is tracked as a ticket-only item.

func TestStart_InvalidID(t *testing.T) {
	SetClient(&testutil.MockClient{})
	err := runStart(&cobra.Command{}, []string{"abc"})
	if err == nil || !strings.Contains(err.Error(), "invalid work package ID") {
		t.Fatalf("expected invalid-ID error, got: %v", err)
	}
}

func TestStart_FetchErrorAbortsBeforeAnyWrite(t *testing.T) {
	// If the ticket can't be fetched there must be no branch creation and no
	// ticket update — start must never half-apply.
	updated := false
	mock := &testutil.MockClient{
		GetWorkPackageFn: func(id int) (*api.WorkPackage, error) {
			return nil, errors.New("not found")
		},
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			updated = true
			return nil, nil
		},
	}
	SetClient(mock)

	var err error
	testutil.CaptureStdout(func() { err = runStart(&cobra.Command{}, []string{"99"}) })
	if err == nil || !strings.Contains(err.Error(), "fetching work package") {
		t.Fatalf("expected fetch error, got: %v", err)
	}
	if updated {
		t.Error("ticket must not be updated when the fetch fails")
	}
}
