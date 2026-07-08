package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chenhuijun/op-cli/pkg/api"
)

// Well-formed is the QUS role+means criterion: full "As a … I want …" passes,
// a partial form warns, and neither warns (advisory — it never blocks the gate).
func TestCheckWellFormed(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"role and means", "As a user, I want to reset my password", Pass},
		{"role only", "As an admin, the dashboard should load", Warn},
		{"means only", "I want a faster checkout", Warn},
		{"neither", "Fix the broken login button", Warn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckWellFormed(makeWP("Feature", tt.desc), 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

// Well-formed also reads the dedicated User Story field, not just the description.
func TestCheckWellFormed_UsesUserStoryField(t *testing.T) {
	wp := makeWP("Feature", "Terse description line")
	wp.UserStory = &api.Formattable{Raw: "As a visitor I want to browse offline"}
	if r := CheckWellFormed(wp, 0); r.Level != Pass {
		t.Errorf("got %s, want Pass (msg: %s)", r.Level, r.Message)
	}
}

// Atomic is a heuristic: a conjunction warns (never fails); a single feature passes.
func TestCheckAtomic(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		level Level
	}{
		{"single feature", "As a user I want to export a report", Pass},
		{"conjunction", "As a user I want to export and email a report", Warn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckAtomic(makeWP("Feature", tt.desc), 0)
			if r.Level != tt.level {
				t.Errorf("got %s, want %s (msg: %s)", r.Level, tt.level, r.Message)
			}
		})
	}
}

// The default DoR must preserve the historical rule sets for bug and task
// exactly (parity), guarding against accidental drift when the config layer
// changed how rules are resolved.
func TestDefaultDoR_ParityForBugAndTask(t *testing.T) {
	if got := len(defaultDoR.Rules("bug")); got != 8 {
		t.Errorf("bug default has %d checks, want 8", got)
	}
	if got := len(defaultDoR.Rules("task")); got != 7 {
		t.Errorf("task default has %d checks, want 7", got)
	}
	// Unknown types fall back to the "" entry.
	if got := len(defaultDoR.Rules("Milestone")); got != len(defaultDoR.Types[""]) {
		t.Errorf("unknown type got %d checks, want fallback %d", got, len(defaultDoR.Types[""]))
	}
}

func TestLoadDoR_DefaultWhenUnset(t *testing.T) {
	t.Setenv("OP_DOR_CONFIG", "")
	cfg, err := LoadDoR()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != defaultDoR {
		t.Error("expected baked-in default when OP_DOR_CONFIG unset")
	}
}

func TestLoadDoR_ReadsAndValidatesFile(t *testing.T) {
	dir := t.TempDir()

	// A valid custom DoR: a lean bug bar plus the opt-in QUS atomic check.
	good := filepath.Join(dir, "dor.json")
	if err := os.WriteFile(good, []byte(`{"types":{"bug":["description","reproduction_steps"],"feature":["description","well_formed","atomic"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OP_DOR_CONFIG", good)
	cfg, err := LoadDoR()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(cfg.Rules("bug")); got != 2 {
		t.Errorf("custom bug rules = %d, want 2", got)
	}
	if got := len(cfg.Rules("story")); got != 3 { // story folds onto feature
		t.Errorf("story should fold onto feature (3 rules), got %d", got)
	}
}

func TestLoadDoR_FailsLoudOnBadConfig(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "nope.json")
	t.Setenv("OP_DOR_CONFIG", missing)
	if _, err := LoadDoR(); err == nil {
		t.Error("expected error for missing config file, got nil")
	}

	badJSON := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(badJSON, []byte(`{not json`), 0o600)
	t.Setenv("OP_DOR_CONFIG", badJSON)
	if _, err := LoadDoR(); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}

	unknownID := filepath.Join(dir, "unknown.json")
	_ = os.WriteFile(unknownID, []byte(`{"types":{"bug":["description","made_up_check"]}}`), 0o600)
	t.Setenv("OP_DOR_CONFIG", unknownID)
	if _, err := LoadDoR(); err == nil {
		t.Error("expected error for unknown check ID, got nil")
	}
}
