# Contributing to sls

Thanks for your interest in contributing to `sls`!

## Getting Started

### Prerequisites

- Go 1.23+
- Access to an SSH config (`~/.ssh/config`)

### Build

```bash
go build -o sls .
```

### Run locally

```bash
go run . [command]
```

## How to Contribute

### Reporting Issues

Open an issue on GitHub with:
- Steps to reproduce
- Expected vs actual behavior
- Go version and OS

### Submitting a PR

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Ensure `go build` succeeds with no errors
4. Open a PR against `main`

### PR Guidelines

- **Title**: Use conventional commits format — `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`
- **Description**: Explain what and why, not just how
- **Scope**: Keep PRs focused — one feature or fix per PR
- **Review**: All PRs require owner approval before merge
- **Merge**: Squash merge only (enforced by repo settings)

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use typed interfaces — avoid `any` type
- Validate container names/aliases with the existing regex (`^[a-zA-Z0-9._-]+$`)
- Write commit messages and PR content in English

## Architecture

See [CLAUDE.md](./CLAUDE.md) for detailed architecture documentation.

## License

By contributing, you agree that your contributions will be licensed under the project's license.
