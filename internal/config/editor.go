package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinmugo/sls/internal/consts"
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

	cfg, err := sshconfig.Decode(f)

	hosts := []*sshconfig.Host{}
	for _, h := range cfg.Hosts {
		if len(h.Patterns) > 0 && h.Patterns[0].String() != "*" {
			hosts = append(hosts, h)
		}
	}
	cfg.Hosts = hosts

	return cfg, path, err
}

func SaveAST(cfg *sshconfig.Config, path string) error {
	var buf bytes.Buffer
	for _, h := range cfg.Hosts {
		buf.WriteString(fmt.Sprintf("Host %s\n", h.Patterns[0].String()))
		for _, n := range h.Nodes {
			if kv, ok := n.(*sshconfig.KV); ok {
				buf.WriteString(fmt.Sprintf("    %s\t%s\n", kv.Key, kv.Value))
			}
		}
		buf.WriteString("\n")
	}
	// SSH config files should be 0600 for security
	return os.WriteFile(path, buf.Bytes(), 0o600)
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
	for _, n := range h.Nodes {
		if kv, ok := n.(*sshconfig.KV); ok && kv.Key == key {
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

