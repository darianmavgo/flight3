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
		ConfigureRouting(se.App)

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
