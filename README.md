# loggerMCP

Go MCP server for reading and searching Ubuntu syslog.

## Features

- Read syslog entries with pagination
- Filter by date range (start/end)
- Substring search with wildcard (`*`) support
- Access key authentication
- AES-256-GCM response encryption
- TLS with auto-generated self-signed certificate
- SSE transport on port 7777

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/AlexeySpiridonov/loggerMCP/main/install.sh | sudo bash
```

After installation:

```bash
# Set access key
sudo nano /etc/loggermcp/config.yaml

# Start
sudo systemctl start loggermcp

# Check status
sudo systemctl status loggermcp
```

## Configuration

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml`:

```yaml
access_key: "your-secret-key"
syslog_path: "/var/log/syslog"
port: 7777
```

## Build & Run

```bash
go build -o loggerMCP .
./loggerMCP
# or with a custom config path:
./loggerMCP /path/to/config.yaml
```

## MCP Tool: `read_logs`

| Parameter   | Type    | Required | Description                                                    |
|-------------|---------|----------|----------------------------------------------------------------|
| access_key  | string  | yes      | Access key                                                     |
| start_date  | string  | no       | Start date (`2006-01-02` or `2006-01-02T15:04:05`)            |
| end_date    | string  | no       | End date (`2006-01-02` or `2006-01-02T15:04:05`)              |
| pattern     | string  | no       | Substring filter, `*` = wildcard. Example: `error*disk`        |
| page        | number  | no       | Page number (default: 1)                                       |
| page_size   | number  | no       | Entries per page (default: 100, max: 1000)                     |
| encrypt     | boolean | no       | Encrypt response with AES-256-GCM (key from config)            |

## TLS

Set `tls: true` in `config.yaml` — the server will auto-generate a self-signed certificate on first run. Or provide your own:

```yaml
tls: true
cert_file: "/path/to/cert.pem"
key_file: "/path/to/key.pem"
```

## Encryption

Set `encryption_key` in config. When a client passes `encrypt: true`, the response is encrypted with AES-256-GCM and returned as `ENC:<base64>`.

Decryption example (Python):

```python
import base64, hashlib
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

data = base64.b64decode(response.removeprefix("ENC:"))
key = hashlib.sha256(b"my-secret-encryption-key").digest()
gcm = AESGCM(key)
plaintext = gcm.decrypt(data[:12], data[12:], None).decode()
```

## Install via APT

### Add repository

```bash
# Import GPG key
curl -fsSL https://AlexeySpiridonov.github.io/loggerMCP/public.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/loggermcp.gpg

# Add source
echo "deb [signed-by=/usr/share/keyrings/loggermcp.gpg] https://AlexeySpiridonov.github.io/loggerMCP stable main" \
  | sudo tee /etc/apt/sources.list.d/loggermcp.list

# Install
sudo apt update
sudo apt install loggermcp
```

After installation:
- Binary: `/usr/bin/loggermcp`
- Config: `/etc/loggermcp/config.yaml` (created from example, **edit `access_key`**)
- Systemd service: `loggermcp.service`

```bash
# Edit config
sudo nano /etc/loggermcp/config.yaml

# Start
sudo systemctl start loggermcp

# Check status
sudo systemctl status loggermcp

# Service logs
journalctl -u loggermcp -f
```

### Update

```bash
sudo apt update && sudo apt upgrade loggermcp
```

## Publishing a Release (APT deploy)

### One-time setup

1. **GPG key** for repository signing:

```bash
gpg --full-generate-key
gpg --armor --export-secret-keys YOUR_KEY_ID
```

2. **GitHub Secrets** (Settings → Secrets and variables → Actions):
   - `GPG_PRIVATE_KEY` — private key (output of the command above)
   - `GPG_PASSPHRASE` — key passphrase
   - `GPG_KEY_ID` — key ID

3. **GitHub Pages** — Settings → Pages → Source: `gh-pages` branch

### Release

Create a tag — GitHub Actions will automatically build `.deb` (amd64 + arm64), publish the APT repository to GitHub Pages, and create a GitHub Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```
