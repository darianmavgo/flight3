//go:build mage
// +build mage

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
var Default = Build

// Build compiles the flight binary
func Build() error {
	fmt.Println("🔨 Building flight...")
	return sh.Run("go", "build", "-o", "flight", "./cmd/flight")
}

// Test runs all tests
func Test() error {
	fmt.Println("🧪 Running tests...")
	return sh.Run("go", "test", "-v", "./...")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("🧹 Cleaning...")
	os.Remove("flight")
	os.RemoveAll("pb_data")
	os.RemoveAll("pb_public")
	return nil
}

// Kill terminates any running flight processes
func Kill() error {
	fmt.Println("🔪 Killing running flight processes...")
	if err := sh.Run("killall", "flight"); err != nil {
		fmt.Println("  No running flight processes found (or failed to kill).")
	} else {
		fmt.Println("  ✅ Flight processes killed.")
	}
	return nil
}

// Install builds and installs flight to macOS conventional locations
// Binary: /usr/local/bin/flight
// Data: ~/Library/Application Support/Flight3/
func Install() error {
	mg.Deps(Build)

	fmt.Println("📦 Installing Flight3 to macOS conventional locations...")

	// Determine installation paths based on OS
	var binPath, dataPath string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try to find GOPATH/bin or fallback to ~/go/bin
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = filepath.Join(homeDir, "go")
	}
	binDir := filepath.Join(goPath, "bin")

	// Create bin dir if it doesn't exist
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	binPath = filepath.Join(binDir, "flight")

	if runtime.GOOS == "darwin" {
		// macOS conventional data path (user-local)
		dataPath = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	} else {
		// Linux/Unix conventional data path
		dataPath = filepath.Join(homeDir, ".local", "share", "flight3")
	}

	// Install binary (no sudo needed for user folders)
	fmt.Printf("  Installing binary to: %s\n", binPath)
	if err := sh.Run("cp", "flight", binPath); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	// Make binary executable
	if err := sh.Run("chmod", "+x", binPath); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Create data directory structure
	fmt.Printf("  Creating data directory: %s\n", dataPath)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"pb_data", "pb_public", "cache", "temp"}
	for _, subdir := range subdirs {
		path := filepath.Join(dataPath, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", subdir, err)
		}
	}

	// Copy any existing data if present
	// Copy pb_data only if it doesn't exist in destination (preserve user data)
	destPbData := filepath.Join(dataPath, "pb_data")
	destDataDB := filepath.Join(destPbData, "data.db")
	if _, err := os.Stat(destDataDB); err == nil {
		fmt.Printf("  ⚠️  Destination 'pb_data/data.db' exists in %s, skipping data overwrite.\n", destPbData)
	} else {
		// Destination doesn't exist or is empty, copy from source if present
		if _, err := os.Stat("pb_data"); err == nil {
			fmt.Println("  Copying initial pb_data...")
			if err := sh.Run("cp", "-r", "pb_data/.", destPbData); err != nil {
				fmt.Printf("  Warning: failed to copy pb_data: %v\n", err)
			}
		} else {
			fmt.Println("  ⚠️  Source 'pb_data' not found, skipping data copy.")
		}
	}

	if _, err := os.Stat("pb_public"); err == nil {
		fmt.Println("  Copying existing pb_public...")
		if err := sh.Run("cp", "-r", "pb_public/.", filepath.Join(dataPath, "pb_public")); err != nil {
			fmt.Printf("  Warning: failed to copy pb_public: %v\n", err)
		}
	} else {
		fmt.Println("  ⚠️  Source 'pb_public' not found, skipping public assets copy.")
	}

	// Create a launch script that uses the data directory
	fmt.Println("  Creating launch configuration...")
	configPath := filepath.Join(dataPath, "flight.env")
	configContent := fmt.Sprintf("# Flight3 Configuration\nDATA_DIR=%s\n", dataPath)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		fmt.Printf("  Warning: failed to create config file: %v\n", err)
	}

	fmt.Println("\n✅ Installation complete!")
	fmt.Printf("\n📍 Locations:\n")
	fmt.Printf("   Binary: %s\n", binPath)
	fmt.Printf("   Data:   %s\n", dataPath)
	fmt.Printf("\n🚀 To run: flight serve\n")
	fmt.Printf("   (Flight will use data directory: %s)\n", dataPath)

	return nil
}

