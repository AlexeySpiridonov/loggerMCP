#!/bin/bash
set -e

REPO="AlexeySpiridonov/loggerMCP"
INSTALL_DIR="/usr/bin"
CONFIG_DIR="/etc/loggermcp"
SERVICE_NAME="loggermcp"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[✗]${NC} $1"; exit 1; }

# Check root
if [ "$(id -u)" -ne 0 ]; then
    error "Run with sudo: curl -fsSL ... | sudo bash"
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    aarch64) GOARCH="arm64" ;;
    arm64)   GOARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH" ;;
esac

info "Architecture: $GOARCH"

# Get latest release
info "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    error "Failed to determine latest release"
fi
VERSION="${LATEST#v}"
info "Version: $VERSION"

# Look for .deb in release assets
DEB_NAME="loggermcp_${VERSION}_${GOARCH}.deb"
DEB_URL="https://github.com/${REPO}/releases/download/${LATEST}/${DEB_NAME}"

info "Downloading ${DEB_NAME}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL -o "${TMP_DIR}/${DEB_NAME}" "$DEB_URL" || {
    # .deb not found, download binary directly
    warn ".deb package not found, installing binary directly..."

    BIN_URL="https://github.com/${REPO}/releases/download/${LATEST}/loggermcp-linux-${GOARCH}"
    curl -fsSL -o "${TMP_DIR}/loggermcp" "$BIN_URL" || error "Failed to download binary"

    chmod 755 "${TMP_DIR}/loggermcp"
    cp "${TMP_DIR}/loggermcp" "${INSTALL_DIR}/loggermcp"
    info "Binary installed: ${INSTALL_DIR}/loggermcp"

    # Create service user
    if ! getent passwd $SERVICE_NAME > /dev/null 2>&1; then
        adduser --system --group --no-create-home --shell /usr/sbin/nologin $SERVICE_NAME
        info "Created user: $SERVICE_NAME"
    fi
    usermod -aG adm $SERVICE_NAME || true

    # Config
    mkdir -p "$CONFIG_DIR"
    if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
        cat > "${CONFIG_DIR}/config.yaml" <<'EOF'
access_key: "CHANGE-ME"
syslog_path: "/var/log/syslog"
port: 7777
tls: true
encryption_key: ""
EOF
        chown ${SERVICE_NAME}:${SERVICE_NAME} "${CONFIG_DIR}/config.yaml"
        chmod 600 "${CONFIG_DIR}/config.yaml"
        warn "Config created: ${CONFIG_DIR}/config.yaml — EDIT access_key!"
    else
        info "Config already exists, skipping"
    fi

    # Systemd unit
    cat > /lib/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=loggerMCP - MCP server for Ubuntu syslog
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/loggermcp ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=5
User=${SERVICE_NAME}
Group=${SERVICE_NAME}
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true
ReadOnlyPaths=/var/log

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    info "Systemd service installed and enabled"
    info "Done! Edit the config and start the service:"
    echo ""
    echo "  sudo nano ${CONFIG_DIR}/config.yaml"
    echo "  sudo systemctl start ${SERVICE_NAME}"
    echo ""
    exit 0
}

# Install .deb
info "Installing .deb package..."
dpkg -i "${TMP_DIR}/${DEB_NAME}" || {
    warn "Fixing dependencies..."
    apt-get install -f -y
}

info "Done! Edit the config and start the service:"
echo ""
echo "  sudo nano ${CONFIG_DIR}/config.yaml"
echo "  sudo systemctl start ${SERVICE_NAME}"
echo ""
