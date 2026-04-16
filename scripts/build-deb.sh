#!/bin/bash
set -euo pipefail

VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"
PKG_NAME="loggermcp"
BUILD_DIR="build/${PKG_NAME}_${VERSION}_${ARCH}"
DEBIAN_VERSION_REGEX='^([0-9]+:)?[0-9][A-Za-z0-9.+~-]*(-[A-Za-z0-9.+~]+)?$'

require_command() {
	command -v "$1" > /dev/null 2>&1 || {
		echo "Missing required command: $1" >&2
		exit 1
	}
}

if [[ ! "${VERSION}" =~ ${DEBIAN_VERSION_REGEX} ]]; then
	echo "Invalid Debian package version: ${VERSION}" >&2
	echo "Version must start with a digit and use Debian version characters only." >&2
	exit 1
fi

echo "==> Building ${PKG_NAME} v${VERSION} (${ARCH})"

require_command go
require_command dpkg-deb
require_command sed

# Build Go binary
echo "==> Compiling..."
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags="-s -w" -o build/loggermcp .
cp build/loggermcp "build/loggermcp-linux-${ARCH}"

# Create .deb package structure
echo "==> Creating package..."
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/etc/loggermcp"
mkdir -p "${BUILD_DIR}/lib/systemd/system"

# Binary
cp build/loggermcp "${BUILD_DIR}/usr/bin/loggermcp"
chmod 755 "${BUILD_DIR}/usr/bin/loggermcp"

# Config
cp config.yaml.example "${BUILD_DIR}/etc/loggermcp/config.yaml.example"

# Systemd unit
cp debian/loggermcp.service "${BUILD_DIR}/lib/systemd/system/loggermcp.service"

# DEBIAN control
sed \
	-e "s/VERSION_PLACEHOLDER/${VERSION}/g" \
	-e "s/Architecture: amd64/Architecture: ${ARCH}/g" \
	debian/control > "${BUILD_DIR}/DEBIAN/control"

# Scripts
cp debian/postinst "${BUILD_DIR}/DEBIAN/postinst"
cp debian/prerm "${BUILD_DIR}/DEBIAN/prerm"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/DEBIAN/prerm"

# Build .deb
dpkg-deb --build "${BUILD_DIR}"
echo "==> Done: ${BUILD_DIR}.deb"
