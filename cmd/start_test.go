package cmd

import "testing"

func TestBranchName(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		subject string
		want    string
	}{
		{"simple", 12345, "Crash on save", "wp-12345-crash-on-save"},
		{"punctuation collapses", 81477, "Document new CLI tool!", "wp-81477-document-new-cli-tool"},
		{"mixed case and symbols", 42, "Fix: API 500 @ /login", "wp-42-fix-api-500-login"},
		{"leading/trailing junk trimmed", 7, "  ...Hello...  ", "wp-7-hello"},
		// Subject with no usable characters falls back to id-only so the branch
		// is still valid and unique.
		{"empty slug falls back to id only", 99, "！！！", "wp-99"},
		// Long subjects are capped to 50 slug chars; this subject is built so the
		// cut lands exactly on a dash, proving the trailing dash gets trimmed.
		{"long subject capped, trailing dash trimmed", 1,
			"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk",
			"wp-1-aaaa-bbbb-cccc-dddd-eeee-ffff-gggg-hhhh-iiii-jjjj"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchName(tt.id, tt.subject)
			if got != tt.want {
				t.Errorf("branchName(%d, %q) = %q, want %q", tt.id, tt.subject, got, tt.want)
			}
		})
	}
}
