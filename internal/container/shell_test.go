package container

import "testing"

func TestShellLabel(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{ShellUnknown, ""},
		{ShellNone, "no shell"},
		{ShellSh, "sh"},
		{ShellBash, "bash"},
		{ShellAsh, "ash"},
	}

	for _, tt := range tests {
		c := Container{Shell: tt.shell}
		got := c.ShellLabel()
		if got != tt.want {
			t.Errorf("ShellLabel() for %q = %q, want %q", tt.shell, got, tt.want)
		}
	}
}

func TestSaveShellToCache(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/containers.json"

	cache, _ := LoadCache(path)
	cache.Update("server", []Container{
		{Name: "nginx", Host: "server"},
		{Name: "redis", Host: "server"},
	})

	// Update nginx's shell
	containers := cache.GetContainers("server")
	for i, c := range containers {
		if c.Name == "nginx" {
			containers[i].Shell = ShellBash
		}
	}
	cache.Update("server", containers)
	cache.Save()

	// Reload and verify
	cache2, _ := LoadCache(path)
	for _, c := range cache2.GetContainers("server") {
		if c.Name == "nginx" && c.Shell != ShellBash {
			t.Errorf("expected nginx shell %q, got %q", ShellBash, c.Shell)
		}
		if c.Name == "redis" && c.Shell != ShellUnknown {
			t.Errorf("expected redis shell empty, got %q", c.Shell)
		}
	}
}

func TestShellCandidatesOrder(t *testing.T) {
	// Verify the fallback order is bash → sh → ash
	if len(ShellCandidates) != 3 {
		t.Fatalf("expected 3 shell candidates, got %d", len(ShellCandidates))
	}
	if ShellCandidates[0] != ShellBash {
		t.Errorf("first candidate should be /bin/bash, got %s", ShellCandidates[0])
	}
	if ShellCandidates[1] != ShellSh {
		t.Errorf("second candidate should be /bin/sh, got %s", ShellCandidates[1])
	}
	if ShellCandidates[2] != ShellAsh {
		t.Errorf("third candidate should be /bin/ash, got %s", ShellCandidates[2])
	}
}
