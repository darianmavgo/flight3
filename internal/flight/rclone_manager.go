package flight

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"
)

// RcloneManager manages VFS instances and caching
type RcloneManager struct {
	vfsCache map[string]*vfs.VFS
	cacheDir string
	mu       sync.RWMutex
}

var globalRcloneManager *RcloneManager

// InitRclone initializes the global rclone manager
func InitRclone(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	globalRcloneManager = &RcloneManager{
		vfsCache: make(map[string]*vfs.VFS),
		cacheDir: cacheDir,
	}

	log.Printf("[RCLONE] Initialized with cache directory: %s", cacheDir)
	return nil
}

// GetRcloneManager returns the global rclone manager instance
func GetRcloneManager() *RcloneManager {
	return globalRcloneManager
}

// generateVFSHash creates a unique hash for VFS caching based on remote config
func generateVFSHash(remoteConfig map[string]interface{}) string {
	// Serialize config to JSON for consistent hashing
	configJSON, err := json.Marshal(remoteConfig)
	if err != nil {
		log.Printf("[RCLONE] Warning: failed to marshal config for hashing: %v", err)
		return fmt.Sprintf("%v", remoteConfig)
	}

	hash := md5.Sum(configJSON)
	return fmt.Sprintf("%x", hash)
}

// GetVFS gets or creates a VFS instance for the given remote configuration
func (rm *RcloneManager) GetVFS(remoteRecord *core.Record) (*vfs.VFS, error) {
	// Extract configuration from PocketBase record
	remoteType := remoteRecord.GetString("type")
	configData := remoteRecord.Get("config")

	var config map[string]interface{}
	switch v := configData.(type) {
	case map[string]interface{}:
		config = v
	case string:
		if err := json.Unmarshal([]byte(v), &config); err != nil {
			return nil, fmt.Errorf("failed to parse config JSON: %w", err)
		}
	case []byte:
		// Handle types.JSONRaw (which is []byte under the hood)
		if err := json.Unmarshal(v, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config JSON from bytes: %w", err)
		}
	default:
		// Try to marshal and unmarshal as a fallback for types.JSONRaw
		jsonBytes, err := json.Marshal(configData)
		if err != nil {
			return nil, fmt.Errorf("invalid config type: %T", configData)
		}
		if err := json.Unmarshal(jsonBytes, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config JSON from type %T: %w", configData, err)
		}
	}

	// Generate hash for this configuration
	configHash := generateVFSHash(config)

	// Check if VFS already exists in cache
	rm.mu.RLock()
	if existingVFS, ok := rm.vfsCache[configHash]; ok {
		rm.mu.RUnlock()
		log.Printf("[RCLONE] VFS cache hit for hash: %s", configHash)
		return existingVFS, nil
	}
	rm.mu.RUnlock()

	// Create new VFS instance
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double-check after acquiring write lock
	if existingVFS, ok := rm.vfsCache[configHash]; ok {
		return existingVFS, nil
	}

	log.Printf("[RCLONE] Creating new VFS for type: %s, hash: %s", remoteType, configHash)

	// Create rclone filesystem
	f, err := rm.createFilesystem(remoteType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}

	// Get VFS settings (use defaults or from record)
	vfsOpts := rm.getVFSOptions(remoteRecord)

	// Create VFS instance
	newVFS := vfs.New(f, &vfsOpts)

	// Cache the VFS instance
	rm.vfsCache[configHash] = newVFS

	log.Printf("[RCLONE] VFS created and cached for hash: %s", configHash)
	return newVFS, nil
}

// createFilesystem creates an rclone filesystem from type and config
func (rm *RcloneManager) createFilesystem(remoteType string, config map[string]interface{}) (fs.Fs, error) {
	// Convert config map to configmap.Simple
	m := configmap.Simple{}
	for k, v := range config {
		m[k] = fmt.Sprintf("%v", v)
	}

	// Find the filesystem registry info
	fsInfo, err := fs.Find(remoteType)
	if err != nil {
		return nil, fmt.Errorf("unknown remote type '%s': %w", remoteType, err)
	}

	// Create the filesystem
	// The path is typically empty or "/" for root access
	ctx := context.Background()
	f, err := fsInfo.NewFs(ctx, "", "", m)
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}

	return f, nil
}

