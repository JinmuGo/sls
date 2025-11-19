package validator

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/jinmugo/sls/internal/config"
)

var (
	// Alias must be alphanumeric with optional hyphens, underscores, and dots
	aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
)

// ValidateAlias checks if the given alias is valid for SSH config
func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if alias == "*" {
		return fmt.Errorf("alias cannot be '*' (wildcard)")
	}
	if strings.Contains(alias, " ") {
		return fmt.Errorf("alias cannot contain spaces")
	}
	if !aliasRegex.MatchString(alias) {
		return fmt.Errorf("alias must contain only alphanumeric characters, hyphens, underscores, and dots")
	}
	if len(alias) > 255 {
		return fmt.Errorf("alias too long (max 255 characters)")
	}
	return nil
}

// ValidatePort checks if the port number is in valid range
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidateHostname checks if the hostname is valid
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Check if it's an IP address
	if net.ParseIP(hostname) != nil {
		return nil
	}

	// Check if it's a valid hostname
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long (max 253 characters)")
	}

	// Basic hostname validation
	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return fmt.Errorf("invalid hostname format: each label must be 1-63 characters")
		}
		// Allow alphanumeric and hyphens, but not starting/ending with hyphen
		if part[0] == '-' || part[len(part)-1] == '-' {
			return fmt.Errorf("invalid hostname format: labels cannot start or end with hyphen")
		}
	}

	return nil
}

// ValidateUser checks if the user name is valid
func ValidateUser(user string) error {
	if user == "" {
		return nil // User is optional in SSH config
	}
	if len(user) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}
	// Allow alphanumeric, underscore, hyphen
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(user) {
		return fmt.Errorf("username must contain only alphanumeric characters, underscores, and hyphens")
	}
	return nil
}

// ValidateHostExists checks if a host exists in SSH config.
// Returns an error if the host is not found in ~/.ssh/config.
func ValidateHostExists(host string) error {
	hosts, err := config.Parse("")
	if err != nil {
		return fmt.Errorf("parse ssh config: %w", err)
	}

	for _, h := range hosts {
		if len(h.Patterns) > 0 && h.Patterns[0].String() == host {
			return nil
		}
	}

	return fmt.Errorf("host %q not found in ~/.ssh/config", host)
}
