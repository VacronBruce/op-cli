package cmd

import (
	"fmt"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// resolveActivityAuthors fills each activity's user title. The activities API
// returns only a user href, so names are looked up once per distinct user and
// fall back to "user #<id>" when the lookup fails.
func resolveActivityAuthors(ac *api.ActivityCollection) {
	names := map[int]string{}
	for i := range ac.Embedded.Elements {
		a := &ac.Embedded.Elements[i]
		if a.Links.User.Title != "" {
			continue
		}
		uid := a.UserID()
		if uid == 0 {
			continue
		}
		name, ok := names[uid]
		if !ok {
			var u api.User
			if err := client.Get(fmt.Sprintf("/users/%d", uid), &u); err == nil && u.Name != "" {
				name = u.Name
			} else {
				name = fmt.Sprintf("user #%d", uid)
			}
			names[uid] = name
		}
		a.Links.User.Title = name
	}
}
