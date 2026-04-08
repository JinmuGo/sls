package hostinfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCacheNotExist(t *testing.T) {
	c, err := LoadCache("/nonexistent/path/hostinfo.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Hosts) != 0 {
		t.Errorf("expected empty hosts, got %d", len(c.Hosts))
	}
}

func TestLoadCacheCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hostinfo.json")
	os.WriteFile(path, []byte("{corrupt"), 0o600)

	c, err := LoadCache(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Hosts) != 0 {
		t.Errorf("expected empty hosts after corrupt, got %d", len(c.Hosts))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hostinfo.json")

	c := &Cache{
		Hosts: map[string]*HostInfo{
			"prod": {
				Hostname:  "prod",
				OS:        "Ubuntu 22.04",
				FetchedAt: time.Now(),
			},
		},
		path: path,
	}

	if err := c.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadCache(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Hosts["prod"] == nil {
		t.Fatal("expected 'prod' in loaded cache")
	}
	if loaded.Hosts["prod"].OS != "Ubuntu 22.04" {
		t.Errorf("OS = %q, want Ubuntu 22.04", loaded.Hosts["prod"].OS)
	}
}

func TestGetFreshAndStale(t *testing.T) {
	c := &Cache{
		Hosts: map[string]*HostInfo{
			"fresh": {
				Hostname:  "fresh",
				FetchedAt: time.Now(),
			},
			"stale": {
				Hostname:  "stale",
				FetchedAt: time.Now().Add(-2 * time.Hour),
			},
		},
	}

	if c.Get("fresh") == nil {
		t.Error("expected fresh entry to be returned")
	}
	if c.Get("stale") != nil {
		t.Error("expected stale entry to return nil")
	}
	if c.Get("missing") != nil {
		t.Error("expected missing entry to return nil")
	}
}

func TestGetErrorTTL(t *testing.T) {
	c := &Cache{
		Hosts: map[string]*HostInfo{
			"err-fresh": {
				Hostname:  "err-fresh",
				Error:     "connection failed",
				FetchedAt: time.Now().Add(-3 * time.Minute),
			},
			"err-stale": {
				Hostname:  "err-stale",
				Error:     "connection failed",
				FetchedAt: time.Now().Add(-10 * time.Minute),
			},
		},
	}

	if c.Get("err-fresh") == nil {
		t.Error("expected fresh error to be returned (within 5min)")
	}
	if c.Get("err-stale") != nil {
		t.Error("expected stale error to return nil (beyond 5min)")
	}
}

func TestRenameHost(t *testing.T) {
	c := &Cache{
		Hosts: map[string]*HostInfo{
			"old": {Hostname: "old", OS: "Ubuntu"},
		},
	}

	c.RenameHost("old", "new")

	if c.Hosts["old"] != nil {
		t.Error("old alias should be removed")
	}
	if c.Hosts["new"] == nil {
		t.Fatal("new alias should exist")
	}
	if c.Hosts["new"].Hostname != "new" {
		t.Errorf("hostname = %q, want new", c.Hosts["new"].Hostname)
	}
}

func TestDeleteHost(t *testing.T) {
	c := &Cache{
		Hosts: map[string]*HostInfo{
			"host": {Hostname: "host"},
		},
	}

	c.DeleteHost("host")
	if c.Hosts["host"] != nil {
		t.Error("host should be deleted")
	}
}

func TestCacheAge(t *testing.T) {
	c := &Cache{
		Hosts: map[string]*HostInfo{
			"recent": {FetchedAt: time.Now().Add(-30 * time.Second)},
			"old":    {FetchedAt: time.Now().Add(-45 * time.Minute)},
		},
	}

	if age := c.CacheAge("recent"); age == "" {
		t.Error("expected non-empty age for recent")
	}
	if age := c.CacheAge("missing"); age != "" {
		t.Errorf("expected empty age for missing, got %q", age)
	}
}
