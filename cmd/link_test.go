package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

// runLinkWith injects mock, runs runLink with flags, and returns stdout + error.
func runLinkWith(t *testing.T, mock *testutil.MockClient, args []string) (string, error) {
	t.Helper()
	SetClient(mock)

	cmd := &cobra.Command{}
	cmd.Flags().String("parent", "", "")
	cmd.Flags().Bool("no-parent", false, "")
	for _, rf := range relationFlags {
		cmd.Flags().String(rf.flag, "", "")
	}

	// Parse flags from args
	var positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			parts := strings.SplitN(args[i], "=", 2)
			flagName := strings.TrimPrefix(parts[0], "--")
			if len(parts) == 2 {
				cmd.Flags().Set(flagName, parts[1])
			} else {
				// Bool flag
				cmd.Flags().Set(flagName, "true")
			}
		} else {
			positional = append(positional, args[i])
		}
	}

	var err error
	out := testutil.CaptureStdout(func() {
		err = runLink(cmd, positional)
	})
	return out, err
}

// --- Parent linking tests ---

func TestLink_SetParent_Success(t *testing.T) {
	var capturedID int
	var capturedReq *api.UpdateWPRequest

	mock := &testutil.MockClient{
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			capturedID = id
			capturedReq = req
			return &api.WorkPackage{ID: id, Subject: "Child task"}, nil
		},
	}

	out, err := runLinkWith(t, mock, []string{"81482", "--parent=81477"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != 81482 {
		t.Errorf("expected ID=81482, got %d", capturedID)
	}
	parentLink, ok := capturedReq.Links["parent"].(api.Link)
	if !ok {
		t.Fatalf("expected parent link to be api.Link, got %T", capturedReq.Links["parent"])
	}
	if parentLink.Href != "/api/v3/work_packages/81477" {
		t.Errorf("expected parent href /api/v3/work_packages/81477, got %s", parentLink.Href)
	}
	if !strings.Contains(out, "#81482") {
		t.Errorf("expected output to contain #81482, got: %s", out)
	}
}

func TestLink_RemoveParent_Success(t *testing.T) {
	var capturedReq *api.UpdateWPRequest

	mock := &testutil.MockClient{
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			capturedReq = req
			return &api.WorkPackage{ID: id, Subject: "Child task"}, nil
		},
	}

	_, err := runLinkWith(t, mock, []string{"81482", "--no-parent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parentLink, ok := capturedReq.Links["parent"].(api.Link)
	if !ok {
		t.Fatalf("expected parent link to be api.Link, got %T", capturedReq.Links["parent"])
	}
	if parentLink.Href != "" {
		t.Errorf("expected empty href to remove parent, got %s", parentLink.Href)
	}
}

func TestLink_SetParent_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		UpdateWorkPackageFn: func(id int, req *api.UpdateWPRequest) (*api.WorkPackage, error) {
			return nil, errors.New("forbidden")
		},
	}

	_, err := runLinkWith(t, mock, []string{"81482", "--parent=81477"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "setting parent") {
		t.Errorf("expected error to mention 'setting parent', got: %v", err)
	}
}

func TestLink_InvalidID(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runLinkWith(t, mock, []string{"abc", "--parent=123"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid work package ID") {
		t.Errorf("expected invalid ID error, got: %v", err)
	}
}

func TestLink_InvalidParentID(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runLinkWith(t, mock, []string{"81482", "--parent=abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric parent ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid parent ID") {
		t.Errorf("expected invalid parent ID error, got: %v", err)
	}
}

func TestLink_NoFlags(t *testing.T) {
	mock := &testutil.MockClient{}

	_, err := runLinkWith(t, mock, []string{"81482"})
	if err == nil {
		t.Fatal("expected error when no link flags specified")
	}
	if !strings.Contains(err.Error(), "specify") {
		t.Errorf("expected error about specifying a link type, got: %v", err)
	}
}

// --- Relation tests ---

func TestLink_RelatesTo_Success(t *testing.T) {
	var capturedFrom, capturedTo int
	var capturedType string

	mock := &testutil.MockClient{
		CreateRelationFn: func(fromID int, relType string, toID int) error {
			capturedFrom = fromID
			capturedTo = toID
			capturedType = relType
			return nil
		},
	}

	out, err := runLinkWith(t, mock, []string{"81482", "--relates-to=81483"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFrom != 81482 {
		t.Errorf("expected from=81482, got %d", capturedFrom)
	}
	if capturedTo != 81483 {
		t.Errorf("expected to=81483, got %d", capturedTo)
	}
	if capturedType != "relates" {
		t.Errorf("expected type=relates, got %s", capturedType)
	}
	if !strings.Contains(out, "relates") {
		t.Errorf("expected output to mention 'relates', got: %s", out)
	}
}

func TestLink_Blocks_Success(t *testing.T) {
	var capturedType string

	mock := &testutil.MockClient{
		CreateRelationFn: func(fromID int, relType string, toID int) error {
			capturedType = relType
			return nil
		},
	}

	_, err := runLinkWith(t, mock, []string{"81482", "--blocks=81485"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedType != "blocks" {
		t.Errorf("expected type=blocks, got %s", capturedType)
	}
}

func TestLink_Relation_APIError(t *testing.T) {
	mock := &testutil.MockClient{
		CreateRelationFn: func(fromID int, relType string, toID int) error {
			return errors.New("conflict")
		},
	}

	_, err := runLinkWith(t, mock, []string{"81482", "--relates-to=81483"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating relation") {
		t.Errorf("expected error to mention 'creating relation', got: %v", err)
	}
}
