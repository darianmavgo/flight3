#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 for WebAssembly"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Get Go's wasm_exec.js
echo "Locating Go installation..."
GOROOT=$(go env GOROOT)
echo "GOROOT: $GOROOT"

# Try both old and new locations
declare -a WASM_PATHS=(
    "$GOROOT/lib/wasm/wasm_exec.js"           # Go 1.21+
    "$GOROOT/misc/wasm/wasm_exec.js"          # Go < 1.21
    "/opt/homebrew/opt/go/libexec/lib/wasm/wasm_exec.js"
    "/opt/homebrew/opt/go/libexec/misc/wasm/wasm_exec.js"
    "/usr/local/opt/go/libexec/lib/wasm/wasm_exec.js"
    "/usr/local/opt/go/libexec/misc/wasm/wasm_exec.js"
)

WASM_EXEC_JS=""
for path in "${WASM_PATHS[@]}"; do
    if [ -f "$path" ]; then
        WASM_EXEC_JS="$path"
        echo "Found wasm_exec.js at: $path"
        break
    fi
done

if [ -z "$WASM_EXEC_JS" ]; then
    echo "Error: Could not locate wasm_exec.js"
    echo "Searched in:"
    printf '%s\n' "${WASM_PATHS[@]}"
    echo "Please ensure Go is properly installed"
    exit 1
fi

echo "Copying wasm_exec.js from Go installation..."
cp "$WASM_EXEC_JS" "$BUILD_DIR/"

# Build the WebAssembly binary
echo "Compiling Go to WebAssembly..."
cd "$PROJECT_ROOT"

GOOS=js GOARCH=wasm go build \
    -o "$BUILD_DIR/flight3.wasm" \
    -ldflags="-s -w" \
    ./cmd/flight

echo "✓ WASM binary compiled: $(du -h "$BUILD_DIR/flight3.wasm" | cut -f1)"

# Create a simple HTTP server script for testing
cat > "$BUILD_DIR/serve.sh" << 'EOF'
#!/bin/bash
PORT=${1:-8000}

echo "Starting HTTP server for WASM testing..."
echo "Open: http://localhost:$PORT"
echo "Press Ctrl+C to stop"
echo ""

# Try different server options
if command -v python3 &> /dev/null; then
    python3 -m http.server $PORT
elif command -v python &> /dev/null; then
    python -m SimpleHTTPServer $PORT
elif command -v php &> /dev/null; then
    php -S localhost:$PORT
else
    echo "Error: No HTTP server available"
    echo "Install Python or PHP to serve WASM files"
    exit 1
fi
EOF

chmod +x "$BUILD_DIR/serve.sh"

# Create htaccess for proper WASM MIME types (if using Apache)
cat > "$BUILD_DIR/.htaccess" << 'EOF'
# Set proper MIME type for WebAssembly
AddType application/wasm .wasm

# Enable CORS for development
<IfModule mod_headers.c>
    Header set Access-Control-Allow-Origin "*"
    Header set Access-Control-Allow-Methods "GET, POST, OPTIONS"
    Header set Access-Control-Allow-Headers "Content-Type"
</IfModule>

# Compression
<IfModule mod_deflate.c>
    AddOutputFilterByType DEFLATE application/wasm
    AddOutputFilterByType DEFLATE application/javascript
    AddOutputFilterByType DEFLATE text/html
    AddOutputFilterByType DEFLATE text/css
</IfModule>
EOF

# Create README
cat > "$BUILD_DIR/README.md" << 'EOF'
# Flight3 - WebAssembly Build

This is an experimental WebAssembly build of Flight3 that runs entirely in the browser.

## Quick Start

### Option 1: Local Server
```bash
# Start local server
./serve.sh

# Open in browser
open http://localhost:8000
```

### Option 2: Python Server
```bash
python3 -m http.server 8000
```

### Option 3: Go Server
```bash
go run server.go
```

Then open http://localhost:8000 in your browser.

## Files

- `flight3.wasm` - Compiled WebAssembly binary
- `wasm_exec.js` - Go's WebAssembly runtime 
- `index.html` - Web interface
- `flight3.js` - Application controller
- `serve.sh` - Simple HTTP server script

