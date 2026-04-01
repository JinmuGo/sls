package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jinmugo/sls/internal/util"
)

// Cache stores discovered container data locally.
type Cache struct {
	Hosts map[string]HostCache `json:"hosts"`
	path  string
}

// HostCache holds cached containers for a single host.
type HostCache struct {
	Containers []Container `json:"containers"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// DefaultCachePath returns the default cache file path.
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "sls", "containers.json"), nil
}

// LoadCache loads the container cache from disk. Returns an empty cache if the
// file doesn't exist or is corrupted.
func LoadCache(path string) (*Cache, error) {
	c := &Cache{
		Hosts: make(map[string]HostCache),
		path:  path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("read cache file: %w", err)
	}

	if err := json.Unmarshal(data, &c.Hosts); err != nil {
		// Corrupt cache — reset silently
		fmt.Fprintf(os.Stderr, "Warning: cache file corrupted, resetting: %v\n", err)
		c.Hosts = make(map[string]HostCache)
		return c, nil
	}

	return c, nil
}

// Save writes the cache to disk atomically.
func (c *Cache) Save() error {
	data, err := json.MarshalIndent(c.Hosts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	return util.AtomicWriteFile(c.path, data, 0o600)
}

// Update replaces the cached containers for a host.
func (c *Cache) Update(hostAlias string, containers []Container) {
	c.Hosts[hostAlias] = HostCache{
		Containers: containers,
		UpdatedAt:  time.Now(),
	}
}

// MergeUpdate merges newly discovered containers with existing cached data.
// Keeps aliases for containers that are still running, adds new containers,
// and marks stopped containers with empty status.
func (c *Cache) MergeUpdate(hostAlias string, discovered []Container) {
	existing := c.GetContainers(hostAlias)

	// Build lookup from existing containers by Docker name
	aliasMap := make(map[string]string) // Docker name → alias
	for _, e := range existing {
		if e.Alias != "" {
			aliasMap[e.Name] = e.Alias
		}
	}

	// Apply existing aliases to discovered containers
	for i, d := range discovered {
		if alias, ok := aliasMap[d.Name]; ok {
			discovered[i].Alias = alias
		}
	}

	c.Hosts[hostAlias] = HostCache{
		Containers: discovered,
		UpdatedAt:  time.Now(),
	}
}

// RenameHost moves all cached containers from oldAlias to newAlias.
func (c *Cache) RenameHost(oldAlias, newAlias string) {
	hc, ok := c.Hosts[oldAlias]
	if !ok {
		return
	}
	// Update Host field on each container
	for i := range hc.Containers {
		hc.Containers[i].Host = newAlias
	}
	c.Hosts[newAlias] = hc
	delete(c.Hosts, oldAlias)
}

// GetContainers returns cached containers for a host.
func (c *Cache) GetContainers(hostAlias string) []Container {
	if hc, ok := c.Hosts[hostAlias]; ok {
		return hc.Containers
	}
	return nil
}

// AllContainers returns all cached containers across all hosts.
func (c *Cache) AllContainers() []Container {
	var all []Container
	for _, hc := range c.Hosts {
		all = append(all, hc.Containers...)
	}
	return all
}

// IsStale returns true if the cache for the given host is older than maxAge,
// or if the host is not in the cache.
func (c *Cache) IsStale(hostAlias string, maxAge time.Duration) bool {
	hc, ok := c.Hosts[hostAlias]
	if !ok {
		return true
	}
	return time.Since(hc.UpdatedAt) > maxAge
}

// RemoveStaleHosts removes cached entries for hosts not in the provided list.
func (c *Cache) RemoveStaleHosts(validHosts []string) {
	valid := make(map[string]bool)
	for _, h := range validHosts {
		valid[h] = true
	}
	for host := range c.Hosts {
		if !valid[host] {
			delete(c.Hosts, host)
		}
	}
}

// CacheAge returns a human-readable age string for a host's cache.
func (c *Cache) CacheAge(hostAlias string) string {
	hc, ok := c.Hosts[hostAlias]
	if !ok {
		return "no cache"
	}
	d := time.Since(hc.UpdatedAt)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
