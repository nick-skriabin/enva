# enva

![/screenshots/enva.png](Enva)

Per-directory environment variable manager with automatic shell integration.

enva stores environment variables in a SQLite database and automatically loads/unloads them as you navigate directories. Variables are inherited from parent directories, allowing you to set project-wide defaults and override them in subdirectories.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap nick-skriabin/tap
brew install enva
```

### Go Install

```bash
go install github.com/nick-skriabin/enva/cmd/enva@latest
```

### Build from Source

```bash
git clone https://github.com/nick-skriabin/enva.git
cd enva
go build -o enva ./cmd/enva
mv enva /usr/local/bin/
```

## Shell Integration

Add to your shell configuration:

**Zsh** (`~/.zshrc`):
```zsh
eval "$(enva hook zsh)"
```

**Bash** (`~/.bashrc`):
```bash
eval "$(enva hook bash)"
```

**Fish** (`~/.config/fish/config.fish`):
```fish
enva hook fish | source
```

Restart your shell or source the config file.

## Usage

### Setting Variables

```bash
# Set a variable in the current directory
enva set API_KEY=secret123
enva set DATABASE_URL=postgres://localhost/mydb

# Variables are scoped to the current directory
cd ~/projects/myapp
enva set DEBUG=true
```

### Viewing Variables

```bash
# List effective variables (inherited + local)
enva ls

# Show what would be exported
enva export
```

### Removing Variables

```bash
# Remove a variable from current directory
enva unset API_KEY
```

### Running Commands

```bash
# Run a command with the effective environment
enva run -- npm start
enva run -- docker-compose up
```

### Editing Variables

```bash
# Open $EDITOR to edit local variables
enva edit
```

### Interactive TUI

```bash
enva tui
```

The TUI provides:
- Fuzzy search across keys and values
- View inherited vs local variables
- Add, edit, and delete variables
- Bulk import from env files

**Keybindings:**
| Key | Action |
|-----|--------|
| `j/k` | Navigate |
| `/` | Search |
| `e` | Edit selected |
| `a` | Add new |
| `A` | Bulk import |
| `x` | Delete |
| `t` | Toggle effective/local view |
| `?` | Help |
| `q` | Quit |

## How It Works

### Directory Inheritance

Variables are inherited from parent directories down to the current directory:

```
/projects/                    # DATABASE_URL=postgres://...
/projects/myapp/              # API_KEY=abc123 (inherits DATABASE_URL)
/projects/myapp/backend/      # DEBUG=true (inherits both above)
```

When you `cd` into `/projects/myapp/backend/`, you get all three variables.

### Root Boundary Discovery

enva determines the project root by looking for (in order):

1. `.enva` marker file (closest ancestor wins)
2. `.git` directory (closest ancestor wins)
3. Filesystem root `/`

Variables are only inherited within the project root boundary.

### Automatic Loading

With shell integration, enva automatically:
- Loads variables when entering a directory
- Unloads variables when leaving
- Shows status messages: `enva: loaded 5 var(s)`

## Profiles

enva supports multiple profiles for different environments:

```bash
# Use a specific profile
ENVA_PROFILE=production enva set API_URL=https://api.example.com
ENVA_PROFILE=production enva ls

# Default profile is "default"
enva set API_URL=http://localhost:3000
```

## Database Location

Variables are stored in:
```
~/.local/share/enva/enva.db
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `enva hook <shell>` | Print shell hook (bash/zsh/fish) |
| `enva set KEY=VALUE` | Set variable in current directory |
| `enva unset KEY` | Remove variable from current directory |
| `enva ls` | List effective variables |
| `enva export` | Print shell export commands |
| `enva edit` | Edit local variables in $EDITOR |
| `enva run -- CMD` | Run command with environment |
| `enva tui` | Interactive terminal UI |

## License

MIT
