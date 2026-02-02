# enva

![enva](screenshots/enva.jpg)

**Stop juggling `.env` files. Start managing environment variables like a pro.**

enva is a per-directory environment variable manager that stores your vars in SQLite and automatically loads them as you navigate your filesystem. Set variables once, inherit them everywhere, override when needed.

## Why enva?

- **Automatic loading** — Variables load/unload as you `cd` between directories
- **Directory inheritance** — Set project-wide defaults, override in subdirectories
- **Beautiful TUI** — Fuzzy search, bulk import, visual editing
- **Profile support** — Switch between dev/staging/production configs
- **No more `.env` files** — Centralized storage, no secrets in git

## Quick Start

### Install

```bash
# Homebrew (macOS/Linux)
brew tap nick-skriabin/tap
brew install enva

# Or with Go
go install github.com/nick-skriabin/enva/cmd/enva@latest
```

### Set up your shell

Add to your shell config and restart:

```bash
# Zsh (~/.zshrc)
eval "$(enva hook zsh)"

# Bash (~/.bashrc)
eval "$(enva hook bash)"

# Fish (~/.config/fish/config.fish)
enva hook fish | source
```

### Start using it

```bash
cd ~/projects/myapp

# Set some variables
enva set DATABASE_URL=postgres://localhost/mydb
enva set API_KEY=sk-123456

# That's it! Variables auto-load when you enter this directory
```

## The TUI

Just run `enva` to launch the interactive interface:

```bash
enva
```

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `/` | Fuzzy search |
| `a` | Add variable |
| `e` | Edit selected |
| `x` | Delete |
| `A` | Bulk import from clipboard/file |
| `t` | Toggle all/local view |
| `?` | Help |
| `q` | Quit |

## CLI Commands

| Command | Description |
|---------|-------------|
| `enva` | Launch interactive TUI |
| `enva set KEY=VALUE` | Set variable in current directory |
| `enva unset KEY` | Remove variable |
| `enva ls` | List effective variables |
| `enva edit` | Edit in `$EDITOR` |
| `enva run -- cmd` | Run command with env loaded |
| `enva export` | Print export statements |
| `enva hook <shell>` | Print shell integration code |

## How Inheritance Works

Variables cascade down from parent directories:

```
~/projects/                     DATABASE_URL=postgres://...
└── myapp/                      API_KEY=abc123
    └── backend/                DEBUG=true
```

When you `cd ~/projects/myapp/backend`, you get all three variables merged together. Child directories can override parent values.

### Project Boundaries

enva looks for project roots in this order:

1. `.enva` marker file
2. `.git` directory
3. Filesystem root

Variables only inherit within the same project boundary.

## Profiles

Manage multiple environments with profiles:

```bash
# Production config
ENVA_PROFILE=production enva set API_URL=https://api.example.com

# Development (default profile)
enva set API_URL=http://localhost:3000

# Switch profiles
export ENVA_PROFILE=production
enva ls  # Shows production vars
```

## Variable Descriptions

Add descriptions to document your variables:

```bash
# In the TUI, each variable has an optional description field
# In exports, descriptions appear as comments:
export API_KEY='sk-123' # Main API key for auth service
```

## Storage

All variables live in a single SQLite database:

```
~/.local/share/enva/enva.db
```

No more scattered `.env` files. One source of truth.

## Build from Source

```bash
git clone https://github.com/nick-skriabin/enva.git
cd enva
go build -o enva ./cmd/enva
sudo mv enva /usr/local/bin/
```

## License

MIT
