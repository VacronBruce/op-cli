package check

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DoRConfig is a tunable Definition of Ready: it maps a work-package type
// (lowercased) to the ordered list of check IDs that define readiness for that
// type. It lets a team add or drop checks per type without a code change —
// each ID must exist in the check registry (see registry.go).
type DoRConfig struct {
	Types map[string][]string `json:"types"`
}

// defaultDoR is the Definition of Ready that ships with op. bug, feature/story,
// and task carry the advisory INVEST "no_blockers" (Independent) check; feature/
// story additionally carry the advisory QUS "well_formed" check. epic omits
// no_blockers — epics aggregate dependent work by design. The "" entry is the
// fallback for unknown types. The noisier QUS "atomic" check is registered but
// intentionally left out of the defaults — enable it per team via a config file.
var defaultDoR = &DoRConfig{
	Types: map[string][]string{
		"bug":     {"description", "reproduction_steps", "story_points", "assignee", "priority", "attachments", "parent_epic", "component", "no_blockers"},
		"feature": {"description", "acceptance_criteria", "use_case", "business_value", "well_formed", "story_points", "assignee", "priority", "attachments", "parent_epic", "component", "no_blockers"},
		"task":    {"description", "acceptance_criteria", "story_points", "assignee", "priority", "parent_epic", "component", "no_blockers"},
		"epic":    {"description", "acceptance_criteria", "business_value", "component"},
		"":        {"description", "story_points", "assignee", "priority", "component"},
	},
}

// canonicalType folds type synonyms onto the canonical config keys. "user story"
// and "story" are treated as "feature", matching the historical RulesForType.
func canonicalType(typeName string) string {
	t := strings.ToLower(strings.TrimSpace(typeName))
	switch t {
	case "user story", "story":
		return "feature"
	default:
		return t
	}
}

// Rules resolves the check functions for a work-package type. Unknown types fall
// back to the "" entry. IDs with no registered check are skipped defensively
// (LoadDoR validates a config up front so this cannot happen for loaded files).
func (c *DoRConfig) Rules(typeName string) []CheckFunc {
	ids, ok := c.Types[canonicalType(typeName)]
	if !ok {
		ids = c.Types[""]
	}
	funcs := make([]CheckFunc, 0, len(ids))
	for _, id := range ids {
		if f, ok := registry[id]; ok {
			funcs = append(funcs, f)
		}
	}
	return funcs
}

// LoadDoR returns the DoR config to use. With OP_DOR_CONFIG unset it returns the
// baked-in default. When OP_DOR_CONFIG points to a file, that file is read,
// parsed, and validated; a missing file, bad JSON, or an unknown check ID is a
// loud error — never a silent fallback to defaults.
func LoadDoR() (*DoRConfig, error) {
	path := os.Getenv("OP_DOR_CONFIG")
	if path == "" {
		return defaultDoR, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading DoR config %s: %w", path, err)
	}
	var cfg DoRConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing DoR config %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid DoR config %s: %w", path, err)
	}
	return &cfg, nil
}

// validate rejects a config that references a check ID with no registered
// implementation, so a typo fails loudly at load rather than silently dropping a
// check at runtime.
func (c *DoRConfig) validate() error {
	if len(c.Types) == 0 {
		return fmt.Errorf("no types defined")
	}
	for typ, ids := range c.Types {
		for _, id := range ids {
			if _, ok := registry[id]; !ok {
				return fmt.Errorf("type %q references unknown check %q", typ, id)
			}
		}
	}
	return nil
}
