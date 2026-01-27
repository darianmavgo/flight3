#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 Linux Service"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
OUTPUT_DIR="$BUILD_DIR/dist"
BINARY_NAME="flight3"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build the Go binary for Linux
echo "Compiling Go binary for Linux (amd64)..."
cd "$PROJECT_ROOT"

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -o "$OUTPUT_DIR/$BINARY_NAME" \
    -ldflags="-s -w" \
    ./cmd/flight

echo "✓ Binary compiled: $(du -h "$OUTPUT_DIR/$BINARY_NAME" | cut -f1)"

# Copy service file
echo "Copying systemd service file..."
cp "$BUILD_DIR/flight3.service" "$OUTPUT_DIR/"

# Copy public assets
echo "Copying public assets..."
if [ -d "$PROJECT_ROOT/pb_public" ]; then
    cp -r "$PROJECT_ROOT/pb_public" "$OUTPUT_DIR/"
fi

# Create installation script
cat > "$OUTPUT_DIR/install.sh" << 'EOF'
#!/bin/bash
set -e

echo "Installing Flight3 Service..."

# Check for root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (sudo)"
    exit 1
fi

# Create user and group
if ! id -u flight3 > /dev/null 2>&1; then
    echo "Creating flight3 user..."
    useradd -r -s /bin/false flight3
fi

# Create directories
echo "Creating installation directories..."
mkdir -p /opt/flight3/{pb_data,logs,pb_public}
mkdir -p /usr/local/bin

# Copy binary
echo "Installing binary..."
cp flight3 /usr/local/bin/
chmod +x /usr/local/bin/flight3

# Copy assets
if [ -d "pb_public" ]; then
    cp -r pb_public/* /opt/flight3/pb_public/
fi

# Set permissions
chown -R flight3:flight3 /opt/flight3
chmod 755 /usr/local/bin/flight3

# Install systemd service
echo "Installing systemd service..."
cp flight3.service /etc/systemd/system/
systemctl daemon-reload

echo ""
echo "========================================="
echo "✓ Flight3 installed successfully!"
echo ""
echo "To start the service:"
echo "  sudo systemctl start flight3"
echo ""
echo "To enable at boot:"
echo "  sudo systemctl enable flight3"
echo ""
echo "To check status:"
echo "  sudo systemctl status flight3"
echo ""
echo "To view logs:"
echo "  sudo journalctl -u flight3 -f"
echo "========================================="
EOF

chmod +x "$OUTPUT_DIR/install.sh"

# Create uninstall script
cat > "$OUTPUT_DIR/uninstall.sh" << 'EOF'
#!/bin/bash
set -e

echo "Uninstalling Flight3 Service..."

# Check for root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (sudo)"
    exit 1
fi

# Stop and disable service
systemctl stop flight3 2>/dev/null || true
systemctl disable flight3 2>/dev/null || true

# Remove files
rm -f /etc/systemd/system/flight3.service
rm -f /usr/local/bin/flight3
rm -rf /opt/flight3

# Remove user
userdel flight3 2>/dev/null || true

systemctl daemon-reload

echo "✓ Flight3 uninstalled"
EOF

chmod +x "$OUTPUT_DIR/uninstall.sh"

# Create README
cat > "$OUTPUT_DIR/README.md" << 'EOF'
# Flight3 Linux Service

This package contains Flight3 compiled for Linux (amd64) as a systemd service.

## Installation

```bash
sudo ./install.sh
```

## Starting the Service

```bash
# Start immediately
sudo systemctl start flight3

# Enable at boot
sudo systemctl enable flight3

# Check status
sudo systemctl status flight3
```

## Accessing Flight3

Once started, Flight3 will be available at:
- Admin UI: http://your-server:8090/_/
- Data API: http://your-server:8090/

## Logs

View logs with:
```bash
sudo journalctl -u flight3 -f
```

## Configuration

The service runs as user `flight3` with data stored in:
- Data directory: `/opt/flight3/pb_data`
- Log directory: `/opt/flight3/logs`
- Public assets: `/opt/flight3/pb_public`

## Uninstallation

```bash
sudo ./uninstall.sh
```

## Firewall

If using a firewall, allow port 8090:
```bash
sudo ufw allow 8090/tcp
```

## Requirements

- Linux (amd64)
- systemd
- SQLite support (included)
EOF

# Create tarball
echo "Creating distribution tarball..."
cd "$BUILD_DIR"
tar czf "flight3-linux-service.tar.gz" -C dist .

echo ""
echo "========================================="
echo "✓ Linux service build complete!"
echo ""
echo "Output directory: $OUTPUT_DIR"
echo "Archive: $BUILD_DIR/flight3-linux-service.tar.gz"
echo ""
echo "Contents:"
ls -lh "$OUTPUT_DIR"
echo ""
echo "To install on a Linux server:"
echo "1. Copy flight3-linux-service.tar.gz to the server"
echo "2. Extract: tar xzf flight3-linux-service.tar.gz"
echo "3. Run: sudo ./install.sh"
echo "========================================="
