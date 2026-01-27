#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 for Windows"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
OUTPUT_DIR="$BUILD_DIR/dist"
BINARY_NAME="flight3.exe"
ZIP_FILE="$BUILD_DIR/flight3-windows-amd64.zip"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -rf "$OUTPUT_DIR" "$ZIP_FILE"
mkdir -p "$OUTPUT_DIR"

# Build the Go binary for Windows
echo "Compiling Go binary for Windows (amd64)..."
cd "$PROJECT_ROOT"

# Note: CGO is needed for SQLite support
# On macOS/Linux, we need MinGW-w64 for cross-compilation with CGO
# Check if cross-compiler is available
if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "Using MinGW-w64 cross-compiler..."
    CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build \
        -o "$OUTPUT_DIR/$BINARY_NAME" \
        -ldflags="-s -w -H windowsgui" \
        ./cmd/flight
else
    echo "Warning: MinGW-w64 not found. Building without CGO (using pure Go SQLite)..."
    echo "For full SQLite support, install: brew install mingw-w64"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
        -o "$OUTPUT_DIR/$BINARY_NAME" \
        -ldflags="-s -w" \
        ./cmd/flight
fi

echo "✓ Binary compiled: $(du -h "$OUTPUT_DIR/$BINARY_NAME" | cut -f1)"

# Copy README
cp "$BUILD_DIR/README.txt" "$OUTPUT_DIR/"

# Copy public assets
echo "Copying public assets..."
if [ -d "$PROJECT_ROOT/pb_public" ]; then
    cp -r "$PROJECT_ROOT/pb_public" "$OUTPUT_DIR/"
fi

# Create a batch file for easy launching
cat > "$OUTPUT_DIR/start-flight3.bat" << 'EOF'
@echo off
echo ========================================
echo Starting Flight3 Server
echo ========================================
echo.

REM Create data directories if they don't exist
if not exist pb_data mkdir pb_data
if not exist logs mkdir logs

REM Start Flight3
echo Starting server on http://localhost:8090
echo.
echo Press Ctrl+C to stop the server
echo.

flight3.exe serve --http=0.0.0.0:8090

pause
EOF

# Create a batch file to open the admin UI
cat > "$OUTPUT_DIR/open-admin.bat" << 'EOF'
@echo off
start http://localhost:8090/_/
EOF

# Create installation script
cat > "$OUTPUT_DIR/install-service.bat" << 'EOF'
@echo off
echo This script helps install Flight3 as a Windows service using NSSM
echo.
echo Please ensure you have downloaded NSSM from https://nssm.cc/download
echo and placed nssm.exe in this directory.
echo.

if not exist nssm.exe (
    echo ERROR: nssm.exe not found in current directory
    echo Please download from https://nssm.cc/download
    pause
    exit /b 1
)

echo Installing Flight3 service...
nssm install Flight3 "%~dp0flight3.exe" serve --http=0.0.0.0:8090
nssm set Flight3 AppDirectory "%~dp0"
nssm set Flight3 DisplayName "Flight3 Data Server"
nssm set Flight3 Description "Flight3 - Enhanced Data Serving Platform"
nssm set Flight3 Start SERVICE_AUTO_START

echo.
echo Service installed successfully!
echo.
echo To start the service, run: nssm start Flight3
echo To stop the service, run: nssm stop Flight3
echo To remove the service, run: nssm remove Flight3 confirm
echo.
pause
EOF

# Create the zip file
echo "Creating Windows distribution archive..."
cd "$BUILD_DIR"
zip -r "$ZIP_FILE" dist/

echo ""
echo "========================================="
echo "✓ Windows build complete!"
echo ""
echo "Output directory: $OUTPUT_DIR"
echo "Archive: $ZIP_FILE"
echo ""
echo "Contents:"
ls -lh "$OUTPUT_DIR"
echo ""
echo "To test (on Windows):"
echo "1. Extract the zip file"
echo "2. Run start-flight3.bat"
echo "3. Open http://localhost:8090 in a browser"
echo "========================================="
