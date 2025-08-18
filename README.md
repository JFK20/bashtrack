# Bash Command Tracker

A powerful, lightweight CLI (Go) that persistently records your interactive bash command history with timestamps and working directory into a local SQLite database. Provides fast fuzzy‑style searches, statistics, configurable exclusions, and safe local storage with zero telemetry.

## Features

- **Automatic Command Tracking**: Records each executed bash command with timestamp + working directory using a prompt hook
- **SQLite Storage**: Small, single-file database; transactional writes (no lost records on crash)
- **Smart Filtering**: Regex exclude patterns to skip noisy or sensitive commands (configurable at runtime)
- **Search & Analytics**: Filter, free‑text search, top commands/directories, basic activity stats
- **Cleanup & Retention**: Built‑in pruning of old entries by age
- **Cross-Platform**: Linux, macOS, Windows (WSL / Git Bash / MSYS2)
- **Privacy-First**: 100% local; no network calls; sensitive patterns excluded by default
- **Zero Shell History Mutation**: Reads, but does not rewrite your original history file

## Installation

### From Source

1. Clone the repository:
```bash
git clone <repository-url>
cd bashtrack
```

2. Build the application:
```bash
go mod tidy
go build -o bashtrack
```

3. Put the binary on your PATH:
```bash
sudo mv bashtrack /usr/local/bin/
# or (no sudo)
install -m 755 bashtrack "$HOME/.local/bin/"  # ensure ~/.local/bin is in PATH
```

## Setup

Two alternative integration methods. Prefer Method 1 (fc) for accuracy & zero race conditions.

Method 1 (Recommended: fc built‑in) \
Add to your ~/.bashrc (append near the end): \
Remove the `2>/dev/null` after `bashtrack record "$last_cmd"` if you want to see errors
```bash
# BashTrack command recording (Method 1)
bashtrack_record() {
    local last_cmd=$(fc -ln -1 2>/dev/null | sed 's/^[ \t]*//')
    if [[ -n "$last_cmd" && "$last_cmd" != bashtrack* ]]; then
        bashtrack record "$last_cmd" 2>/dev/null
    fi
}
export PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND$'\n'}bashtrack_record"
```

Method 2 (Fallback: history -a)
Use if fc is unavailable / restricted:
```bash
# Enable immediate history append
shopt -s histappend
export HISTCONTROL=ignoredups:erasedups
export HISTSIZE=10000
export HISTFILESIZE=20000

bashtrack_record() {
    local last_cmd=$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
    if [[ -n "$last_cmd" && "$last_cmd" != bashtrack* ]]; then
        bashtrack record "$last_cmd" 2>/dev/null
    fi
}
export PROMPT_COMMAND="history -a; ${PROMPT_COMMAND:+$PROMPT_COMMAND$'\n'}bashtrack_record"
```

Reload your shell:
```bash
source ~/.bashrc
```

Verify:
```bash
bashtrack list
```

## Usage

### Basic Commands

```bash
# Record an arbitrary command manually (rarely needed)
bashtrack record "echo hello"

# List recent commands
bashtrack list

# List with custom limit
bashtrack list -l 50

# Filter by substring/pattern within the command
bashtrack list -f docker

# Filter by directory substring
bashtrack list -d "/home/user/projects"

# Search (same as list -f but capped at 50 and optimized)
bashtrack search "docker build"

# Show statistics
bashtrack stats

# Remove commands older than N days (default 90)
bashtrack cleanup -d 120
```

### Configuration Management

```bash
# Show current configuration
bashtrack config show

# Add regex exclude pattern
bashtrack config add-exclude "^vim.*"

# Remove pattern (exact string match)
bashtrack config remove-exclude "^ls.*"
```

## Default Exclude Patterns

The initial config excludes noisy navigation, history invocations, sensitive keywords, and self‑referential tracker usage:

- `^ls.*`, `^cd.*`, `^pwd.*`, `^clear.*`, `^exit.*`
- `^history.*`
- `.*password.*`, `.*secret.*`, `.*token.*`, `.*key.*`
- `.*bashtrack.*` (prevents recursion)

You can relax or extend these via `bashtrack config add-exclude` / `remove-exclude`.

## Configuration

Stored at `~/.bashtrack/config.json` (auto-created on first run):

```json
{
  "exclude_patterns": [
    "^ls.*",
    "^cd.*",
    "^pwd.*",
    "^clear.*",
    "^exit.*",
    "^history.*",
    ".*password.*",
    ".*secret.*",
    ".*token.*",
    ".*key.*",
    ".*bashtrack.*"
  ],
  "database_path": "/home/user/.bashtrack/commands.db"
}
```

Edits can be made manually or through `bashtrack config` subcommands. Invalid JSON will be rejected on next start.

## Privacy & Security

- All data stored locally (single SQLite file)
- Sensitivity patterns (password/secret/token/key) excluded by regex by default
- No network calls; no telemetry
- Easy manual purge: delete `~/.bashtrack` or use `bashtrack cleanup`

## Build Instructions

```bash
# Install / verify dependencies
go mod tidy

# Build (current platform)
go build -o bashtrack

# Cross-compile examples
GOOS=linux   GOARCH=amd64 go build -o bashtrack-linux-amd64
GOOS=linux   GOARCH=arm64 go build -o bashtrack-linux-arm64
GOOS=darwin  GOARCH=amd64 go build -o bashtrack-darwin-amd64
GOOS=darwin  GOARCH=arm64 go build -o bashtrack-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o bashtrack-windows-amd64.exe
GOOS=windows GOARCH=arm64 go build -o bashtrack-windows-arm64.exe
```

For reproducible builds pin Go version (e.g., with `go.mod` toolchain directive) and enable module proxy caching if desired.

## Roadmap

- [ ] Export functionality (JSON, CSV)
- [ ] Additional shell integrations (zsh, fish)
- [ ] Optional local web UI for browsing & charts
