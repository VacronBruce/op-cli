package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
	"github.com/chenhuijun/op-cli/pkg/api"
)

// Agents curled /relations 22 times in 21 days because op show left them out.
// The work package response already embeds them, so show needs no extra call.
const wpWithRelations = `{
  "id": 86156, "subject": "Consent bug",
  "_links": {"type": {"title": "Bug"}, "status": {"title": "Done"}, "project": {"title": "App"}},
  "_embedded": {"relations": {"total": 2, "_embedded": {"elements": [
    {"id": 1, "type": "follows", "_links": {
      "from": {"href": "/api/v3/work_packages/86156", "title": "Consent bug"},
      "to": {"href": "/api/v3/work_packages/85719", "title": "Align consent context"}}},
    {"id": 2, "type": "blocks", "_links": {
      "from": {"href": "/api/v3/work_packages/900", "title": "Release prep"},
      "to": {"href": "/api/v3/work_packages/86156", "title": "Consent bug"}}}
  ]}}}
}`

func TestShow_ListsRelationsWithTheOtherEndOfEach(t *testing.T) {
	var wp api.WorkPackage
	if err := json.Unmarshal([]byte(wpWithRelations), &wp); err != nil {
		t.Fatal(err)
	}
	SetClient(&testutil.MockClient{
		GetWorkPackageFn:  func(int) (*api.WorkPackage, error) { return &wp, nil },
		ListAttachmentsFn: func(int) (*api.AttachmentCollection, error) { return &api.AttachmentCollection{}, nil },
		ListActivitiesFn:  func(int) (*api.ActivityCollection, error) { return &api.ActivityCollection{}, nil },
	})

	out := testutil.CaptureStdout(func() {
		if err := runShow(newShowCmd(), []string{"86156"}); err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		"Relations (2):",
		"follows    -> #85719  Align consent context",
		"blocks     <- #900  Release prep",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}
