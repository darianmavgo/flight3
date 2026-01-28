# Mage Quick Start for Flight3

## Installation (One-Time Setup)

```bash
# Install mage
go install github.com/magefile/mage@latest

# Add mage dependencies to project
go get github.com/magefile/mage/mg github.com/magefile/mage/sh
```

✅ **Done!** Mage is now installed and ready to use.

## Common Commands

### Development
```bash
# Build the project
mage build

# Run with debug mode
mage dev

# Run tests
mage test

# Format code
mage fmt

# Do everything (fmt, tidy, test, build)
mage all
```

### Deployment
```bash
# Build for multiple platforms
mage deploy

# Install to $GOPATH/bin
mage install
```

### Utilities
```bash
# Clean build artifacts
mage clean

# Tidy dependencies
mage tidy

# List all available targets
mage -l
```

## Why This is Better Than npm/npx

### npm/npx Problems:
❌ Requires package.json
❌ Requires node_modules
❌ JavaScript syntax (not Go)
❌ Slow dependency resolution
❌ Platform-specific issues

### Mage Benefits:
✅ **Pure Go** - Use the language you already know
✅ **Type-safe** - Catch errors at compile time
✅ **Fast** - No dependency resolution, just Go compilation
✅ **Cross-platform** - Works everywhere Go works
✅ **IDE support** - Full autocomplete in your editor
✅ **No config files** - Just Go code in `magefile.go`

## Example: Adding a New Task

Just add a function to `magefile.go`:

```go
// MyTask does something custom
func MyTask() error {
    fmt.Println("🎯 Running my custom task...")
    return sh.Run("echo", "Hello from Mage!")
}
```

Then run it:
```bash
mage mytask
```

## Comparison

### Old Way (Makefile)
```makefile
build:
	go build -o flight ./cmd/flight
```
- ❌ Shell syntax
- ❌ Platform-specific
- ❌ No type safety

### Old Way (npm)
```json
{
  "scripts": {
    "build": "go build -o flight ./cmd/flight"
  }
}
```
- ❌ Requires Node.js
- ❌ JSON config
- ❌ Not Go-native

### New Way (Mage)
```go
func Build() error {
    fmt.Println("🔨 Building flight...")
    return sh.Run("go", "build", "-o", "flight", "./cmd/flight")
}
```
- ✅ Pure Go
- ✅ Type-safe
- ✅ Cross-platform
- ✅ IDE autocomplete

## Advanced Features

### Task Dependencies
```go
// Run depends on Build
func Run() error {
    mg.Deps(Build)  // Build will run first
    return sh.Run("./flight", "serve")
}
```

### Environment Variables
```go
func Dev() error {
    env := map[string]string{
        "DEBUG": "true",
    }
    return sh.RunWith(env, "./flight", "serve")
}
```

### Parallel Execution
```go
func All() error {
    mg.Deps(mg.F(Fmt), mg.F(Test))  // Run in parallel
    return Build()
}
```

## Current Available Tasks

Run `mage -l` to see all tasks:

```
Targets:
  all        runs fmt, tidy, test, and build
  build*     compiles the flight binary
  clean      removes build artifacts
  deploy     builds for multiple platforms
  dev        runs the server with debug mode enabled
  fmt        formats all Go code
  install    builds and installs flight to $GOPATH/bin
  launch     builds and opens Chrome (macOS only)
  lint       runs golangci-lint
  run        builds and runs the flight server
  test       runs all tests
  tidy       runs go mod tidy

* default target
```

## Tips

1. **Default target**: Just run `mage` (no args) to run the default target (Build)
2. **Verbose mode**: Use `mage -v build` to see detailed output
3. **Help**: Run `mage -h` for all options
4. **Autocomplete**: Mage supports shell completion (see `mage -h`)

## You'll Love It Because...

1. **No more Makefile syntax** - It's just Go!
2. **No more npm** - No JavaScript tooling needed
3. **Type-safe** - Your IDE helps you write tasks
4. **Fast** - Compiles to native binary
5. **Powerful** - Can import any Go package

Try it now:
```bash
mage build
```

🎉 Welcome to the future of Go build automation!
