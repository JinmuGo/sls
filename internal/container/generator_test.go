package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddIncludeLine(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "config")
	includePath := filepath.Join(dir, "sls", "ssh_config")

	t.Run("adds Include to existing config", func(t *testing.T) {
		original := "Host my-server\n    HostName 10.0.0.1\n    User root\n"
		os.WriteFile(sshConfig, []byte(original), 0o600)

		err := AddIncludeLine(sshConfig, includePath)
		if err != nil {
			t.Fatalf("AddIncludeLine: %v", err)
		}

		data, _ := os.ReadFile(sshConfig)
		content := string(data)
		if !strings.HasPrefix(content, "Include "+includePath) {
			t.Errorf("expected Include at top, got:\n%s", content)
		}
		if !strings.Contains(content, "Host my-server") {
			t.Error("original content should be preserved")
		}
	})

	t.Run("idempotent — second call is no-op", func(t *testing.T) {
		dataBefore, _ := os.ReadFile(sshConfig)
		err := AddIncludeLine(sshConfig, includePath)
		if err != nil {
			t.Fatalf("AddIncludeLine (second): %v", err)
		}
		dataAfter, _ := os.ReadFile(sshConfig)
		if string(dataBefore) != string(dataAfter) {
			t.Error("second call should not modify the file")
		}
	})

	t.Run("error on missing config", func(t *testing.T) {
		err := AddIncludeLine(filepath.Join(dir, "nonexistent"), includePath)
		if err == nil {
			t.Error("expected error for missing config")
		}
	})
}

func TestGenerateIncludeFile(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "ssh_config")
	cachePath := filepath.Join(dir, "containers.json")

	t.Run("empty cache returns error", func(t *testing.T) {
		cache, _ := LoadCache(cachePath)
		err := GenerateIncludeFile(cache, outputPath)
		if err == nil {
			t.Error("expected error for empty cache")
		}
	})

	t.Run("generates valid SSH config", func(t *testing.T) {
		cache, _ := LoadCache(cachePath)
		cache.Update("my-server", []Container{
			{ID: "abc", Name: "nginx", Image: "nginx:alpine", Status: "Up", Host: "my-server"},
			{ID: "def", Name: "postgres", Image: "postgres:16", Status: "Up", Host: "my-server"},
		})
		cache.Hosts["my-server"] = HostCache{
			Containers: cache.Hosts["my-server"].Containers,
			UpdatedAt:  time.Now(),
		}

		err := GenerateIncludeFile(cache, outputPath)
		if err != nil {
			t.Fatalf("GenerateIncludeFile: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		content := string(data)
		if !strings.Contains(content, "Host my-server"+KeySep+"nginx") {
			t.Errorf("expected Host my-server%snginx in generated config, got:\n%s", KeySep, content)
		}
		if !strings.Contains(content, "ProxyJump my-server") {
			t.Error("expected ProxyJump my-server in generated config")
		}
		if !strings.Contains(content, "RemoteCommand docker exec -it nginx /bin/sh") {
			t.Error("expected RemoteCommand in generated config")
		}
		if !strings.Contains(content, "RequestTTY yes") {
			t.Error("expected RequestTTY in generated config")
		}
	})
}
