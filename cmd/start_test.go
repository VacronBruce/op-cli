package cmd

import "testing"

func TestBranchName(t *testing.T) {
	tests := []struct {
		name    string
		project string
		id      int
		subject string
		want    string
	}{
		{"simple", "app", 12345, "Crash on save", "app-12345-crash-on-save"},
		{"punctuation collapses", "web", 81477, "Document new CLI tool!", "web-81477-document-new-cli-tool"},
		{"mixed case and symbols", "app", 42, "Fix: API 500 @ /login", "app-42-fix-api-500-login"},
		{"leading/trailing junk trimmed", "app", 7, "  ...Hello...  ", "app-7-hello"},
		// A project name (vs identifier) is slugified the same way, so spaces and
		// case in the prefix are normalized too.
		{"project name slugified", "NTD App", 5, "Hi", "ntd-app-5-hi"},
		// Empty project identifier falls back to "wp" so the branch stays valid.
		{"empty project falls back to wp", "", 5, "Hello world", "wp-5-hello-world"},
		// Subject with no usable characters falls back to <project>-<id>.
		{"empty slug falls back to project+id", "app", 99, "！！！", "app-99"},
		// Long subjects are capped to 50 slug chars; this subject is built so the
		// cut lands exactly on a dash, proving the trailing dash gets trimmed.
		{"long subject capped, trailing dash trimmed", "app", 1,
			"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk",
			"app-1-aaaa-bbbb-cccc-dddd-eeee-ffff-gggg-hhhh-iiii-jjjj"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchName(tt.project, tt.id, tt.subject)
			if got != tt.want {
				t.Errorf("branchName(%q, %d, %q) = %q, want %q", tt.project, tt.id, tt.subject, got, tt.want)
			}
		})
	}
}
