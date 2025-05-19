package config

import (
	"io"
	"os"
	"path/filepath"

	sshconfig "github.com/kevinburke/ssh_config"
)

func Parse(path string) ([]*sshconfig.Host, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".ssh", "config")
	}
	f, err := os.Open(path)
	if err != nil {
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
	for _, n := range h.Nodes {
		if kv, ok := n.(*sshconfig.KV); ok && kv.Key == key {
			return kv.Value
		}
	}
	return ""
}
