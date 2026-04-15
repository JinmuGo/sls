package container

import "testing"

func TestShellLabel(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{ShellUnknown, ""},
		{ShellNone, "no shell"},
		{"/bin/sh", "sh"},
		{"/bin/bash", "bash"},
		{"/bin/ash", "ash"},
		{"/usr/bin/bash", "bash"},
		{"/usr/local/bin/fish", "fish"},
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
			containers[i].Shell = "/usr/bin/bash"
		}
	}
	cache.Update("server", containers)
	cache.Save()

	// Reload and verify
	cache2, _ := LoadCache(path)
	for _, c := range cache2.GetContainers("server") {
		if c.Name == "nginx" && c.Shell != "/usr/bin/bash" {
			t.Errorf("expected nginx shell %q, got %q", "/usr/bin/bash", c.Shell)
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
	expected := []string{"bash", "sh", "ash"}
	for i, want := range expected {
		if ShellCandidates[i] != want {
			t.Errorf("candidate[%d] should be %s, got %s", i, want, ShellCandidates[i])
		}
	}
}
