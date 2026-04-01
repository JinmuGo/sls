package container

// KeySep is the separator between host alias and container name in composite keys.
// Using "::" because "--" can appear in SSH host aliases.
const KeySep = "::"

// Shell detection results.
const (
	ShellUnknown = ""         // not yet tested
	ShellNone    = "none"     // no shell found
	ShellSh      = "/bin/sh"
	ShellBash    = "/bin/bash"
	ShellAsh     = "/bin/ash"
)

// Shells to try in order during detection.
var ShellCandidates = []string{ShellSh, ShellBash, ShellAsh}

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
	case ShellNone:
		return "no shell"
	case ShellSh:
		return "sh"
	case ShellBash:
		return "bash"
	case ShellAsh:
		return "ash"
	default:
		return ""
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
