package flight

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/mksqlite/converters"
	"github.com/darianmavgo/mksqlite/converters/common"

	// Register all converters
	_ "github.com/darianmavgo/mksqlite/converters/csv"
	_ "github.com/darianmavgo/mksqlite/converters/excel"
	_ "github.com/darianmavgo/mksqlite/converters/filesystem"
	_ "github.com/darianmavgo/mksqlite/converters/html"
	_ "github.com/darianmavgo/mksqlite/converters/json"
	_ "github.com/darianmavgo/mksqlite/converters/markdown"
	_ "github.com/darianmavgo/mksqlite/converters/txt"
	_ "github.com/darianmavgo/mksqlite/converters/zip"
)

// ConvertToSQLite converts a source file or directory to SQLite database using mksqlite library
func ConvertToSQLite(sourcePath, destPath string) error {
	log.Printf("[CONVERTER] Converting %s -> %s", sourcePath, destPath)

	// Check if source exists
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Determine driver name based on file type
	var driverName string

	if fileInfo.IsDir() {
		// Directory - use filesystem converter
		driverName = "filesystem"
		log.Printf("[CONVERTER] Using filesystem converter for directory")
	} else {
		// File - detect type from extension
		ext := strings.ToLower(filepath.Ext(sourcePath))

		switch ext {
		case ".db", ".sqlite", ".sqlite3":
			// Already SQLite, just copy
			log.Printf("[CONVERTER] Source is already SQLite, copying")
			return copyFile(sourcePath, destPath)

		case ".csv":
			driverName = "csv"

		case ".xlsx", ".xls":
			driverName = "excel"

		case ".html", ".htm":
			driverName = "html"

		case ".json":
			driverName = "json"

		case ".md", ".markdown":
			driverName = "markdown"

		case ".txt":
			driverName = "txt"

		case ".zip":
			driverName = "zip"

		default:
			return fmt.Errorf("unsupported file type: %s", ext)
		}

		log.Printf("[CONVERTER] Using %s converter", driverName)
	}

	// Open the source file or directory
	var provider common.RowProvider

	if fileInfo.IsDir() {
		// For directories, use the filesystem converter directly
		// The filesystem converter needs the directory path in InputPath
		provider, err = converters.Open(driverName, nil, &common.ConversionConfig{
			InputPath: sourcePath,
		})
	} else {
		// For files, open as io.Reader
		file, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer file.Close()

		provider, err = converters.Open(driverName, file, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to open converter: %w", err)
	}

	// Create output database file
	dbFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output database: %w", err)
	}
	defer dbFile.Close()

	// Convert to SQLite
	opts := &converters.ImportOptions{
		Verbose: true,
	}

	if err := converters.ImportToSQLite(provider, dbFile, opts); err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	log.Printf("[CONVERTER] Conversion successful")
	return nil
}

// copyFile copies a file from src to dst (for already-SQLite files)
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
