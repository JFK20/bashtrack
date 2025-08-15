# Bash Command Tracker

A powerful CLI tool written in Go that tracks all your bash commands in an SQLite database with intelligent filtering and comprehensive search capabilities.

## Features

- **Automatic Command Tracking**: Captures every bash command with timestamp and working directory
- **SQLite Storage**: Lightweight, local database storage
- **Smart Filtering**: Configurable exclude patterns for sensitive or common commands
- **Search & Analytics**: Powerful search functionality and usage statistics
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Privacy-First**: All data stays local on your machine

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

3. Move to your PATH:
```bash
sudo mv bashtrack /usr/local/bin/
# or
cp bashtrack ~/bin/  # if ~/bin is in your PATH
```

### Pre-built Binaries

Download the latest release from the releases page and place it in your PATH.

## Setup

1. Run the setup command to see installation instructions:
```bash
bashtrack setup
```

2. Add the tracking hook to your `~/.bashrc`:
```bash
export PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND\n'}bashtrack record \"\$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')\""
```

3. Reload your bash configuration:
```bash
source ~/.bashrc
```

## Usage

### Basic Commands

```bash
# List recent commands
bashtrack list

# List with custom limit
bashtrack list -l 50

# Search for commands containing a pattern
bashtrack search "docker"

# Filter by directory
bashtrack list -d "/home/user/projects"

# Show usage statistics
bashtrack stats

# Clean up old commands (older than 90 days)
bashtrack cleanup -d 90
```

### Configuration Management

```bash
# Show current configuration
bashtrack config show

# Add exclude pattern
bashtrack config add-exclude "^vim.*"

# Remove exclude pattern
bashtrack config remove-exclude "^ls$"
```

## Default Exclude Patterns

The tool comes with sensible defaults to exclude:

- Common navigation commands (`ls`, `cd`, `pwd`, `clear`, `exit`)
- History commands
- Sensitive patterns (anything containing `password`, `secret`, `token`, `key`)
- The tracker's own record commands

## Database Schema

The SQLite database stores:

```sql
CREATE TABLE commands (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        timestamp DATETIME NOT NULL,
                        command TEXT NOT NULL,
                        directory TEXT NOT NULL
);
```

## Configuration

Configuration is stored in `~/.bashtrack/config.json`:

```json
{
  "exclude_patterns": [
    "^ls$",
    "^cd$",
    "^pwd$",
    "^clear$",
    "^exit$",
    "^history",
    ".*password.*",
    ".*secret.*",
    ".*token.*",
    ".*key.*",
    "bashtrack record"
  ],
  "database_path": "/home/user/.bashtrack/commands.db"
}
```

## Privacy & Security

- All data is stored locally in SQLite database
- Sensitive patterns are automatically excluded
- No network connectivity required
- No telemetry or data collection

## Troubleshooting

### Commands not being tracked

1. Ensure the PROMPT_COMMAND is set correctly in `~/.bashrc`
2. Check that `bashtrack` is in your PATH
3. Verify permissions on the `~/.bashtrack` directory

### Database issues

The database is automatically created on first use. If you encounter issues:

```bash
# Check database location
bashtrack config show

# Clean up and restart
rm -rf ~/.bashtrack
bashtrack setup
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Alternative Setup Methods

### Using bash-preexec

If you have `bash-preexec` installed, you can use:

```bash
preexec() { bashtrack record "$1"; }
```

### Manual Integration

For custom setups, you can integrate the tracking however you prefer by calling:

```bash
bashtrack record "your command here"
```

## Build Instructions

```bash
# Install dependencies
go mod tidy

# Build for current platform
go build -o bashtrack

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o bashtrack-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o bashtrack-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o bashtrack-windows-amd64.exe
```

## Performance

- Minimal overhead: ~1-2ms per command
- SQLite database grows approximately 100-200 bytes per command
- Automatic cleanup features to manage database size
- Indexed for fast searches even with large datasets

## Roadmap

- [ ] Export functionality (JSON, CSV)
- [ ] Command frequency analysis
- [ ] Integration with other shells (zsh, fish)
- [ ] Web interface for browsing history
- [ ] Sync capabilities across machines