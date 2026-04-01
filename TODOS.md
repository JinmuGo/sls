# TODOS

## P2 — After Shell Router v1 ships

### Multi-runtime discovery (tmux, systemd, k8s)
Extend discovery beyond Docker to detect tmux sessions, systemd services, and Kubernetes pods. The eureka insight from /office-hours was "get me a shell on a thing" regardless of runtime. Docker is the wedge; multi-runtime is the full vision (Approach C from the design doc).
- **Effort:** XL (human) → L with CC
- **Depends on:** Shell Router v1 shipped and demand validated
- **Context:** Design doc at `~/.gstack/projects/JinmuGo-sls/jinmu-unknown-design-20260330-203023.md`

### Fix SaveAST comment preservation
`internal/config/editor.go:46` `SaveAST` drops all comments and non-KV nodes when rewriting SSH config. Running `sls config add/edit/remove` silently deletes user's hand-crafted comments. The function should preserve comments and empty lines during AST round-trip.
- **Effort:** M (human) → S with CC
- **Context:** New `AddIncludeLine()` avoids this by doing raw text manipulation. But existing config CRUD path still has the bug. Found during /plan-eng-review 2026-03-30.

## P3 — Nice to have

### Delight opportunities
Surfaced during /plan-ceo-review 2026-03-30:
- Container health status indicators in fzf list (green = healthy, red = unhealthy)
- `sls discover --watch` mode that auto-refreshes when containers start/stop
- Shell history per container (which commands you ran last time)
- `sls tree` showing server → container hierarchy as ASCII tree
- Auto-detect Compose project names and group containers by stack

### Integration tests against messy SSH configs
Unit tests cover parsing and generation. Integration tests should cover Include directives, Match blocks, comments, and multi-pattern hosts in ~/.ssh/config.
- **Effort:** M (human) → S with CC
