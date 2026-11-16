package cmd

import (
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
)

func TestComment_ListKeywordAndPluralAliasBehaveLikeBareList(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{{ID: 1, Comment: &api.Formattable{Raw: "hello"}, CreatedAt: "2024-01-15T10:00:00Z"}}
	var gotID int
	mock := &testutil.MockClient{
		ListActivitiesFn: func(id int) (*api.ActivityCollection, error) { gotID = id; return ac, nil },
		PostCommentFn:    func(int, string) error { t.Error("must not post"); return nil },
	}
	bare, err := runCommentWith(t, mock, []string{"42"})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := runCommentWith(t, mock, []string{"list", "42"})
	if err != nil {
		t.Fatal(err)
	}
	if listed != bare || gotID != 42 {
		t.Errorf("list keyword differs from bare form:\n%q\n%q (id %d)", listed, bare, gotID)
	}
	if c, _, err := rootCmd.Find([]string{"comments"}); err != nil || c.Name() != "comment" {
		t.Error("'comments' must alias comment")
	}
}
