package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jinmugo/sls/internal/consts"
	"github.com/jinmugo/sls/internal/util"
	sshconfig "github.com/kevinburke/ssh_config"
)

func LoadAST(custom string) (*sshconfig.Config, string, error) {
	path := custom
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".ssh", "config")
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			sshDir := filepath.Dir(path)
			if _, statErr := os.Stat(sshDir); os.IsNotExist(statErr) {
				return nil, path, ErrSSHDirNotExist
			}
			return nil, path, ErrSSHConfigNotExist
		}
		return nil, path, err
	}
	defer f.Close()

	// Decode can return (nil, err) — e.g. when the config contains a Match
	// directive, which the parser does not support. Returning early avoids a
	// nil-pointer panic and surfaces the real error to the caller.
	cfg, err := sshconfig.Decode(f)
	if err != nil {
		return nil, path, err
	}

	// NOTE: the full config (including the implicit/explicit "Host *" block and
	// all global directives) is returned intact. SaveAST round-trips the whole
	// AST, so callers must not strip hosts here or those entries are lost on save.
	return cfg, path, nil
}

func SaveAST(cfg *sshconfig.Config, path string) error {
	// Use the library's faithful serializer so that global directives, the
	// "Host *" block, comments, Include directives, and multi-pattern Host lines
	// are all preserved. SSH config files should be 0600 — use an atomic write.
	return util.AtomicWriteFile(path, []byte(cfg.String()), 0o600)
}

// FormatConfig reads the SSH config file, formats it with consistent style, and writes it back.
// It preserves all hosts (including *), comments, and Include directives.
// A backup is created at <path>.bak before writing.
func FormatConfig(custom string) error {
	path := custom
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".ssh", "config")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	cfg, err := sshconfig.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Create backup
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read original file: %w", err)
	}
	bakPath := path + ".bak"
	if err := os.WriteFile(bakPath, original, 0o600); err != nil {
		return fmt.Errorf("failed to create backup at %s: %w", bakPath, err)
	}

	var buf bytes.Buffer
	for i, h := range cfg.Hosts {
		isImplicit := (i == 0 && len(h.Patterns) > 0 && h.Patterns[0].String() == "*")

		if !isImplicit {
			// Write Host line
			patterns := make([]string, len(h.Patterns))
			for j, p := range h.Patterns {
				patterns[j] = p.String()
			}
			buf.WriteString(fmt.Sprintf("Host %s", strings.Join(patterns, " ")))
			if h.EOLComment != "" {
				buf.WriteString(" #" + h.EOLComment)
			}
			buf.WriteByte('\n')
		}

		// Write nodes, trimming trailing empty lines
		nodes := h.Nodes
		for len(nodes) > 0 {
			if e, ok := nodes[len(nodes)-1].(*sshconfig.Empty); ok && e.Comment == "" {
				nodes = nodes[:len(nodes)-1]
			} else {
				break
			}
		}
		for _, n := range nodes {
			switch node := n.(type) {
			case *sshconfig.KV:
				indent := "    "
				if isImplicit {
					indent = ""
				}
				line := fmt.Sprintf("%s%s %s", indent, node.Key, node.Value)
				if node.Comment != "" {
					line += " #" + node.Comment
				}
				buf.WriteString(line)
				buf.WriteByte('\n')
			case *sshconfig.Empty:
				if node.Comment != "" {
					indent := "    "
					if isImplicit {
						indent = ""
					}
					buf.WriteString(fmt.Sprintf("%s#%s", indent, node.Comment))
				}
				buf.WriteByte('\n')
			case *sshconfig.Include:
				buf.WriteString(node.String())
				buf.WriteByte('\n')
			}
		}

		// Blank line between host blocks
		if i < len(cfg.Hosts)-1 {
			buf.WriteByte('\n')
		}
	}

	if err := util.AtomicWriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write formatted config: %w", err)
	}

	fmt.Printf("✓ Formatted %s (backup: %s)\n", path, bakPath)
	return nil
}

func FindHost(cfg *sshconfig.Config, alias string) (*sshconfig.Host, int) {
	for i, n := range cfg.Hosts {
		if n.Patterns[0].String() == alias {
			return n, i
		}
	}
	return nil, -1
}

func SetKV(h *sshconfig.Host, key, val string) {
	if val == "" {
		return
	}
	// SSH config keywords are case-insensitive; match an existing key regardless
	// of how the user capitalized it to avoid writing a duplicate directive.
	for _, n := range h.Nodes {
		if kv, ok := n.(*sshconfig.KV); ok && strings.EqualFold(kv.Key, key) {
			kv.Value = val
			return
		}
	}
	h.Nodes = append(h.Nodes, &sshconfig.KV{Key: key, Value: val})
}

func UpsertHost(cfg *sshconfig.Config, alias, hostname, user string, port int) (*sshconfig.Host, error) {
	h, _ := FindHost(cfg, alias)
	if h == nil {
		p, err := sshconfig.NewPattern(alias)
		if err != nil {
			return nil, fmt.Errorf("invalid alias pattern %q: %w", alias, err)
		}
		h = &sshconfig.Host{Patterns: []*sshconfig.Pattern{p}}
		cfg.Hosts = append(cfg.Hosts, h)
	}
	SetKV(h, consts.SSHConfigHostName, hostname)
	SetKV(h, consts.SSHConfigUser, user)
	if port > 0 {
		SetKV(h, consts.SSHConfigPort, fmt.Sprint(port))
	}
	return h, nil
}

func DeleteHost(cfg *sshconfig.Config, alias string) bool {
	if alias == "*" {
		return false
	}
	_, idx := FindHost(cfg, alias)
	if idx == -1 {
		return false
	}
	cfg.Hosts = append(cfg.Hosts[:idx], cfg.Hosts[idx+1:]...)
	return true
}

// EnsureSSHConfig creates ~/.ssh directory and ~/.ssh/config file if they don't exist.
// Returns the path to the config file.
func EnsureSSHConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	sshDir := filepath.Join(home, ".ssh")
	configPath := filepath.Join(sshDir, "config")

	// Check and create ~/.ssh directory
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		if err := os.MkdirAll(sshDir, 0o700); err != nil {
			return "", fmt.Errorf("failed to create %s: %w", sshDir, err)
		}
		fmt.Printf("\u2713 Created %s (mode: 700)\n", sshDir)
	}

	// Check and create ~/.ssh/config file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return "", fmt.Errorf("failed to create %s: %w", configPath, err)
		}
		f.Close()
		fmt.Printf("\u2713 Created %s (mode: 600)\n", configPath)
	}

	return configPath, nil
}