## Browser Compatibility

Tested on:
- ✅ Chrome 90+
- ✅ Firefox 89+
- ✅ Safari 14+
- ✅ Edge 90+

## Limitations

WebAssembly builds have some limitations compared to native:

1. **No Native Networking**: WASM runs in browser sandbox
2. **Memory Limits**: Limited to browser memory allocation
3. **File Access**: No direct filesystem access (uses IndexedDB/localStorage)
4. **Performance**: ~50-80% of native performance
5. **Binary Size**: Larger than native due to Go runtime

## Use Cases

Good for:
- ✅ Client-side data processing
- ✅ Demonstration/preview
- ✅ Offline data analysis
- ✅ Browser extension backend

Not ideal for:
- ❌ Production server deployment
- ❌ High-performance requirements
- ❌ Direct file system access
- ❌ Native networking

## Development

### Rebuild WASM
```bash
./build.sh
```

### Debug
Open browser DevTools Console to see Go output and errors.

### File Size Optimization
Current build uses `-ldflags="-s -w"` to strip debug info.

For even smaller builds:
```bash
# Use TinyGo (experimental)
tinygo build -o flight3.wasm -target wasm ./cmd/flight
```

## Deployment

### Static Hosting
Upload these files to any static host:
- Netlify
- Vercel  
- GitHub Pages
- AWS S3 + CloudFront

### CDN
Serve `.wasm` files with proper MIME type:
```
Content-Type: application/wasm
```

### Compression
Enable gzip/brotli compression for `.wasm` files to reduce download size.

## Performance Tips

1. **Use Web Workers**: Offload heavy processing
2. **Stream Data**: Don't load everything at once
3. **Cache WASM**: Browser will cache the .wasm file
4. **Optimize Bundle**: Remove unused code

## Troubleshooting

**WASM fails to load:**
- Check browser console for errors
- Verify MIME type is `application/wasm`
- Check CORS if loading from CDN
- Ensure HTTPS (required for some features)

**Out of memory:**
- Reduce dataset size
- Process data in chunks
- Clear browser cache

**Slow performance:**
- This is expected vs native
- Use native build for production
- Consider WASM as client-side tool only

## Security

WebAssembly runs in browser sandbox with limited privileges:
- No access to local filesystem
- No arbitrary network requests (only fetch API)
- No system calls
- Memory isolated from other tabs

## Future Improvements

- [ ] Web Worker integration
- [ ] Service Worker for offline support  
- [ ] IndexedDB integration for persistence
- [ ] Streaming data support
- [ ] Progressive Web App (PWA) features

## Resources

- [Go WebAssembly Docs](https://github.com/golang/go/wiki/WebAssembly)
- [WebAssembly.org](https://webassembly.org/)
- [MDN WASM Guide](https://developer.mozilla.org/en-US/docs/WebAssembly)
EOF

# Create a minimal Go HTTP server for serving
cat > "$BUILD_DIR/server.go" << 'EOF'
//go:build ignore

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := "8000"
	
	// Serve current directory
	fs := http.FileServer(http.Dir("."))
	
	// Add WASM MIME type
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path[len(r.URL.Path)-5:] == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		fs.ServeHTTP(w, r)
	})
	
	fmt.Printf("Serving WASM at http://localhost:%s\n", port)
	fmt.Println("Press Ctrl+C to stop")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
EOF

echo ""
echo "========================================="
echo "✓ WebAssembly build complete!"
echo ""
echo "Build directory: $BUILD_DIR"
echo "WASM binary: flight3.wasm ($(du -h "$BUILD_DIR/flight3.wasm" | cut -f1))"
echo ""
echo "To test locally:"
echo "  cd $BUILD_DIR"
echo "  ./serve.sh"
echo "  open http://localhost:8000"
echo ""
echo "Files created:"
ls -lh "$BUILD_DIR" | grep -E '\.(wasm|js|html|sh)$'
echo ""
echo "Read README.md for deployment and usage instructions."
echo "========================================="
