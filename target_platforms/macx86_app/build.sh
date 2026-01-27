#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 for macOS Intel (x86_64)"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
APP_NAME="Flight3"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_BUNDLE/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -rf "$APP_BUNDLE"

# Create app bundle structure
echo "Creating app bundle structure..."
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"

# Build the Go binary for macOS AMD64
echo "Compiling Go binary for macOS (amd64)..."
cd "$PROJECT_ROOT"

CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build \
    -o "$MACOS_DIR/flight3" \
    -ldflags="-s -w" \
    ./cmd/flight

echo "✓ Binary compiled: $(du -h "$MACOS_DIR/flight3" | cut -f1)"

# Make binary executable
chmod +x "$MACOS_DIR/flight3"

# Copy Info.plist
echo "Copying Info.plist..."
cp "$BUILD_DIR/Info.plist" "$CONTENTS_DIR/"

# Copy public assets to Resources
echo "Copying public assets..."
if [ -d "$PROJECT_ROOT/pb_public" ]; then
    cp -r "$PROJECT_ROOT/pb_public" "$RESOURCES_DIR/"
fi

# Create a launcher script that sets up the environment
cat > "$MACOS_DIR/launcher.sh" << 'EOF'
#!/bin/bash

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
RESOURCES_DIR="$DIR/../Resources"

# Set up data directory in user's Application Support
DATA_DIR="$HOME/Library/Application Support/Flight3"
mkdir -p "$DATA_DIR"/{pb_data,logs}

# Export environment variables
export FLIGHT3_DATA_DIR="$DATA_DIR/pb_data"
export FLIGHT3_LOG_DIR="$DATA_DIR/logs"
export FLIGHT3_PUBLIC_DIR="$RESOURCES_DIR/pb_public"

# Launch Flight3
cd "$DATA_DIR"
exec "$DIR/flight3" serve
EOF

chmod +x "$MACOS_DIR/launcher.sh"

# Create README
cat > "$BUILD_DIR/README.md" << EOF
# Flight3 for macOS Intel

This is Flight3 compiled for Intel-based Macs (x86_64).

## Installation

1. Copy Flight3.app to your Applications folder
2. On first launch, you may need to right-click and select "Open" to bypass Gatekeeper
   - Admin UI: http://[::1]:PORT/_/ (where PORT is the random port assigned)
   - Data API: http://[::1]:PORT/

Flight3 will automatically attempt to launch Google Chrome to these addresses on startup.

## Data Location

Flight3 stores its data in:
\`~/Library/Application Support/Flight3/\`

This includes:
- \`pb_data/\` - PocketBase database and cache
- \`logs/\` - Application logs

## Running from Terminal

You can also run Flight3 from the command line:

\`\`\`bash
/Applications/Flight3.app/Contents/MacOS/flight3 serve
\`\`\`

## Uninstallation

1. Quit Flight3
2. Delete Flight3.app from Applications
3. (Optional) Remove data: \`rm -rf ~/Library/Application\ Support/Flight3\`

## Requirements

- macOS 10.13 (High Sierra) or later
- Intel processor

## Support

For issues and documentation, visit:
https://github.com/darianmavgo/flight3
EOF

# Create DMG if hdiutil is available
if command -v hdiutil &> /dev/null; then
    echo "Creating disk image..."
    DMG_NAME="Flight3-macOS-Intel.dmg"
    rm -f "$BUILD_DIR/$DMG_NAME"
    
    # Create temporary dmg directory
    TMP_DMG_DIR="$BUILD_DIR/dmg_temp"
    rm -rf "$TMP_DMG_DIR"
    mkdir -p "$TMP_DMG_DIR"
    
    # Copy app to temp directory
    cp -r "$APP_BUNDLE" "$TMP_DMG_DIR/"
    cp "$BUILD_DIR/README.md" "$TMP_DMG_DIR/"
    
    # Create DMG
    hdiutil create -volname "Flight3" -srcfolder "$TMP_DMG_DIR" -ov -format UDZO "$BUILD_DIR/$DMG_NAME"
    
    # Clean up
    rm -rf "$TMP_DMG_DIR"
    
    echo "✓ Disk image created: $DMG_NAME"
fi

echo ""
echo "========================================="
echo "✓ macOS Intel build complete!"
echo ""
echo "App bundle: $APP_BUNDLE"
if [ -f "$BUILD_DIR/Flight3-macOS-Intel.dmg" ]; then
    echo "Disk image: Flight3-macOS-Intel.dmg"
fi
echo ""
echo "To test (if on Intel Mac):"
echo "  open $APP_BUNDLE"
echo ""
echo "Or run from terminal:"
echo "  $MACOS_DIR/flight3 serve"
echo "========================================="
