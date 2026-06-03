package api

import (
	"strings"
	"testing"
)

func TestCustomFieldByName_Defaults(t *testing.T) {
	cf, err := CustomFieldByName("component")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cf.Field != "customField12" {
		t.Errorf("expected customField12, got %s", cf.Field)
	}
	id, err := cf.OptionID("android")
	if err != nil || id != "42" {
		t.Errorf("expected android -> 42, got %s (err %v)", id, err)
	}
}

func TestCustomFieldByName_Unknown(t *testing.T) {
	_, err := CustomFieldByName("nope")
	if err == nil {
		t.Fatal("expected error for unknown custom field")
	}
	// Error lists known names so the user can correct the typo.
	if !strings.Contains(err.Error(), "component") {
		t.Errorf("expected known names in error, got: %v", err)
	}
}

// LoadCustomFields must let ~/.oprc override the field key and replace the
// option set, and register an entirely new logical field — without disturbing
// fields the user didn't mention.
func TestLoadCustomFields_OverrideAndExtend(t *testing.T) {
	// Snapshot and restore global state so other tests stay isolated.
	saved := map[string]*CustomField{}
	for k, v := range customFields {
		cp := *v
		saved[k] = &cp
	}
	t.Cleanup(func() {
		customFields = saved
	})

	LoadCustomFields(map[string]CustomFieldConfig{
		"component": { // override the field key + options
			Field:   "customField99",
			Options: map[string]string{"Backend": "/api/v3/custom_options/900"},
		},
		"team": { // brand-new logical field
			Field:   "customField50",
			Options: map[string]string{"core": "/api/v3/custom_options/500"},
		},
	})

	comp, _ := CustomFieldByName("component")
	if comp.Field != "customField99" {
		t.Errorf("expected overridden field customField99, got %s", comp.Field)
	}
	// Options were replaced, and lookup is case-insensitive on the config key.
	if id, err := comp.OptionID("backend"); err != nil || id != "900" {
		t.Errorf("expected backend -> 900, got %s (err %v)", id, err)
	}
	if _, err := comp.OptionID("android"); err == nil {
		t.Error("expected old 'android' option to be gone after replace")
	}

	team, err := CustomFieldByName("team")
	if err != nil {
		t.Fatalf("expected new 'team' field, got error: %v", err)
	}
	if href, err := team.ResolveHref("core"); err != nil || href != "/api/v3/custom_options/500" {
		t.Errorf("expected core href, got %s (err %v)", href, err)
	}

	// A field the user didn't mention keeps its defaults.
	if label, _ := CustomFieldByName("label"); label.Field != "customField13" {
		t.Errorf("untouched field changed: %s", label.Field)
	}
}

func TestCustomField_OptionNamesSorted(t *testing.T) {
	cf, _ := CustomFieldByName("component")
	got := cf.OptionNames()
	want := []string{"analytics", "android", "engineering", "ios", "ott"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("expected sorted %v, got %v", want, got)
	}
}
