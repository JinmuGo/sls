package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("creates file with correct content", func(t *testing.T) {
		path := filepath.Join(dir, "test1.txt")
		data := []byte("hello world")

		err := AtomicWriteFile(path, data, 0o644)
		if err != nil {
			t.Fatalf("AtomicWriteFile: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != string(data) {
			t.Errorf("content = %q, want %q", string(got), string(data))
		}
	})

	t.Run("sets correct permissions", func(t *testing.T) {
		path := filepath.Join(dir, "test2.txt")
		err := AtomicWriteFile(path, []byte("secret"), 0o600)
		if err != nil {
			t.Fatalf("AtomicWriteFile: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("permissions = %o, want 600", info.Mode().Perm())
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(dir, "test3.txt")
		AtomicWriteFile(path, []byte("first"), 0o644)
		AtomicWriteFile(path, []byte("second"), 0o644)

		got, _ := os.ReadFile(path)
		if string(got) != "second" {
			t.Errorf("content = %q, want %q", string(got), "second")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		path := filepath.Join(dir, "sub", "dir", "test4.txt")
		err := AtomicWriteFile(path, []byte("nested"), 0o644)
		if err != nil {
			t.Fatalf("AtomicWriteFile: %v", err)
		}

		got, _ := os.ReadFile(path)
		if string(got) != "nested" {
			t.Errorf("content = %q, want %q", string(got), "nested")
		}
	})

	t.Run("no temp file left on success", func(t *testing.T) {
		path := filepath.Join(dir, "test5.txt")
		AtomicWriteFile(path, []byte("clean"), 0o644)

		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".txt" && e.Name() != "sub" {
				t.Errorf("unexpected file in dir: %s (possible leftover temp file)", e.Name())
			}
		}
	})
}
