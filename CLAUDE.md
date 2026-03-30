# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`sls` (ssh ls) is a smart fuzzy CLI selector for SSH config hosts written in Go. It provides an interactive interface for selecting and connecting to SSH hosts defined in `~/.ssh/config`, with features like fuzzy search, usage tracking, and favorites management.

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
- Interactive mode launches fzf with host list sorted by favorites and usage count
- Passes selected host to SSH runner with any extra arguments

**SSH Config Parsing (`internal/config/`)**
- `config.go`: Parses `~/.ssh/config` using `github.com/kevinburke/ssh_config`
- `editor.go`: AST manipulation for add/edit/remove operations via `LoadAST`/`SaveAST`
- Filters out wildcard (`*`) hosts from interactive listings

**Favorites & Usage Tracking (`internal/favorites/`)**
- Stores favorites and usage counts in `~/.config/sls/meta.json`
- JSON structure: `{"hostname": {"favorite": bool, "count": int}}`
- Auto-increments count on each connection
- Favorites marked with `⭐︎` symbol and pinned to top

**Interactive Selection (`cmd/root.go:runInteractive`)**
- Builds sorted host list: favorites first, then by usage count
- Launches `fzf` with preview command (`sls preview {}`)
- Requires `fzf` in PATH (declared as Homebrew dependency)

**Commands (`internal/cli/`)**
- `config.go`: Interactive config management (add/edit/remove/list hosts)
  - Add/edit prompts for HostName, User, Port, and optional SSH config fields
  - Uses `go-fuzzyfinder` for optional field selection
  - Validates host existence for edit operations
- `fav.go`: Favorite management (add/remove/list)
- `preview.go`: Host detail preview for fzf (hidden command)
- `completion.go`: Shell completion support

**SSH Execution (`internal/runner/`)**
- Simple wrapper around `ssh` command with stdin/stdout/stderr passthrough
- Accepts extra SSH arguments from initial `sls` invocation

**Constants (`internal/consts/`)**
- `RequiredSSHConfigOptions`: Host, HostName, User
- `AllSSHConfigOptions`: Comprehensive SSH config option map for fuzzy selection

### Data Flow

1. Parse `~/.ssh/config` → filter non-wildcard hosts
2. Load favorites/counts from `~/.config/sls/meta.json`
3. Sort: favorites first, then by usage count descending
4. Launch fzf with preview of host details
5. On selection: increment usage count, exec SSH

### Configuration Files

- **SSH config**: `~/.ssh/config` (read/write)
- **Metadata**: `~/.config/sls/meta.json` (favorites and usage tracking)

## External Dependencies

- `github.com/kevinburke/ssh_config`: SSH config parsing
- `github.com/ktr0731/go-fuzzyfinder`: Interactive terminal fuzzy finder
- `github.com/spf13/cobra`: CLI framework
- `fzf`: External binary (runtime dependency)

## Conventions

- Git commit messages and PR titles/descriptions must always be written in English.

## Important Implementation Notes

- The `*` wildcard host pattern is explicitly filtered from all listings
- Host aliases are taken from the first pattern in multi-pattern hosts
- Config writes completely rewrite `~/.ssh/config` (no incremental edits)
- Format for saved config: `Host <alias>\n    Key\tValue\n\n`
- Usage count increments happen before SSH connection (even if connection fails)

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
