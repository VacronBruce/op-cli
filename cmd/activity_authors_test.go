package cmd

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
)

func activityByUser(id int, userHref string) api.Activity {
	a := api.Activity{ID: id, Comment: &api.Formattable{Raw: "hi"}, CreatedAt: "2026-11-06T09:30:00Z"}
	a.Links.User = api.Link{Href: userHref}
	return a
}

func TestComment_List_ResolvesAuthorNamesFromUserHref(t *testing.T) {
	ac := &api.ActivityCollection{}
	ac.Embedded.Elements = []api.Activity{
		activityByUser(1, "/api/v3/users/36"),
		activityByUser(2, "/api/v3/users/36"),
		activityByUser(3, "/api/v3/users/7"),
	}
	lookups := map[string]int{}
	mock := &testutil.MockClient{
		ListActivitiesFn: func(int) (*api.ActivityCollection, error) { return ac, nil },
		GetFn: func(path string, result interface{}) error {
			lookups[path]++
			switch path {
			case "/users/36":
				return json.Unmarshal([]byte(`{"name":"Ada Lovelace"}`), result)
			case "/users/7":
				return errors.New("forbidden")
			}
			return errors.New("unexpected " + path)
		},
	}

	out, err := runCommentWith(t, mock, []string{"42"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Ada Lovelace (#1)") {
		t.Errorf("expected resolved name, got: %s", out)
	}
	if !strings.Contains(out, "user #7 (#3)") {
		t.Errorf("failed lookup must fall back to 'user #7', got: %s", out)
	}
	if lookups["/users/36"] != 1 {
		t.Errorf("each user must be fetched once per invocation, got %d", lookups["/users/36"])
	}
}
