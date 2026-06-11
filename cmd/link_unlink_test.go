package cmd

import (
	"fmt"
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
	for _, rf := range relationFlags {
		c.Flags().String(rf.flag, "", "")
	}
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
	if err == nil || !strings.Contains(err.Error(), "relation flag") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

// --- Extended relation types (#81747) ---

// relationBetween builds a stored relation of the given type from one WP to another.
func relationBetween(id int, relType string, fromWP, toWP int) api.Relation {
	rel := api.Relation{ID: id, Type: relType}
	rel.Links.From = api.Link{Href: fmt.Sprintf("/api/v3/work_packages/%d", fromWP)}
	rel.Links.To = api.Link{Href: fmt.Sprintf("/api/v3/work_packages/%d", toWP)}
	return rel
}

// Every OpenProject relation type is creatable; the flag's API type is sent
// verbatim so the server canonicalizes direction, not the CLI.
func TestLink_AllRelationTypesSendAPIType(t *testing.T) {
	for _, rf := range relationFlags {
		var gotType string
		var gotFrom, gotTo int
		mock := &testutil.MockClient{
			CreateRelationFn: func(fromID int, relType string, toID int) error {
				gotFrom, gotType, gotTo = fromID, relType, toID
				return nil
			},
		}
		out, err := runLinkWith(t, mock, []string{"100", "--" + rf.flag + "=200"})
		if err != nil {
			t.Fatalf("--%s: unexpected error: %v", rf.flag, err)
		}
		if gotFrom != 100 || gotTo != 200 || gotType != rf.relType {
			t.Errorf("--%s: sent (%d, %q, %d), want (100, %q, 200)", rf.flag, gotFrom, gotType, gotTo, rf.relType)
		}
		if !strings.Contains(out, rf.verb) {
			t.Errorf("--%s: confirmation must phrase the relation (%q), got: %s", rf.flag, rf.verb, out)
		}
	}
}

// A relation created as --blocked-by may be STORED by OpenProject as "blocks"
// from the other side; unlink must still find and remove it.
func TestUnlink_MatchesReverseStoredType(t *testing.T) {
	var deleted int
	col := &api.RelationCollection{}
	col.Embedded.Elements = []api.Relation{
		relationBetween(7, "blocks", 200, 100), // #200 blocks #100, stored canonically
	}
	mock := &testutil.MockClient{
		ListRelationsFn: func(wpID int) (*api.RelationCollection, error) { return col, nil },
		DeleteRelationFn: func(relID int) error {
			deleted = relID
			return nil
		},
	}
	SetClient(mock)

	cmd := newUnlinkCmd()
	_ = cmd.Flags().Set("blocked-by", "200")
	var err error
	testutil.CaptureStdout(func() { err = runUnlink(cmd, []string{"100"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 7 {
		t.Errorf("expected relation 7 deleted, got %d", deleted)
	}
}

// Two relation flags at once is ambiguous — fail loud, never guess.
func TestUnlink_TwoFlagsIsAnError(t *testing.T) {
	SetClient(&testutil.MockClient{})
	cmd := newUnlinkCmd()
	_ = cmd.Flags().Set("blocks", "1")
	_ = cmd.Flags().Set("follows", "2")
	err := runUnlink(cmd, []string{"12"})
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected exactly-one error, got: %v", err)
	}
}
