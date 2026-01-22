# sls (ssh ls)

sls is a smart fuzzy CLI selector for SSH config hosts.  
It provides a convenient way to select, preview, and connect to SSH hosts defined in your `~/.ssh/config`.

<img width="1428" height="338" alt="sls-example" src="https://github.com/user-attachments/assets/b99d18b5-848d-41a3-9b93-8f8ecbe982f4" />

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

### Debian/Ubuntu (apt)

```bash
# Add GPG key
curl -fsSL https://package.jinmu.me/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/sls.gpg

# Add repository
echo "deb [signed-by=/etc/apt/keyrings/sls.gpg] https://package.jinmu.me/deb stable main" | sudo tee /etc/apt/sources.list.d/sls.list

# Install
sudo apt update && sudo apt install sls
```

### Fedora/RHEL (dnf/yum)

```bash
# Add repository
sudo tee /etc/yum.repos.d/sls.repo << EOF
[sls]
name=sls
baseurl=https://package.jinmu.me/rpm
enabled=1
gpgcheck=1
gpgkey=https://package.jinmu.me/gpg.key
EOF

# Install
sudo dnf install sls
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
