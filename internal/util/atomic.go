package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to a file atomically using temp-file-then-rename.
// The temp file is created in the target's own directory so the rename stays on
// the same filesystem. If the target is a symlink (common with dotfile
// managers, e.g. ~/.ssh/config), it is resolved first so the write goes through
// to the real file and the link itself is preserved.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Resolve symlinks for existing targets; EvalSymlinks errors for a path that
	// does not exist yet, in which case we keep the original path.
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".sls-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	defer func() {
		// Clean up temp file on failure
		os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}
	return nil
}
