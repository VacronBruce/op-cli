package api

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

// CustomField is a logical, instance-specific custom field: the OpenProject
// field key (e.g. "customField12") plus its selectable options (lowercased
// name -> option href). The built-in defaults below target the epochbase.com
// instance; a `custom_fields:` section in ~/.oprc overrides them via
// LoadCustomFields (both the field key and the option set are configurable).
type CustomField struct {
	Field   string
	Options map[string]string
}

// customFields is the active registry, keyed by logical name. It starts from
// the built-in defaults (the ComponentOptions/... maps in types.go) and is
// overridden by LoadCustomFields at startup. The option maps are cloned so the
// registry can never mutate the exported default maps.
var customFields = map[string]*CustomField{
	"component": {Field: "customField12", Options: maps.Clone(ComponentOptions)},
	"product":   {Field: "customField4", Options: maps.Clone(ProductOptions)},
	"tech-area": {Field: "customField6", Options: maps.Clone(TechAreaOptions)},
	"label":     {Field: "customField13", Options: maps.Clone(LabelOptions)},
}

// CustomFieldByName returns the registered custom field for a logical name
// (e.g. "component"), or an error listing the known names.
func CustomFieldByName(name string) (*CustomField, error) {
	if cf, ok := customFields[strings.ToLower(name)]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("unknown custom field %q (have: %s)", name, strings.Join(CustomFieldNames(), ", "))
}

// CustomFieldNames returns the logical field names, sorted — for help and
// shell completion.
func CustomFieldNames() []string {
	names := make([]string, 0, len(customFields))
	for k := range customFields {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// OptionNames returns the option keys for this field, sorted — for completion.
func (cf *CustomField) OptionNames() []string {
	names := make([]string, 0, len(cf.Options))
	for k := range cf.Options {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ResolveHref resolves an option name to its href (for create/update links).
func (cf *CustomField) ResolveHref(name string) (string, error) {
	return ResolveCustomOption(cf.Options, name)
}

// OptionID resolves an option name to its numeric ID (for filter values).
func (cf *CustomField) OptionID(name string) (string, error) {
	return OptionID(cf.Options, name)
}

// CustomFieldConfig is the ~/.oprc shape for one logical field under
// `custom_fields:`. Both keys are optional: omitting `field` keeps the
// built-in field key, omitting `options` keeps the built-in option set.
type CustomFieldConfig struct {
	Field   string            `mapstructure:"field"`
	Options map[string]string `mapstructure:"options"`
}

// LoadCustomFields merges user config over the built-in defaults. A logical
// name not present in cfg keeps its defaults; a new name (with a field key)
// registers an entirely new custom field. Option names are lowercased so
// lookups stay case-insensitive.
func LoadCustomFields(cfg map[string]CustomFieldConfig) {
	for name, c := range cfg {
		name = strings.ToLower(name)
		cf := customFields[name]
		if cf == nil {
			cf = &CustomField{}
			customFields[name] = cf
		}
		if c.Field != "" {
			cf.Field = c.Field
		}
		if len(c.Options) > 0 {
			opts := make(map[string]string, len(c.Options))
			for k, v := range c.Options {
				opts[strings.ToLower(k)] = v
			}
			cf.Options = opts
		}
	}
}
