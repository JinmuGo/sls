# Changelog

## [0.1.0.0] - 2026-04-01 — Shell Router v1

Discover Docker containers on your remote servers and open a shell in one keystroke. The interactive selector now handles both SSH hosts and containers in a unified list.

### Added

- **Container discovery.** Press ctrl+s on any host to scan for running Docker containers via SSH. Select which ones to keep, give them friendly names, and they appear nested under their host in the main list.
- **Shell detection with fallback.** On first connect, sls tries /bin/sh, /bin/bash, and /bin/ash in order. The working shell is cached so subsequent connects are instant. Distroless containers (no shell) are detected and marked.
- **Custom Bubbletea TUI.** Replaced the external fzf dependency with a built-in fuzzy finder. Supports tree-style container nesting (├─/└─), delete confirmation (y/n), status flash messages, adaptive hint bar, and cursor position restoration.
- **Multi-select with memory.** When re-scanning a host, previously selected containers are pre-checked.
- **Container management.** Rename (ctrl+r), delete (ctrl+d), and star (ctrl+f) work for both hosts and containers. Host renames cascade to container cache and favorites.
- **SSH config generation.** `sls gen ssh-config` creates an includable SSH config using ProxyJump to inherit all parent host settings automatically.
- **`sls connect` and `sls discover` subcommands** for scriptable access to container connections and discovery.

### Changed

- **Finder upgraded.** Moved from go-fuzzyfinder to charmbracelet/bubbletea + lipgloss with sahilm/fuzzy for matching. Shared color palette in styles.go.
- **Atomic file writes.** SSH config and container cache writes use temp-file-then-rename to prevent corruption.
- **SSH exit code handling.** Exit code 255 is only suppressed for interactive sessions, not for docker exec (where it indicates a real failure).
- **Discovery security.** Container scanning no longer writes to known_hosts (StrictHostKeyChecking=no + UserKnownHostsFile=/dev/null).
- **Separator change.** Container composite keys use `::` instead of `--` to avoid conflicts with SSH host aliases containing dashes.

### Fixed

- **Stale cache calculation.** Fixed duration bug that marked caches as stale after ~17 minutes instead of the intended 1 hour.
