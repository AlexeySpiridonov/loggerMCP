#!/bin/bash
set -e

VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"
PKG_NAME="loggermcp"
BUILD_DIR="build/${PKG_NAME}_${VERSION}_${ARCH}"

echo "==> Сборка ${PKG_NAME} v${VERSION} (${ARCH})"

# Собираем Go-бинарник
echo "==> Компиляция..."
CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags="-s -w" -o build/loggermcp .

# Формируем структуру .deb пакета
echo "==> Формируем пакет..."
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/etc/loggermcp"
mkdir -p "${BUILD_DIR}/lib/systemd/system"

# Бинарник
cp build/loggermcp "${BUILD_DIR}/usr/bin/loggermcp"
chmod 755 "${BUILD_DIR}/usr/bin/loggermcp"

# Конфиг
cp config.yaml.example "${BUILD_DIR}/etc/loggermcp/config.yaml.example"

# Systemd unit
cp debian/loggermcp.service "${BUILD_DIR}/lib/systemd/system/loggermcp.service"

# DEBIAN control
sed "s/VERSION_PLACEHOLDER/${VERSION}/g" debian/control > "${BUILD_DIR}/DEBIAN/control"
sed -i "s/amd64/${ARCH}/g" "${BUILD_DIR}/DEBIAN/control"

# Скрипты
cp debian/postinst "${BUILD_DIR}/DEBIAN/postinst"
cp debian/prerm "${BUILD_DIR}/DEBIAN/prerm"
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/DEBIAN/prerm"

# conffiles — чтобы apt не перезаписывал конфиг при апгрейде
cat > "${BUILD_DIR}/DEBIAN/conffiles" <<EOF
/etc/loggermcp/config.yaml
EOF

# Собираем .deb
dpkg-deb --build "${BUILD_DIR}"
echo "==> Готово: ${BUILD_DIR}.deb"
