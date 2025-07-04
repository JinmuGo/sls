# sls (ssh ls)

sls is a smart fuzzy CLI selector for SSH config hosts.  
It provides a convenient way to select, preview, and connect to SSH hosts defined in your `~/.ssh/config`.

## Key Features

- Fuzzy search over SSH host entries using fzf
- Preview detailed host information before connecting
- Usage-based sorting: frequently used hosts appear higher
- Mark and unmark favorites to pin entries to the top
- Interactive configuration commands: add, edit, and remove hosts
- Compatible with standard OpenSSH and SSH config syntax

## Installation

### Homebrew (macOS)

```bash
brew tap jinmugo/tap
brew install sls
```

### From Source

```bash
go install github.com/jinmugo/sls@latest
```

### Binary Release

Download platform-specific binaries from the [Releases](https://github.com/jinmugo/sls/releases) page.

## Usage

```bash
sls
```

This opens an interactive fuzzy selector listing all SSH hosts. Selecting a host immediately initiates an SSH connection.

### Other Commands

```bash
sls config list
sls config add
sls config edit <alias>
sls config remove <alias>

sls fav add <alias>
sls fav remove <alias>
sls fav list
```

Favorites are marked with a `⭐︎` symbol and shown at the top of the list. Other hosts are sorted based on how frequently they are used.

## Configuration

- SSH config files:
  - `~/.ssh/config`
- User-specific data:
  - Usage counts and favorites are saved to `~/.config/sls/meta.json`
- fzf must be installed and available in the system PATH
