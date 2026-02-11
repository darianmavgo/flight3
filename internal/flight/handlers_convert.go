package flight

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// HandleConvertToSQLite accepts a file upload (CSV, Excel, JSON, etc.)
// and converts it to SQLite using the mksqlite converters
func HandleConvertToSQLite(e *core.RequestEvent) error {
	log.Printf("[CONVERT] Received conversion request")

	// Get uploaded file
	file, header, err := e.Request.FormFile("file")
	if err != nil {
		log.Printf("[CONVERT] Error: No file provided - %v", err)
		return e.JSON(400, map[string]interface{}{
			"error": "No file provided",
			"code":  "missing_file",
		})
	}
	defer file.Close()

	log.Printf("[CONVERT] Received file: %s (%d bytes)", header.Filename, header.Size)

	// Validate file size (e.g., 100MB limit)
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if header.Size > maxFileSize {
		return e.JSON(400, map[string]interface{}{
			"error": fmt.Sprintf("File too large. Maximum size is %d MB", maxFileSize/(1024*1024)),
			"code":  "file_too_large",
		})
	}

	// Create temp directory for conversion
	tempDir := filepath.Join(os.TempDir(), "flight3-convert")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("[CONVERT] Error creating temp dir: %v", err)
		return e.JSON(500, map[string]interface{}{
			"error": "Failed to create temporary directory",
			"code":  "temp_dir_error",
		})
	}

	// Save uploaded file to temp location
	tempInputPath := filepath.Join(tempDir, header.Filename)
	log.Printf("[CONVERT] Saving to temp: %s", tempInputPath)

	out, err := os.Create(tempInputPath)
	if err != nil {
		log.Printf("[CONVERT] Error creating temp file: %v", err)
		return e.JSON(500, map[string]interface{}{
			"error": "Failed to save uploaded file",
			"code":  "file_save_error",
		})
	}

	written, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		log.Printf("[CONVERT] Error writing file: %v", err)
		os.Remove(tempInputPath)
		return e.JSON(500, map[string]interface{}{
			"error": "Failed to write uploaded file",
			"code":  "file_write_error",
		})
	}

	log.Printf("[CONVERT] Saved %d bytes", written)

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	supportedExts := []string{".csv", ".xlsx", ".xls", ".json", ".html", ".htm", ".md", ".markdown", ".txt", ".zip"}
	isSupported := false
	for _, supExt := range supportedExts {
		if ext == supExt {
			isSupported = true
			break
		}
	}

	// If already SQLite, just return it
	if ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" {
		log.Printf("[CONVERT] File is already SQLite, returning as-is")
		defer os.Remove(tempInputPath)
		http.ServeFile(e.Response, e.Request, tempInputPath)
		return nil
	}

	// If unsupported, return error
	if !isSupported {
		os.Remove(tempInputPath)
		return e.JSON(400, map[string]interface{}{
			"error":             fmt.Sprintf("Unsupported file type: %s", ext),
			"code":              "unsupported_format",
			"supported_formats": supportedExts,
		})
	}

	// Generate output SQLite filename
	baseName := strings.TrimSuffix(header.Filename, ext)
	tempOutputPath := filepath.Join(tempDir, baseName+".db")

	log.Printf("[CONVERT] Converting %s -> %s", tempInputPath, tempOutputPath)

	// Convert using mksqlite
	if err := ConvertToSQLite(tempInputPath, tempOutputPath); err != nil {
		log.Printf("[CONVERT] Conversion failed: %v", err)
		os.Remove(tempInputPath)
		return e.JSON(500, map[string]interface{}{
			"error":  "Conversion failed",
			"code":   "conversion_error",
			"detail": err.Error(),
		})
	}

	// Clean up input file
	os.Remove(tempInputPath)

	// Check if output file exists
	if _, err := os.Stat(tempOutputPath); err != nil {
		log.Printf("[CONVERT] Output file not found: %v", err)
		return e.JSON(500, map[string]interface{}{
			"error": "Conversion failed to produce output file",
			"code":  "output_missing",
		})
	}

	log.Printf("[CONVERT] Conversion successful, serving file: %s", tempOutputPath)

	// Serve the SQLite file
	// Note: ServeFile handles content-type and range requests automatically
	// TODO: Implement cleanup job or defer deletion after serving
	http.ServeFile(e.Response, e.Request, tempOutputPath)
	return nil
}
