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
bash ./scripts/build-deb.sh 0.1.0 amd64
```

This creates:

- `build/loggermcp_0.1.0_amd64.deb`
- `build/loggermcp-linux-amd64`

Supported architectures:

- `amd64`
- `arm64`

Example for ARM64:

```bash
bash ./scripts/build-deb.sh 0.1.0 arm64
```

Notes:

- local `.deb` building requires `dpkg-deb`
- on macOS this usually fails unless Debian packaging tools are installed separately
- the script now exits early with a clear error if required tools are missing

## Build Package Artifacts In GitHub Actions

The repository now has a package build workflow that runs on:

- push to `main`
- pull requests
- manual `workflow_dispatch`

That workflow builds:

- `.deb` packages for `amd64` and `arm64`
- standalone Linux binaries for `amd64` and `arm64`

The outputs are uploaded as GitHub Actions artifacts.

Each workflow run also writes the exact `.deb` and binary filenames to the GitHub Actions job summary.

Use this path when you need package artifacts without creating a tagged release.

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
gpg --list-secret-keys --keyid-format=long
gpg --armor --export-secret-keys YOUR_KEY_ID > gpg-private-key.asc
```

Set the GitHub Actions secrets as:

- `GPG_PRIVATE_KEY`: the full contents of `gpg-private-key.asc`, including the
  `-----BEGIN PGP PRIVATE KEY BLOCK-----` and `-----END PGP PRIVATE KEY BLOCK-----`
  lines
- `GPG_PASSPHRASE`: the passphrase for that private key
- `GPG_KEY_ID`: the key ID from `gpg --list-secret-keys --keyid-format=long`
