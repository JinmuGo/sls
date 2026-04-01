package container

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	// Load empty cache (file doesn't exist)
	cache, err := LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if len(cache.Hosts) != 0 {
		t.Errorf("expected empty cache, got %d hosts", len(cache.Hosts))
	}

	// Update and save
	cache.Update("my-server", []Container{
		{ID: "abc", Name: "nginx", Image: "nginx:alpine", Status: "Up 3 days", Host: "my-server"},
		{ID: "def", Name: "postgres", Image: "postgres:16", Status: "Up 3 days", Host: "my-server"},
	})
	if err := cache.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload and verify
	cache2, err := LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache after save: %v", err)
	}
	containers := cache2.GetContainers("my-server")
	if len(containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].Name != "nginx" {
		t.Errorf("container[0].Name = %q, want %q", containers[0].Name, "nginx")
	}
}

func TestCacheCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache, err := LoadCache(path)
	if err != nil {
		t.Fatalf("expected no error on corrupt cache, got: %v", err)
	}
	if len(cache.Hosts) != 0 {
		t.Errorf("expected empty cache after corruption, got %d hosts", len(cache.Hosts))
	}
}

func TestCacheIsStale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)

	// Missing host is stale
	if !cache.IsStale("unknown", time.Hour) {
		t.Error("expected missing host to be stale")
	}

	// Fresh cache is not stale
	cache.Update("my-server", nil)
	if cache.IsStale("my-server", time.Hour) {
		t.Error("expected fresh cache to not be stale")
	}

	// Simulate old cache
	hc := cache.Hosts["my-server"]
	hc.UpdatedAt = time.Now().Add(-2 * time.Hour)
	cache.Hosts["my-server"] = hc
	if !cache.IsStale("my-server", time.Hour) {
		t.Error("expected 2h old cache to be stale with 1h TTL")
	}
}

func TestCacheAllContainers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)
	cache.Update("server-1", []Container{
		{Name: "nginx", Host: "server-1"},
	})
	cache.Update("server-2", []Container{
		{Name: "postgres", Host: "server-2"},
		{Name: "redis", Host: "server-2"},
	})

	all := cache.AllContainers()
	if len(all) != 3 {
		t.Errorf("expected 3 total containers, got %d", len(all))
	}
}

func TestCacheRemoveStaleHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)
	cache.Update("active", []Container{{Name: "app"}})
	cache.Update("removed", []Container{{Name: "old"}})

	cache.RemoveStaleHosts([]string{"active"})

	if _, ok := cache.Hosts["removed"]; ok {
		t.Error("expected 'removed' host to be cleaned up")
	}
	if _, ok := cache.Hosts["active"]; !ok {
		t.Error("expected 'active' host to remain")
	}
}

func TestCacheMergeUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)

	// Set up existing containers with aliases
	cache.Update("server", []Container{
		{Name: "nginx", Alias: "web", Host: "server"},
		{Name: "postgres", Alias: "db", Host: "server"},
	})

	// Merge with new discovery (nginx still running, postgres stopped, redis new)
	discovered := []Container{
		{Name: "nginx", Image: "nginx:latest", Host: "server"},
		{Name: "redis", Image: "redis:7", Host: "server"},
	}

	cache.MergeUpdate("server", discovered)
	containers := cache.GetContainers("server")

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	// nginx should keep its alias
	var nginx *Container
	for i, c := range containers {
		if c.Name == "nginx" {
			nginx = &containers[i]
			break
		}
	}
	if nginx == nil {
		t.Fatal("expected nginx in merged containers")
	}
	if nginx.Alias != "web" {
		t.Errorf("expected nginx alias 'web', got %q", nginx.Alias)
	}

	// redis should have no alias (new)
	var redis *Container
	for i, c := range containers {
		if c.Name == "redis" {
			redis = &containers[i]
			break
		}
	}
	if redis == nil {
		t.Fatal("expected redis in merged containers")
	}
	if redis.Alias != "" {
		t.Errorf("expected empty alias for new container, got %q", redis.Alias)
	}
}

func TestCacheRenameHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)
	cache.Update("old-server", []Container{
		{Name: "nginx", Host: "old-server"},
	})

	cache.RenameHost("old-server", "new-server")

	// Old key should be gone
	if cache.GetContainers("old-server") != nil {
		t.Error("expected old host key to be removed")
	}

	// New key should have the container
	containers := cache.GetContainers("new-server")
	if len(containers) != 1 {
		t.Fatalf("expected 1 container under new key, got %d", len(containers))
	}
	if containers[0].Host != "new-server" {
		t.Errorf("expected container Host to be 'new-server', got %q", containers[0].Host)
	}
}

func TestCacheAge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "containers.json")

	cache, _ := LoadCache(path)

	if cache.CacheAge("unknown") != "no cache" {
		t.Errorf("expected 'no cache' for unknown host")
	}

	cache.Update("my-server", nil)
	age := cache.CacheAge("my-server")
	if age == "no cache" {
		t.Error("expected a valid age for cached host")
	}
}
