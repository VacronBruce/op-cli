package cmd

import (
	"strings"
	"testing"

	"github.com/chenhuijun/op-cli/internal/testutil"
)

// `op fields` is the discovery path for --component/--product/... values: the
// overview must name every logical field with its OpenProject field key so
// users (and the skill) never have to read source or guess.
func TestFields_OverviewListsAllFieldsWithKeys(t *testing.T) {
	var err error
	out := testutil.CaptureStdout(func() { err = runFields(fieldsCmd, nil) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"component", "product", "tech-area", "label", "jira-id"} {
		if !strings.Contains(out, name) {
			t.Errorf("overview must list %q, got: %s", name, out)
		}
	}
	if !strings.Contains(out, "customField12") {
		t.Errorf("overview must show the OpenProject field key, got: %s", out)
	}
	// jira-id has no fixed options — it must be marked free text, not "0 options".
	if !strings.Contains(out, "free text") {
		t.Errorf("free-text fields must be marked as such, got: %s", out)
	}
}

// `op fields <name>` shows exactly the values create/update/board accept —
// the same registry the flags resolve against, including ~/.oprc overrides.
func TestFields_DescribeListsOptions(t *testing.T) {
	var err error
	out := testutil.CaptureStdout(func() { err = runFields(fieldsCmd, []string{"component"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "component (customField12)") {
		t.Errorf("describe must show name and field key, got: %s", out)
	}
	if !strings.Contains(out, "android") {
		t.Errorf("describe must list the allowed values, got: %s", out)
	}
}

func TestFields_DescribeFreeTextField(t *testing.T) {
	var err error
	out := testutil.CaptureStdout(func() { err = runFields(fieldsCmd, []string{"jira-id"}) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "free-text") {
		t.Errorf("jira-id must be described as free-text, got: %s", out)
	}
}

func TestFields_UnknownFieldFailsListingKnown(t *testing.T) {
	err := runFields(fieldsCmd, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown custom field") {
		t.Fatalf("expected unknown-field error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "component") {
		t.Errorf("error must list the known field names, got: %v", err)
	}
}
