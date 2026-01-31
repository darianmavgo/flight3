package flight

// deliberately import everything here as the primary location of orchestration.
import (
	"log"
	"net"
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

func Flight() {

	// Default to "serve" command if no arguments are provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	// Detect if we are serving and if --http is already set
	isServe := false
	httpAddr := ""
	for i, arg := range os.Args {
		if arg == "serve" {
			isServe = true
		}
		if strings.HasPrefix(arg, "--http=") {
			httpAddr = strings.TrimPrefix(arg, "--http=")
		} else if arg == "--http" && i+1 < len(os.Args) {
			httpAddr = os.Args[i+1]
		}
	}

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
	sqliterConfig.Verbose = true
	sqliterServer := sqliter.NewServer(sqliterConfig)

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

		// Handler function for banquet requests
		banquetHandler := func(e *core.RequestEvent) error {
			path := e.Request.URL.Path

			// Don't intercept PocketBase paths
			if strings.HasPrefix(path, "/_/") || strings.HasPrefix(path, "/api/") {
				return e.Next()
			}

			// Don't intercept common web standards
			if path == "/favicon.ico" || path == "/robots.txt" || path == "/sitemap.xml" {
				return e.Next()
			}

			// Pass to BanquetHandler and handle errors
			// Flight3 handles: Scheme → DataSetPath
			err := HandleBanquet(e, true)
			if err != nil {
				return HandleBanquetError(e, err)
			}
			return nil
		}

		// Mount SQLiter for data rendering
		// SQLiter handles: ColumnSetPath → Query
		se.Router.Any("/_/data", func(e *core.RequestEvent) error {
			sqliterServer.ServeHTTP(e.Response, e.Request)
			return nil
		})
		se.Router.Any("/_/data/*", func(e *core.RequestEvent) error {
			sqliterServer.ServeHTTP(e.Response, e.Request)
			return nil
		})

		log.Printf("[FLIGHT] SQLiter mounted at /_/data/")

		// Rclone config UI and API routes (must be before catch-all)
		se.Router.GET("/_/rclone_config", HandleRcloneConfigUI)
		se.Router.GET("/_/rclone_config/api/providers", HandleListProviders)
		se.Router.GET("/_/rclone_config/api/provider/{type}", HandleGetProviderSchema)
		se.Router.GET("/_/rclone_config/api/remotes", HandleListRemotes)
		se.Router.POST("/_/rclone_config/api/remotes", HandleCreateRemote)
		se.Router.PUT("/_/rclone_config/api/remotes/{id}", HandleUpdateRemote)
		se.Router.DELETE("/_/rclone_config/api/remotes/{id}", HandleDeleteRemote)
		se.Router.POST("/_/rclone_config/api/test", HandleTestRemote)

		// Register auto-login handler
		se.Router.GET("/api/auto_login", HandleAutoLogin)

		// Register root path handler
		se.Router.Any("/", banquetHandler)

		// Register catch-all route for all other paths
		se.Router.Any("/*", banquetHandler)

		// Launch Chrome on macOS if we are serving
		if isServe && httpAddr != "" && runtime.GOOS == "darwin" {
			go func() {
				// Give the server a moment to bind and start listening
				time.Sleep(1 * time.Second)
				// Use auto-login route to ensure user is authenticated
				url := "http://" + httpAddr + "/api/auto_login"
				log.Printf("[SILICON] Enjoying Flight3: Launching Google Chrome to %s", url)
				err := exec.Command("open", "-a", "Google Chrome", url).Start()
				if err != nil {
					log.Printf("[SILICON] Failed to launch Google Chrome: %v (falling back to default browser)", err)
					exec.Command("open", url).Start()
				}
			}()
		}

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
