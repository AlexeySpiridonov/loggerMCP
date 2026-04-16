#!/bin/bash
set -euo pipefail

VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"
PKG_NAME="loggermcp"
BUILD_DIR="build/${PKG_NAME}_${VERSION}_${ARCH}"

echo "==> Building ${PKG_NAME} v${VERSION} (${ARCH})"

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
sed "s/VERSION_PLACEHOLDER/${VERSION}/g" debian/control > "${BUILD_DIR}/DEBIAN/control"
sed -i "s/amd64/${ARCH}/g" "${BUILD_DIR}/DEBIAN/control"

# Scripts
cp debian/postinst "${BUILD_DIR}/DEBIAN/postinst"
cp debian/prerm "${BUILD_DIR}/DEBIAN/prerm"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/DEBIAN/prerm"

# conffiles — prevent apt from overwriting config on upgrade
cat > "${BUILD_DIR}/DEBIAN/conffiles" <<EOF
/etc/loggermcp/config.yaml
EOF

# Build .deb
dpkg-deb --build "${BUILD_DIR}"
echo "==> Done: ${BUILD_DIR}.deb"
