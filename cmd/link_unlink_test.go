package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
)

func relationsFixture() *api.RelationCollection {
	col := &api.RelationCollection{Total: 2}
	rel1 := api.Relation{ID: 7, Type: "relates"}
	rel1.Links.From = api.Link{Href: "/api/v3/work_packages/12", Title: "This one"}
	rel1.Links.To = api.Link{Href: "/api/v3/work_packages/34", Title: "That one"}
	rel2 := api.Relation{ID: 8, Type: "blocks"}
	rel2.Links.From = api.Link{Href: "/api/v3/work_packages/12", Title: "This one"}
	rel2.Links.To = api.Link{Href: "/api/v3/work_packages/56", Title: "Blocked one"}
	col.Embedded.Elements = []api.Relation{rel1, rel2}
	return col
}

func newLinkListCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().String("parent", "", "")
	c.Flags().Bool("no-parent", false, "")
	c.Flags().String("relates-to", "", "")
	c.Flags().String("blocks", "", "")
	c.Flags().Bool("list", false, "")
	return c
}

func TestLink_ListShowsRelationsWithIDs(t *testing.T) {
	// --list is how users discover what to unlink: each line must show the
	// relation type, the other end, and be resolvable to an unlink call.
	mock := &testutil.MockClient{
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) {
			if wpID != 12 {
				t.Errorf("expected wpID 12, got %d", wpID)
			}
			return relationsFixture(), nil
		},
	}
	SetClient(mock)

	cmd := newLinkListCmd()
	_ = cmd.Flags().Set("list", "true")
	out := testutil.CaptureStdout(func() {
		if err := runLink(cmd, []string{"12"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "relates") || !strings.Contains(out, "#34") || !strings.Contains(out, "That one") {
		t.Errorf("expected relates line with target, got: %s", out)
	}
	if !strings.Contains(out, "blocks") || !strings.Contains(out, "#56") {
		t.Errorf("expected blocks line, got: %s", out)
	}
}

func TestLink_ListEmpty(t *testing.T) {
	SetClient(&testutil.MockClient{})
	cmd := newLinkListCmd()
	_ = cmd.Flags().Set("list", "true")
	out := testutil.CaptureStdout(func() {
		if err := runLink(cmd, []string{"12"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No relations") {
		t.Errorf("expected no-relations message, got: %s", out)
	}
}

func newUnlinkCmd() *cobra.Command {
	c := &cobra.Command{}
	c.Flags().String("relates-to", "", "")
	c.Flags().String("blocks", "", "")
	return c
}

func TestUnlink_RelatesToDeletesMatchingRelation(t *testing.T) {
	// unlink resolves (type, other-end WP id) to the RELATION id and deletes
	// that — users never have to know relation ids exist.
	var deleted int
	mock := &testutil.MockClient{
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) {
			return relationsFixture(), nil
		},
		DeleteRelationFn: func(relID int) error {
			deleted = relID
			return nil
		},
	}
	SetClient(mock)

	cmd := newUnlinkCmd()
	_ = cmd.Flags().Set("relates-to", "34")
	out := testutil.CaptureStdout(func() {
		if err := runUnlink(cmd, []string{"12"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if deleted != 7 {
		t.Errorf("expected relation 7 deleted, got %d", deleted)
	}
	if !strings.Contains(out, "Removed relates relation") {
		t.Errorf("expected confirmation, got: %s", out)
	}
}

func TestUnlink_NoMatchFailsLoudListingExisting(t *testing.T) {
	// A wrong target must not delete anything and must show what IS linked so
	// the user can correct the command.
	deleted := false
	mock := &testutil.MockClient{
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) {
			return relationsFixture(), nil
		},
		DeleteRelationFn: func(relID int) error {
			deleted = true
			return nil
		},
	}
	SetClient(mock)

	cmd := newUnlinkCmd()
	_ = cmd.Flags().Set("relates-to", "999")
	err := runUnlink(cmd, []string{"12"})
	if err == nil || !strings.Contains(err.Error(), "no relates relation") {
		t.Fatalf("expected no-match error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "#34") {
		t.Errorf("error should list existing relations, got: %v", err)
	}
	if deleted {
		t.Error("nothing must be deleted on no-match")
	}
}

func TestUnlink_RequiresExactlyOneFlag(t *testing.T) {
	SetClient(&testutil.MockClient{})
	err := runUnlink(newUnlinkCmd(), []string{"12"})
	if err == nil || !strings.Contains(err.Error(), "--relates-to or --blocks") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}