// getVFSOptions returns VFS options, using defaults or custom settings from record
func (rm *RcloneManager) getVFSOptions(remoteRecord *core.Record) vfscommon.Options {
	opts := vfscommon.Options{
		CacheMode:         vfscommon.CacheModeFull, // Critical for random access
		DirCacheTime:      10 * 60,                 // 10 minutes
		CacheMaxAge:       24 * 60 * 60,            // 24 hours
		CachePollInterval: 60,                      // 1 minute
		ChunkSize:         128 * 1024 * 1024,       // 128 MB
		ReadAhead:         0,                       // Disable read-ahead by default
	}

	// Check for custom VFS settings in the record
	vfsSettingsData := remoteRecord.Get("vfs_settings")
	if vfsSettingsData != nil {
		var customSettings map[string]interface{}
		switch v := vfsSettingsData.(type) {
		case map[string]interface{}:
			customSettings = v
		case string:
			if err := json.Unmarshal([]byte(v), &customSettings); err != nil {
				log.Printf("[RCLONE] Warning: failed to parse vfs_settings: %v", err)
				return opts
			}
		}

		// Apply custom settings if present
		if cacheMode, ok := customSettings["cache_mode"].(string); ok {
			switch cacheMode {
			case "off":
				opts.CacheMode = vfscommon.CacheModeOff
			case "minimal":
				opts.CacheMode = vfscommon.CacheModeMinimal
			case "writes":
				opts.CacheMode = vfscommon.CacheModeWrites
			case "full":
				opts.CacheMode = vfscommon.CacheModeFull
			}
		}

		if chunkSize, ok := customSettings["chunk_size"].(float64); ok {
			opts.ChunkSize = fs.SizeSuffix(chunkSize)
		}
	}

	return opts
}

// LookupRemote queries PocketBase for remote configuration by hostname
func LookupRemote(app core.App, hostname string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter(
		"rclone_remotes",
		"name = {:hostname} && enabled = true",
		map[string]interface{}{"hostname": hostname},
	)

	if err != nil {
		return nil, fmt.Errorf("remote '%s' not found or disabled: %w", hostname, err)
	}

	return record, nil
}

// Stat returns metadata for a remote path
func (rm *RcloneManager) Stat(v *vfs.VFS, remotePath string) (vfs.Node, error) {
	return v.Stat(remotePath)
}

// FetchFile downloads a file from remote to local cache using VFS
func (rm *RcloneManager) FetchFile(v *vfs.VFS, remotePath string, localCachePath string) error {
	log.Printf("[RCLONE] Fetching file: %s -> %s", remotePath, localCachePath)

	// Ensure local cache directory exists
	if err := os.MkdirAll(filepath.Dir(localCachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open remote file via VFS
	remoteFile, err := v.OpenFile(remotePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Create local file
	localFile, err := os.Create(localCachePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy contents (VFS handles caching internally with CacheModeFull)
	written, err := localFile.ReadFrom(remoteFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	log.Printf("[RCLONE] Fetched %d bytes successfully", written)
	return nil
}

// IndexDirectory creates a SQLite database containing the listing of a remote directory
// This matches the schema and table name used by mksqlite's filesystem converter
func (rm *RcloneManager) IndexDirectory(v *vfs.VFS, remotePath string, localCachePath string) error {
	log.Printf("[RCLONE] Indexing directory: %s -> %s", remotePath, localCachePath)

	// Ensure local cache directory exists
	if err := os.MkdirAll(filepath.Dir(localCachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Open/Create SQLite database
	// We use the same name "tb0" and same columns as mksqlite filesystem converter
	db, err := sql.Open("sqlite3", localCachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache database: %w", err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tb0 (
		path TEXT,
		name TEXT,
		size TEXT,
		extension TEXT,
		mod_time TEXT,
		is_dir TEXT
	)`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Clear existing data (if any)
	_, err = db.Exec("DELETE FROM tb0")
	if err != nil {
		return fmt.Errorf("failed to clear table: %w", err)
	}

	// Get directory node
	node, err := v.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote directory: %w", err)
	}

	dir, ok := node.(*vfs.Dir)
	if !ok {
		return fmt.Errorf("path is not a directory")
	}

	// Read entries
	entries, err := dir.ReadDirAll()
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Prepare insert statement
	stmt, err := db.Prepare("INSERT INTO tb0 (path, name, size, extension, mod_time, is_dir) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert entries
	for _, entry := range entries {
		name := entry.Name()
		relPath := filepath.Join(remotePath, name)
		size := fmt.Sprintf("%d", entry.Size())
		ext := filepath.Ext(name)
		modTime := entry.ModTime().Format(time.RFC3339)
		isDir := "0"
		if entry.IsDir() {
			isDir = "1"
			size = "0" // Traditionally 0 or entry count for dirs in some tools, mksqlite uses size
		}

		_, err = stmt.Exec(relPath, name, size, ext, modTime, isDir)
		if err != nil {
			log.Printf("[RCLONE] Warning: failed to index entry %s: %v", name, err)
		}
	}

	log.Printf("[RCLONE] Indexed %d entries successfully", len(entries))
	return nil
}

// GetConfigHash returns the hash of a remote record's configuration
func (rm *RcloneManager) GetConfigHash(remoteRecord *core.Record) string {
	configData := remoteRecord.Get("config")

	var config map[string]interface{}
	switch v := configData.(type) {
	case map[string]interface{}:
		config = v
	case string:
		if err := json.Unmarshal([]byte(v), &config); err != nil {
			log.Printf("[RCLONE] Warning: failed to parse config for hashing: %v", err)
			return ""
		}
	default:
		return ""
	}

	return generateVFSHash(config)
}
