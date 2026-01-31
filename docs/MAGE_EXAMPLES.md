# Extending Magefile - Examples

Here are some useful tasks you can add to your `magefile.go`:

## Database Tasks

```go
// ResetDB removes and recreates the database
func ResetDB() error {
    fmt.Println("🗑️  Resetting database...")
    os.RemoveAll("pb_data")
    return nil
}

// Backup creates a backup of the database
func Backup() error {
    fmt.Println("💾 Creating database backup...")
    timestamp := time.Now().Format("2006-01-02-15-04-05")
    backupDir := fmt.Sprintf("backups/pb_data-%s", timestamp)
    return sh.Run("cp", "-r", "pb_data", backupDir)
}
```

## Docker Tasks

```go
// DockerBuild builds a Docker image
func DockerBuild() error {
    fmt.Println("🐳 Building Docker image...")
    return sh.Run("docker", "build", "-t", "flight3:latest", ".")
}

// DockerRun runs the Docker container
func DockerRun() error {
    mg.Deps(DockerBuild)
    fmt.Println("🐳 Running Docker container...")
    return sh.Run("docker", "run", "-p", "8080:8080", "flight3:latest")
}
```

## Release Tasks

```go
// Release creates a new release with version tag
func Release() error {
    version := os.Getenv("VERSION")
    if version == "" {
        return fmt.Errorf("VERSION environment variable not set")
    }
    
    fmt.Printf("📦 Creating release %s...\n", version)
    
    // Build for all platforms
    if err := Deploy(); err != nil {
        return err
    }
    
    // Create git tag
    if err := sh.Run("git", "tag", version); err != nil {
        return err
    }
    
    fmt.Printf("✅ Release %s created!\n", version)
    return nil
}
```

## Benchmark Tasks

```go
// Bench runs benchmarks
func Bench() error {
    fmt.Println("⚡ Running benchmarks...")
    return sh.Run("go", "test", "-bench=.", "-benchmem", "./...")
}

// BenchCompare compares benchmarks with previous run
func BenchCompare() error {
    fmt.Println("📊 Running benchmark comparison...")
    
    // Run benchmarks and save to file
    if err := sh.Run("go", "test", "-bench=.", "-benchmem", "./...", ">", "bench-new.txt"); err != nil {
        return err
    }
    
    // Compare with previous
    return sh.Run("benchstat", "bench-old.txt", "bench-new.txt")
}
```

## Code Quality Tasks

```go
// Vet runs go vet
func Vet() error {
    fmt.Println("🔍 Running go vet...")
    return sh.Run("go", "vet", "./...")
}

// Coverage generates test coverage report
func Coverage() error {
    fmt.Println("📈 Generating coverage report...")
    
    if err := sh.Run("go", "test", "-coverprofile=coverage.out", "./..."); err != nil {
        return err
    }
    
    return sh.Run("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
}

// SecurityScan runs gosec security scanner
func SecurityScan() error {
    fmt.Println("🔒 Running security scan...")
    return sh.Run("gosec", "./...")
}
```

## Development Helpers

```go
// Watch rebuilds on file changes (requires entr or similar)
func Watch() error {
    fmt.Println("👀 Watching for changes...")
    return sh.Run("find", ".", "-name", "*.go", "|", "entr", "-r", "mage", "run")
}

// Deps downloads and verifies dependencies
func Deps() error {
    fmt.Println("📦 Downloading dependencies...")
    
    if err := sh.Run("go", "mod", "download"); err != nil {
        return err
    }
    
    return sh.Run("go", "mod", "verify")
}

// Update updates all dependencies
func Update() error {
    fmt.Println("⬆️  Updating dependencies...")
    return sh.Run("go", "get", "-u", "./...")
}
```

## CI/CD Tasks

```go
// CI runs all CI checks
func CI() error {
    fmt.Println("🤖 Running CI checks...")
    
    // Run tasks in sequence
    tasks := []interface{}{
        Fmt,
        Vet,
        Lint,
        Test,
        Build,
    }
    
    for _, task := range tasks {
        if err := mg.Deps(task); err != nil {
            return err
        }
    }
    
    fmt.Println("✅ All CI checks passed!")
    return nil
}
```

## Platform-Specific Tasks

```go
// MacOS specific task
func MacSetup() error {
    if runtime.GOOS != "darwin" {
        return fmt.Errorf("This task is only for macOS")
    }
    
    fmt.Println("🍎 Setting up macOS environment...")
    return sh.Run("brew", "install", "golangci-lint")
}

// Linux specific task
func LinuxSetup() error {
    if runtime.GOOS != "linux" {
        return fmt.Errorf("This task is only for Linux")
    }
    
    fmt.Println("🐧 Setting up Linux environment...")
    return sh.Run("apt-get", "install", "-y", "golangci-lint")
}
```

## Utility Functions

```go
// Helper function to check if command exists
func commandExists(cmd string) bool {
    _, err := exec.LookPath(cmd)
    return err == nil
}

// Helper function to run command with output
func runWithOutput(cmd string, args ...string) error {
    c := exec.Command(cmd, args...)
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    return c.Run()
}

// Helper function to get git commit hash
func getGitCommit() (string, error) {
    output, err := sh.Output("git", "rev-parse", "--short", "HEAD")
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(output), nil
}

// Build with version info
func BuildWithVersion() error {
    commit, err := getGitCommit()
    if err != nil {
        return err
    }
    
    version := os.Getenv("VERSION")
    if version == "" {
        version = "dev"
    }
    
    ldflags := fmt.Sprintf("-X main.Version=%s -X main.Commit=%s", version, commit)
    
    return sh.Run("go", "build", "-ldflags", ldflags, "-o", "flight", "./cmd/flight")
}
```

## Complex Task Example

```go
// FullDeploy does a complete deployment
func FullDeploy() error {
    fmt.Println("🚀 Starting full deployment...")
    
    // 1. Clean
    if err := Clean(); err != nil {
        return err
    }
    
    // 2. Run tests
    if err := Test(); err != nil {
        return err
    }
    
    // 3. Build for all platforms
    if err := Deploy(); err != nil {
        return err
    }
    
    // 4. Create checksums
    files, err := filepath.Glob("flight-*")
    if err != nil {
        return err
    }
    
    for _, file := range files {
        fmt.Printf("  Creating checksum for %s...\n", file)
        if err := sh.Run("shasum", "-a", "256", file, ">", file+".sha256"); err != nil {
            return err
        }
    }
    
    // 5. Create archive
    fmt.Println("  Creating release archive...")
    if err := sh.Run("tar", "czf", "flight-release.tar.gz", "flight-*"); err != nil {
        return err
    }
    
    fmt.Println("✅ Full deployment complete!")
    return nil
}
```

## Using These Examples

1. Copy the functions you want into your `magefile.go`
2. Add any necessary imports at the top
3. Run `mage -l` to see your new tasks
4. Execute with `mage taskname`

## Pro Tips

1. **Namespace tasks** using struct methods:
```go
type Docker mg.Namespace

func (Docker) Build() error { ... }
func (Docker) Run() error { ... }

// Use: mage docker:build
```

2. **Add descriptions** with comments:
```go
// Build compiles the application
// This is shown in mage -l
func Build() error { ... }
```

3. **Use context** for cancellation:
```go
func LongTask(ctx context.Context) error {
    // Check ctx.Done() for cancellation
}
```

4. **Return early** for better error messages:
```go
func Task() error {
    if !commandExists("docker") {
        return fmt.Errorf("docker not found, please install it")
    }
    // ... rest of task
}
```
