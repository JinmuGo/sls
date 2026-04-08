package hostinfo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jinmugo/sls/internal/util"
)

const (
	// DefaultTTL is the cache TTL for successful results.
	DefaultTTL = 1 * time.Hour
	// ErrorTTL is the cache TTL for error results.
	ErrorTTL = 5 * time.Minute
)

// Cache stores host info on disk as JSON.
type Cache struct {
	Hosts map[string]*HostInfo `json:"hosts"`
	path  string
}

// DefaultCachePath returns the default hostinfo cache path.
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "sls", "hostinfo.json"), nil
}

// LoadCache loads the hostinfo cache from disk.
func LoadCache(path string) (*Cache, error) {
	c := &Cache{
		Hosts: make(map[string]*HostInfo),
		path:  path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("read hostinfo cache: %w", err)
	}

	if err := json.Unmarshal(data, &c.Hosts); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: hostinfo cache corrupted, resetting: %v\n", err)
		c.Hosts = make(map[string]*HostInfo)
		return c, nil
	}

	return c, nil
}

// Save writes the cache to disk atomically.
func (c *Cache) Save() error {
	data, err := json.MarshalIndent(c.Hosts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hostinfo cache: %w", err)
	}
	return util.AtomicWriteFile(c.path, data, 0o600)
}

// Get returns cached info for a host, or nil if missing/stale.
func (c *Cache) Get(host string) *HostInfo {
	info, ok := c.Hosts[host]
	if !ok {
		return nil
	}

	ttl := DefaultTTL
	if info.Error != "" {
		ttl = ErrorTTL
	}
	if time.Since(info.FetchedAt) > ttl {
		return nil
	}
	info.FillMemoryFields()
	return info
}

// Set stores host info in the cache.
func (c *Cache) Set(host string, info *HostInfo) {
	c.Hosts[host] = info
}

// RenameHost moves cached info from old alias to new alias.
func (c *Cache) RenameHost(oldAlias, newAlias string) {
	info, ok := c.Hosts[oldAlias]
	if !ok {
		return
	}
	info.Hostname = newAlias
	c.Hosts[newAlias] = info
	delete(c.Hosts, oldAlias)
}

// DeleteHost removes cached info for a host.
func (c *Cache) DeleteHost(alias string) {
	delete(c.Hosts, alias)
}

// CacheAge returns a human-readable age for a host's cached data.
func (c *Cache) CacheAge(host string) string {
	info, ok := c.Hosts[host]
	if !ok {
		return ""
	}
	d := time.Since(info.FetchedAt)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
