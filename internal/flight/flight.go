package flight

// deliberately import everything here as the primary location of orchestration.
import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// getDataDirectory determines the appropriate data directory
// Priority: 1. Conventional location if exists, 2. Current directory
func getDataDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./pb_data" // Fallback to current directory
	}

	var conventionalPath string
	if runtime.GOOS == "darwin" {
		conventionalPath = filepath.Join(homeDir, "Library", "Application Support", "Flight3")
	} else {
		conventionalPath = filepath.Join(homeDir, ".local", "share", "flight3")
	}

	// Check if conventional path exists and has pb_data subdirectory
	pbDataPath := filepath.Join(conventionalPath, "pb_data")
	if _, err := os.Stat(pbDataPath); err == nil {
		return pbDataPath
	}

	// Fallback to current directory
	return "./pb_data"
}

// Global SQLiter server instance
var globalSQLiterServer *sqliter.Server

// SetSQLiterServer sets the global SQLiter server instance
func SetSQLiterServer(server *sqliter.Server) {
	globalSQLiterServer = server
}

// GetSQLiterServer returns the global SQLiter server instance
func GetSQLiterServer() *sqliter.Server {
	return globalSQLiterServer
}

func Flight() {

	// Default to "serve" command if no arguments are provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	// Detect if we are serving and checked for start URL
	isServe := false
	httpAddr := ""
	startRequestURL := ""

	var newArgs []string
	newArgs = append(newArgs, os.Args[0])

	// First pass: detecting 'serve' or help to avoid interfering with standard behavior
	for _, arg := range os.Args {
		if arg == "serve" {
			isServe = true
		}
		if arg == "--help" || arg == "-h" {
			// If help is requested, just pass everything through to PocketBase
			// so it prints the standard help text.
			return
		}
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Handle flags
		if strings.HasPrefix(arg, "-") {
			newArgs = append(newArgs, arg)

			// Check for flags that take arguments
			// Handle --http separate arg
			if arg == "--http" && i+1 < len(os.Args) {
				httpAddr = os.Args[i+1]
				newArgs = append(newArgs, httpAddr)
				i++ // Consume next
				continue
			}
			// Handle --http=...
			if strings.HasPrefix(arg, "--http=") {
				httpAddr = strings.TrimPrefix(arg, "--http=")
			}
			continue
		}

		// Handle known commands
		if arg == "serve" || arg == "migrate" || arg == "admin" || arg == "upgrade" {
			newArgs = append(newArgs, arg)
			continue
		}

		// If it's not a flag and not a known command, assume it's our start URL
		// This captures "https://..." or "/path/to/file" or even "file.db"
		if startRequestURL == "" {
			startRequestURL = arg
			isServe = true
			continue // Do NOT add to newArgs
		}

		// If we already have a start URL, treat subsequent args as potential valid args (weird, but safe)
		newArgs = append(newArgs, arg)
	}

	// Ensure "serve" command is present if we decided we are serving and it's missing
	// (e.g. user ran `flight http://url`)
	hasServe := false
	for _, arg := range newArgs {
		if arg == "serve" {
			hasServe = true
			break
		}
	}
	if isServe && !hasServe {
		// Insert "serve" after program name
		newArgs = append(newArgs[:1], append([]string{"serve"}, newArgs[1:]...)...)
	}

	os.Args = newArgs

	// If serving but no --http address specified, find a random high port on [::1]
	// This makes it enjoyable on macOS as requested.
	if isServe && httpAddr == "" {
		l, err := net.Listen("tcp", "[::1]:0")
		if err == nil {
			httpAddr = l.Addr().String()
			l.Close()
			os.Args = append(os.Args, "--http="+httpAddr)
		}
	}

	// Determine data directory
	// Priority: 1. Conventional location if exists, 2. Current directory
	dataDir := getDataDirectory()

	// Create PocketBase app with custom data directory
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: dataDir,
	})

	log.Printf("Using data directory: %s", app.DataDir())

	// Initialize SQLiter server
	// SQLiter handles everything from ColumnSetPath → Query
	sqliterConfig := sqliter.DefaultConfig()
	sqliterConfig.ServeFolder = filepath.Join(app.DataDir(), "cache")
	sqliterConfig.RemoteFetcher = func(urlStr string, destFolder string) (string, error) {
		log.Printf("[FLIGHT] Fetching remote file: %s", urlStr)
		// 1. Determine destination path
		// Use SHA256 of URL as filename to avoid collisions and length issues
		hash := sha256.Sum256([]byte(urlStr))
		hashStr := hex.EncodeToString(hash[:])[:16]

		// Try to get extension from URL
		u, err := url.Parse(urlStr)
		ext := ""
		if err == nil {
			ext = filepath.Ext(u.Path)
		}
		if ext == "" {
			ext = ".db" // Fallback
		}

		fileName := hashStr + ext
		destPath := filepath.Join(destFolder, fileName)

		// 2. Check if exists
		if _, err := os.Stat(destPath); err == nil {
			log.Printf("[FLIGHT] Cache hit: %s", destPath)
			return destPath, nil
		}

		// 3. Download
		log.Printf("[FLIGHT] Downloading to: %s", destPath)
		resp, err := http.Get(urlStr)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", err
		}

		out, err := os.Create(destPath)
		if err != nil {
			return "", err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", err
		}

		return destPath, nil
	}
	sqliterConfig.Verbose = true
	sqliterConfig.BaseURL = "/sqliter/"
	sqliterServer := sqliter.NewServer(sqliterConfig)
	SetSQLiterServer(sqliterServer) // Make it globally accessible

	log.Printf("[FLIGHT] SQLiter server initialized, serving from: %s", sqliterConfig.ServeFolder)

	// Initialize rclone early (doesn't need database)
	cacheDir := filepath.Join(app.DataDir(), "cache")
	if err := InitRclone(cacheDir); err != nil {
		log.Fatalf("Error initializing rclone: %v", err)
	}
	log.Printf("Rclone manager initialized with cache dir: %s", cacheDir)

	// OnServe: Setup collections when server starts (database is ready by then)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Ensure collections exist (database is ready now)
		if err := EnsureCollections(se.App); err != nil {
			log.Printf("Error ensuring collections: %v", err)
			return err
		}
		log.Printf("PocketBase collections ensured")

		// Ensure superuser exists
		if err := EnsureSuperUser(se.App, "admin@example.com", "password123"); err != nil {
			log.Printf("Error ensuring superuser: %v", err)
		}

		// Configure centralized routing
		// Configure centralized routing
		ConfigureRouting(se.App, sqliterServer)

		// Launch Chrome on macOS if we are serving
		if isServe && httpAddr != "" && runtime.GOOS == "darwin" {
			go func() {
				// Give the server a moment to bind and start listening
				time.Sleep(1 * time.Second)
				// Open the URL directly
				targetURL := "http://" + httpAddr + "/" // Start at root

				if startRequestURL != "" {
					// We want to open http://localhost:port/<startRequestURL>
					// Be careful with slashes
					targetURL += strings.TrimPrefix(startRequestURL, "/")
				}

				log.Printf("[SILICON] Enjoying Flight3: Launching Google Chrome to %s", targetURL)
				err := exec.Command("open", "-a", "Google Chrome", targetURL).Start()
				if err != nil {
					log.Printf("[SILICON] Failed to launch Google Chrome: %v (falling back to default browser)", err)
					exec.Command("open", targetURL).Start()
				}
			}()
		}

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
