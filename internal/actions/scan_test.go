package actions

import (
	"path/filepath"
	"testing"

	"github.com/jinmugo/sls/internal/container"
)

func TestScanPreCheckedBuildsCorrectly(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "containers.json")

	cache, _ := container.LoadCache(cachePath)

	// Simulate: user previously scanned and selected nginx + postgres
	cache.Update("my-server", []container.Container{
		{Name: "nginx", Image: "nginx:alpine", Host: "my-server"},
		{Name: "postgres", Image: "postgres:16", Host: "my-server"},
	})

	// Now simulate new discovery: nginx still running, postgres stopped, redis new
	discovered := []container.Container{
		{Name: "nginx", Image: "nginx:alpine", Host: "my-server"},
		{Name: "redis", Image: "redis:7", Host: "my-server"},
	}

	// Build preChecked list (same logic as scan.go)
	existing := cache.GetContainers("my-server")
	existingSet := make(map[string]bool, len(existing))
	for _, c := range existing {
		existingSet[c.Name] = true
	}

	var preChecked []string
	for _, c := range discovered {
		if existingSet[c.Name] {
			preChecked = append(preChecked, c.Name)
		}
	}

	// nginx was previously selected and is still running → should be pre-checked
	if len(preChecked) != 1 {
		t.Fatalf("expected 1 pre-checked item, got %d: %v", len(preChecked), preChecked)
	}
	if preChecked[0] != "nginx" {
		t.Errorf("expected 'nginx' to be pre-checked, got %q", preChecked[0])
	}
}

func TestScanPreCheckedEmptyCacheReturnsNone(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "containers.json")

	cache, _ := container.LoadCache(cachePath)

	existing := cache.GetContainers("my-server")
	if len(existing) != 0 {
		t.Fatalf("expected empty existing, got %d", len(existing))
	}

	discovered := []container.Container{
		{Name: "nginx"},
		{Name: "redis"},
	}

	existingSet := make(map[string]bool)
	var preChecked []string
	for _, c := range discovered {
		if existingSet[c.Name] {
			preChecked = append(preChecked, c.Name)
		}
	}

	if len(preChecked) != 0 {
		t.Errorf("expected 0 pre-checked for fresh host, got %d", len(preChecked))
	}
}

// TestScanPreCheckedAfterCacheRoundtrip verifies that after saving cache to disk
// and reloading it, existing containers are found for pre-checking.
func TestScanPreCheckedAfterCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "containers.json")

	// First scan: save nginx and postgres
	cache, _ := container.LoadCache(cachePath)
	cache.MergeUpdate("my-server", []container.Container{
		{Name: "nginx", Image: "nginx:alpine", Host: "my-server"},
		{Name: "postgres", Image: "postgres:16", Host: "my-server"},
	})
	if err := cache.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Simulate root.go reload: load cache from disk again
	cache2, err := container.LoadCache(cachePath)
	if err != nil {
		t.Fatalf("LoadCache after save: %v", err)
	}

	// Second scan: redis is new, nginx still running, postgres stopped
	discovered := []container.Container{
		{Name: "nginx", Image: "nginx:alpine", Host: "my-server"},
		{Name: "redis", Image: "redis:7", Host: "my-server"},
	}

	existing := cache2.GetContainers("my-server")
	if len(existing) != 2 {
		t.Fatalf("expected 2 existing containers from disk, got %d", len(existing))
	}

	existingSet := make(map[string]bool, len(existing))
	for _, c := range existing {
		existingSet[c.Name] = true
	}

	var preChecked []string
	for _, c := range discovered {
		if existingSet[c.Name] {
			preChecked = append(preChecked, c.Name)
		}
	}

	// nginx should be pre-checked (was in cache and still discovered)
	if len(preChecked) != 1 || preChecked[0] != "nginx" {
		t.Errorf("expected ['nginx'] pre-checked, got %v", preChecked)
	}
}
