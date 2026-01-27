#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 Ubuntu/Debian Package"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
PKG_NAME="flight3"
PKG_DIR="$BUILD_DIR/package"
DEB_FILE="$BUILD_DIR/flight3_1.0.0_amd64.deb"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -rf "$PKG_DIR" "$DEB_FILE"

# Create package directory structure
echo "Creating package structure..."
mkdir -p "$PKG_DIR"/{DEBIAN,usr/local/bin,etc/systemd/system,opt/flight3/pb_public}

# Copy DEBIAN control files
cp "$BUILD_DIR/DEBIAN/control" "$PKG_DIR/DEBIAN/"
cp "$BUILD_DIR/DEBIAN/postinst" "$PKG_DIR/DEBIAN/"
cp "$BUILD_DIR/DEBIAN/prerm" "$PKG_DIR/DEBIAN/"
chmod 755 "$PKG_DIR/DEBIAN/postinst" "$PKG_DIR/DEBIAN/prerm"

# Build the Go binary for Linux
echo "Compiling Go binary for Linux (amd64)..."
cd "$PROJECT_ROOT"

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -o "$PKG_DIR/usr/local/bin/flight3" \
    -ldflags="-s -w" \
    ./cmd/flight

echo "✓ Binary compiled: $(du -h "$PKG_DIR/usr/local/bin/flight3" | cut -f1)"

# Make binary executable
chmod 755 "$PKG_DIR/usr/local/bin/flight3"

# Copy systemd service file
cat > "$PKG_DIR/etc/systemd/system/flight3.service" << 'EOF'
[Unit]
Description=Flight3 Data Serving Platform
Documentation=https://github.com/darianmavgo/flight3
After=network.target

[Service]
Type=simple
User=flight3
Group=flight3
WorkingDirectory=/opt/flight3
ExecStart=/usr/local/bin/flight3 serve --http=0.0.0.0:8090
Restart=always
RestartSec=10

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/flight3/pb_data /opt/flight3/logs

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=flight3

# Environment
Environment="FLIGHT3_DATA_DIR=/opt/flight3/pb_data"
Environment="FLIGHT3_LOG_DIR=/opt/flight3/logs"

[Install]
WantedBy=multi-user.target
EOF

# Copy public assets
echo "Copying public assets..."
if [ -d "$PROJECT_ROOT/pb_public" ]; then
    cp -r "$PROJECT_ROOT/pb_public"/* "$PKG_DIR/opt/flight3/pb_public/" 2>/dev/null || true
fi

# Build the deb package
echo "Building .deb package..."
dpkg-deb --build "$PKG_DIR" "$DEB_FILE"

# Clean up package directory
rm -rf "$PKG_DIR"

echo ""
echo "========================================="
echo "✓ Ubuntu package build complete!"
echo ""
echo "Package: $DEB_FILE"
echo "Size: $(du -h "$DEB_FILE" | cut -f1)"
echo ""
echo "To install:"
echo "  sudo dpkg -i flight3_1.0.0_amd64.deb"
echo ""
echo "To remove:"
echo "  sudo dpkg -r flight3"
echo ""
echo "Package info:"
dpkg-deb --info "$DEB_FILE"
echo "========================================="
