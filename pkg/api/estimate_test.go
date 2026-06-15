package api

import "testing"

func TestParseEstimate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		// Why: the UI's "2d 0h" is 16h at 8h/day; the CLI must produce the same
		// hours the server stores, not an ISO P2D (which OpenProject would read
		// against a configurable, possibly non-8h day).
		{name: "days", in: "2d", want: "PT16H"},
		{name: "hours", in: "16h", want: "PT16H"},
		{name: "bare number is hours", in: "16", want: "PT16H"},
		{name: "days and hours", in: "2d 4h", want: "PT20H"},
		{name: "fractional day", in: "0.5d", want: "PT4H"},
		{name: "fractional hour to minutes", in: "1.5h", want: "PT1H30M"},
		{name: "minutes only", in: "0.5h", want: "PT30M"},
		{name: "no-space combined unsupported", in: "2d4h", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "zero", in: "0h", wantErr: true},
		{name: "garbage", in: "soon", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEstimate(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseEstimate(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEstimate(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseEstimate(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatEstimate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "PT16H", want: "2d 0h"},
		{in: "PT20H", want: "2d 4h"},
		{in: "PT4H", want: "4h"},
		{in: "PT1H30M", want: "1.5h"},
		{in: "", want: ""},
		{in: "garbage", want: ""},
	}
	for _, tt := range tests {
		if got := FormatEstimate(tt.in); got != tt.want {
			t.Errorf("FormatEstimate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
