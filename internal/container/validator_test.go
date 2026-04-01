package container

import "testing"

func TestValidateName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple", "nginx", true},
		{"with hyphen", "my-app", true},
		{"with underscore", "app_v2", true},
		{"with dot", "my.service", true},
		{"with numbers", "app123", true},
		{"mixed", "My-App_v2.0", true},
		{"empty", "", false},
		{"with space", "my app", false},
		{"with newline", "nginx\nProxyCommand evil", false},
		{"with semicolon", "app;rm -rf /", false},
		{"with pipe", "app|cat /etc/passwd", false},
		{"with backtick", "app`whoami`", false},
		{"with dollar", "app$HOME", false},
		{"with slash", "app/sub", false},
		{"with tab", "app\tsecond", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateName(tt.input)
			if got != tt.expected {
				t.Errorf("ValidateName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