// Uninstall removes flight from system locations
func Uninstall() error {
	fmt.Println("🗑️  Uninstalling Flight3...")

	// Determine paths matches Install
	var binPath, dataPath string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = filepath.Join(homeDir, "go")
	}
	binPath = filepath.Join(goPath, "bin", "flight")

	if runtime.GOOS == "darwin" {
		dataPath = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	} else {
		dataPath = filepath.Join(homeDir, ".local", "share", "flight3")
	}

	// Remove binary
	fmt.Printf("  Removing binary: %s\n", binPath)
	if err := os.Remove(binPath); err != nil {
		fmt.Printf("  Warning: failed to remove binary: %v\n", err)
	}

	// Ask before removing data
	fmt.Printf("\n⚠️  Data directory: %s\n", dataPath)
	fmt.Print("   Remove data directory? (y/N): ")

	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		fmt.Println("  Removing data directory...")
		if err := os.RemoveAll(dataPath); err != nil {
			fmt.Printf("  Warning: failed to remove data directory: %v\n", err)
		} else {
			fmt.Println("  Data directory removed")
		}
	} else {
		fmt.Println("  Data directory preserved")
	}

	return nil
}

// Service installs Flight3 as a macOS launchd service (startup item)
// Logs will be located at ~/Library/Logs/Flight3/flight.log
func Service() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Service management is only supported on macOS")
	}

	mg.Deps(Install)
	fmt.Println("🚀 Setting up Flight3 as a background service...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 1. Define Paths
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = filepath.Join(homeDir, "go")
	}
	binPath := filepath.Join(goPath, "bin", "flight")

	logDir := filepath.Join(homeDir, "Library", "Logs", "Flight3")
	logPath := filepath.Join(logDir, "flight.log")
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.darianmavgo.flight3.plist")

	// 2. Create Log Directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	// 3. Create Plist
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.darianmavgo.flight3</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>serve</string>
        <string>--http=[::1]:8090</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>`, binPath, logPath, logPath)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	// 4. Load Service
	// Unload first just in case
	sh.Run("launchctl", "unload", plistPath)
	if err := sh.Run("launchctl", "load", plistPath); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	fmt.Println("✅ Service installed and started!")
	fmt.Printf("   Logs: %s\n", logPath)
	fmt.Printf("   URL:  http://[::1]:8090\n")
	fmt.Println("   Use 'mage logs' to view output.")
	return nil
}

// Unservice removes the Flight3 launchd service
func Unservice() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Service management is only supported on macOS")
	}

	fmt.Println("🛑 Stopping and removing Flight3 service...")
	homeDir, _ := os.UserHomeDir()
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.darianmavgo.flight3.plist")

	sh.Run("launchctl", "unload", plistPath)
	os.Remove(plistPath)

	fmt.Println("✅ Service removed.")
	return nil
}

// Logs tails the Flight3 service logs
func Logs() error {
	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, "Library", "Logs", "Flight3", "flight.log")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found at %s. Is the service running?", logPath)
	}

	fmt.Printf("📋 Tailing logs at %s...\n", logPath)
	return sh.RunV("tail", "-f", logPath)
}

// Run builds and runs the flight server
func Run() error {
	mg.Deps(Build)
	setTerminalTitle("✈️ Flight Server")
	fmt.Println("🚀 Starting flight server...")
	notify("Flight Server Started", "Ready for boarding at http://localhost:8090")
	return sh.Run("./flight", "serve")
}

// Dev runs the server with debug mode enabled
func Dev() error {
	mg.Deps(Build)
	setTerminalTitle("✈️ Flight Server (Debug)")
	fmt.Println("🔧 Starting flight server in DEBUG mode...")
	notify("Flight Debug Mode", "Debug server running...")
	env := map[string]string{
		"DEBUG": "true",
	}
	return sh.RunWith(env, "./flight", "serve")
}

// Helper to set the terminal window title
func setTerminalTitle(title string) {
	fmt.Printf("\033]0;%s\007", title)
}

// Helper to send macOS notification
func notify(title, message string) {
	if runtime.GOOS == "darwin" {
		exec.Command("osascript", "-e", fmt.Sprintf("display notification \"%s\" with title \"%s\"", message, title)).Run()
	}
}

// Deploy builds for multiple platforms
func Deploy() error {
	fmt.Println("🌍 Building for multiple platforms...")

	platforms := []struct {
		os   string
		arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
	}

	for _, p := range platforms {
		output := fmt.Sprintf("flight-%s-%s", p.os, p.arch)
		if p.os == "windows" {
			output += ".exe"
		}

		fmt.Printf("  Building %s...\n", output)
		env := map[string]string{
			"GOOS":        p.os,
			"GOARCH":      p.arch,
			"CGO_ENABLED": "0",
		}

		if err := sh.RunWith(env, "go", "build", "-o", output, "./cmd/flight"); err != nil {
			return err
		}
	}

	fmt.Println("✅ All platforms built successfully!")
	return nil
}

// Launch builds and opens Chrome (macOS only)
func Launch() error {
	mg.Deps(Build)

	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Launch is only supported on macOS")
	}

	fmt.Println("🚀 Launching flight and opening Chrome...")

	var listener net.Listener
	var err error
	var port int

	// 1. Try sticky ports (80, 8090-8099) on IPv6
	ports := []int{80}
	for i := 8090; i <= 8099; i++ {
		ports = append(ports, i)
	}

	for _, p := range ports {
		addr := fmt.Sprintf("[::1]:%d", p)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			port = p
			break
		}
	}

	// 2. Fallback to random free port if sticky ports failed
	if listener == nil {
		fmt.Println("⚠️  Preferred ports 8090-8099 are busy, falling back to random port.")
		listener, err = net.Listen("tcp", "[::1]:0")
		if err != nil {
			// Try IPv4 if IPv6 fails
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return fmt.Errorf("failed to find free port: %w", err)
			}
		}
		port = listener.Addr().(*net.TCPAddr).Port
	}

	listener.Close() // Close it so the server can use it

	// Determine address string for Flight
	addr := fmt.Sprintf("[::1]:%d", port) // Default to IPv4 localhost for stability

	// Create Launch URL
	url := fmt.Sprintf("http://[::1]:%d", port) // localhost implies 127.0.0.1 usuallly

	fmt.Printf("\n🔗 App URL: %s\n\n", url)

	// Start server in background with DEBUG enabled
	cmd := exec.Command("./flight", "serve", "--http", addr)
	cmd.Env = append(os.Environ(), "DEBUG=true")
	if err := cmd.Start(); err != nil {
		return err
	}

	fmt.Printf("✅ Flight started (PID: %d)\n", cmd.Process.Pid)
	return nil
}

// Fmt formats all Go code
func Fmt() error {
	fmt.Println("💅 Formatting code...")
	return sh.Run("go", "fmt", "./...")
}

// Lint runs golangci-lint
func Lint() error {
	fmt.Println("🔍 Running linter...")
	return sh.Run("golangci-lint", "run")
}

// Tidy runs go mod tidy
func Tidy() error {
	fmt.Println("📚 Tidying dependencies...")
	return sh.Run("go", "mod", "tidy")
}

// All runs fmt, tidy, test, and build
func All() error {
	mg.Deps(Fmt, Tidy, Test, Build)
	fmt.Println("✅ All tasks completed!")
	return nil
}
