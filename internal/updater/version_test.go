package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "1.1.1", "1.1.1", 0},
		{"equal with v prefix on one side", "v1.1.1", "1.1.1", 0},
		{"equal with v prefix on both", "v1.1.1", "v1.1.1", 0},
		{"older patch", "1.1.0", "1.1.1", -1},
		{"newer patch", "1.1.2", "1.1.1", 1},
		{"minor beats patch", "1.2.0", "1.1.9", 1},
		{"major beats minor", "2.0.0", "1.9.9", 1},
		{"missing patch treated as zero", "1.1", "1.1.0", 0},
		{"missing minor treated as zero", "1", "1.0.0", 0},
		{"double digit patch", "1.1.10", "1.1.9", 1},
		{"prerelease lower than release", "1.1.0-rc1", "1.1.0", -1},
		{"release higher than prerelease", "1.1.0", "1.1.0-rc1", 1},
		{"prerelease single digit ordering", "1.1.0-rc1", "1.1.0-rc2", -1},
		{"prerelease multi digit ordering", "1.1.0-rc9", "1.1.0-rc10", -1},
		{"prerelease multi digit reverse", "1.1.0-rc10", "1.1.0-rc2", 1},
		{"prerelease equal", "1.1.0-rc3", "1.1.0-rc3", 0},
		{"prerelease dotted numeric identifiers", "1.0.0-alpha.2", "1.0.0-alpha.10", -1},
		{"prerelease numeric lower than alpha identifier", "1.0.0-1", "1.0.0-alpha", -1},
		{"prerelease fewer identifiers lower", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"prerelease alpha lexical", "1.0.0-alpha", "1.0.0-beta", -1},
		{"v prefixed newer", "v1.1.2", "v1.1.1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
