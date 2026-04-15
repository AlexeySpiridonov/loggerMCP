# loggerMCP

Golang [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for Ubuntu system logs. Exposes tools for reading and searching log files and the systemd journal so that AI assistants can inspect your system.

## Features

| Tool | Description |
|---|---|
| `list_log_files` | List all readable log files under `/var/log` |
| `read_log_file` | Read the last N lines of any log file under `/var/log` |
| `search_log_file` | Search a log file for lines matching a pattern (case-insensitive) |
| `read_journal` | Read systemd journal entries via `journalctl` (optionally filtered by unit / time) |
| `search_journal` | Search systemd journal entries for lines matching a regex pattern |

## Requirements

- Go 1.21+
- Ubuntu (or any Linux system with `/var/log` and `journalctl`)

## Build

```bash
go build -o loggerMCP .
```

## Usage

Run the server and connect it to an MCP-compatible client (e.g. Claude Desktop, VS Code with Copilot, or any MCP client).

### stdio transport (default)

```bash
./loggerMCP
```

The server communicates over stdin/stdout using the MCP JSON-RPC protocol.

### Connecting from Claude Desktop

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "loggerMCP": {
      "command": "/path/to/loggerMCP"
    }
  }
}
```

### Tool parameters

#### `read_log_file`
| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | ✅ | Absolute path to the log file (must be inside `/var/log`) |
| `lines` | number | | Number of lines to return from the end (default: 100, max: 10 000) |

#### `search_log_file`
| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | ✅ | Absolute path to the log file |
| `pattern` | string | ✅ | Text pattern to search for (case-insensitive substring) |
| `max_results` | number | | Maximum matching lines to return (default: 100, max: 10 000) |

#### `read_journal`
| Parameter | Type | Required | Description |
|---|---|---|---|
| `lines` | number | | Number of recent entries to return (default: 100, max: 10 000) |
| `unit` | string | | Systemd unit filter, e.g. `ssh.service` or `nginx` |
| `since` | string | | Show entries since this time, e.g. `"1 hour ago"`, `"today"`, `"2024-01-15 12:00:00"` |
| `current_boot` | boolean | | If `true`, show entries from the current boot only |

#### `search_journal`
| Parameter | Type | Required | Description |
|---|---|---|---|
| `pattern` | string | ✅ | Regex pattern to match in journal messages |
| `lines` | number | | Maximum matching entries to return (default: 100, max: 10 000) |
| `unit` | string | | Systemd unit filter |
| `since` | string | | Time filter as above |

## Security

- `read_log_file` and `search_log_file` reject any path outside `/var/log`, including symlinks that resolve outside that directory.
- `unit` and `since` parameters are validated against an allow-list of safe characters before being passed to `journalctl`.

## License

See [LICENSE](LICENSE).

