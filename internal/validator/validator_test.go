package validator

import "testing"

func TestValidateAliasRejectsLeadingDash(t *testing.T) {
	// A leading '-' would be parsed by ssh as a command-line flag, not a host.
	for _, bad := range []string{"-foo", "-oProxyCommand", "-"} {
		if err := ValidateAlias(bad); err == nil {
			t.Errorf("ValidateAlias(%q) = nil, want error", bad)
		}
	}
	for _, good := range []string{"foo", "web-prod", "db_1", "a.b.c"} {
		if err := ValidateAlias(good); err != nil {
			t.Errorf("ValidateAlias(%q) = %v, want nil", good, err)
		}
	}
}

func TestValidateHostnameRejectsInjection(t *testing.T) {
	// Whitespace, newlines and special characters must be rejected — otherwise a
	// hostname could smuggle extra directives into ~/.ssh/config.
	bad := []string{
		"foo bar",
		"foo\n    ProxyCommand evil",
		"foo;rm -rf /",
		"foo\tbar",
		"foo/../bar",
		"foo$(whoami)",
	}
	for _, h := range bad {
		if err := ValidateHostname(h); err == nil {
			t.Errorf("ValidateHostname(%q) = nil, want error", h)
		}
	}
	good := []string{"example.com", "10.0.0.1", "web-prod", "my_host.internal", "::1", "2001:db8::1"}
	for _, h := range good {
		if err := ValidateHostname(h); err != nil {
			t.Errorf("ValidateHostname(%q) = %v, want nil", h, err)
		}
	}
}
