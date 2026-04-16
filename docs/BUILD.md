# Build From Source

## Requirements

- Go `1.23`
- access to the target syslog file
- Linux or Ubuntu target for actual deployment

## Local Development Build

Create a local config:

```bash
cp config.yaml.example config.yaml
```

Minimal example:

```yaml
access_key: "your-secret-key"
syslog_path: "/var/log/syslog"
port: 7777
tls: false
encryption_key: ""
```

Build and run:

```bash
go build -o loggerMCP .
./loggerMCP
```

Run with a custom config path:

```bash
./loggerMCP /path/to/config.yaml
```

## Local Verification

Health endpoint:

```bash
curl -fsSL http://localhost:7777/health
```

Manifest endpoint:

```bash
curl -fsSL http://localhost:7777/manifest
```

## Build Debian Package Locally

Use the helper script:

```bash
./scripts/build-deb.sh 0.1.0 amd64
```

This creates:

- `build/loggermcp_0.1.0_amd64.deb`
- `build/loggermcp-linux-amd64`

Supported architectures:

- `amd64`
- `arm64`

Example for ARM64:

```bash
./scripts/build-deb.sh 0.1.0 arm64
```

## Install Local Package

```bash
sudo dpkg -i build/loggermcp_0.1.0_amd64.deb
sudo systemctl start loggermcp
sudo systemctl status loggermcp
```

## Release Build and Publish

The GitHub Actions workflow does the following on a tag push like `v0.1.0`:

- builds `.deb` packages for `amd64` and `arm64`
- builds standalone Linux binaries for `amd64` and `arm64`
- publishes the APT repository to GitHub Pages
- creates a GitHub release with all artifacts attached

Trigger a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Release Prerequisites

You need:

- a GPG key for APT repository signing
- GitHub Actions secrets:
  - `GPG_PRIVATE_KEY`
  - `GPG_PASSPHRASE`
  - `GPG_KEY_ID`
- GitHub Pages enabled on the `gh-pages` branch

Create and export a GPG key:

```bash
gpg --full-generate-key
gpg --armor --export-secret-keys YOUR_KEY_ID
```
