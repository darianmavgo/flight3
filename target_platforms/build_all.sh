#!/bin/bash
set -e

echo "========================================"
echo "Flight3 Multi-Platform Build System"
echo "========================================"
echo ""

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Project: Flight3"
echo "Build directory: $SCRIPT_DIR"
echo "Timestamp: $(date)"
echo ""

# Track build results
SUCCESSFUL_BUILDS=()
FAILED_BUILDS=()

# Function to run a build
run_build() {
    local platform=$1
    local build_script=$2
    
    echo "========================================"
    echo "Building: $platform"
    echo "========================================"
    
    if [ -f "$build_script" ]; then
        chmod +x "$build_script"
        if bash "$build_script"; then
            SUCCESSFUL_BUILDS+=("$platform")
            echo "✓ $platform build succeeded"
        else
            FAILED_BUILDS+=("$platform")
            echo "✗ $platform build failed"
        fi
    else
        FAILED_BUILDS+=("$platform (script not found)")
        echo "✗ Build script not found: $build_script"
    fi
    
    echo ""
}

# Build each platform
run_build "Chrome App" "$SCRIPT_DIR/chrome_app/build.sh"
run_build "Linux Service" "$SCRIPT_DIR/linux_service/build.sh"
run_build "macOS Apple Silicon" "$SCRIPT_DIR/macsilicon_app/build.sh"
run_build "macOS Intel" "$SCRIPT_DIR/macx86_app/build.sh"
run_build "Ubuntu Package" "$SCRIPT_DIR/ubuntu_app/build.sh"
run_build "Windows" "$SCRIPT_DIR/windows_app/build.sh"

# Print summary
echo "========================================"
echo "Build Summary"
echo "========================================"
echo ""

if [ ${#SUCCESSFUL_BUILDS[@]} -gt 0 ]; then
    echo "✓ Successful builds (${#SUCCESSFUL_BUILDS[@]}):"
    for build in "${SUCCESSFUL_BUILDS[@]}"; do
        echo "  - $build"
    done
    echo ""
fi

if [ ${#FAILED_BUILDS[@]} -gt 0 ]; then
    echo "✗ Failed builds (${#FAILED_BUILDS[@]}):"
    for build in "${FAILED_BUILDS[@]}"; do
        echo "  - $build"
    done
    echo ""
fi

# List all outputs
echo "========================================"
echo "Build Artifacts"
echo "========================================"
echo ""

echo "Chrome App:"
[ -f "$SCRIPT_DIR/chrome_app/flight3-chrome-app.zip" ] && \
    ls -lh "$SCRIPT_DIR/chrome_app/flight3-chrome-app.zip" || \
    echo "  Not found"
echo ""

echo "Linux Service:"
[ -f "$SCRIPT_DIR/linux_service/flight3-linux-service.tar.gz" ] && \
    ls -lh "$SCRIPT_DIR/linux_service/flight3-linux-service.tar.gz" || \
    echo "  Not found"
echo ""

echo "macOS Apple Silicon:"
[ -f "$SCRIPT_DIR/macsilicon_app/Flight3-macOS-AppleSilicon.dmg" ] && \
    ls -lh "$SCRIPT_DIR/macsilicon_app/Flight3-macOS-AppleSilicon.dmg" || \
    [ -d "$SCRIPT_DIR/macsilicon_app/Flight3.app" ] && \
    echo "  Flight3.app ($(du -sh "$SCRIPT_DIR/macsilicon_app/Flight3.app" | cut -f1))" || \
    echo "  Not found"
echo ""

echo "macOS Intel:"
[ -f "$SCRIPT_DIR/macx86_app/Flight3-macOS-Intel.dmg" ] && \
    ls -lh "$SCRIPT_DIR/macx86_app/Flight3-macOS-Intel.dmg" || \
    [ -d "$SCRIPT_DIR/macx86_app/Flight3.app" ] && \
    echo "  Flight3.app ($(du -sh "$SCRIPT_DIR/macx86_app/Flight3.app" | cut -f1))" || \
    echo "  Not found"
echo ""

echo "Ubuntu Package:"
[ -f "$SCRIPT_DIR/ubuntu_app/flight3_1.0.0_amd64.deb" ] && \
    ls -lh "$SCRIPT_DIR/ubuntu_app/flight3_1.0.0_amd64.deb" || \
    echo "  Not found"
echo ""

echo "Windows:"
[ -f "$SCRIPT_DIR/windows_app/flight3-windows-amd64.zip" ] && \
    ls -lh "$SCRIPT_DIR/windows_app/flight3-windows-amd64.zip" || \
    echo "  Not found"
echo ""

# Exit with error if any builds failed
if [ ${#FAILED_BUILDS[@]} -gt 0 ]; then
    echo "========================================"
    echo "⚠ Some builds failed. See above for details."
    echo "========================================"
    exit 1
else
    echo "========================================"
    echo "✓ All builds completed successfully!"
    echo "========================================"
    exit 0
fi
