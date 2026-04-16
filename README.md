# loggerMCP

`loggerMCP` is a Go MCP server for reading and searching Ubuntu syslog files.

It exposes an SSE MCP endpoint, a public manifest endpoint, and a health endpoint. The server supports transport-level access-key authentication, optional TLS, pagination, date filtering, wildcard matching, and optional AES-256-GCM response encryption.

## Endpoints

- `/sse`: MCP SSE transport endpoint.
- `/manifest`: public MCP manifest generated from config.
- `/health`: public health endpoint for readiness and monitoring.

## Quick Start

### Local build

```bash
cp config.yaml.example config.yaml
go build -o loggerMCP .
./loggerMCP
```

Minimal config:

```yaml
access_key: "your-secret-key"
syslog_path: "/var/log/syslog"
port: 7777
tls: false
```

### One-line install

```bash
curl -fsSL https://raw.githubusercontent.com/AlexeySpiridonov/loggerMCP/main/install.sh | sudo bash
```

The installer creates `/etc/loggermcp/config.yaml`, generates random `access_key` and `encryption_key` values, installs the systemd service, and keeps runtime state in `/var/lib/loggermcp`.

After installation:

```bash
sudo nano /etc/loggermcp/config.yaml
sudo systemctl start loggermcp
sudo systemctl status loggermcp
```

## Configuration Reference

- `access_key`: transport authentication secret.
- `syslog_path`: absolute path to the log file to read.
- `port`: HTTP listen port.
- `tls`: enable HTTPS.
- `cert_file`: certificate path. Relative paths are resolved from the current working directory.
- `key_file`: private key path. Relative paths are resolved from the current working directory.
- `encryption_key`: optional key for `read_logs` response encryption.
- `public_base_url`: optional public URL used in manifest and health responses.
- `manifest_name`: manifest `name` field.
- `manifest_title`: manifest `title` field.
- `manifest_description`: manifest `description` field.
- `manifest_version`: manifest `version` field.
- `manifest_path`: HTTP path for the manifest endpoint.
- `manifest_remote_type`: manifest remote type. The built-in server transport is `sse`.
- `manifest_remote_url`: optional explicit remote URL override.
- `health_path`: HTTP path for the health endpoint.

Example config:

```yaml
access_key: "your-secret-key"
syslog_path: "/var/log/syslog"
port: 7777
tls: true
encryption_key: ""
# public_base_url: "https://logger.example.com"
manifest_name: "logger.local/mcp"
manifest_title: "loggerMCP"
manifest_description: "Remote MCP server for Ubuntu syslog search workflows."
manifest_version: "1.0.0"
manifest_path: "/manifest"
manifest_remote_type: "sse"
health_path: "/health"
```

## Authentication

`access_key` is enforced at the transport level. The server accepts it in any of these forms:

- `Authorization: Bearer your-secret-key`
- `X-Access-Key: your-secret-key`
- `?access_key=your-secret-key` on the SSE URL

Recommended client config with a header:

```json
{
  "url": "https://logger.example.com/sse",
  "headers": {
    "Authorization": "Bearer your-secret-key"
  }
}
```

Alternative client config with a query parameter:

```json
{
  "url": "https://logger.example.com/sse?access_key=your-secret-key"
}
```

The `read_logs` tool also accepts `access_key` as a legacy fallback when the client cannot set transport headers or query parameters.

## TLS

Set `tls: true` to enable HTTPS. If `cert_file` and `key_file` do not exist, the server auto-generates a self-signed certificate.

When running under the packaged systemd service, relative certificate paths resolve from `/var/lib/loggermcp`.

## MCP Tool: `read_logs`

Parameters:

- `access_key` (`string`, optional): legacy fallback access key.
- `start_date` (`string`, optional): `2006-01-02` or `2006-01-02T15:04:05`.
- `end_date` (`string`, optional): `2006-01-02` or `2006-01-02T15:04:05`.
- `pattern` (`string`, optional): wildcard search string such as `error*disk`.
- `page` (`number`, optional): page number, default `1`.
- `page_size` (`number`, optional): entries per page, default `100`, max `1000`.
- `encrypt` (`boolean`, optional): encrypt the returned payload with AES-256-GCM.

## Manifest and Health

The live manifest is served from `/manifest` and is generated from config.

Example manifest shape:

```json
{
  "description": "Remote MCP server for Ubuntu syslog search workflows.",
  "name": "logger.local/mcp",
  "remotes": [
    {
      "type": "sse",
      "url": "https://logger.example.com/sse"
    }
  ],
  "title": "loggerMCP",
  "version": "1.0.0"
}
```

Health example:

```bash
curl -fsSL http://localhost:7777/health
```

`/health` returns JSON and reports `degraded` with HTTP `503` when the configured log file cannot be opened.

The public manifest does not include secrets. Configure access keys in the MCP client, not in the manifest.

The repository also includes [mcp.json](mcp.json) as a static manifest example matching the default server shape.

## Response Encryption

Set `encryption_key` in config to enable encrypted responses for `read_logs` when the client passes `encrypt: true`.

Encrypted responses are returned as `ENC:<base64>`, where the payload is `nonce + ciphertext` encrypted with AES-256-GCM.

Python decryption example:

```python
import base64
import hashlib
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

data = base64.b64decode(response.removeprefix("ENC:"))
key = hashlib.sha256(b"my-secret-encryption-key").digest()
plaintext = AESGCM(key).decrypt(data[:12], data[12:], None).decode()
```

## Install via APT

Add the repository and install the package:

```bash
curl -fsSL https://AlexeySpiridonov.github.io/loggerMCP/public.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/loggermcp.gpg

echo "deb [signed-by=/usr/share/keyrings/loggermcp.gpg] https://AlexeySpiridonov.github.io/loggerMCP stable main" \
  | sudo tee /etc/apt/sources.list.d/loggermcp.list

sudo apt update
sudo apt install loggermcp
```

Installed paths:

- Binary: `/usr/bin/loggermcp`
- Config: `/etc/loggermcp/config.yaml`
- Service: `loggermcp.service`
- Runtime state: `/var/lib/loggermcp`

Useful commands:

```bash
sudo nano /etc/loggermcp/config.yaml
sudo systemctl start loggermcp
sudo systemctl status loggermcp
journalctl -u loggermcp -f
```

Update with:

```bash
sudo apt update
sudo apt upgrade loggermcp
```

## Publishing Releases

One-time setup:

1. Create a GPG key for repository signing.

```bash
gpg --full-generate-key
gpg --armor --export-secret-keys YOUR_KEY_ID
```

1. Add GitHub Actions secrets:

- `GPG_PRIVATE_KEY`
- `GPG_PASSPHRASE`
- `GPG_KEY_ID`

1. Enable GitHub Pages and point it at the `gh-pages` branch.

Release flow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow builds `.deb` packages for `amd64` and `arm64`, publishes the APT repository to GitHub Pages, and attaches both `.deb` packages and standalone Linux binaries to the GitHub release.
