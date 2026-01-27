package flight

// deliberately import everything here as the primary location of orchestration.
import (
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func Flight() {

	// Default to "serve" command if no arguments are provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	app := pocketbase.New()

	// Initialize SQLiter with Embedded Templates
	// We use the templates embedded in the sqliter library.
	tpl, err := sqliter.GetEmbeddedTemplates()
	if err != nil {
		log.Fatal("Failed to load embedded templates from sqliter:", err)
	}

	// Initialize TableWriter with embedded templates
	tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

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

			// Pass to BanquetHandler
			return HandleBanquet(e, tw, tpl, true)
		}

		// Register root path handler
		se.Router.Any("/", banquetHandler)

		// Register catch-all route for all other paths
		se.Router.Any("/*", banquetHandler)

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
