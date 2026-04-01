package container

import (
	"testing"
)

func TestParseDockerOutput(t *testing.T) {
	tests := []struct {
		name           string
		hostAlias      string
		output         string
		expectedCount  int
		expectedNames  []string
	}{
		{
			name:      "normal output",
			hostAlias: "my-server",
			output:    "abc123|||nginx|||nginx:alpine|||Up 3 days\ndef456|||postgres|||postgres:16|||Up 3 days\n",
			expectedCount: 2,
			expectedNames: []string{"nginx", "postgres"},
		},
		{
			name:          "empty output",
			hostAlias:     "my-server",
			output:        "",
			expectedCount: 0,
		},
		{
			name:          "only whitespace",
			hostAlias:     "my-server",
			output:        "\n  \n\n",
			expectedCount: 0,
		},
		{
			name:      "malformed line skipped",
			hostAlias: "my-server",
			output:    "WARNING: something deprecated\nabc123|||nginx|||nginx:alpine|||Up 3 days\n",
			expectedCount: 1,
			expectedNames: []string{"nginx"},
		},
		{
			name:      "unsafe container name skipped",
			hostAlias: "my-server",
			output:    "abc123|||nginx|||nginx:alpine|||Up 3 days\ndef456|||evil;rm -rf /|||malicious:latest|||Up 1 hour\n",
			expectedCount: 1,
			expectedNames: []string{"nginx"},
		},
		{
			name:      "host alias propagated",
			hostAlias: "prod-01",
			output:    "abc123|||app|||app:latest|||Up 1 hour\n",
			expectedCount: 1,
		},
		{
			name:          "all lines malformed",
			hostAlias:     "my-server",
			output:        "not a valid line\nalso bad\n",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers, err := parseDockerOutput(tt.hostAlias, tt.output, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(containers) != tt.expectedCount {
				t.Errorf("expected %d containers, got %d", tt.expectedCount, len(containers))
			}
			for i, name := range tt.expectedNames {
				if i < len(containers) && containers[i].Name != name {
					t.Errorf("container[%d].Name = %q, want %q", i, containers[i].Name, name)
				}
			}
			// Verify host alias is set
			for _, c := range containers {
				if c.Host != tt.hostAlias {
					t.Errorf("container.Host = %q, want %q", c.Host, tt.hostAlias)
				}
			}
		})
	}
}
