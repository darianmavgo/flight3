#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 Chrome App"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
OUTPUT_ZIP="$BUILD_DIR/flight3-chrome-app.zip"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -f "$OUTPUT_ZIP"

# Copy sample data (if needed for demo)
if [ -d "$PROJECT_ROOT/pb_public/sample_data" ]; then
    echo "Copying sample data..."
    cp -r "$PROJECT_ROOT/pb_public/sample_data" "$BUILD_DIR/"
fi

# Create placeholder icons (simple colored squares)
echo "Creating placeholder icons..."
for size in 16 48 128; do
    # Create a simple SVG and convert to PNG if imagemagick is available
    if command -v convert &> /dev/null; then
        convert -size ${size}x${size} xc:'#667eea' "$BUILD_DIR/icon${size}.png"
    else
        # Create placeholder text file if imagemagick not available
        echo "Icon placeholder - ${size}x${size}" > "$BUILD_DIR/icon${size}.png.txt"
        echo "Warning: ImageMagick not found. Icon placeholders created as text files."
    fi
done

# Create README
cat > "$BUILD_DIR/README.txt" << 'EOF'
Flight3 Chrome Extension
========================

This is a Chrome extension that provides quick access to your local Flight3 server.

Installation:
1. Ensure Flight3 server is running (./flight serve)
2. Open Chrome and navigate to chrome://extensions/
3. Enable "Developer mode" in the top right
4. Click "Load unpacked"
5. Select this directory

Usage:
- Click the Flight3 icon in your Chrome toolbar
- Check server status
- Quick links to Admin UI and Data Browser

Note: The server must be running on http://localhost:8090 for this extension to work.
EOF

# Create the zip file (excluding the build script itself and any existing zip)
echo "Creating Chrome app package..."
cd "$BUILD_DIR"
zip -r "$OUTPUT_ZIP" \
    manifest.json \
    popup.html \
    popup.js \
    background.js \
    README.txt \
    icon*.png* \
    -x "build.sh" -x "*.zip"

echo ""
echo "========================================="
echo "✓ Chrome app build complete!"
echo "Output: $OUTPUT_ZIP"
echo ""
echo "To load in Chrome:"
echo "1. Open chrome://extensions/"
echo "2. Enable Developer mode"
echo "3. Click 'Load unpacked'"
echo "4. Select: $BUILD_DIR"
echo "========================================="
