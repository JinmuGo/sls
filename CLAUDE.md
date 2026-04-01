# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`sls` (ssh ls) is a smart fuzzy CLI selector for SSH config hosts and Docker containers written in Go. It provides an interactive Bubbletea TUI for selecting and connecting to SSH hosts and remote containers, with fuzzy search, usage tracking, favorites, container discovery, and shell detection.

## Build and Development Commands

### Building
```bash
go build -o sls .
```

### Running locally
```bash
go run . [command]
```

### Testing installation
```bash
go install github.com/jinmugo/sls@latest
```

### Release build
Uses GoReleaser for cross-platform builds and Homebrew tap publishing:
```bash
goreleaser release --clean
```

## Architecture

### Core Components

**Entry Point (`cmd/root.go`)**
- Main command orchestration using cobra
- Interactive mode runs a main loop: show finder → handle action → loop back
- Builds sorted item list: favorited containers → favorited hosts → normal hosts (with nested containers)
- Selective reload: only re-reads SSH config, favorites, or cache when the previous action modified them

**SSH Config Parsing (`internal/config/`)**
- `config.go`: Parses `~/.ssh/config` using `github.com/kevinburke/ssh_config`
- `editor.go`: AST manipulation for add/edit/remove operations via `LoadAST`/`SaveAST`
- Filters out wildcard (`*`) hosts from interactive listings

**Favorites & Usage Tracking (`internal/favorites/`)**
- Stores favorites and usage counts in `~/.config/sls/meta.json`
- JSON structure: `{"hostname": {"favorite": bool, "count": int}}`
- Auto-increments count on each connection
- Favorites marked with `⭐︎` symbol and pinned to top

**Interactive Finder (`internal/finder/`)**
- `finder.go`: Custom Bubbletea TUI with fuzzy search, tree-style container nesting (├─/└─), delete confirmation, status flash messages, adaptive hint bar, cursor restoration
- `multi.go`: Multi-select TUI for container selection during scan (supports pre-checked items)
- `prompt.go`: Inline rename prompt with alias validation
- `styles.go`: Shared color palette (accent orange, cyan prompt, yellow match, dim gray, green success, red error)

**Action Handlers (`internal/actions/`)**
- `connect.go`: Unified connect with shell fallback (/bin/sh → /bin/bash → /bin/ash) and refresh-on-miss
- `scan.go`: Container discovery → multi-select → rename → cache update flow
- `rename.go`: Host rename with cascade to container cache keys and favorites
- `delete.go`: Host delete (blocked if containers exist) and container delete
- `star.go`: Favorite toggle for hosts and containers

**Container Management (`internal/container/`)**
- `container.go`: Container struct with shell detection, `::` separator for composite keys
- `cache.go`: JSON cache at `~/.config/sls/containers.json` with merge-update and host rename
- `discovery.go`: SSH → `docker ps` parsing with name validation and concurrent multi-host scan
- `generator.go`: SSH config generation using ProxyJump to inherit parent host settings
- `validator.go`: Regex validation for container names and aliases (prevents injection)

**Commands (`internal/cli/`)**
- `config.go`: Interactive config management (add/edit/remove/list hosts)
- `connect.go`: `sls connect <host::container>` for scriptable container access
- `discover.go`: `sls discover [host]` and `sls discover --hosts` for container scanning
- `gen.go`: `sls gen ssh-config` generates includable SSH config for containers
- `fav.go`: Favorite management (add/remove/list)
- `completion.go`: Shell completion support

**SSH Execution (`internal/runner/`)**
- `SSH()`: Interactive sessions (exit 255 suppressed as normal termination)
- `SSHWithCmd()`: Remote commands like docker exec (exit 255 = real failure)

**Constants (`internal/consts/`)**
- `RequiredSSHConfigOptions`: Host, HostName, User
- `AllSSHConfigOptions`: Comprehensive SSH config option map for fuzzy selection

### Data Flow

1. Parse `~/.ssh/config` → filter non-wildcard hosts
2. Load favorites/counts from `~/.config/sls/meta.json`
3. Load container cache from `~/.config/sls/containers.json`
4. Build item list: fav containers → fav hosts → normal hosts (with nested containers)
5. Show Bubbletea finder → user selects action
6. Execute action (connect/scan/rename/delete/star) → loop back to step 1 (selective reload)
7. On connect: shell fallback detection → cache working shell → exec SSH

### Configuration Files

- **SSH config**: `~/.ssh/config` (read/write)
- **Metadata**: `~/.config/sls/meta.json` (favorites and usage tracking)
- **Container cache**: `~/.config/sls/containers.json` (discovered containers with shell info)
- **Generated SSH config**: `~/.config/sls/ssh_config` (includable container entries)

## External Dependencies

- `github.com/kevinburke/ssh_config`: SSH config parsing
- `github.com/charmbracelet/bubbletea`: Terminal TUI framework
- `github.com/charmbracelet/lipgloss`: TUI styling
- `github.com/sahilm/fuzzy`: Fuzzy string matching
- `github.com/ktr0731/go-fuzzyfinder`: Used for SSH config field selection in `sls config`
- `github.com/spf13/cobra`: CLI framework
- `golang.org/x/sync`: Concurrent container discovery

## Conventions

- Git commit messages and PR titles/descriptions must always be written in English.

## Important Implementation Notes

- The `*` wildcard host pattern is explicitly filtered from all listings
- Host aliases are taken from the first pattern in multi-pattern hosts
- Config writes completely rewrite `~/.ssh/config` (no incremental edits) using atomic temp+rename
- Format for saved config: `Host <alias>\n    Key\tValue\n\n`
- Usage count increments happen before SSH connection (even if connection fails)
- Container composite keys use `::` separator (e.g., `prod-server::nginx`), not `--`
- Container names and aliases validated with `^[a-zA-Z0-9._-]+$` to prevent SSH config injection
- Shell detection caches results: `""` = unknown, `"/bin/sh"` = sh, `"none"` = no shell available
- Generated SSH config uses `ProxyJump` to route through parent host (inherits all SSH settings)
- Discovery uses `StrictHostKeyChecking=no` + `UserKnownHostsFile=/dev/null` to avoid writing host keys

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
