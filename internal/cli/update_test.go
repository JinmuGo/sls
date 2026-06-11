package cli

import (
	"strings"
	"testing"

	"github.com/jinmugo/sls/internal/updater"
)

func TestFormatCheck(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		ok      bool
		want    string // substring that must appear
	}{
		{"network failure", "1.1.1", "", false, "could not determine"},
		{"update available", "1.1.1", "v1.1.2", true, "1.1.1 → v1.1.2"},
		{"up to date", "1.1.2", "v1.1.2", true, "up to date"},
		{"current newer", "1.2.0", "v1.1.2", true, "up to date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCheck(tt.current, tt.latest, tt.ok)
			if !strings.Contains(strings.ToLower(got), strings.ToLower(tt.want)) {
				t.Errorf("formatCheck(%q, %q, %v) = %q, want substring %q",
					tt.current, tt.latest, tt.ok, got, tt.want)
			}
		})
	}
}

func TestManualInstructions(t *testing.T) {
	for _, m := range []updater.Method{updater.MethodBinary, updater.MethodUnknown} {
		got := manualInstructions(m)
		if !strings.Contains(got, updater.ReleasesURL) {
			t.Errorf("manualInstructions(%v) = %q, want it to include releases URL %q",
				m, got, updater.ReleasesURL)
		}
	}
}
