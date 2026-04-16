# API Documentation

## Overview

`loggerMCP` exposes three HTTP endpoints and one MCP tool.

Endpoints:

- `/sse`: MCP SSE transport endpoint
- `/manifest`: public MCP manifest JSON
- `/health`: public health endpoint

Tool:

- `read_logs`

## Authentication

Transport authentication is controlled by `access_key` in `config.yaml`.

Supported ways to pass the key:

- `Authorization: Bearer <access_key>`
- `X-Access-Key: <access_key>`
- `?access_key=<access_key>` on the SSE URL

The `read_logs` tool also accepts `access_key` as a legacy fallback if the client cannot set transport-level authentication.

## Manifest Endpoint

Path:

```text
GET /manifest
```

Default response shape:

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

Relevant config fields:

- `public_base_url`
- `manifest_name`
- `manifest_title`
- `manifest_description`
- `manifest_version`
- `manifest_path`
- `manifest_remote_type`
- `manifest_remote_url`

Notes:

- The built-in transport is `sse`.
- If you advertise another remote type, set `manifest_remote_url` explicitly to a compatible endpoint.
- The manifest does not include secrets.

## Health Endpoint

Path:

```text
GET /health
```

Example response:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "time": "2026-04-16T12:00:00Z",
  "auth_required": true,
  "tls": true,
  "log_file": "/var/log/syslog",
  "log_file_accessible": true,
  "manifest_url": "https://logger.example.com/manifest",
  "remote_url": "https://logger.example.com/sse"
}
```

Behavior:

- returns HTTP `200` with `status: "ok"` when the configured log file is accessible
- returns HTTP `503` with `status: "degraded"` when the configured log file cannot be opened

## MCP Tool: `read_logs`

Description:

Read and search syslog entries with date filtering, wildcard matching, pagination, and optional response encryption.

Parameters:

- `access_key` (`string`, optional): legacy fallback access key
- `start_date` (`string`, optional): `2006-01-02` or `2006-01-02T15:04:05`
- `end_date` (`string`, optional): `2006-01-02` or `2006-01-02T15:04:05`
- `pattern` (`string`, optional): wildcard search string such as `error*disk`
- `page` (`number`, optional): page number, default `1`
- `page_size` (`number`, optional): entries per page, default `100`, max `1000`
- `encrypt` (`boolean`, optional): encrypt the response using AES-256-GCM

### Date Filtering

Supported input formats:

- `2006-01-02`
- `2006-01-02T15:04:05`
- `2006-01-02 15:04:05`

Syslog timestamps are parsed from the log line prefix, for example:

```text
Apr 15 10:30:00
```

The server infers the year and handles year rollover around New Year.

### Pattern Matching

The `pattern` field supports `*` as a wildcard.

Examples:

- `error*disk`
- `ssh*failed`
- `*panic*`

### Pagination

- default `page`: `1`
- default `page_size`: `100`
- maximum `page_size`: `1000`

Response header text format:

```text
Total: 250 entries | Page 2/3 (size: 100)
---
```

### Encryption

If `encrypt: true` is passed and `encryption_key` is configured, the response is returned as:

```text
ENC:<base64>
```

The payload is `nonce + ciphertext`, encrypted with AES-256-GCM using `SHA-256(encryption_key)` as the AES-256 key.

Python decryption example:

```python
import base64
import hashlib
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

data = base64.b64decode(response.removeprefix("ENC:"))
key = hashlib.sha256(b"my-secret-encryption-key").digest()
plaintext = AESGCM(key).decrypt(data[:12], data[12:], None).decode()
```

## Configuration Summary

Main config fields affecting API behavior:

- `access_key`
- `syslog_path`
- `tls`
- `cert_file`
- `key_file`
- `encryption_key`
- `public_base_url`
- `manifest_*`
- `health_path`
