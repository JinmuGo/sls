package container

import "path/filepath"

// KeySep is the separator between host alias and container name in composite keys.
// Using "::" because "--" can appear in SSH host aliases.
const KeySep = "::"

// Shell detection results.
const (
	ShellUnknown = ""     // not yet tested
	ShellNone    = "none" // no shell found
)

// ShellCandidates lists shell names to search via "command -v" during detection.
var ShellCandidates = []string{"bash", "sh", "ash"}

// Container represents a Docker container discovered on a remote host.
type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`           // original Docker container name
	Alias  string `json:"alias,omitempty"` // user-defined display name (defaults to Name)
	Image  string `json:"image"`
	Status string `json:"status"`
	Host   string `json:"host"`            // parent SSH host alias
	Shell  string `json:"shell,omitempty"` // detected shell path, empty = unknown, "none" = no shell
}

// ShellLabel returns a short label for display in the finder.
func (c Container) ShellLabel() string {
	switch c.Shell {
	case ShellUnknown:
		return ""
	case ShellNone:
		return "no shell"
	default:
		return filepath.Base(c.Shell)
	}
}

// DisplayName returns the alias if set, otherwise the original name.
func (c Container) DisplayName() string {
	if c.Alias != "" {
		return c.Alias
	}
	return c.Name
}

// Key returns the composite key for this container: "host::name".
func (c Container) Key() string {
	return c.Host + KeySep + c.Name
}
