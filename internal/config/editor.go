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
	return os.WriteFile(path, buf.Bytes(), 0o644)
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

func UpsertHost(cfg *sshconfig.Config, alias, hostname, user string, port int) *sshconfig.Host {
	h, _ := FindHost(cfg, alias)
	if h == nil {
		h = &sshconfig.Host{Patterns: []*sshconfig.Pattern{mustPattern(alias)}}
		cfg.Hosts = append(cfg.Hosts, h)
	}
	SetKV(h, consts.SSHConfigHostName, hostname)
	SetKV(h, consts.SSHConfigUser, user)
	if port > 0 {
		SetKV(h, consts.SSHConfigPort, fmt.Sprint(port))
	}
	return h
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

func mustPattern(s string) *sshconfig.Pattern {
	p, err := sshconfig.NewPattern(s)
	if err != nil {
		panic(err)
	}
	return p
}
