package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jinmugo/sls/internal/consts"
)

// writeTempConfig writes content to a temp file and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

const sampleConfig = `# Global defaults
Compression yes
ServerAliveInterval 60

Host *
    ForwardAgent no

Host web web-prod
    HostName 10.0.0.1
    User deploy

Host db
    HostName 10.0.0.2
    User admin
`

// TestSaveASTPreservesGlobalsAndComments verifies that editing a single host
// via LoadAST/SaveAST does not destroy global directives, the explicit Host *
// block, comments, or secondary host patterns.
func TestSaveASTPreservesGlobalsAndComments(t *testing.T) {
	path := writeTempConfig(t, sampleConfig)

	cfg, p, err := LoadAST(path)
	if err != nil {
		t.Fatalf("LoadAST: %v", err)
	}

	h, _ := FindHost(cfg, "web")
	if h == nil {
		t.Fatal("FindHost(web) returned nil")
	}
	SetKV(h, consts.SSHConfigHostName, "10.9.9.9")

	if err := SaveAST(cfg, p); err != nil {
		t.Fatalf("SaveAST: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := string(out)

	mustContain := []string{
		"Compression yes",        // global directive (implicit host)
		"ServerAliveInterval 60", // global directive
		"Host *",                 // explicit wildcard block
		"ForwardAgent no",        // wildcard block body
		"web web-prod",           // secondary host pattern preserved
		"Host db",                // unrelated host preserved
		"10.9.9.9",               // the edit was applied
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("output lost %q.\n--- output ---\n%s", want, got)
		}
	}
	// The old HostName for web must be gone (replaced, not duplicated).
	if strings.Contains(got, "10.0.0.1") {
		t.Errorf("old web HostName 10.0.0.1 still present (duplicate KV?).\n%s", got)
	}
}

// TestDeleteHostPreservesRest verifies DeleteHost only removes the target.
func TestDeleteHostPreservesRest(t *testing.T) {
	path := writeTempConfig(t, sampleConfig)

	cfg, p, err := LoadAST(path)
	if err != nil {
		t.Fatalf("LoadAST: %v", err)
	}
	if !DeleteHost(cfg, "web") {
		t.Fatal("DeleteHost(web) returned false")
	}
	if err := SaveAST(cfg, p); err != nil {
		t.Fatalf("SaveAST: %v", err)
	}

	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "web-prod") {
		t.Errorf("web host not deleted:\n%s", s)
	}
	for _, want := range []string{"Compression yes", "Host *", "Host db"} {
		if !strings.Contains(s, want) {
			t.Errorf("delete lost unrelated content %q:\n%s", want, s)
		}
	}
}

// TestLoadASTMatchDirectiveDoesNotPanic verifies a Match block (which the
// underlying parser cannot decode) returns an error instead of panicking.
func TestLoadASTMatchDirectiveDoesNotPanic(t *testing.T) {
	path := writeTempConfig(t, "Host foo\n    HostName 1.2.3.4\n\nMatch host bar\n    User baz\n")

	cfg, _, err := LoadAST(path)
	if err == nil {
		t.Fatal("expected error for Match directive, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil cfg on error, got %+v", cfg)
	}
}

// TestGetSetKVCaseInsensitive verifies SSH keyword matching ignores case, as
// the ssh_config spec requires (keywords are case-insensitive).
func TestGetSetKVCaseInsensitive(t *testing.T) {
	path := writeTempConfig(t, "Host foo\n    hostname 1.2.3.4\n")

	cfg, p, err := LoadAST(path)
	if err != nil {
		t.Fatalf("LoadAST: %v", err)
	}
	h, _ := FindHost(cfg, "foo")
	if h == nil {
		t.Fatal("FindHost(foo) nil")
	}
	if v := GetKV(h, consts.SSHConfigHostName); v != "1.2.3.4" {
		t.Errorf("GetKV(HostName) = %q, want 1.2.3.4 (case-insensitive match failed)", v)
	}
	// Setting HostName must update the existing lowercase key, not append a dup.
	SetKV(h, consts.SSHConfigHostName, "5.6.7.8")
	if err := SaveAST(cfg, p); err != nil {
		t.Fatalf("SaveAST: %v", err)
	}
	out, _ := os.ReadFile(path)
	if n := strings.Count(strings.ToLower(string(out)), "hostname"); n != 1 {
		t.Errorf("expected exactly 1 hostname line, got %d:\n%s", n, out)
	}
	if !strings.Contains(string(out), "5.6.7.8") {
		t.Errorf("updated value not written:\n%s", out)
	}
}
