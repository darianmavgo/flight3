# Flight3 Multi-Platform Build System

This directory contains build configurations and scripts for creating Flight3 distributions across multiple platforms.

## Available Platforms

### ✅ Successfully Built

1. **Chrome Extension** (`chrome_app/`)
   - Output: `flight3-chrome-app.zip`
   - Provides browser-based access to Flight3 server
   - Load unpacked in Chrome Developer Mode

2. **macOS Apple Silicon** (`macsilicon_app/`)
   - Output: `Flight3-macOS-AppleSilicon.dmg` (36MB)
   - Native app bundle for M1/M2/M3/M4 Macs
   - Requires macOS 11.0 (Big Sur) or later

3. **macOS Intel** (`macx86_app/`)
   - Output: `Flight3-macOS-Intel.dmg` (38MB)
   - App bundle for Intel-based Macs
   - Requires macOS 10.13 (High Sierra) or later

4. **Windows** (`windows_app/`)
   - Output: `flight3-windows-amd64.zip` (37MB exe)
   - Standalone executable with helper batch scripts
   - Includes service installer guide
   - Built without CGO (pure Go SQLite fallback)

### ⚠️ Requires Linux Build Environment

5. **Linux Service** (`linux_service/`)
   - Build script ready, requires Linux host
   - Includes systemd service file
   - Installation/uninstallation scripts included

6. **Ubuntu Package** (`ubuntu_app/`)
   - Build script ready, requires Linux host
   - Full Debian package structure with postinst/prerm scripts
   - Creates `.deb` package when built on Linux

### ✅ Cloud & Browser Platforms

7. **Google App Engine** (`gae_service/`)
   - Output: `deploy/` directory (ready for GCP deployment)
   - Includes app.yaml, deployment scripts, documentation
   - Auto-scaling configuration included
   - Health checks configured

8. **WebAssembly** (`wasm_app/`)
   - ⚠️ Build infrastructure complete but not compilable
   - Blocked by SQLite dependency (modernc.org/sqlite doesn't support js/wasm)
   - Full browser UI with modern design ready
   - See wasm_app/README.md for alternatives

## Build Instructions

### Build All Platforms (from macOS)

```bash
cd /Users/darianhickman/Documents/flight3/builds
./build_all.sh
```

This will build all platforms that can be built from macOS. Linux/Ubuntu builds will fail with cross-compilation errors but their scripts are ready. WASM will fail due to SQLite dependency constraints.

### Build Individual Platforms

```bash
# Chrome extension
./chrome_app/build.sh

# macOS Apple Silicon
./macsilicon_app/build.sh

# macOS Intel
./macx86_app/build.sh

# Windows
./windows_app/build.sh

# Google App Engine
./gae_service/build.sh

# WebAssembly (will fail - see wasm_app/README.md)
./wasm_app/build.sh

# Linux (requires Linux environment)
./linux_service/build.sh

# Ubuntu (requires Linux environment)
./ubuntu_app/build.sh
```

## Build Outputs Summary

```
builds/
├── chrome_app/
│   └── flight3-chrome-app.zip              # Chrome extension
├── macsilicon_app/
│   ├── Flight3.app/                        # App bundle
│   └── Flight3-macOS-AppleSilicon.dmg      # Disk image (36MB)
├── macx86_app/
│   ├── Flight3.app/                        # App bundle
│   └── Flight3-macOS-Intel.dmg             # Disk image (38MB)
├── windows_app/
│   └── flight3-windows-amd64.zip           # Contains 37MB .exe
├── linux_service/
│   └── flight3-linux-service.tar.gz        # (requires Linux build)
└── ubuntu_app/
    └── flight3_1.0.0_amd64.deb             # (requires Linux build)
```

## Building Linux Targets

Due to CGO requirements for SQLite support, Linux builds must be compiled on a Linux system or using Docker:

### Option 1: Using Docker

```bash
# From project root
docker run --rm -v "$PWD":/usr/src/flight3 -w /usr/src/flight3 \
  golang:1.25 bash -c "cd builds/linux_service && ./build.sh"
```

### Option 2: On an Ubuntu/Linux Machine

```bash
# Transfer the builds directory to a Linux machine, then:
cd builds/linux_service
./build.sh

cd ../ubuntu_app
./build.sh
```

## Cross-Compilation Limitations

- **macOS → Linux**: CGO cross-compilation requires complex toolchain setup
- **Solution**: Use Docker or build on actual Linux hardware
- **Windows from macOS**: Works with pure Go SQLite (CGO disabled)
  - For full CGO support: `brew install mingw-w64`

## Platform-Specific Notes

### Chrome App
- No server binary (web-only)
- Connects to local Flight3 server at `http://localhost:8090`
- Provides quick access popup and status checking

### macOS Apps
- Data stored in `~/Library/Application Support/Flight3/`
- Includes launcher script for environment setup
- May require "Open" from right-click menu on first launch (Gatekeeper)

### Linux Service
- Systemd integration included
- Runs as dedicated `flight3` user
- Auto-restart on failure
- Data in `/opt/flight3/`

### Ubuntu Package
- Standard `.deb` package
- Post-installation scripts create user and directories
- Integrates with systemd
- Install: `sudo dpkg -i flight3_1.0.0_amd64.deb`

### Windows
- Portable installation (extract anywhere)
- Batch files for easy launching
- NSSM service installer guide included
- Data stored alongside executable

## Testing Builds

### macOS (Current Platform)
```bash
# Test Silicon version
open ./macsilicon_app/Flight3.app

# Or run from terminal
./macsilicon_app/Flight3.app/Contents/MacOS/flight3 serve
```

### Chrome Extension
1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `builds/chrome_app/` directory

### Windows/Linux
Transfer packages to target platform and follow platform-specific README files.

## Troubleshooting

**Linux build fails on macOS:**
- Expected behavior due to CGO cross-compilation
- Use Docker or Linux machine (see "Building Linux Targets" above)

**Windows build missing CGO:**
- Install MinGW: `brew install mingw-w64`
- Rebuild with `./windows_app/build.sh`

**macOS app won't open:**
- Right-click → "Open" (first time only)
- Or: System Preferences → Security & Privacy → Allow

## Architecture

All builds include:
- Flight3 binary (or web assets for Chrome)
- PocketBase embedded server
- SQLite database support
- Public web assets from `pb_public/`
- Platform-specific launch helpers

## Version

Current build version: 1.0.0

## Support

For issues or questions:
- Project: https://github.com/darianmavgo/flight3
- Documentation: `/Users/darianhickman/Documents/flight3/README.md`
