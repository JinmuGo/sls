package container

import "regexp"

// validNamePattern allows alphanumeric, dots, hyphens, and underscores.
// This prevents SSH config injection via malicious container names.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateName checks if a container name is safe to use in SSH config
// and shell commands. Also used for user-provided aliases.
func ValidateName(name string) bool {
	if name == "" {
		return false
	}
	return validNamePattern.MatchString(name)
}
