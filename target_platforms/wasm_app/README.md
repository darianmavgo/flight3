# Flight3 WebAssembly Build

⚠️ **Important Limitation**: Flight3 cannot currently be compiled to WebAssembly due to SQLite dependency limitations.

## The Problem

Flight3 uses `modernc.org/sqlite` which is a pure-Go reimplementation of SQLite but with platform-specific code that doesn't support the `js/wasm` target. When attempting to compile to WASM, you'll encounter:

```
build constraints exclude all Go files in modernc.org/libc@v1.67.6/...
```

## Potential Solutions

### Option 1: Use Alternative SQLite Library

Replace `modernc.org/sqlite` with a WASM-compatible alternative:

**crawshaw.io/sqlite** supports WASM but requires building SQLite to WASM first:
```bash
# Complex setup required
git clone https://github.com/sqlite/sqlite
# Compile SQLite to WASM
# Link with crawshaw.io/sqlite
```

**zombiezen.com/go/sqlite** - Another option with potential WASM support

### Option 2: Use Browser Storage APIs

For a browser-only version, replace SQLite with:
- **IndexedDB** via `github.com/idb` 
- **LocalStorage** for simple key-value
- **In-memory** data structures

This would require significant code changes to abstract the storage layer.

### Option 3: Server-Side Only

Keep Flight3 as a server application and use the Chrome Extension instead:
- Chrome extension (already built) provides browser UI
- Connects to Flight3 server running locally or on GAE
- No WASM needed

## What's Included

Even though compilation fails, this directory contains a complete WASM build infrastructure ready for when/if SQLite support becomes available:

- ✅ `index.html` - Modern browser UI with status monitoring
- ✅ `flight3.js` - WASM loader and controller
- ✅ `build.sh` - Automated build script
- ✅ `serve.sh` - Local testing server
- ✅ `server.go` - Go-based development server
- ✅ `.htaccess` - Proper MIME types for production

## Current Recommendation

**Use the Chrome Extension instead**: It provides browser-based access to Flight3 without the WASM limitations.

```bash
# Install Chrome extension
cd ../chrome_app
# Follow installation instructions in that directory
```

Or deploy Flight3 to Google App Engine:

```bash
# Deploy to cloud
cd ../gae_service/deploy
./deploy.sh
```

## For Future Development

If PocketBase/SQLite dependency can be abstracted or replaced:

1. Update `internal/flight/` to use an interface for storage
2. Implement browser-specific storage backend
3. Build tags to select storage implementation:
   ```go
   //go:build !js
   // Use SQLite
   
   //go:build js
   // Use IndexedDB
   ```

## Build Attempts

The build script is functional and will:
1. ✅ Locate and copy `wasm_exec.js` from Go installation
2. ✅ Attempt to compile with `GOOS=js GOARCH=wasm`
3. ❌ Fail due to SQLite dependency constraints

You can try running it yourself:

```bash
./build.sh
```

Expected error:
```
build constraints exclude all Go files in modernc.org/libc
```

## Alternative: Simplified Demo

To create a WebAssembly demo of Flight3's data processing capabilities (without PocketBase):

1. Create a separate `cmd/flight-wasm/` entry point
2. Remove PocketBase dependency
3. Use only the data processing parts (mksqlite, sqliter)
4. Implement minimal in-memory storage

This would be a subset of Flight3's functionality but would work in WASM.

## Resources

- [Go WASM Documentation](https://github.com/golang/go/wiki/WebAssembly)
- [SQLite WASM](https://sqlite.org/wasm/doc/tip/about.md)
- [PocketBase Discussion on WASM](https://github.com/pocketbase/pocketbase/discussions)

## Status

🔴 **Not buildable** - Infrastructure ready, awaiting dependency solution

For production use, deploy to:
- ✅ Google App Engine (`gae_service/`)
- ✅ Linux/Ubuntu servers (`linux_service/`, `ubuntu_app/`)
- ✅ Windows machines (`windows_app/`)
- ✅ macOS (`macsilicon_app/`, `macx86_app/`)
