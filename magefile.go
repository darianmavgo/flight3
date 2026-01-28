//go:build mage
// +build mage

package main

import (
	"fmt"
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

// Install builds and installs flight to macOS conventional locations
// Binary: /usr/local/bin/flight
// Data: ~/Library/Application Support/Flight3/
func Install() error {
	mg.Deps(Build)

	fmt.Println("📦 Installing Flight3 to macOS conventional locations...")

	// Determine installation paths based on OS
	var binPath, dataPath string

	if runtime.GOOS == "darwin" {
		// macOS conventional paths
		binPath = "/usr/local/bin/flight"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataPath = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	} else {
		// Linux/Unix conventional paths
		binPath = "/usr/local/bin/flight"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dataPath = filepath.Join(homeDir, ".local", "share", "flight3")
	}

	// Install binary
	fmt.Printf("  Installing binary to: %s\n", binPath)
	if err := sh.Run("sudo", "cp", "flight", binPath); err != nil {
		return fmt.Errorf("failed to install binary (may need sudo): %w", err)
	}

	// Make binary executable
	if err := sh.Run("sudo", "chmod", "+x", binPath); err != nil {
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
	if _, err := os.Stat("pb_data"); err == nil {
		fmt.Println("  Copying existing pb_data...")
		if err := sh.Run("cp", "-r", "pb_data/.", filepath.Join(dataPath, "pb_data")); err != nil {
			fmt.Printf("  Warning: failed to copy pb_data: %v\n", err)
		}
	}

	if _, err := os.Stat("pb_public"); err == nil {
		fmt.Println("  Copying existing pb_public...")
		if err := sh.Run("cp", "-r", "pb_public/.", filepath.Join(dataPath, "pb_public")); err != nil {
			fmt.Printf("  Warning: failed to copy pb_public: %v\n", err)
		}
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

	var binPath, dataPath string

	if runtime.GOOS == "darwin" {
		binPath = "/usr/local/bin/flight"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataPath = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	} else {
		binPath = "/usr/local/bin/flight"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataPath = filepath.Join(homeDir, ".local", "share", "flight3")
	}

	// Remove binary
	fmt.Printf("  Removing binary: %s\n", binPath)
	if err := sh.Run("sudo", "rm", "-f", binPath); err != nil {
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

	fmt.Println("\n✅ Uninstall complete!")
	return nil
}

// Run builds and runs the flight server
func Run() error {
	mg.Deps(Build)
	fmt.Println("🚀 Starting flight server...")
	return sh.Run("./flight", "serve")
}

// Dev runs the server with debug mode enabled
func Dev() error {
	mg.Deps(Build)
	fmt.Println("🔧 Starting flight server in DEBUG mode...")
	env := map[string]string{
		"DEBUG": "true",
	}
	return sh.RunWith(env, "./flight", "serve")
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

	// Start server in background with DEBUG enabled
	cmd := exec.Command("./flight", "serve")
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
