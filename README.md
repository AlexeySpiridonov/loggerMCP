# loggerMCP

## 1. What This Project Is

`loggerMCP` is an MCP server for Ubuntu syslog.

It exists to give AI clients a controlled way to read and search system logs without giving them unrestricted shell access. Instead of asking an AI to run ad hoc `grep`, `tail`, or `journalctl` commands, you expose a narrow MCP API with filtering, pagination, authentication, and optional encryption.

Use it when you want an AI client to:

- inspect syslog remotely
- search logs by date range
- filter by wildcard patterns
- consume results page by page
- connect through MCP over SSE with an access key

## 2. How To Install It

### Option A: one-line installer

```bash
curl -fsSL https://raw.githubusercontent.com/AlexeySpiridonov/loggerMCP/main/install.sh | sudo bash
```

What it does:

- installs the binary
- creates `/etc/loggermcp/config.yaml`
- generates random `access_key` and `encryption_key`
- sets the manifest URL to `https://<server-ip>:7777/sse`
- installs and enables the systemd service
- uses `/var/lib/loggermcp` for runtime state and generated TLS files

After installation:

```bash
sudo nano /etc/loggermcp/config.yaml
sudo systemctl start loggermcp
sudo systemctl status loggermcp
```

### Option B: install via APT

```bash
curl -fsSL https://AlexeySpiridonov.github.io/loggerMCP/public.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/loggermcp.gpg

echo "deb [signed-by=/usr/share/keyrings/loggermcp.gpg] https://AlexeySpiridonov.github.io/loggerMCP stable main" \
  | sudo tee /etc/apt/sources.list.d/loggermcp.list

sudo apt update
sudo apt install loggermcp
```

Installed paths:

- binary: `/usr/bin/loggermcp`
- config: `/etc/loggermcp/config.yaml`
- service: `loggermcp.service`
- runtime state: `/var/lib/loggermcp`

### Update loggerMCP

If you installed with the one-line installer, rerun it:

```bash
curl -fsSL https://raw.githubusercontent.com/AlexeySpiridonov/loggerMCP/main/install.sh | sudo bash
sudo systemctl restart loggermcp
```

If you installed via APT:

```bash
sudo apt update
sudo apt install --only-upgrade loggermcp
sudo systemctl restart loggermcp
```

The existing `/etc/loggermcp/config.yaml` is preserved during updates.

## 3. How To Add It To An AI Client

The server exposes these endpoints:

- `/sse` for MCP transport
- `/.well-known/mcp-manifest.json` for MCP manifest autodiscovery
- `/manifest` as a legacy manifest alias
- `/health` for health status

The server requires `access_key` at the transport layer. Supported forms:

- `Authorization: Bearer <access_key>`
- `X-Access-Key: <access_key>`
- `?access_key=<access_key>` on the SSE URL

### Generic MCP client example with header auth

```json
{
  "url": "https://logger.example.com:7777/sse",
  "headers": {
    "Authorization": "Bearer your-secret-key"
  }
}
```

### Generic MCP client example with query auth

```json
{
  "url": "https://logger.example.com:7777/sse?access_key=your-secret-key"
}
```

### Static manifest example

The repository includes [mcp.json](mcp.json) as a default manifest example.

### Notes

- The manifest is public and does not contain secrets.
- Keep the access key in local client configuration, not in the manifest.
- If you run behind a reverse proxy, set `public_base_url` in config so the manifest returns correct URLs.

## 4. API Documentation

Detailed API and endpoint documentation was moved to [docs/API.md](docs/API.md).

It covers:

- authentication
- manifest response and health check
- the `read_logs` tool
- filtering, pagination, and encryption behavior

## 5. Build From Source

Build-from-source instructions were moved to [docs/BUILD.md](docs/BUILD.md).

It covers:

- local build and run
- configuration for local development
- package build
- release publishing flow
