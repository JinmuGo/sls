package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	sshconfig "github.com/kevinburke/ssh_config"
)

func Parse(path string) ([]*sshconfig.Host, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".ssh", "config")
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			sshDir := filepath.Dir(path)
			if _, statErr := os.Stat(sshDir); os.IsNotExist(statErr) {
				return nil, ErrSSHDirNotExist
			}
			return nil, ErrSSHConfigNotExist
		}
		return nil, err
	}
	defer f.Close()
	return parseReader(f)
}

func parseReader(r io.Reader) ([]*sshconfig.Host, error) {
	cfg, err := sshconfig.Decode(r)
	if err != nil {
		return nil, err
	}
	hosts := append([]*sshconfig.Host{}, cfg.Hosts...)

	return hosts, nil
}

func GetKV(h *sshconfig.Host, key string) string {
	// SSH config keywords are case-insensitive, so match regardless of the
	// capitalization used in the file (e.g. "hostname" vs "HostName").
	for _, n := range h.Nodes {
		if kv, ok := n.(*sshconfig.KV); ok && strings.EqualFold(kv.Key, key) {
			return kv.Value
		}
	}
	return ""
}
