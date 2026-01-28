# Build Tool Options for Go Projects

## 1. Mage (Recommended) ⭐

**What**: Make-like build tool using pure Go code
**Why**: Type-safe, cross-platform, IDE-friendly

### Installation
```bash
go install github.com/magefile/mage@latest
```

### Usage
```bash
# List available targets
mage -l

# Run specific target
mage build
mage test
mage dev
mage deploy

# Run default target
mage
```

### Available Targets in magefile.go
- `mage build` - Build the flight binary
- `mage test` - Run all tests
- `mage clean` - Remove build artifacts
- `mage install` - Install to $GOPATH/bin
- `mage run` - Build and run server
- `mage dev` - Run with DEBUG=true
- `mage deploy` - Build for multiple platforms
- `mage launch` - Build and open Chrome (macOS)
- `mage fmt` - Format all code
- `mage lint` - Run linter
- `mage tidy` - Run go mod tidy
- `mage all` - Run fmt, tidy, test, build

### Pros
✅ Pure Go (no shell scripts)
✅ Type-safe with autocomplete
✅ Cross-platform
✅ Dependency management between tasks
✅ Can be vendored (no external dependency)

### Cons
❌ Requires installing mage binary
❌ One more tool to learn (but it's just Go)

---

## 2. Task (Taskfile)

**What**: Modern Make alternative with YAML config
**Why**: Simple, fast, cross-platform

### Installation
```bash
brew install go-task/tap/go-task
# or
go install github.com/go-task/task/v3/cmd/task@latest
```

### Example Taskfile.yml
```yaml
version: '3'

tasks:
  build:
    desc: Build the flight binary
    cmds:
      - go build -o flight ./cmd/flight

  test:
    desc: Run all tests
    cmds:
      - go test -v ./...

  dev:
    desc: Run with debug mode
    env:
      DEBUG: "true"
    cmds:
      - task: build
      - ./flight serve

  deploy:
    desc: Build for multiple platforms
    cmds:
      - GOOS=darwin GOARCH=amd64 go build -o flight-darwin-amd64 ./cmd/flight
      - GOOS=darwin GOARCH=arm64 go build -o flight-darwin-arm64 ./cmd/flight
      - GOOS=linux GOARCH=amd64 go build -o flight-linux-amd64 ./cmd/flight
```

### Usage
```bash
task build
task test
task dev
```

### Pros
✅ Simple YAML syntax
✅ Fast (written in Go)
✅ Good documentation
✅ Cross-platform

### Cons
❌ YAML (not type-safe)
❌ Another config file format

---

## 3. Just

**What**: Command runner (like Make but better)
**Why**: Simple, no dependencies

### Installation
```bash
brew install just
```

### Example justfile
```just
# Build the flight binary
build:
    go build -o flight ./cmd/flight

# Run all tests
test:
    go test -v ./...

# Run with debug mode
dev: build
    DEBUG=true ./flight serve

# Build for multiple platforms
deploy:
    GOOS=darwin GOARCH=amd64 go build -o flight-darwin-amd64 ./cmd/flight
    GOOS=darwin GOARCH=arm64 go build -o flight-darwin-arm64 ./cmd/flight
    GOOS=linux GOARCH=amd64 go build -o flight-linux-amd64 ./cmd/flight
```

### Usage
```bash
just build
just test
just dev
```

### Pros
✅ Very simple syntax
✅ Fast
✅ No dependencies

### Cons
❌ Not Go-specific
❌ Less powerful than Mage

---

## 4. Plain Go Scripts (No External Tool)

**What**: Use `go run` with build scripts
**Why**: Zero dependencies, pure Go

### Example: scripts/build.go
```go
//go:build ignore

package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
)

func main() {
    action := flag.String("action", "build", "Action to perform")
    flag.Parse()

    switch *action {
    case "build":
        build()
    case "test":
        test()
    case "dev":
        dev()
    default:
        fmt.Printf("Unknown action: %s\n", *action)
        os.Exit(1)
    }
}

func build() {
    cmd := exec.Command("go", "build", "-o", "flight", "./cmd/flight")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        os.Exit(1)
    }
}

func test() {
    cmd := exec.Command("go", "test", "-v", "./...")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        os.Exit(1)
    }
}

func dev() {
    build()
    cmd := exec.Command("./flight", "serve")
    cmd.Env = append(os.Environ(), "DEBUG=true")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        os.Exit(1)
    }
}
```

### Usage
```bash
go run scripts/build.go -action=build
go run scripts/build.go -action=test
go run scripts/build.go -action=dev
```

### Pros
✅ Zero external dependencies
✅ Pure Go
✅ Version controlled

### Cons
❌ More verbose
❌ No task listing
❌ Manual dependency management

---

## 5. Traditional Makefile (Still Works)

If you want to stick with Make but make it better:

```makefile
.PHONY: build test dev deploy clean

build:
	go build -o flight ./cmd/flight

test:
	go test -v ./...

dev: build
	DEBUG=true ./flight serve

deploy:
	GOOS=darwin GOARCH=amd64 go build -o flight-darwin-amd64 ./cmd/flight
	GOOS=darwin GOARCH=arm64 go build -o flight-darwin-arm64 ./cmd/flight
	GOOS=linux GOARCH=amd64 go build -o flight-linux-amd64 ./cmd/flight

clean:
	rm -f flight flight-*
```

---

## My Recommendation for Flight3

**Use Mage** because:

1. ✅ **Pure Go** - You already know Go, no new syntax
2. ✅ **Type-safe** - Catch errors at compile time
3. ✅ **Cross-platform** - Works on macOS, Linux, Windows
4. ✅ **IDE support** - Full autocomplete and refactoring
5. ✅ **Powerful** - Can do complex logic easily
6. ✅ **Go ecosystem** - Can import any Go package

### Quick Start
```bash
# Install mage
go install github.com/magefile/mage@latest

# List available targets
mage -l

# Build
mage build

# Run with debug
mage dev

# Deploy to multiple platforms
mage deploy
```

The `magefile.go` I created has all the common tasks you need. You can extend it easily since it's just Go code!

---

## Comparison Table

| Tool | Language | Type-Safe | Cross-Platform | Learning Curve | Power |
|------|----------|-----------|----------------|----------------|-------|
| **Mage** | Go | ✅ | ✅ | Low (it's Go!) | ⭐⭐⭐⭐⭐ |
| Task | YAML | ❌ | ✅ | Very Low | ⭐⭐⭐ |
| Just | Custom | ❌ | ✅ | Very Low | ⭐⭐ |
| Go Scripts | Go | ✅ | ✅ | Low | ⭐⭐⭐⭐ |
| Make | Shell | ❌ | ⚠️ | Medium | ⭐⭐⭐ |

**Winner**: Mage 🏆
